package gates

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fow830/ratchet/pkg/antidrift"
	"github.com/fow830/ratchet/pkg/benchlock"
	"github.com/fow830/ratchet/pkg/docs"
	"github.com/fow830/ratchet/pkg/fitness"
	"github.com/fow830/ratchet/pkg/plugins"
	"github.com/fow830/ratchet/pkg/tokens"
)

// Failure is a single gate failure.
type Failure struct {
	Gate    string `json:"gate"`
	Message string `json:"message"`
}

func (f Failure) String() string { return fmt.Sprintf("[%s] %s", f.Gate, f.Message) }

// Result aggregates all gate outcomes.
type Result struct {
	OK             bool               `json:"ok"`
	Violations     []fitness.Violation `json:"violations"`
	Drift          antidrift.Diff      `json:"drift"`
	DocsViolations []docs.Violation    `json:"docs_violations"`
	Failures       []Failure           `json:"failures"`
	Profile        string              `json:"profile"`
	GatesRun       []string            `json:"gates_run"`
}

// Options configures a gate run.
type Options struct {
	Root       string
	Config     tokens.Config
	Profile    string
	ConfigPath string
	Runner     CommandRunner
}

// CommandRunner executes external tools (testable).
type CommandRunner interface {
	Run(ctx context.Context, dir, name string, args ...string) (string, error)
}

// ExecRunner runs real commands.
type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s %v: %w\n%s", name, args, err, out)
	}
	return string(out), nil
}

// Run executes enabled gates for the profile.
func Run(ctx context.Context, opts Options) (Result, error) {
	if opts.Runner == nil {
		opts.Runner = ExecRunner{}
	}
	profile := opts.Config.Profile
	if opts.Profile != "" {
		profile = opts.Profile
	}
	profile = tokens.NormalizeProfile(profile)

	var g tokens.GateFlags
	if opts.Profile != "" {
		g = tokens.ProfileGates(profile)
	} else {
		g = opts.Config.EffectiveGates()
		profile = tokens.NormalizeProfile(opts.Config.Profile)
	}

	res := Result{Profile: profile, OK: true}
	fail := func(gate, msg string) {
		res.Failures = append(res.Failures, Failure{Gate: gate, Message: msg})
		res.OK = false
	}

	an := fitness.NewAnalyzer(opts.Config)

	if g.Arch {
		res.GatesRun = append(res.GatesRun, "arch")
		v, err := an.AnalyzeAll(ctx, opts.Root)
		if err != nil {
			return res, err
		}
		if g.LayerPaths {
			res.GatesRun = append(res.GatesRun, "layer_paths")
			lp, err := an.CheckLayerPaths(ctx, opts.Root)
			if err != nil {
				return res, err
			}
			v = append(v, lp...)
		}
		if g.External {
			res.GatesRun = append(res.GatesRun, "external")
			ext, err := an.CheckExternal(ctx, opts.Root)
			if err != nil {
				return res, err
			}
			v = append(v, ext...)
		}
		tv, err := an.CheckTestImports(ctx, opts.Root)
		if err != nil {
			return res, err
		}
		if len(tv) > 0 {
			res.GatesRun = append(res.GatesRun, "test_imports")
			v = append(v, tv...)
		}
		if g.Cycles {
			res.GatesRun = append(res.GatesRun, "cycles")
			cycles, err := fitness.DetectCycles(ctx, opts.Root, opts.Config.Module)
			if err != nil {
				fail("cycles", err.Error())
			} else {
				for _, c := range cycles {
					fail("cycles", fitness.FormatCycle(c))
				}
			}
		}
		if len(v) > 0 {
			res.Violations = v
			res.OK = false
		}
	}

	if g.Lock {
		res.GatesRun = append(res.GatesRun, "lock")
		eng := antidrift.New(opts.Root)
		eng.ConfigPath = opts.ConfigPath
		diff, err := eng.VerifyAll(ctx, opts.Config)
		if err != nil {
			return res, err
		}
		res.Drift = diff
		if !diff.OK() {
			res.OK = false
		}
		if len(opts.Config.Plugins) > 0 {
			res.GatesRun = append(res.GatesRun, "plugin_lock")
			if err := plugins.VerifyLock(ctx, opts.Root, opts.Config.Plugins); err != nil {
				fail("plugin_lock", err.Error())
			}
		}
	}

	if g.Docs {
		res.GatesRun = append(res.GatesRun, "docs")
		dv, err := docs.Check(ctx, opts.Root, opts.Config)
		if err != nil {
			return res, err
		}
		if len(dv) > 0 {
			res.DocsViolations = dv
			res.OK = false
		}
	}

	if g.Contracts {
		res.GatesRun = append(res.GatesRun, "contracts")
		dir := filepath.Join(opts.Root, opts.Config.ContractsRoot())
		if _, err := os.Stat(dir); err == nil {
			if _, err := opts.Runner.Run(ctx, opts.Root, "go", "test", "./"+opts.Config.ContractsRoot()+"/..."); err != nil {
				fail("contracts", err.Error())
			}
		}
	}

	runTool := func(gate, name string, args ...string) (string, error) {
		res.GatesRun = append(res.GatesRun, gate)
		out, err := opts.Runner.Run(ctx, opts.Root, name, args...)
		if err != nil {
			if isNotInstalled(err) {
				fail(gate, name+" not installed")
			} else {
				fail(gate, err.Error())
			}
			return out, err
		}
		return out, nil
	}

	if g.Vet {
		_, _ = runTool("vet", "go", "vet", "./...")
	}
	if g.Race {
		_, _ = runTool("race", "go", "test", "-race", "-count=1", "./...")
	}
	if g.Fuzz {
		out, err := opts.Runner.Run(ctx, opts.Root, "go", "test", "-list=^Fuzz", "./...")
		if err == nil && !strings.Contains(out, "Fuzz") {
			res.GatesRun = append(res.GatesRun, "fuzz_skipped")
		} else {
			_, _ = runTool("fuzz", "go", "test", "-fuzz=Fuzz", "-fuzztime=10s", "./...")
		}
	}
	if g.Staticcheck {
		_, _ = runTool(tokens.ToolStaticcheck, tokens.ToolStaticcheck, "./...")
	}
	if g.Govuln {
		_, _ = runTool(tokens.ToolGovulncheck, tokens.ToolGovulncheck, "./...")
	}
	if g.PBT {
		_, _ = runTool("pbt", "go", "test", "-run", "Property|PBT|Quick", "./...")
	}
	if g.Testcontainers {
		res.GatesRun = append(res.GatesRun, "testcontainers")
		tags := opts.Config.TestcontainersTags
		if len(tags) == 0 {
			tags = []string{"integration"}
		}
		args := []string{"test", "-count=1", "-tags=" + strings.Join(tags, ",")}
		args = append(args, "./"+opts.Config.ContractsRoot()+"/...")
		if _, err := opts.Runner.Run(ctx, opts.Root, "go", args...); err != nil {
			if !isNotInstalled(err) && !strings.Contains(err.Error(), "no test files") && !strings.Contains(err.Error(), "build constraints") {
				fail("testcontainers", err.Error())
			}
		}
	}
	if g.Mutation {
		out, err := runTool("mutation", tokens.ToolGoMutesting, "./...")
		if err == nil && opts.Config.Quality.MutationMinPct > 0 {
			score, perr := ParseMutationScore(out)
			if perr != nil {
				fail("mutation", perr.Error())
			} else if score < opts.Config.Quality.MutationMinPct {
				fail("mutation", fmt.Sprintf("score %.1f%% < min %.1f%%", score, opts.Config.Quality.MutationMinPct))
			}
		}
	}
	if g.Bench {
		res.GatesRun = append(res.GatesRun, "bench")
		if err := runBenchGate(ctx, opts); err != nil {
			fail("bench", err.Error())
		}
	}
	if g.Coverage {
		_, err := runTool("coverage", "go", "test", "-coverprofile="+tokens.CoverageOutFile, "./...")
		if err == nil && opts.Config.Quality.CoverageMinPct > 0 {
			coverOut, cerr := opts.Runner.Run(ctx, opts.Root, "go", "tool", "cover", "-func="+tokens.CoverageOutFile)
			var pct float64
			var perr error
			if cerr == nil {
				pct, perr = ParseCoverFuncTotal(coverOut)
			} else {
				perr = cerr
			}
			if perr != nil {
				fail("coverage", perr.Error())
			} else if pct < opts.Config.Quality.CoverageMinPct {
				fail("coverage", fmt.Sprintf("coverage %.1f%% < min %.1f%%", pct, opts.Config.Quality.CoverageMinPct))
			}
		}
	}
	if g.SQLC && opts.Config.Codegen.SQLCPath != "" {
		if _, err := os.Stat(filepath.Join(opts.Root, opts.Config.Codegen.SQLCPath)); err != nil {
			fail("sqlc", "missing "+opts.Config.Codegen.SQLCPath)
		} else {
			_, _ = runTool("sqlc", "sqlc", "diff")
		}
	}
	if g.Buf && opts.Config.Codegen.BufPath != "" {
		if _, err := os.Stat(filepath.Join(opts.Root, opts.Config.Codegen.BufPath)); err != nil {
			fail("buf", "missing "+opts.Config.Codegen.BufPath)
		} else {
			_, _ = runTool("buf", "buf", "breaking", "--against", tokens.BufAgainstGit())
		}
	}
	if g.OpenAPI && opts.Config.Codegen.OpenAPIPath != "" {
		if _, err := os.Stat(filepath.Join(opts.Root, opts.Config.Codegen.OpenAPIPath)); err != nil {
			fail("openapi", "missing "+opts.Config.Codegen.OpenAPIPath)
		} else {
			_, _ = runTool("openapi", "oapi-codegen", "-generate", "types", opts.Config.Codegen.OpenAPIPath)
		}
	}
	if g.CUE && opts.Config.Codegen.CUEPath != "" {
		if _, err := os.Stat(filepath.Join(opts.Root, opts.Config.Codegen.CUEPath)); err != nil {
			fail("cue", "missing "+opts.Config.Codegen.CUEPath)
		} else {
			_, _ = runTool("cue", "cue", "vet", opts.Config.Codegen.CUEPath)
		}
	}
	if g.WASM && len(opts.Config.Plugins) > 0 {
		res.GatesRun = append(res.GatesRun, "wasm")
		if err := runWASMPlugins(ctx, opts); err != nil {
			fail("wasm", err.Error())
		}
	}

	return res, nil
}

func runWASMPlugins(ctx context.Context, opts Options) error {
	eng, err := plugins.NewEngine(ctx)
	if err != nil {
		return err
	}
	defer eng.Close(ctx)
	var paths []string
	for _, p := range opts.Config.Plugins {
		paths = append(paths, filepath.Join(opts.Root, p.Path))
	}
	return plugins.RunAll(ctx, eng, paths)
}

func runBenchGate(ctx context.Context, opts Options) error {
	out, err := opts.Runner.Run(ctx, opts.Root, "go", "test", "-bench=.", "-benchmem", "-count=1", "./...")
	if err != nil {
		return err
	}
	entries, err := benchlock.ParseOutput(out)
	if err != nil {
		return err
	}
	marked := map[string]struct{}{}
	for _, n := range opts.Config.ZeroAllocBenches {
		marked[n] = struct{}{}
	}
	var za []BenchEntry
	for _, e := range entries {
		_, ok := marked[e.Name]
		za = append(za, BenchEntry{Name: e.Name, AllocsPerOp: e.AllocsPerOp, MarkedZeroAlloc: ok})
	}
	if err := CheckZeroAlloc(za); err != nil {
		return err
	}
	benchPath := filepath.Join(opts.Root, benchlock.FileName)
	if _, err := os.Stat(benchPath); err == nil {
		diff, err := benchlock.New(opts.Root).Verify(ctx, entries)
		if err != nil {
			return err
		}
		if !diff.OK() {
			return fmt.Errorf("%s", diff.String())
		}
	}
	return nil
}

func isNotInstalled(err error) bool {
	return strings.Contains(err.Error(), "executable file not found") ||
		strings.Contains(err.Error(), "not installed")
}
