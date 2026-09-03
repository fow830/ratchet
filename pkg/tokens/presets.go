package tokens

// PresetConfig returns architecture SSOT for a named preset and module path.
func PresetConfig(preset, module string) Config {
	switch preset {
	case PresetVitek:
		return vitekPreset(module)
	case PresetHex:
		return hexPreset(module)
	case PresetClean, "":
		return cleanPreset(module)
	default:
		cfg := cleanPreset(module)
		cfg.Preset = ""
		return cfg
	}
}

func cleanPreset(module string) Config {
	return Config{
		Module: module,
		Preset: PresetClean,
		Layers: map[string]string{
			SuffixDomain:   LayerDomain,
			SuffixUsecase:  LayerUsecase,
			SuffixDelivery: LayerDelivery,
		},
		AllowedEdges: map[string][]string{
			LayerDomain:   {},
			LayerUsecase:  {LayerDomain},
			LayerDelivery: {LayerUsecase, LayerDomain},
		},
		ForbiddenEdges: []ForbiddenEdge{
			{From: LayerDomain, To: LayerDelivery},
			{From: LayerDomain, To: LayerUsecase},
		},
		LayerPaths: []string{
			DirInternal + "/" + LayerDomain,
			DirInternal + "/" + LayerUsecase,
			DirInternal + "/" + LayerDelivery,
		},
		ContractFiles: []string{
			CursorRules,
			ConfigFileName,
		},
		AllowedProseDocs: []string{READMEFileName},
		ContractsDir:     ContractsDirDefault,
	}
}

func vitekPreset(module string) Config {
	return Config{
		Module: module,
		Preset: PresetVitek,
		Layers: map[string]string{
			"/transport":  LayerTransport,
			"/service":    LayerService,
			"/repository": LayerRepository,
			"/domain":     LayerDomain,
		},
		AllowedEdges: map[string][]string{
			LayerDomain:     {},
			LayerRepository: {LayerDomain},
			LayerService:    {LayerRepository, LayerDomain},
			LayerTransport:  {LayerService, LayerDomain},
		},
		ForbiddenEdges: []ForbiddenEdge{
			{From: LayerTransport, To: LayerRepository},
			{From: LayerDomain, To: LayerTransport},
			{From: LayerDomain, To: LayerService},
			{From: LayerDomain, To: LayerRepository},
		},
		LayerPaths: []string{
			DirInternal + "/" + LayerTransport,
			DirInternal + "/" + LayerService,
			DirInternal + "/" + LayerRepository,
			DirInternal + "/" + LayerDomain,
		},
		BoundaryRules: []BoundaryRule{
			{ImporterGlob: "**/" + DirInternal + "/" + LayerDomain + "/**", ImportGlob: "**/" + DirInternal + "/" + LayerTransport + "/**"},
		},
		ContractFiles: []string{
			CursorRules,
			ConfigFileName,
		},
		AllowedProseDocs: []string{READMEFileName, DeployFileName},
		ContractsDir:     ContractsDirDefault,
		Codegen: CodegenHooks{
			SQLCPath:  "sqlc.yaml",
			TokensGen: "go run ./" + CmdTokensGenRel,
		},
	}
}

func hexPreset(module string) Config {
	return Config{
		Module: module,
		Preset: PresetHex,
		Layers: map[string]string{
			"/domain":      LayerDomain,
			"/application": LayerApplication,
			"/adapter":     LayerAdapter,
			SuffixDelivery: LayerDelivery,
		},
		AllowedEdges: map[string][]string{
			LayerDomain:      {},
			LayerApplication: {LayerDomain},
			LayerAdapter:     {LayerApplication, LayerDomain},
			LayerDelivery:    {LayerAdapter, LayerApplication, LayerDomain},
		},
		LayerPaths: []string{
			DirInternal + "/" + LayerDomain,
			DirInternal + "/" + LayerApplication,
			DirInternal + "/" + LayerAdapter,
			DirInternal + "/" + LayerDelivery,
		},
		ContractFiles: []string{
			CursorRules,
			ConfigFileName,
		},
		AllowedProseDocs: []string{READMEFileName},
		ContractsDir:     ContractsDirDefault,
	}
}
