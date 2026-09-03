package gates

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	coverageRe = regexp.MustCompile(`coverage:\s+([0-9.]+)%`)
	mutationRe = regexp.MustCompile(`(?i)Score:\s*([0-9.]+)%`)
)

// BenchEntry is a parsed benchmark with optional zero-alloc mark.
type BenchEntry struct {
	Name            string
	AllocsPerOp     int64
	MarkedZeroAlloc bool
}

// ParseCoverageTotal averages coverage percentages from go test -cover output.
func ParseCoverageTotal(out string) (float64, error) {
	matches := coverageRe.FindAllStringSubmatch(out, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("no coverage percentages found")
	}
	var sum float64
	for _, m := range matches {
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, err
		}
		sum += v
	}
	return sum / float64(len(matches)), nil
}

// ParseMutationScore extracts mutation score percent from go-mutesting output.
func ParseMutationScore(out string) (float64, error) {
	m := mutationRe.FindStringSubmatch(out)
	if len(m) != 2 {
		return 0, fmt.Errorf("no mutation score found")
	}
	return strconv.ParseFloat(m[1], 64)
}

// CheckZeroAlloc fails when marked benches allocate.
func CheckZeroAlloc(entries []BenchEntry) error {
	var bad []string
	for _, e := range entries {
		if e.MarkedZeroAlloc && e.AllocsPerOp > 0 {
			bad = append(bad, fmt.Sprintf("%s allocs/op=%d", e.Name, e.AllocsPerOp))
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf("zero-alloc budget exceeded: %s", strings.Join(bad, "; "))
	}
	return nil
}
