// Package analyze provides escape-analysis hints from go build -gcflags=-m.
package analyze

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var escapeLine = regexp.MustCompile(`(.*?):(\d+):\d+: (.* escapes to heap)`)

// Hint is one escape-analysis finding.
type Hint struct {
	File    string `json:"file"`
	Line    string `json:"line"`
	Message string `json:"message"`
}

// ParseHints extracts escape-to-heap lines from compiler output.
func ParseHints(text string) []Hint {
	var hints []Hint
	for _, line := range strings.Split(text, "\n") {
		if m := escapeLine.FindStringSubmatch(line); len(m) == 4 {
			hints = append(hints, Hint{File: m[1], Line: m[2], Message: m[3]})
		}
	}
	return hints
}

// Run executes go build with escape analysis and parses hints.
func Run(ctx context.Context, root string) ([]Hint, error) {
	cmd := exec.CommandContext(ctx, "go", "build", "-gcflags=all=-m", "./...")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	text := string(out)
	if err != nil && !strings.Contains(text, "escapes to heap") && len(ParseHints(text)) == 0 {
		return nil, fmt.Errorf("analyze: %w\n%s", err, text)
	}
	return ParseHints(text), nil
}
