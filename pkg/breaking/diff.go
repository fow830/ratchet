// Package breaking detects breaking architecture/config changes.
package breaking

import (
	"fmt"
	"sort"

	"github.com/fow830/ratchet/pkg/tokens"
)

// Change describes a breaking delta.
type Change struct {
	Field  string `json:"field"`
	Before string `json:"before"`
	After  string `json:"after"`
}

// DiffConfig compares two configs and returns breaking changes.
func DiffConfig(old, newCfg tokens.Config) []Change {
	var out []Change
	out = append(out, diffEdges(old.AllowedEdges, newCfg.AllowedEdges)...)
	out = append(out, diffLayers(old.Layers, newCfg.Layers)...)
	return out
}

func diffEdges(old, new map[string][]string) []Change {
	var out []Change
	keys := map[string]struct{}{}
	for k := range old {
		keys[k] = struct{}{}
	}
	for k := range new {
		keys[k] = struct{}{}
	}
	for k := range keys {
		if fmt.Sprint(old[k]) != fmt.Sprint(new[k]) {
			out = append(out, Change{Field: "allowed_edges." + k, Before: fmt.Sprint(old[k]), After: fmt.Sprint(new[k])})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Field < out[j].Field })
	return out
}

func diffLayers(old, new map[string]string) []Change {
	var out []Change
	for suffix, layer := range old {
		if new[suffix] != layer {
			out = append(out, Change{Field: "layers" + suffix, Before: layer, After: new[suffix]})
		}
	}
	for suffix, layer := range new {
		if old[suffix] == "" {
			out = append(out, Change{Field: "layers" + suffix, Before: "", After: layer})
		}
	}
	return out
}
