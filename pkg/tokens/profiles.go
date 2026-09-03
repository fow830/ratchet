package tokens

import "fmt"

// NormalizeProfile returns a canonical profile name.
func NormalizeProfile(p string) string {
	switch p {
	case ProfileMinimal, ProfileStandard, ProfileService, ProfileAPI, ProfileStrict, ProfileParanoid:
		return p
	case "":
		return ProfileStandard
	default:
		return ProfileStandard
	}
}

// ProfileGates returns default hard gates for a profile.
func ProfileGates(profile string) GateFlags {
	switch NormalizeProfile(profile) {
	case ProfileMinimal:
		return GateFlags{Arch: true, Lock: true}
	case ProfileService:
		return GateFlags{
			Arch: true, Lock: true, Docs: true, Contracts: true, Vet: true,
			LayerPaths: true, Cycles: true, SQLC: true, Testcontainers: true,
		}
	case ProfileAPI:
		return GateFlags{
			Arch: true, Lock: true, Docs: true, Contracts: true, Vet: true,
			LayerPaths: true, Cycles: true, Buf: true, OpenAPI: true, CUE: true,
		}
	case ProfileStrict:
		return GateFlags{
			Arch: true, Lock: true, Docs: true, Contracts: true, Vet: true,
			Staticcheck: true, Race: true, Fuzz: true, PBT: true, Cycles: true,
			LayerPaths: true, External: true, Coverage: true, Govuln: true,
		}
	case ProfileParanoid:
		g := ProfileGates(ProfileStrict)
		g.Mutation = true
		g.Bench = true
		g.WASM = true
		g.Testcontainers = true
		g.SQLC = true
		g.Buf = true
		return g
	default: // standard
		return GateFlags{
			Arch: true, Lock: true, Docs: true, Vet: true, Cycles: true, LayerPaths: true,
		}
	}
}

// AllProfiles lists supported profile names.
func AllProfiles() []string {
	return []string{
		ProfileMinimal, ProfileStandard, ProfileService, ProfileAPI, ProfileStrict, ProfileParanoid,
	}
}

// ValidatePreset reports whether preset name is known.
func ValidatePreset(preset string) error {
	switch preset {
	case "", PresetClean, PresetVitek, PresetHex:
		return nil
	default:
		return fmt.Errorf("unknown preset %q (want %s|%s|%s)", preset, PresetClean, PresetVitek, PresetHex)
	}
}
