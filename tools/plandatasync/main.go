// Command plandatasync makes docs/plan/data a generated mirror of internal/plandata (the single source).
// `go run ./tools/plandatasync` rewrites every mirror file that exists; `-check` only reports drift (exit 1).
// A mirror is docs/plan/data/<name>, else docs/plan/data/builds-endgame/<name without "build-">. It never creates
// files and never touches the docs-only tables that have no plandata twin. sharedgen runs it at merge.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"d2lod-bot/internal/plandata"
)

func mirrorPath(root, name string) string {
	dir := filepath.Join(root, "docs", "plan", "data")
	for _, p := range []string{filepath.Join(dir, name), filepath.Join(dir, "builds-endgame", strings.TrimPrefix(name, "build-"))} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// sync returns the mirrors that differ from the source; it rewrites them unless check is set.
func sync(root string, check bool) ([]string, error) {
	ents, err := plandata.FS.ReadDir(".")
	if err != nil {
		return nil, err
	}
	var changed []string
	for _, e := range ents {
		if e.Name() == plandata.CatalogFile {
			continue // UI text, no mirror
		}
		p := mirrorPath(root, e.Name())
		if p == "" {
			continue
		}
		want, _ := plandata.FS.ReadFile(e.Name())
		have, _ := os.ReadFile(p)
		if bytes.Equal(want, have) {
			continue
		}
		changed = append(changed, p)
		if !check {
			if err := os.WriteFile(p, want, 0o644); err != nil {
				return changed, err
			}
		}
	}
	return changed, nil
}

func main() {
	check := flag.Bool("check", false, "report drift only")
	flag.Parse()
	ch, err := sync(".", *check)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	for _, p := range ch {
		fmt.Println(p)
	}
	if *check && len(ch) > 0 {
		os.Exit(1)
	}
}
