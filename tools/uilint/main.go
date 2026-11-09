// Command uilint checks the watch UI against the panel contract (docs/ui-design-system.md sections 6 and 9, the rule
// list in docs/ui-regression-plan.md section 2): layered CSS, panel scope, no colour literals outside tokens, no inline
// styles or scripts, valid panel manifests. Stdlib only, no CSS parser.
//
//	go run ./tools/uilint [-allow tools/uilint/allow.txt] [-go internal/watch] [internal/watch/web]
//
// Prints one `file:line: rule: message` per finding and exits 1 when any finding is not on the allow-list.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	allowFile := flag.String("allow", "tools/uilint/allow.txt", "allow-list of known violations (`file rule` per line); empty for none")
	goDir := flag.String("go", "internal/watch", "directory whose Go template strings are checked as HTML; empty to skip")
	repo := flag.String("repo", ".", "repository root (manifest doc paths and the unlock ladder are read from it)")
	flag.Parse()
	root := "internal/watch/web"
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}
	l := &Linter{Root: root, GoDir: *goDir, Repo: *repo, Ladder: filepath.Join(*repo, "data/unlocks/ladder.json")}
	fs, err := l.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "uilint:", err)
		os.Exit(2)
	}
	allow, err := readAllow(*allowFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "uilint:", err)
		os.Exit(2)
	}
	bad, used := 0, map[string]bool{}
	for _, f := range fs {
		k := f.File + " " + f.Rule
		if allow[k] {
			used[k] = true
			continue
		}
		fmt.Println(f)
		bad++
	}
	for k := range allow {
		if !used[k] {
			// a warning only: another branch may have fixed the file first (the list only shrinks)
			fmt.Fprintf(os.Stderr, "%s: warning: allow-list entry %q matches nothing; remove it\n", *allowFile, k)
		}
	}
	if bad > 0 {
		fmt.Printf("uilint: %d finding(s)\n", bad)
		os.Exit(1)
	}
}

// readAllow reads `file rule` pairs; '#' starts a comment. The list only shrinks (docs/ui-regression-plan.md PR 2).
func readAllow(path string) (map[string]bool, error) {
	out := map[string]bool{}
	if path == "" {
		return out, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		t, _, _ := strings.Cut(sc.Text(), "#")
		if fs := strings.Fields(t); len(fs) == 2 {
			out[fs[0]+" "+fs[1]] = true
		} else if len(fs) != 0 {
			return nil, fmt.Errorf("%s: bad line %q (want `file rule`)", path, sc.Text())
		}
	}
	return out, sc.Err()
}
