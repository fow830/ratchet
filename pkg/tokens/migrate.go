package tokens

// Migrate upgrades config to CurrentSchemaVersion applying defaults.
func Migrate(cfg Config) (Config, error) {
	if cfg.SchemaVersion == 0 {
		cfg.SchemaVersion = 1
	}
	if cfg.SchemaVersion < CurrentSchemaVersion {
		if cfg.Profile == "" {
			cfg.Profile = ProfileStandard
		}
		if cfg.ContractsDir == "" {
			cfg.ContractsDir = ContractsDirDefault
		}
		if cfg.AllowedProseDocs == nil {
			cfg.AllowedProseDocs = []string{READMEFileName}
		}
		if cfg.AllowedEdges == nil {
			cfg.AllowedEdges = map[string][]string{}
			seen := map[string]struct{}{}
			for _, layer := range cfg.Layers {
				if _, ok := seen[layer]; ok {
					continue
				}
				seen[layer] = struct{}{}
				cfg.AllowedEdges[layer] = []string{}
			}
		}
		cfg.SchemaVersion = CurrentSchemaVersion
	}
	if err := Validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
