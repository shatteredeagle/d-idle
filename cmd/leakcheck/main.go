// Command leakcheck scans the tracked files for private strings (hostnames, machine and people names, private IPs,
// home paths, token shapes) so the repo stays public-clean by construction. See docs/process/public-split.md.
//
//	go run ./cmd/leakcheck                         # warn mode: report, exit 0
//	go run ./cmd/leakcheck -mode fail              # exit 1 on any hit (switched on at go-live)
//	go run ./cmd/leakcheck -staged                 # only files in the index; with -range A..B only files changed there
//	go run ./cmd/leakcheck -json | -list
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
)

const maxSize = 1 << 20

type ClassCount struct {
	Hits  int `json:"hits"`
	Files int `json:"files"`
}

type Report struct {
	Mode           string                `json:"mode"`
	FilesScanned   int                   `json:"files_scanned"`
	SkippedPrivate int                   `json:"skipped_private"`
	SkippedAllowed int                   `json:"skipped_allowed"`
	Counts         map[string]ClassCount `json:"counts"`
	TotalHits      int                   `json:"total_hits"`
	FilesWithHits  int                   `json:"files_with_hits"`
	// ScrubStatus counts files with hits by manifest class (public means: not listed, so a new leak).
	ScrubStatus map[string]int `json:"files_with_hits_by_manifest_class"`
	Hits        []Hit          `json:"hits,omitempty"`
}

func git(args ...string) ([]string, error) {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, err
	}
	var l []string
	for _, s := range strings.Split(string(out), "\n") {
		if s != "" {
			l = append(l, s)
		}
	}
	return l, nil
}

func isText(b []byte) bool { return !bytes.Contains(b[:min(len(b), 8000)], []byte{0}) }

// run is the testable core: files are read from root.
func run(root string, files []string, allow *Allow, man *Manifest, mode string) *Report {
	rep := &Report{Mode: mode, Counts: map[string]ClassCount{}, ScrubStatus: map[string]int{}}
	perClassFiles := map[string]map[string]bool{}
	for _, f := range files {
		if allow.skipPath(f) {
			rep.SkippedAllowed++
			continue
		}
		cls := Public
		if r := man.Lookup(f); r != nil {
			cls = r.Class
		}
		if cls == Priv {
			rep.SkippedPrivate++
			continue
		}
		b, err := os.ReadFile(root + "/" + f)
		if err != nil || len(b) > maxSize || !isText(b) {
			continue
		}
		rep.FilesScanned++
		hits := scanText(f, string(b), allow.Strs)
		if len(hits) == 0 {
			continue
		}
		rep.FilesWithHits++
		rep.ScrubStatus[cls]++
		for _, h := range hits {
			c := rep.Counts[h.Class]
			c.Hits++
			if perClassFiles[h.Class] == nil {
				perClassFiles[h.Class] = map[string]bool{}
			}
			if !perClassFiles[h.Class][f] {
				perClassFiles[h.Class][f] = true
				c.Files++
			}
			rep.Counts[h.Class] = c
			rep.TotalHits++
		}
		rep.Hits = append(rep.Hits, hits...)
	}
	return rep
}

func (r *Report) print(w io.Writer, list bool) {
	if list {
		for _, h := range r.Hits {
			fmt.Fprintf(w, "%s:%d: %s %s\n", h.Path, h.Line, h.Class, h.Text)
		}
	}
	var cl []string
	for c := range r.Counts {
		cl = append(cl, c)
	}
	sort.Strings(cl)
	fmt.Fprintf(w, "leakcheck (%s): %d files scanned, %d private-listed and %d allow-listed skipped, %d hits in %d files\n",
		r.Mode, r.FilesScanned, r.SkippedPrivate, r.SkippedAllowed, r.TotalHits, r.FilesWithHits)
	for _, c := range cl {
		fmt.Fprintf(w, "  %-9s %5d hits %4d files\n", c, r.Counts[c].Hits, r.Counts[c].Files)
	}
	if len(r.ScrubStatus) > 0 {
		fmt.Fprintf(w, "  files with hits by manifest class: %v\n", r.ScrubStatus)
	}
}

func main() {
	mode := flag.String("mode", "warn", "warn (report, exit 0) or fail (exit 1 on any hit)")
	staged := flag.Bool("staged", false, "scan only changed files: the index, or the files changed in -range")
	rng := flag.String("range", "", "with -staged: commit range A..B (a push)")
	asJSON := flag.Bool("json", false, "JSON output")
	list := flag.Bool("list", false, "list every hit, not only the counts")
	allowPath := flag.String("allow", "docs/process/leakcheck-allow.txt", "allow list")
	gen := flag.Bool("gen-manifest", false, "print the manifest with the generated scrub-needed entries refreshed from the current hits")
	manPath := flag.String("manifest", "docs/process/public-manifest.json", "manifest (files classed private are not scanned)")
	flag.Parse()
	if *mode != "warn" && *mode != "fail" {
		fmt.Fprintln(os.Stderr, "leakcheck: -mode must be warn or fail")
		os.Exit(2)
	}
	top, err := git("rev-parse", "--show-toplevel")
	if err != nil || len(top) == 0 {
		fmt.Fprintln(os.Stderr, "leakcheck: not in a git repository")
		os.Exit(2)
	}
	root := top[0]
	var files []string
	switch {
	case *staged && *rng != "" && !strings.HasPrefix(*rng, "0000000000"):
		files, err = git("diff", "--name-only", "--diff-filter=d", *rng)
	case *staged:
		files, err = git("diff", "--name-only", "--cached", "--diff-filter=d")
	default:
		files, err = git("-C", root, "ls-files")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "leakcheck: git:", err)
		os.Exit(2)
	}
	allow := &Allow{}
	if b, err := os.ReadFile(root + "/" + *allowPath); err == nil {
		allow = parseAllow(string(b))
	}
	man, _ := loadManifest(root + "/" + *manPath)
	if *gen {
		// hits of every non-private file, so run without the manifest's private skips being re-decided by stale generated rules
		rep := run(root, files, allow, &Manifest{Rules: filterHand(man)}, *mode)
		if err := genManifest(os.Stdout, man, rep); err != nil {
			fmt.Fprintln(os.Stderr, "leakcheck:", err)
			os.Exit(2)
		}
		return
	}
	rep := run(root, files, allow, man, *mode)
	if *asJSON {
		json.NewEncoder(os.Stdout).Encode(rep)
	} else {
		rep.print(os.Stdout, *list)
	}
	if *mode == "fail" && rep.TotalHits > 0 {
		os.Exit(1)
	}
}

// filterHand returns the hand-kept rules only.
func filterHand(m *Manifest) []*ManifestRule {
	var l []*ManifestRule
	if m != nil {
		for _, r := range m.Rules {
			if r.Note != "generated" {
				l = append(l, r)
			}
		}
	}
	return l
}
