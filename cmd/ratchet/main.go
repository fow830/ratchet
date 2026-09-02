package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fow830/ratchet/pkg/antidrift"
	"github.com/fow830/ratchet/pkg/fitness"
	"github.com/fow830/ratchet/pkg/github"
	"github.com/fow830/ratchet/pkg/hooks"
	"github.com/fow830/ratchet/pkg/report"
	"github.com/fow830/ratchet/pkg/skills"
	"github.com/fow830/ratchet/pkg/tokens"
)

const commandTimeout = 2 * time.Minute

type rootFlags struct {
	config  string
	jsonOut bool
	verbose bool
	dryRun  bool
}

func main() {
	var rf rootFlags
	root := &cobra.Command{
		Use:   "ratchet",
		Short: "Deterministic AI-native anti-drift framework for Go",
		Long: `ratchet enforces Zero Architectural Regression (anti-drift) for Go repositories.

Exit codes:
  0  success — no architecture violations or drift
  1  violations — layer isolation or contract drift detected
  2  error — invalid flags, I/O, parse, or other system failures`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&rf.config, "config", "", "path to ratchet.json (default: ./ratchet.json)")
	root.PersistentFlags().BoolVar(&rf.jsonOut, "json", false, "emit machine-readable JSON on stdout where applicable")
	root.PersistentFlags().BoolVarP(&rf.verbose, "verbose", "v", false, "print diagnostic details to stderr")
	root.PersistentFlags().BoolVar(&rf.dryRun, "dry-run", false, "show actions without writing files")

	root.AddCommand(
		newInitCmd(&rf),
		newCheckCmd(&rf),
		newGenCmd(&rf),
		newInitCICmd(&rf),
		newInitHooksCmd(&rf),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(exitCode(err))
	}
}

func newInitCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Bootstrap .cursorrules, ratchet.json, Claude skill, and lock file",
		Long: `Create default SSOT config and agent rule artifacts in the current module.

Writes ratchet.json, .cursorrules, .claude/skills/ratchet.md, and ratchet.lock.
Use --dry-run to preview without writing.`,
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
			cfg := tokens.DefaultConfig(module)
			if rf.dryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "ratchet init (dry-run): would write %s, %s, %s, %s\n",
					tokens.CursorRules, tokens.ConfigFileName, tokens.ClaudeSkillRel, tokens.LockFileName)
				return nil
			}
			if err := tokens.Save(ctx, wd, cfg); err != nil {
				return systemErr(err)
			}
			if err := skills.NewGenerator(cfg).Generate(wd); err != nil {
				return systemErr(err)
			}
			if err := antidrift.New(wd).Lock(ctx, cfg.ContractFiles); err != nil {
				return systemErr(err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ratchet init: wrote %s, %s, %s, %s\n",
				tokens.CursorRules, tokens.ConfigFileName, tokens.ClaudeSkillRel, tokens.LockFileName)
			return nil
		},
	}
}

func newCheckCmd(rf *rootFlags) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run AST fitness functions and anti-drift checks",
		Long: `Analyze Go packages for illegal layer imports and verify locked contract hashes.

Output:
  --format=human (default)  human-readable lines on stderr
  --format=llm              structured RULE_VIOLATION blocks for agents
  --json                    JSON summary on stdout (violations + drift)

Exit 1 when violations or drift are found; exit 2 on system/parse errors.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := commandContext()
			defer cancel()

			format = strings.ToLower(strings.TrimSpace(format))
			switch format {
			case report.FormatHuman, report.FormatLLM:
			default:
				return systemErr(fmt.Errorf("unsupported --format %q (want human|llm)", format))
			}

			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			cfg, cfgPath, err := loadConfig(ctx, wd, rf.config)
			if err != nil {
				return systemErr(err)
			}
			if rf.verbose {
				fmt.Fprintf(cmd.ErrOrStderr(), "config=%s module=%s\n", cfgPath, cfg.Module)
			}

			violations, err := fitness.NewAnalyzer(cfg).Analyze(ctx, wd)
			if err != nil {
				return systemErr(err)
			}
			if violations == nil {
				violations = []fitness.Violation{}
			}

			eng := antidrift.New(wd)
			eng.ConfigPath = cfgPath
			diff, err := eng.Verify(ctx)
			if err != nil {
				return systemErr(err)
			}
			normalizeDiff(&diff)

			failed := len(violations) > 0 || !diff.OK()
			errOut := cmd.ErrOrStderr()
			out := cmd.OutOrStdout()

			if rf.jsonOut {
				payload := checkJSON{
					OK:         !failed,
					Violations: violations,
					Drift:      diff,
				}
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				if err := enc.Encode(payload); err != nil {
					return systemErr(err)
				}
			} else {
				for _, v := range violations {
					if format == report.FormatLLM {
						fmt.Fprintln(errOut, report.FormatLLMViolation(v))
						fmt.Fprintln(errOut)
						continue
					}
					fmt.Fprintln(errOut, report.FormatHumanViolation(v))
				}
				if !diff.OK() {
					if format == report.FormatLLM {
						fmt.Fprintln(errOut, report.FormatLLMDiff(diff))
					} else {
						fmt.Fprint(errOut, report.FormatHumanDiff(diff))
					}
				}
			}

			if failed {
				return violationErr(fmt.Errorf("ratchet check failed"))
			}
			if !rf.jsonOut {
				fmt.Fprintln(out, "ratchet check: ok")
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&format, "format", "f", report.FormatHuman, "output format: human|llm")
	return cmd
}

func newGenCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "gen",
		Short: "Generate agent skill rules and re-lock contracts",
		Long: `Regenerate .cursorrules / Claude skill from SSOT and refresh ratchet.lock.

If ratchet.json is missing, creates a default config from go.mod first.
Use --dry-run to preview without writing.`,
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
				fmt.Fprintln(cmd.OutOrStdout(), "ratchet gen (dry-run): would regenerate contracts and lock file")
				return nil
			}
			if err := skills.NewGenerator(cfg).Generate(wd); err != nil {
				return systemErr(err)
			}
			if err := antidrift.New(wd).Lock(ctx, cfg.ContractFiles); err != nil {
				return systemErr(err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ratchet gen: regenerated contracts and lock file")
			return nil
		},
	}
}

func newInitCICmd(rf *rootFlags) *cobra.Command {
	var protectMain bool
	cmd := &cobra.Command{
		Use:   "init-ci",
		Short: "Generate GitHub Actions workflow for hard CI enforcement",
		Long: `Write .github/workflows/ratchet.yml (go test + ratchet check --format=llm).

--protect-main attempts to enable required status checks via gh api.
Private personal repos may require GitHub Pro for branch protection.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			if rf.dryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "ratchet init-ci (dry-run): would write %s\n", gha.WorkflowRel)
				return nil
			}
			path, err := gha.WriteWorkflow(wd)
			if err != nil {
				return systemErr(err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ratchet init-ci: wrote %s\n", path)
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
	ghPath, lookErr := exec.LookPath("gh")
	argsAPI, body := gha.ProtectMainArgs(owner, repo)
	manual := gha.ProtectMainCommand(owner, repo)
	if lookErr != nil {
		fmt.Fprintln(cmd.OutOrStdout(), "gh CLI not found; run:")
		fmt.Fprintln(cmd.OutOrStdout(), manual)
		return fmt.Errorf("protect-main: gh not installed")
	}

	var stderr bytes.Buffer
	c := exec.Command(ghPath, argsAPI...)
	c.Stdin = bytes.NewBufferString(body)
	c.Stdout = cmd.OutOrStdout()
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		msg := stderr.String()
		fmt.Fprint(cmd.ErrOrStderr(), msg)
		if gha.IsProtectionUnavailable(msg) {
			return fmt.Errorf("protect-main unavailable: private repos need GitHub Pro (or make the repo public); CI workflow is still the hard gate via exit code 1")
		}
		fmt.Fprintln(cmd.OutOrStdout(), "run manually:")
		fmt.Fprintln(cmd.OutOrStdout(), manual)
		return fmt.Errorf("protect-main: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "ratchet init-ci: required status check %q enabled on %s/%s main\n", gha.StatusCheckName, owner, repo)
	return nil
}

func newInitHooksCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "init-hooks",
		Short: "Install local pre-commit hook (soft friction)",
		Long: `Install .git/hooks/pre-commit that runs ratchet check --format=llm.

This is soft friction only; CI exit code 1 remains the hard constraint.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			if rf.dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), "ratchet init-hooks (dry-run): would install .git/hooks/pre-commit")
				return nil
			}
			path, err := hooks.Install(wd)
			if err != nil {
				return systemErr(err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ratchet init-hooks: installed %s\n", path)
			return nil
		},
	}
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

func normalizeDiff(d *antidrift.Diff) {
	if d.Changed == nil {
		d.Changed = []antidrift.ChangedFile{}
	}
	if d.Missing == nil {
		d.Missing = []string{}
	}
	if d.Extra == nil {
		d.Extra = []string{}
	}
}

type checkJSON struct {
	OK         bool                `json:"ok"`
	Violations []fitness.Violation `json:"violations"`
	Drift      antidrift.Diff      `json:"drift"`
}
