package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fow830/ratchet/pkg/antidrift"
	"github.com/fow830/ratchet/pkg/fitness"
	githubci "github.com/fow830/ratchet/pkg/github"
	"github.com/fow830/ratchet/pkg/hooks"
	"github.com/fow830/ratchet/pkg/report"
	"github.com/fow830/ratchet/pkg/skills"
	"github.com/fow830/ratchet/pkg/tokens"
)

func main() {
	root := &cobra.Command{
		Use:           "ratchet",
		Short:         "Deterministic AI-native anti-drift framework for Go",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		newInitCmd(),
		newCheckCmd(),
		newGenCmd(),
		newInitCICmd(),
		newInitHooksCmd(),
	)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Bootstrap .cursorrules, ratchet.json, and lock file",
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			module, err := tokens.ModuleFromGoMod(wd)
			if err != nil {
				return err
			}
			cfg := tokens.DefaultConfig(module)
			if err := tokens.Save(wd, cfg); err != nil {
				return err
			}
			if err := skills.NewGenerator(cfg).Generate(wd); err != nil {
				return err
			}
			if err := antidrift.New(wd).Lock(cfg.ContractFiles); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ratchet init: wrote %s, %s, %s, %s\n",
				tokens.CursorRules, tokens.ConfigFileName, tokens.ClaudeSkillRel, tokens.LockFileName)
			return nil
		},
	}
}

func newCheckCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run AST fitness functions and anti-drift checks",
		RunE: func(cmd *cobra.Command, args []string) error {
			format = strings.ToLower(strings.TrimSpace(format))
			switch format {
			case report.FormatHuman, report.FormatLLM:
			default:
				return fmt.Errorf("unsupported --format %q (want human|llm)", format)
			}

			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			cfg, err := tokens.Load(wd)
			if err != nil {
				return err
			}

			violations, err := fitness.NewAnalyzer(cfg).Analyze(wd)
			if err != nil {
				return err
			}

			errOut := cmd.ErrOrStderr()
			for _, v := range violations {
				if format == report.FormatLLM {
					fmt.Fprintln(errOut, report.FormatLLMViolation(v))
					fmt.Fprintln(errOut)
					continue
				}
				fmt.Fprintln(errOut, report.FormatHumanViolation(v))
			}

			diff, err := antidrift.New(wd).Verify()
			if err != nil {
				return err
			}
			if !diff.OK() {
				if format == report.FormatLLM {
					fmt.Fprintln(errOut, report.FormatLLMDiff(diff))
				} else {
					fmt.Fprint(errOut, report.FormatHumanDiff(diff))
				}
			}

			if len(violations) > 0 || !diff.OK() {
				return fmt.Errorf("ratchet check failed")
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ratchet check: ok")
			return nil
		},
	}
	cmd.Flags().StringVarP(&format, "format", "f", report.FormatHuman, "output format: human|llm")
	return cmd
}

func newGenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gen",
		Short: "Generate agent skill rules and re-lock contracts",
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			cfg, err := tokens.Load(wd)
			if err != nil {
				module, merr := tokens.ModuleFromGoMod(wd)
				if merr != nil {
					return err
				}
				cfg = tokens.DefaultConfig(module)
				if err := tokens.Save(wd, cfg); err != nil {
					return err
				}
			}
			if err := skills.NewGenerator(cfg).Generate(wd); err != nil {
				return err
			}
			if err := antidrift.New(wd).Lock(cfg.ContractFiles); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ratchet gen: regenerated contracts and lock file")
			return nil
		},
	}
}

func newInitCICmd() *cobra.Command {
	var protectMain bool
	cmd := &cobra.Command{
		Use:   "init-ci",
		Short: "Generate GitHub Actions workflow for hard CI enforcement",
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			path, err := githubci.WriteWorkflow(wd)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ratchet init-ci: wrote %s\n", path)
			if !protectMain {
				return nil
			}
			return enableProtectMain(cmd, wd)
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
	argsAPI, body := githubci.ProtectMainArgs(owner, repo)
	manual := githubci.ProtectMainCommand(owner, repo)
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
		if githubci.IsProtectionUnavailable(msg) {
			return fmt.Errorf("protect-main unavailable: private repos need GitHub Pro (or make the repo public); CI workflow is still the hard gate via exit code 1")
		}
		fmt.Fprintln(cmd.OutOrStdout(), "run manually:")
		fmt.Fprintln(cmd.OutOrStdout(), manual)
		return fmt.Errorf("protect-main: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "ratchet init-ci: required status check %q enabled on %s/%s main\n", githubci.StatusCheckName, owner, repo)
	return nil
}

func newInitHooksCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init-hooks",
		Short: "Install local pre-commit hook (soft friction)",
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			path, err := hooks.Install(wd)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ratchet init-hooks: installed %s\n", path)
			return nil
		},
	}
}

func resolveOwnerRepo(wd string) (string, string, error) {
	out, err := exec.Command("git", "-C", wd, "remote", "get-url", "origin").Output()
	if err != nil {
		return "", "", fmt.Errorf("resolve origin remote: %w", err)
	}
	return githubci.ParseOwnerRepo(string(out))
}
