package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
		Use:   "ratchet",
		Short: "Deterministic AI-native anti-drift framework for Go",
	}

	root.AddCommand(
		newInitCmd(),
		newCheckCmd(),
		newGenCmd(),
		newInitCICmd(),
		newInitHooksCmd(),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Bootstrap .cursorrules and default ratchet.go config",
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			module := detectModule(wd)
			cfg := tokens.DefaultConfig(module)

			if err := writeRatchetGo(wd, cfg); err != nil {
				return err
			}

			gen := skills.NewGenerator(cfg)
			if err := gen.Generate(wd); err != nil {
				return err
			}

			eng := antidrift.New(wd)
			if err := eng.Lock(cfg.ContractFiles); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "ratchet init: wrote .cursorrules, ratchet.go, .claude/skills/ratchet.md, ratchet.lock")
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
			cfg, err := loadConfig(wd)
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
				} else {
					fmt.Fprintln(errOut, report.FormatHumanViolation(v))
				}
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
		Short: "Generate boilerplate contracts and agent skill rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			cfg, err := loadConfig(wd)
			if err != nil {
				cfg = tokens.DefaultConfig(detectModule(wd))
				if err := writeRatchetGo(wd, cfg); err != nil {
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

			owner, repo, err := resolveOwnerRepo(wd)
			if err != nil {
				return err
			}

			ghPath, lookErr := exec.LookPath("gh")
			argsAPI, body := githubci.ProtectMainArgs(owner, repo)
			if lookErr != nil {
				fmt.Fprintln(cmd.OutOrStdout(), "gh CLI not found; run this command to enable branch protection:")
				fmt.Fprintln(cmd.OutOrStdout(), githubci.ProtectMainCommand(owner, repo))
				return nil
			}

			c := exec.Command(ghPath, argsAPI...)
			c.Stdin = bytes.NewBufferString(body)
			c.Stdout = cmd.OutOrStdout()
			c.Stderr = cmd.ErrOrStderr()
			if err := c.Run(); err != nil {
				fmt.Fprintln(cmd.OutOrStdout(), "branch protection API call failed; run manually:")
				fmt.Fprintln(cmd.OutOrStdout(), githubci.ProtectMainCommand(owner, repo))
				return fmt.Errorf("protect-main: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ratchet init-ci: enabled required status check on %s/%s main\n", owner, repo)
			return nil
		},
	}
	cmd.Flags().BoolVar(&protectMain, "protect-main", false, "enable GitHub required status checks on main via gh api")
	return cmd
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
	c := exec.Command("git", "-C", wd, "remote", "get-url", "origin")
	out, err := c.Output()
	if err != nil {
		return "", "", fmt.Errorf("resolve origin remote: %w", err)
	}
	return githubci.ParseOwnerRepo(string(out))
}

func detectModule(wd string) string {
	data, err := os.ReadFile(filepath.Join(wd, "go.mod"))
	if err != nil {
		return "github.com/fow830/ratchet"
	}
	for _, line := range splitLines(string(data)) {
		if len(line) >= 7 && line[:7] == "module " {
			return line[7:]
		}
	}
	return "github.com/fow830/ratchet"
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			out = append(out, line)
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func writeRatchetGo(wd string, cfg tokens.Config) error {
	marker := `// Code generated by ratchet. DO NOT EDIT MANUALLY without re-running ratchet gen.
package ratchetconfig

// This package is a Pure Go SSOT marker. Runtime config lives in ratchet.json.
`
	if err := os.WriteFile(filepath.Join(wd, "ratchet.go"), []byte(marker), 0o644); err != nil {
		return fmt.Errorf("write ratchet.go: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(wd, "ratchet.json"), data, 0o644); err != nil {
		return fmt.Errorf("write ratchet.json: %w", err)
	}
	return nil
}

func loadConfig(wd string) (tokens.Config, error) {
	data, err := os.ReadFile(filepath.Join(wd, "ratchet.json"))
	if err != nil {
		return tokens.Config{}, fmt.Errorf("load ratchet.json: %w", err)
	}
	var cfg tokens.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return tokens.Config{}, fmt.Errorf("parse ratchet.json: %w", err)
	}
	return cfg, nil
}
