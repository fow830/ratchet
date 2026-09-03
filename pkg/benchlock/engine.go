package benchlock

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/fow830/ratchet/pkg/tokens"
)

// FileName is the on-disk bench baseline ledger.
const FileName = tokens.BenchFileName

var benchLine = regexp.MustCompile(`^(Benchmark\S+?)(?:-\d+)?\s+\d+\s+([0-9.]+)\s+ns/op\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op`)

// Entry is one benchmark measurement.
type Entry struct {
	Name        string `json:"name"`
	NsPerOp     int64  `json:"ns_per_op"`
	BytesPerOp  int64  `json:"bytes_per_op"`
	AllocsPerOp int64  `json:"allocs_per_op"`
}

// RegressedEntry describes a benchmark regression.
type RegressedEntry struct {
	Name     string `json:"name"`
	Baseline int64  `json:"baseline_ns_per_op"`
	Current  int64  `json:"current_ns_per_op"`
}

// Diff is the verify result.
type Diff struct {
	Regressed []RegressedEntry `json:"regressed"`
	Missing   []string         `json:"missing"`
}

// OK reports no regressions.
func (d Diff) OK() bool { return len(d.Regressed) == 0 && len(d.Missing) == 0 }

// Engine manages ratchet.bench.
type Engine struct {
	Root string
}

// New returns an Engine for root.
func New(root string) *Engine { return &Engine{Root: root} }

type lockFile struct {
	Version int     `json:"version"`
	Entries []Entry `json:"entries"`
}

// Lock writes baseline entries to ratchet.bench.
func (e *Engine) Lock(ctx context.Context, entries []Entry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	sorted := append([]Entry(nil), entries...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	lf := lockFile{Version: tokens.LockVersion, Entries: sorted}
	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(e.Root, FileName), data, tokens.FileModeFile)
}

// Verify compares current entries against ratchet.bench.
func (e *Engine) Verify(ctx context.Context, current []Entry) (Diff, error) {
	if err := ctx.Err(); err != nil {
		return Diff{}, err
	}
	data, err := os.ReadFile(filepath.Join(e.Root, FileName))
	if err != nil {
		return Diff{}, err
	}
	var lf lockFile
	if err := json.Unmarshal(data, &lf); err != nil {
		return Diff{}, err
	}
	base := map[string]Entry{}
	for _, ent := range lf.Entries {
		base[ent.Name] = ent
	}
	cur := map[string]Entry{}
	for _, ent := range current {
		cur[ent.Name] = ent
	}
	var diff Diff
	for name, b := range base {
		c, ok := cur[name]
		if !ok {
			diff.Missing = append(diff.Missing, name)
			continue
		}
		if c.NsPerOp > b.NsPerOp {
			diff.Regressed = append(diff.Regressed, RegressedEntry{
				Name: name, Baseline: b.NsPerOp, Current: c.NsPerOp,
			})
		}
	}
	sort.Strings(diff.Missing)
	sort.Slice(diff.Regressed, func(i, j int) bool { return diff.Regressed[i].Name < diff.Regressed[j].Name })
	return diff, nil
}

// ParseOutput parses go test -bench output lines.
func ParseOutput(text string) ([]Entry, error) {
	var out []Entry
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		m := benchLine.FindStringSubmatch(line)
		if len(m) != 5 {
			continue
		}
		nsFloat, err := strconv.ParseFloat(m[2], 64)
		if err != nil {
			return nil, err
		}
		ns := int64(nsFloat + 0.5)
		b, err := strconv.ParseInt(m[3], 10, 64)
		if err != nil {
			return nil, err
		}
		a, err := strconv.ParseInt(m[4], 10, 64)
		if err != nil {
			return nil, err
		}
		name := m[1]
		if i := strings.LastIndex(name, "-"); i > 0 {
			// strip CPU suffix if still present
			if _, err := strconv.Atoi(name[i+1:]); err == nil {
				name = name[:i]
			}
		}
		out = append(out, Entry{Name: name, NsPerOp: ns, BytesPerOp: b, AllocsPerOp: a})
	}
	return out, sc.Err()
}

func (d Diff) String() string {
	if d.OK() {
		return "benchlock: ok"
	}
	var b strings.Builder
	b.WriteString("benchlock: regression detected\n")
	for _, r := range d.Regressed {
		fmt.Fprintf(&b, "  regressed %s baseline=%dns current=%dns\n", r.Name, r.Baseline, r.Current)
	}
	for _, m := range d.Missing {
		fmt.Fprintf(&b, "  missing %s\n", m)
	}
	return b.String()
}
