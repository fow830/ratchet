package report

import (
	"encoding/json"
	"fmt"
)

// ValidateSARIF checks SARIF payload has required top-level fields.
func ValidateSARIF(data []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("sarif json: %w", err)
	}
	if raw["version"] == nil && raw["$schema"] == nil {
		return fmt.Errorf("sarif: missing version/$schema")
	}
	runs, ok := raw["runs"].([]any)
	if !ok || len(runs) == 0 {
		return fmt.Errorf("sarif: missing runs")
	}
	return nil
}
