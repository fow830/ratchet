// Package tokens holds the pure-Go SSOT configuration for ratchet.
package tokens

import "io/fs"

// Published identity — single source; all other artifact names derive from ToolName.
const (
	ToolName      = "ratchet"
	ModulePath    = "github.com/fow830/" + ToolName
	DefaultBranch = "main"
)

// On-disk artifact names and paths derived from ToolName.
const (
	DirVendor    = "vendor"
	DirTestdata  = "testdata"
	DirBin       = "bin"
	DirGenerated = "generated"
	DirExamples  = "examples"
	DirDist      = "dist"
	DirCoverage  = "coverage"
	DirClaude    = ".claude"
	DirGitHub    = ".github"
	DirInternal  = "internal"
	DirCmd       = "cmd"
	SchemaDir    = "schema"
	GitDir       = ".git"

	ConfigFileName      = ToolName + ".json"
	LockFileName        = ToolName + ".lock"
	BenchFileName       = ToolName + ".bench"
	PluginsLockFileName = ToolName + ".plugins.lock"
	SchemaFileName      = ToolName + ".schema.json"
	CursorRules         = ".cursorrules"
	ClaudeSkillRel      = DirClaude + "/skills/" + ToolName + ".md"
	BinaryRel           = DirBin + "/" + ToolName
	CmdRel              = DirCmd + "/" + ToolName
	CmdTokensGenRel     = DirCmd + "/tokensgen"
	GoModFileName       = "go.mod"
	GoWorkFileName      = "go.work"
	PreCommitRel        = GitDir + "/hooks/pre-commit"
	CommitMsgRel        = GitDir + "/hooks/commit-msg"
	SchemaRel           = SchemaDir + "/" + SchemaFileName
	ContractsDirDefault = "tests/contracts"
	EnvExampleRel       = ".env.example"
	READMEFileName      = "README.md"
	DeployFileName      = "DEPLOY.md"
	FuzzCorpusRel       = DirTestdata + "/fuzz/FuzzSeed"
	FuzzSeedFileName    = "seed.txt"
	CoverageOutFile     = "coverage.out"
	CPUProfileFile      = "cpu.pprof"
	ContractSuiteFile   = "suite_test.go"
	ContractTestSuffix  = "_contract_test.go"
	DocGoFileName       = "doc.go"
	ExampleServiceDir   = "service"
	GoFileExt           = ".go"
	GoTestSuffix        = "_test.go"
)

// PresetSkillRel returns .claude/skills/<tool>-<preset>.md.
func PresetSkillRel(preset string) string {
	return DirClaude + "/skills/" + ToolName + "-" + preset + ".md"
}

// BufAgainstGit is the buf breaking --against ref for the default branch.
func BufAgainstGit() string {
	return GitDir + "#branch=" + DefaultBranch
}

// Analysis / quality tool binary names and install module paths.
const (
	ToolStaticcheck   = "staticcheck"
	ToolGovulncheck   = "govulncheck"
	ToolGoMutesting   = "go-mutesting"
	InstallStaticcheck = "honnef.co/go/tools/cmd/" + ToolStaticcheck + "@latest"
	InstallGovulncheck = "golang.org/x/vuln/cmd/" + ToolGovulncheck + "@latest"
	FitnessPkgRel     = "pkg/fitness"
)

// Generated Dockerfile base images.
const (
	DockerGoImage      = "golang:1.22-alpine"
	DockerRuntimeImage = "alpine:3.20"
	ExampleGoVersion   = "1.22"
	ExampleModulePath  = "example.com/" + ToolName + "-service"
)

// Default clean-architecture layer names and import-path suffixes.
const (
	LayerDomain   = "domain"
	LayerUsecase  = "usecase"
	LayerDelivery = "delivery"

	SuffixDomain   = "/" + LayerDomain
	SuffixUsecase  = "/" + LayerUsecase
	SuffixDelivery = "/" + LayerDelivery
)

// Filesystem and lock ledger constants.
const (
	FileModeFile fs.FileMode = 0o644
	FileModeExec fs.FileMode = 0o755
	FileModeDir  fs.FileMode = 0o755

	LockVersion = 1
)

// ModuleHTTPSURL returns the canonical https:// URL for ModulePath.
func ModuleHTTPSURL() string {
	return "https://" + ModulePath
}

// Config is the root SSOT for a ratchet-managed repository.
type Config struct {
	SchemaVersion int    `json:"schema_version,omitempty"`
	Module        string `json:"module"`
	Preset        string `json:"preset,omitempty"`
	Profile       string `json:"profile,omitempty"`

	Layers       map[string]string   `json:"layers"`
	AllowedEdges map[string][]string `json:"allowed_edges"`

	ForbiddenEdges     []ForbiddenEdge `json:"forbidden_edges,omitempty"`
	TestForbiddenEdges []ForbiddenEdge `json:"test_forbidden_edges,omitempty"`
	BoundaryRules      []BoundaryRule  `json:"boundary_rules,omitempty"`
	LayerPaths         []string        `json:"layer_paths,omitempty"`
	AllowedModules     []string        `json:"allowed_modules,omitempty"`
	AllowedExternal    []string        `json:"allowed_external_imports,omitempty"`
	TestcontainersTags []string        `json:"testcontainers_tags,omitempty"`
	ZeroAllocBenches   []string        `json:"zero_alloc_benches,omitempty"`

	ContractFiles []string       `json:"contract_files"`
	ContractLocks []ContractLock `json:"contract_locks,omitempty"`

	AllowedProseDocs []string `json:"allowed_prose_docs,omitempty"`
	ContractsDir     string   `json:"contracts_dir,omitempty"`

	Plugins  []PluginRef   `json:"plugins,omitempty"`
	Gates    *GateFlags    `json:"gates,omitempty"`
	Quality  QualityBudget `json:"quality,omitempty"`
	Modules  []WorkspaceModule `json:"workspace_modules,omitempty"`

	Codegen CodegenHooks `json:"codegen,omitempty"`
}

// CodegenHooks configures optional drift checks for Go ecosystem generators.
type CodegenHooks struct {
	SQLCPath    string `json:"sqlc_path,omitempty"`
	BufPath     string `json:"buf_path,omitempty"`
	OpenAPIPath string `json:"openapi_path,omitempty"`
	CUEPath     string `json:"cue_path,omitempty"`
	TokensGen   string `json:"tokens_gen_cmd,omitempty"`
}

// DefaultConfig returns a clean-architecture baseline for module.
func DefaultConfig(module string) Config {
	cfg := PresetConfig(PresetClean, module)
	cfg.SchemaVersion = CurrentSchemaVersion
	cfg.Profile = ProfileStandard
	return cfg
}

// Preset returns the preset name if set on config.
func (c Config) PresetName() (string, bool) {
	if c.Preset == "" {
		return "", false
	}
	return c.Preset, true
}

// EffectiveGates merges profile defaults with explicit gate overrides.
func (c Config) EffectiveGates() GateFlags {
	g := ProfileGates(NormalizeProfile(c.Profile))
	if c.Gates == nil {
		return g
	}
	ov := c.Gates
	return GateFlags{
		Arch:           pick(ov.Arch, g.Arch),
		Lock:           pick(ov.Lock, g.Lock),
		Docs:           pick(ov.Docs, g.Docs),
		Contracts:      pick(ov.Contracts, g.Contracts),
		Vet:            pick(ov.Vet, g.Vet),
		Staticcheck:    pick(ov.Staticcheck, g.Staticcheck),
		Race:           pick(ov.Race, g.Race),
		Fuzz:           pick(ov.Fuzz, g.Fuzz),
		PBT:            pick(ov.PBT, g.PBT),
		Mutation:       pick(ov.Mutation, g.Mutation),
		Bench:          pick(ov.Bench, g.Bench),
		Coverage:       pick(ov.Coverage, g.Coverage),
		Govuln:         pick(ov.Govuln, g.Govuln),
		Testcontainers: pick(ov.Testcontainers, g.Testcontainers),
		SQLC:           pick(ov.SQLC, g.SQLC),
		Buf:            pick(ov.Buf, g.Buf),
		OpenAPI:        pick(ov.OpenAPI, g.OpenAPI),
		CUE:            pick(ov.CUE, g.CUE),
		WASM:           pick(ov.WASM, g.WASM),
		Cycles:         pick(ov.Cycles, g.Cycles),
		LayerPaths:     pick(ov.LayerPaths, g.LayerPaths),
		External:       pick(ov.External, g.External),
	}
}

func pick(explicit, fallback bool) bool {
	if explicit {
		return true
	}
	return fallback
}

// ContractsRoot returns the contract test directory.
func (c Config) ContractsRoot() string {
	if c.ContractsDir != "" {
		return c.ContractsDir
	}
	return ContractsDirDefault
}
