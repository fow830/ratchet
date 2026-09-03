package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fow830/ratchet/pkg/antidrift"
	"github.com/fow830/ratchet/pkg/contracts"
	"github.com/fow830/ratchet/pkg/generate"
	gha "github.com/fow830/ratchet/pkg/github"
	"github.com/fow830/ratchet/pkg/hooks"
	"github.com/fow830/ratchet/pkg/report"
	"github.com/fow830/ratchet/pkg/skills"
	"github.com/fow830/ratchet/pkg/tokens"
)

// Set by goreleaser ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const commandTimeout = 2 * time.Minute

type rootFlags struct {
	config  string
	jsonOut bool
	verbose bool
	dryRun  bool
}

func main() {
	os.Exit(run())
}

func run() int {
	root := newRootCommand()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return exitCode(err)
	}
	return exitOK
}

func newRootCommand() *cobra.Command {
	var rf rootFlags
	tool := tokens.ToolName
	root := &cobra.Command{
		Use:     tool,
		Short:   "Deterministic AI-native anti-drift framework for Go",
		Version: version,
		Long: fmt.Sprintf(`%s enforces Zero Architectural Regression (anti-drift) for Go repositories.

Exit codes:
  0  success — no architecture violations or drift
  1  violations — layer isolation or contract drift detected
  2  error — invalid flags, I/O, parse, or other system failures`, tool),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetVersionTemplate(fmt.Sprintf("%s {{.Version}} (commit=%s date=%s)\n", tool, commit, date))
	root.PersistentFlags().StringVar(&rf.config, "config", "", "path to "+tokens.ConfigFileName+" (default: ./"+tokens.ConfigFileName+")")
	root.PersistentFlags().BoolVar(&rf.jsonOut, "json", false, "alias for --format="+report.FormatJSON+" on check")
	root.PersistentFlags().BoolVarP(&rf.verbose, "verbose", "v", false, "print diagnostic details to stderr")
	root.PersistentFlags().BoolVar(&rf.dryRun, "dry-run", false, "show actions without writing files")

	root.AddCommand(
		newInitCmd(&rf),
		newCheckCmd(&rf),
		newGenCmd(&rf),
		newInitCICmd(&rf),
		newInitHooksCmd(&rf),
	)
	registerExtraCommands(root, &rf)
	root.AddCommand(newCompletionCmd())
	return root
}

func toolMsg(cmd, msg string) string {
	return tokens.ToolName + " " + cmd + ": " + msg
}

func newInitCmd(rf *rootFlags) *cobra.Command {
	var preset, profile string
	var withContracts bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: fmt.Sprintf("Bootstrap %s, %s, Claude skill, and lock file", tokens.CursorRules, tokens.ConfigFileName),
		Long: fmt.Sprintf(`Create default SSOT config and agent rule artifacts in the current module.

Writes %s, %s, %s, and %s.
Use --preset=clean|vitek|hex and --with-contracts for superproduction scaffold.
Use --dry-run to preview without writing.`,
			tokens.ConfigFileName, tokens.CursorRules, tokens.ClaudeSkillRel, tokens.LockFileName),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := commandContext()
			defer cancel()
			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			module, err := tokens.ModuleFromGoMod(ctx, wd)
			if err != nil {
				return systemErr(err)
			}
			cfg := tokens.PresetConfig(preset, module)
			cfg.SchemaVersion = tokens.CurrentSchemaVersion
			if profile != "" {
				cfg.Profile = profile
			} else {
				cfg.Profile = tokens.ProfileStandard
			}
			if rf.dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), toolMsg("init (dry-run)", fmt.Sprintf(
					"would write %s, %s, %s, %s preset=%s profile=%s",
					tokens.CursorRules, tokens.ConfigFileName, tokens.ClaudeSkillRel, tokens.LockFileName, preset, cfg.Profile,
				)))
				return nil
			}
			if err := tokens.Save(ctx, wd, cfg); err != nil {
				return systemErr(err)
			}
			if err := skills.NewGenerator(cfg).Generate(wd); err != nil {
				return systemErr(err)
			}
			if withContracts {
				dir := filepath.Join(wd, cfg.ContractsRoot())
				if _, err := contracts.ScaffoldSuite(dir); err != nil {
					return systemErr(err)
				}
				_, err := contracts.Scaffold(wd, contracts.ScaffoldOpts{ID: "ARCH-001", Title: "Architecture layer isolation"})
				if err != nil {
					return systemErr(err)
				}
			}
			if err := antidrift.New(wd).LockAll(ctx, cfg); err != nil {
				return systemErr(err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), toolMsg("init", fmt.Sprintf(
				"wrote %s, %s, %s, %s",
				tokens.CursorRules, tokens.ConfigFileName, tokens.ClaudeSkillRel, tokens.LockFileName,
			)))
			return nil
		},
	}
	cmd.Flags().StringVar(&preset, "preset", tokens.PresetClean, "architecture preset: clean|vitek|hex")
	cmd.Flags().StringVar(&profile, "profile", tokens.ProfileStandard, "default check profile")
	cmd.Flags().BoolVar(&withContracts, "with-contracts", false, "scaffold "+tokens.ContractsDirDefault)
	return cmd
}

func newCheckCmd(rf *rootFlags) *cobra.Command {
	var format, profile string
	var workspaceMode bool
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run profile-based architecture and quality gates",
		Long: fmt.Sprintf(`Analyze Go packages for illegal layer imports, contract drift, and profile gates.

Profiles (--profile):
  %s, %s, %s, %s, %s, %s

Output formats (--format / -f):
  %s   ANSI-colored terminal output (default; alias: %s)
  %s   structured JSON on stdout
  %s  SARIF 2.1.0 for GitHub Code Scanning
  %s    dry RULE_VIOLATION blocks for agents

Exit 1 when violations or drift are found; exit 2 on system/parse errors.`,
			tokens.ProfileMinimal, tokens.ProfileStandard, tokens.ProfileService, tokens.ProfileAPI, tokens.ProfileStrict, tokens.ProfileParanoid,
			report.FormatText, report.FormatHuman, report.FormatJSON, report.FormatSARIF, report.FormatLLM),
		RunE: func(cmd *cobra.Command, args []string) error {
			if rf.jsonOut {
				format = report.FormatJSON
			}
			nfmt, err := report.NormalizeFormat(format)
			if err != nil {
				return systemErr(err)
			}
			return runCheck(cmd, rf, nfmt, profile, workspaceMode)
		},
	}
	cmd.Flags().StringVarP(&format, "format", "f", report.FormatText, "output format: "+report.SupportedFormats)
	cmd.Flags().StringVar(&profile, "profile", "", "gate profile (overrides ratchet.json profile)")
	cmd.Flags().BoolVar(&workspaceMode, "workspace", false, "check all modules in go.work monorepo")
	return cmd
}

func newGenCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "gen",
		Short: "Generate agent skill rules and re-lock contracts",
		Long: fmt.Sprintf(`Regenerate %s / Claude skill from SSOT and refresh %s.

If %s is missing, creates a default config from %s first.
Use --dry-run to preview without writing.`,
			tokens.CursorRules, tokens.LockFileName, tokens.ConfigFileName, tokens.GoModFileName),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := commandContext()
			defer cancel()
			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			cfg, _, err := loadConfig(ctx, wd, rf.config)
			if err != nil {
				module, merr := tokens.ModuleFromGoMod(ctx, wd)
				if merr != nil {
					return systemErr(err)
				}
				cfg = tokens.DefaultConfig(module)
				if !rf.dryRun {
					if err := tokens.Save(ctx, wd, cfg); err != nil {
						return systemErr(err)
					}
				}
			}
			if rf.dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), toolMsg("gen (dry-run)", "would regenerate contracts and lock file"))
				return nil
			}
			if err := skills.NewGenerator(cfg).Generate(wd); err != nil {
				return systemErr(err)
			}
			if _, err := generate.WriteAll(ctx, wd, cfg); err != nil {
				return systemErr(err)
			}
			if err := antidrift.New(wd).LockAll(ctx, cfg); err != nil {
				return systemErr(err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), toolMsg("gen", "regenerated contracts, tokens, and lock file"))
			return nil
		},
	}
}

func newInitCICmd(rf *rootFlags) *cobra.Command {
	var protectMain bool
	cmd := &cobra.Command{
		Use:   "init-ci",
		Short: "Generate GitHub Actions workflow for hard CI enforcement",
		Long: fmt.Sprintf(`Write %s (go test/vet + %s check + goreleaser on tags).

--protect-main attempts to enable required status checks via gh api.
Private personal repos may require GitHub Pro for branch protection.`,
			gha.WorkflowRel, tokens.ToolName),
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			if rf.dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), toolMsg("init-ci (dry-run)", "would write "+gha.WorkflowRel))
				return nil
			}
			path, err := gha.WriteWorkflow(wd)
			if err != nil {
				return systemErr(err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), toolMsg("init-ci", "wrote "+path))
			if !protectMain {
				return nil
			}
			if err := enableProtectMain(cmd, wd); err != nil {
				return systemErr(err)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&protectMain, "protect-main", false, "enable GitHub required status checks on main via gh api")
	return cmd
}

func enableProtectMain(cmd *cobra.Command, wd string) error {
	owner, repo, err := resolveOwnerRepo(wd)
	if err != nil {
		return err
	}
	client := gha.NewClient()
	manual := gha.ProtectMainCommand(owner, repo)
	var stderr bytes.Buffer
	if err := client.ProtectMain(owner, repo, cmd.OutOrStdout(), &stderr); err != nil {
		msg := stderr.String()
		fmt.Fprint(cmd.ErrOrStderr(), msg)
		if strings.Contains(err.Error(), gha.GhBinary+" not installed") {
			fmt.Fprintln(cmd.OutOrStdout(), gha.GhBinary+" CLI not found; run:")
			fmt.Fprintln(cmd.OutOrStdout(), manual)
			return err
		}
		if gha.IsProtectionUnavailable(msg) {
			return fmt.Errorf("protect-main unavailable: private repos need GitHub Pro (or make the repo public); CI workflow is still the hard gate via exit code 1")
		}
		fmt.Fprintln(cmd.OutOrStdout(), "run manually:")
		fmt.Fprintln(cmd.OutOrStdout(), manual)
		return fmt.Errorf("protect-main: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), toolMsg("init-ci", fmt.Sprintf(
		"required status check %q enabled on %s/%s main", gha.StatusCheckName, owner, repo,
	)))
	return nil
}

func newInitHooksCmd(rf *rootFlags) *cobra.Command {
	var lrtVerify bool
	cmd := &cobra.Command{
		Use:   "init-hooks",
		Short: "Install local pre-commit hook (soft friction)",
		Long: fmt.Sprintf(`Install %s that runs %s check --format=%s.

This is soft friction only; CI exit code 1 remains the hard constraint.`,
			tokens.PreCommitRel, tokens.ToolName, report.FormatLLM),
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			if rf.dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), toolMsg("init-hooks (dry-run)", "would install "+tokens.PreCommitRel))
				return nil
			}
			path, err := hooks.NewInstaller().Install(wd, hooks.InstallOptions{LRTVerify: lrtVerify})
			if err != nil {
				return systemErr(err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), toolMsg("init-hooks", "installed "+path))
			return nil
		},
	}
	cmd.Flags().BoolVar(&lrtVerify, "lrt-verify", false, "require LRT-VERIFY in commit message when contracts change")
	return cmd
}

func loadConfig(ctx context.Context, wd, configFlag string) (tokens.Config, string, error) {
	path := configFlag
	if path == "" {
		path = filepath.Join(wd, tokens.ConfigFileName)
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(wd, path)
	}
	cfg, err := tokens.LoadFile(ctx, path)
	if err != nil {
		return tokens.Config{}, "", err
	}
	return cfg, path, nil
}

func resolveOwnerRepo(wd string) (string, string, error) {
	out, err := exec.Command("git", "-C", wd, "remote", "get-url", "origin").Output()
	if err != nil {
		return "", "", fmt.Errorf("resolve origin remote: %w", err)
	}
	return gha.ParseOwnerRepo(string(out))
}

func commandContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), commandTimeout)
}
