package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/fow830/ratchet/pkg/analyze"
	"github.com/fow830/ratchet/pkg/antidrift"
	"github.com/fow830/ratchet/pkg/benchlock"
	"github.com/fow830/ratchet/pkg/breaking"
	"github.com/fow830/ratchet/pkg/contracts"
	"github.com/fow830/ratchet/pkg/doctor"
	"github.com/fow830/ratchet/pkg/fuzzinit"
	"github.com/fow830/ratchet/pkg/gates"
	"github.com/fow830/ratchet/pkg/generate"
	"github.com/fow830/ratchet/pkg/graph"
	"github.com/fow830/ratchet/pkg/observe"
	"github.com/fow830/ratchet/pkg/plugins"
	"github.com/fow830/ratchet/pkg/report"
	"github.com/fow830/ratchet/pkg/smoke"
	"github.com/fow830/ratchet/pkg/tokens"
)

func registerExtraCommands(root *cobra.Command, rf *rootFlags) {
	root.AddCommand(
		newDoctorCmd(rf),
		newValidateConfigCmd(rf),
		newMigrateConfigCmd(rf),
		newGraphCmd(rf),
		newNewContractCmd(rf),
		newDiffLockCmd(rf),
		newExplainCmd(rf),
		newFuzzInitCmd(rf),
		newAnalyzeCmd(rf),
		newLockCmd(rf),
		newBenchLockCmd(rf),
		newInitExampleCmd(rf),
		newObserveCmd(rf),
		newGenTokensCmd(rf),
		newPluginLockCmd(rf),
		newSmokeCmd(rf),
		newValidateSARIFCmd(),
	)
}

func newValidateSARIFCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate-sarif [file]",
		Short: "Validate a SARIF file structure",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			b, err := os.ReadFile(args[0])
			if err != nil {
				return systemErr(err)
			}
			if err := report.ValidateSARIF(b); err != nil {
				return violationErr(err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "sarif: ok")
			return nil
		},
	}
}

func newDoctorCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose ratchet project setup",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := commandContext()
			defer cancel()
			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			cfg, cfgPath, err := loadConfig(ctx, wd, rf.config)
			if err != nil {
				return systemErr(err)
			}
			rep, err := doctor.Run(ctx, wd, cfg, cfgPath)
			if err != nil {
				return systemErr(err)
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			if err := enc.Encode(rep); err != nil {
				return systemErr(err)
			}
			if !rep.Healthy() {
				return violationErr(fmt.Errorf("%s doctor: setup incomplete", tokens.ToolName))
			}
			return nil
		},
	}
}

func newValidateConfigCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "validate-config",
		Short: "Validate ratchet.json schema and invariants",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := commandContext()
			defer cancel()
			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			cfg, _, err := loadConfig(ctx, wd, rf.config)
			if err != nil {
				return systemErr(err)
			}
			if err := tokens.Validate(cfg); err != nil {
				return violationErr(err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), toolMsg("validate-config", "ok"))
			return nil
		},
	}
}

func newMigrateConfigCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "migrate-config",
		Short: "Migrate ratchet.json to the current schema version",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := commandContext()
			defer cancel()
			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			cfg, cfgPath, err := loadConfig(ctx, wd, rf.config)
			if err != nil {
				return systemErr(err)
			}
			migrated, err := tokens.Migrate(cfg)
			if err != nil {
				return systemErr(err)
			}
			if rf.dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), toolMsg("migrate-config (dry-run)", "would write "+cfgPath))
				return nil
			}
			if err := tokens.SaveFile(ctx, cfgPath, migrated); err != nil {
				return systemErr(err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), toolMsg("migrate-config", "migrated to schema v"+fmt.Sprint(migrated.SchemaVersion)))
			return nil
		},
	}
}

func newGraphCmd(rf *rootFlags) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Export import dependency graph",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := commandContext()
			defer cancel()
			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			cfg, _, err := loadConfig(ctx, wd, rf.config)
			if err != nil {
				return systemErr(err)
			}
			switch format {
			case "mermaid", "":
				out, err := graph.ExportMermaid(ctx, wd, cfg)
				if err != nil {
					return systemErr(err)
				}
				fmt.Fprint(cmd.OutOrStdout(), out)
			default:
				return systemErr(fmt.Errorf("unsupported graph format %q", format))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "mermaid", "graph format: mermaid")
	return cmd
}

func newNewContractCmd(rf *rootFlags) *cobra.Command {
	var title string
	var negative bool
	cmd := &cobra.Command{
		Use:   "new-contract ID",
		Short: "Scaffold a contract test file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			if rf.dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), toolMsg("new-contract (dry-run)", "would scaffold "+args[0]))
				return nil
			}
			path, err := contracts.Scaffold(wd, contracts.ScaffoldOpts{
				ID: args[0], Title: title, Negative: negative,
			})
			if err != nil {
				return systemErr(err)
			}
			_, _ = contracts.ScaffoldSuite(contracts.PackagePath(wd))
			fmt.Fprintln(cmd.OutOrStdout(), toolMsg("new-contract", "wrote "+path))
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "contract proof", "human-readable contract title")
	cmd.Flags().BoolVar(&negative, "negative", false, "scaffold negative contract")
	return cmd
}

func newDiffLockCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "diff-lock",
		Short: "Show breaking config deltas vs git HEAD",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := commandContext()
			defer cancel()
			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			cfgPath := rf.config
			if cfgPath == "" {
				cfgPath = filepath.Join(wd, tokens.ConfigFileName)
			} else if !filepath.IsAbs(cfgPath) {
				cfgPath = filepath.Join(wd, cfgPath)
			}
			changes, err := breaking.DiffAgainstGit(ctx, wd, cfgPath)
			if err != nil {
				// Fallback when git history unavailable: compare to default preset.
				cfg, _, lerr := loadConfig(ctx, wd, rf.config)
				if lerr != nil {
					return systemErr(err)
				}
				baseline := tokens.DefaultConfig(cfg.Module)
				changes = breaking.DiffConfig(baseline, cfg)
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			if err := enc.Encode(changes); err != nil {
				return systemErr(err)
			}
			if len(changes) > 0 {
				return violationErr(fmt.Errorf("%s diff-lock: breaking changes detected", tokens.ToolName))
			}
			return nil
		},
	}
}

func newExplainCmd(rf *rootFlags) *cobra.Command {
	var format, profile string
	cmd := &cobra.Command{
		Use:   "explain",
		Short: "Explain check failures with fix guidance",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := commandContext()
			defer cancel()
			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			cfg, cfgPath, err := loadConfig(ctx, wd, rf.config)
			if err != nil {
				return systemErr(err)
			}
			gr, err := gates.Run(ctx, gates.Options{Root: wd, Config: cfg, Profile: profile, ConfigPath: cfgPath})
			if err != nil {
				return systemErr(err)
			}
			er := report.FromGateResult(gr)
			fmt.Fprint(cmd.OutOrStdout(), report.ExplainResult(er))
			if !er.OK {
				return violationErr(fmt.Errorf("%s explain: failures present", tokens.ToolName))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "", "gate profile")
	cmd.Flags().StringVar(&format, "format", "", "unused; kept for flag compatibility")
	return cmd
}

func newFuzzInitCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "fuzz-init",
		Short: "Scaffold fuzz corpus seed files",
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			if rf.dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), toolMsg("fuzz-init (dry-run)", "would scaffold "+tokens.FuzzCorpusRel))
				return nil
			}
			path, err := fuzzinit.Init(wd)
			if err != nil {
				return systemErr(err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), toolMsg("fuzz-init", path))
			return nil
		},
	}
}

func newAnalyzeCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "analyze",
		Short: "Run escape analysis hints (-gcflags=-m)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := commandContext()
			defer cancel()
			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			hints, err := analyze.Run(ctx, wd)
			if err != nil {
				return systemErr(err)
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			if err := enc.Encode(hints); err != nil {
				return systemErr(err)
			}
			return nil
		},
	}
}

func newLockCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "lock",
		Short: "Re-lock contract files without regenerating skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := commandContext()
			defer cancel()
			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			cfg, _, err := loadConfig(ctx, wd, rf.config)
			if err != nil {
				return systemErr(err)
			}
			if rf.dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), toolMsg("lock (dry-run)", "would update "+tokens.LockFileName))
				return nil
			}
			if err := antidrift.New(wd).LockAll(ctx, cfg); err != nil {
				return systemErr(err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), toolMsg("lock", "updated "+tokens.LockFileName))
			return nil
		},
	}
}

func runCheck(cmd *cobra.Command, rf *rootFlags, format, profile string, workspaceMode bool) error {
	ctx, cancel := commandContext()
	defer cancel()

	wd, err := os.Getwd()
	if err != nil {
		return systemErr(err)
	}

	if workspaceMode {
		results, err := gates.RunWorkspace(ctx, gates.WorkspaceOptions{
			Root: wd, Profile: profile,
		})
		if err != nil {
			return systemErr(err)
		}
		if !gates.WorkspaceOK(results) {
			return violationErr(fmt.Errorf("%s workspace check failed", tokens.ToolName))
		}
		fmt.Fprintln(cmd.OutOrStdout(), toolMsg("check", "workspace ok"))
		return nil
	}

	cfg, cfgPath, err := loadConfig(ctx, wd, rf.config)
	if err != nil {
		return systemErr(err)
	}
	if rf.verbose {
		fmt.Fprintf(cmd.ErrOrStderr(), "config=%s module=%s format=%s profile=%s\n", cfgPath, cfg.Module, format, profile)
	}

	gr, err := gates.Run(ctx, gates.Options{
		Root:       wd,
		Config:     cfg,
		Profile:    profile,
		ConfigPath: cfgPath,
	})
	if err != nil {
		return systemErr(err)
	}
	result := report.FromGateResult(gr)
	errOut := cmd.ErrOrStderr()
	out := cmd.OutOrStdout()

	switch format {
	case report.FormatJSON:
		result.NormalizeExtended()
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return systemErr(err)
		}
		fmt.Fprintln(out, string(data))
	case report.FormatSARIF:
		data, err := report.MarshalSARIF(result.Result)
		if err != nil {
			return systemErr(err)
		}
		fmt.Fprintln(out, string(data))
	case report.FormatLLM:
		for _, v := range result.Violations {
			fmt.Fprintln(errOut, report.FormatLLMViolation(v))
			fmt.Fprintln(errOut)
		}
		if !result.Drift.OK() {
			fmt.Fprint(errOut, report.FormatLLMDiff(result.Drift))
		}
		for _, d := range result.DocsViolations {
			fmt.Fprintln(errOut, report.FormatLLMDocs(d))
			fmt.Fprintln(errOut)
		}
		for _, f := range result.Failures {
			fmt.Fprintln(errOut, report.FormatLLMFailure(f))
			fmt.Fprintln(errOut)
		}
	default:
		for _, v := range result.Violations {
			fmt.Fprintln(errOut, report.FormatTextViolation(v))
		}
		if !result.Drift.OK() {
			fmt.Fprint(errOut, report.FormatTextDiff(result.Drift))
		}
		for _, d := range result.DocsViolations {
			fmt.Fprintln(errOut, d.String())
		}
		for _, f := range result.Failures {
			fmt.Fprintln(errOut, f.String())
		}
	}

	if !result.OK {
		return violationErr(fmt.Errorf("%s check failed (profile=%s)", tokens.ToolName, gr.Profile))
	}
	if format == report.FormatText || format == report.FormatLLM {
		fmt.Fprintln(out, toolMsg("check", "ok"))
	}
	return nil
}

func newGenTokensCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "gen-tokens",
		Short: "Generate env/compose/dockerfile/sqlc stubs from SSOT",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := commandContext()
			defer cancel()
			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			cfg, _, err := loadConfig(ctx, wd, rf.config)
			if err != nil {
				return systemErr(err)
			}
			if rf.dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), toolMsg("gen-tokens (dry-run)", "would write generate.Outputs"))
				return nil
			}
			written, err := generate.WriteAll(ctx, wd, cfg)
			if err != nil {
				return systemErr(err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), toolMsg("gen-tokens", fmt.Sprintf("wrote %v", written)))
			return nil
		},
	}
}

func newPluginLockCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "plugin-lock",
		Short: "Lock WASM plugin hashes to " + plugins.LockFileName,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := commandContext()
			defer cancel()
			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			cfg, _, err := loadConfig(ctx, wd, rf.config)
			if err != nil {
				return systemErr(err)
			}
			if len(cfg.Plugins) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), toolMsg("plugin-lock", "no plugins configured"))
				return nil
			}
			if rf.dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), toolMsg("plugin-lock (dry-run)", "would write "+plugins.LockFileName))
				return nil
			}
			if err := plugins.LockPlugins(ctx, wd, cfg.Plugins); err != nil {
				return systemErr(err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), toolMsg("plugin-lock", "updated "+plugins.LockFileName))
			return nil
		},
	}
}

func newSmokeCmd(rf *rootFlags) *cobra.Command {
	var url string
	var wantStatus int
	cmd := &cobra.Command{
		Use:   "smoke",
		Short: "HTTP smoke probe against a live URL",
		RunE: func(cmd *cobra.Command, args []string) error {
			if url == "" {
				return systemErr(fmt.Errorf("smoke: --url is required"))
			}
			res, err := smoke.GET(url, smoke.Options{WantStatus: wantStatus})
			if err != nil {
				return violationErr(err)
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			if err := enc.Encode(res); err != nil {
				return systemErr(err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&url, "url", "", "URL to probe")
	cmd.Flags().IntVar(&wantStatus, "want-status", 200, "expected HTTP status")
	return cmd
}

func newObserveCmd(rf *rootFlags) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "observe",
		Short: "Probe optional runtime observe tools (go/pprof/cilium/hubble)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := commandContext()
			defer cancel()
			rep := observe.Probe(ctx)
			if err := observe.RequireGo(rep); err != nil {
				return systemErr(err)
			}
			switch format {
			case report.FormatJSON, "":
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(rep); err != nil {
					return systemErr(err)
				}
			default:
				return systemErr(fmt.Errorf("unsupported format %q", format))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", report.FormatJSON, "output format: "+report.FormatJSON)
	return cmd
}

func newBenchLockCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "bench-lock",
		Short: "Capture benchmark baseline to " + benchlock.FileName,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := commandContext()
			defer cancel()
			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			if rf.dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), toolMsg("bench-lock (dry-run)", "would write "+benchlock.FileName))
				return nil
			}
			out, err := exec.CommandContext(ctx, "go", "test", "-bench=.", "-benchmem", "-count=1", "./...").CombinedOutput()
			if err != nil {
				return systemErr(fmt.Errorf("bench: %w\n%s", err, out))
			}
			entries, err := benchlock.ParseOutput(string(out))
			if err != nil {
				return systemErr(err)
			}
			if err := benchlock.New(wd).Lock(ctx, entries); err != nil {
				return systemErr(err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), toolMsg("bench-lock", "updated "+benchlock.FileName))
			return nil
		},
	}
}

func newInitExampleCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "init-example",
		Short: "Scaffold examples/service reference module",
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return systemErr(err)
			}
			if rf.dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), toolMsg("init-example (dry-run)", "would write examples/service"))
				return nil
			}
			if err := scaffoldExampleService(wd); err != nil {
				return systemErr(err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), toolMsg("init-example", "wrote examples/service"))
			return nil
		},
	}
}

func scaffoldExampleService(root string) error {
	dir := filepath.Join(root, tokens.DirExamples, tokens.ExampleServiceDir)
	if err := os.MkdirAll(filepath.Join(dir, tokens.DirInternal, tokens.LayerDomain), tokens.FileModeDir); err != nil {
		return err
	}
	files := map[string]string{
		tokens.GoModFileName: fmt.Sprintf("module %s\n\ngo %s\n", tokens.ExampleModulePath, tokens.ExampleGoVersion),
		tokens.READMEFileName: "# ratchet-service\n\nReference service module (vitek preset).\n",
		filepath.Join(tokens.DirInternal, tokens.LayerDomain, tokens.DocGoFileName): "package domain\n",
	}
	for rel, body := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), tokens.FileModeDir); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(body), tokens.FileModeFile); err != nil {
			return err
		}
	}
	ctx := context.Background()
	cfg := tokens.PresetConfig(tokens.PresetVitek, tokens.ExampleModulePath)
	cfg.Profile = tokens.ProfileService
	if err := tokens.Save(ctx, dir, cfg); err != nil {
		return err
	}
	contractsDir := filepath.Join(dir, filepath.FromSlash(tokens.ContractsDirDefault))
	if err := os.MkdirAll(contractsDir, tokens.FileModeDir); err != nil {
		return err
	}
	if _, err := contracts.ScaffoldSuite(contractsDir); err != nil {
		return err
	}
	_, err := contracts.Scaffold(dir, contracts.ScaffoldOpts{ID: "ARCH-001", Title: "Layer isolation"})
	return err
}
