// Package workspace supports go.work monorepo discovery.
package workspace

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/fow830/ratchet/pkg/tokens"
)

// ModulesFromGoWork parses use directives from go.work.
func ModulesFromGoWork(root string) ([]string, error) {
	path := filepath.Join(root, tokens.GoWorkFileName)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var mods []string
	inUse := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "use (" {
			inUse = true
			continue
		}
		if inUse && line == ")" {
			break
		}
		if inUse && strings.HasPrefix(line, "./") {
			mods = append(mods, strings.TrimPrefix(line, "./"))
		}
	}
	return mods, sc.Err()
}
