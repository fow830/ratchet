package tokens

// Schema and preset identity.
const (
	CurrentSchemaVersion = 2

	PresetClean = "clean"
	PresetVitek = "vitek"
	PresetHex   = "hex"

	ProfileMinimal  = "minimal"
	ProfileStandard = "standard"
	ProfileService  = "service"
	ProfileAPI      = "api"
	ProfileStrict   = "strict"
	ProfileParanoid = "paranoid"

	LockModeSHA    = "sha"
	LockModeRender = "render"
)

// Vitek layer names.
const (
	LayerTransport   = "transport"
	LayerService     = "service"
	LayerRepository  = "repository"
	LayerAdapter     = "adapter"
	LayerApplication = "application"
)

// ForbiddenEdge is an explicit deny rule (stronger than allowed_edges alone).
type ForbiddenEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// BoundaryRule forbids imports matching a path pattern.
type BoundaryRule struct {
	ImporterGlob string `json:"importer_glob"`
	ImportGlob   string `json:"import_glob"`
}

// ContractLock describes per-file lock strategy.
type ContractLock struct {
	Path          string `json:"path"`
	Mode          string `json:"mode"` // sha | render
	RenderPackage string `json:"render_package,omitempty"`
	RenderFunc    string `json:"render_func,omitempty"`
}

// PluginRef is a WASM fitness plugin entry.
type PluginRef struct {
	Path string `json:"path"`
	Name string `json:"name,omitempty"`
}

// GateFlags toggles optional hard gates (profile defaults + overrides).
type GateFlags struct {
	Arch        bool    `json:"arch"`
	Lock        bool    `json:"lock"`
	Docs        bool    `json:"docs"`
	Contracts   bool    `json:"contracts"`
	Vet         bool    `json:"vet"`
	Staticcheck bool    `json:"staticcheck"`
	Race        bool    `json:"race"`
	Fuzz        bool    `json:"fuzz"`
	PBT         bool    `json:"pbt"`
	Mutation    bool    `json:"mutation"`
	Bench       bool    `json:"bench"`
	Coverage    bool    `json:"coverage"`
	Govuln      bool    `json:"govulncheck"`
	Testcontainers bool `json:"testcontainers"`
	SQLC        bool    `json:"sqlc"`
	Buf         bool    `json:"buf"`
	OpenAPI     bool    `json:"openapi"`
	CUE         bool    `json:"cue"`
	WASM        bool    `json:"wasm"`
	Cycles      bool    `json:"cycles"`
	LayerPaths  bool    `json:"layer_paths"`
	External    bool    `json:"external_modules"`
}

// QualityBudget holds thresholds for strict/paranoid profiles.
type QualityBudget struct {
	MutationMinPct float64 `json:"mutation_min_pct,omitempty"`
	CoverageMinPct float64 `json:"coverage_min_pct,omitempty"`
	BenchBaseline  string  `json:"bench_baseline,omitempty"`
}

// WorkspaceModule overrides config for a submodule path in go.work monorepos.
type WorkspaceModule struct {
	Path   string `json:"path"`
	Config Config `json:"config"`
}
