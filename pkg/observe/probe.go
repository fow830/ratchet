// Package observe provides optional runtime observability probes (pprof/eBPF tools).
package observe

import (
	"context"
	"fmt"
	"os/exec"
)

// Required / optional tool names on PATH.
const (
	ToolGo     = "go"
	ToolPprof  = "pprof"
	ToolCilium = "cilium"
	ToolHubble = "hubble"
)

// ToolStatus reports whether an optional observe tool is on PATH.
type ToolStatus struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Path      string `json:"path,omitempty"`
}

// Report aggregates probe results.
type Report struct {
	Tools []ToolStatus `json:"tools"`
}

// Probe checks availability of runtime observe tooling.
// eBPF (cilium/hubble) is optional — missing tools are reported, not fatal by default.
func Probe(ctx context.Context) Report {
	names := []string{ToolGo, ToolPprof, ToolCilium, ToolHubble}
	var tools []ToolStatus
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			break
		}
		path, err := exec.LookPath(name)
		tools = append(tools, ToolStatus{Name: name, Available: err == nil, Path: path})
	}
	return Report{Tools: tools}
}

// RequireGo fails when go toolchain is missing (always required).
func RequireGo(r Report) error {
	for _, t := range r.Tools {
		if t.Name == ToolGo && t.Available {
			return nil
		}
	}
	return fmt.Errorf("%s toolchain not found on PATH", ToolGo)
}
