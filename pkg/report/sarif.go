package report

import (
	"encoding/json"
	"fmt"

	"github.com/fow830/ratchet/pkg/tokens"
)

const (
	sarifSchema  = "https://json.schemastore.org/sarif-2.1.0.json"
	sarifVersion = "2.1.0"
)

// MarshalSARIF builds a SARIF 2.1.0 document for GitHub Code Scanning.
func MarshalSARIF(r Result) ([]byte, error) {
	r.Normalize()
	runs := []sarifRun{{
		Tool: sarifTool{Driver: sarifDriver{
			Name:           tokens.ToolName,
			InformationURI: tokens.ModuleHTTPSURL(),
			Rules: []sarifRule{
				{ID: RuleLayerIsolation, Name: RuleLayerIsolation, ShortDescription: sarifText{Text: "Illegal cross-layer import"}},
				{ID: RuleAntiDrift, Name: RuleAntiDrift, ShortDescription: sarifText{Text: "Generated contract drift"}},
			},
		}},
		Results: make([]sarifResult, 0, len(r.Violations)+len(r.Drift.Changed)+len(r.Drift.Missing)+len(r.Drift.Extra)),
	}}

	for _, v := range r.Violations {
		runs[0].Results = append(runs[0].Results, sarifResult{
			RuleID:  RuleLayerIsolation,
			Level:   "error",
			Message: sarifText{Text: v.String()},
			Locations: []sarifLocation{{
				PhysicalLocation: sarifPhysical{
					ArtifactLocation: sarifArtifact{URI: toFileURI(v.File)},
					Region:           sarifRegion{StartLine: max(1, v.Line)},
				},
			}},
		})
	}
	for _, c := range r.Drift.Changed {
		runs[0].Results = append(runs[0].Results, sarifResult{
			RuleID:  RuleAntiDrift,
			Level:   "error",
			Message: sarifText{Text: fmt.Sprintf("contract hash mismatch for %s", c.Path)},
			Locations: []sarifLocation{{
				PhysicalLocation: sarifPhysical{
					ArtifactLocation: sarifArtifact{URI: toFileURI(c.Path)},
					Region:           sarifRegion{StartLine: 1},
				},
			}},
		})
	}
	for _, m := range r.Drift.Missing {
		runs[0].Results = append(runs[0].Results, sarifResult{
			RuleID:  RuleAntiDrift,
			Level:   "error",
			Message: sarifText{Text: fmt.Sprintf("locked contract missing: %s", m)},
			Locations: []sarifLocation{{
				PhysicalLocation: sarifPhysical{
					ArtifactLocation: sarifArtifact{URI: toFileURI(m)},
					Region:           sarifRegion{StartLine: 1},
				},
			}},
		})
	}
	for _, e := range r.Drift.Extra {
		runs[0].Results = append(runs[0].Results, sarifResult{
			RuleID:  RuleAntiDrift,
			Level:   "warning",
			Message: sarifText{Text: fmt.Sprintf("unlocked contract present: %s", e)},
			Locations: []sarifLocation{{
				PhysicalLocation: sarifPhysical{
					ArtifactLocation: sarifArtifact{URI: toFileURI(e)},
					Region:           sarifRegion{StartLine: 1},
				},
			}},
		})
	}

	doc := sarifDocument{
		Schema:  sarifSchema,
		Version: sarifVersion,
		Runs:    runs,
	}
	return json.MarshalIndent(doc, "", "  ")
}

type sarifDocument struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	InformationURI string      `json:"informationUri,omitempty"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	ShortDescription sarifText `json:"shortDescription"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifText       `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifText struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysical `json:"physicalLocation"`
}

type sarifPhysical struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
	Region           sarifRegion   `json:"region"`
}

type sarifArtifact struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}
