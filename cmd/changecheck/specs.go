package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Finding is one warning; Check names the rule (card, overlap, spec, merge, owners, status, diff).
type Finding struct{ Check, Msg string }

var (
	reqRe      = regexp.MustCompile(`^### Requirement:\s*(.+?)\s*#*\s*$`)
	scenarioRe = regexp.MustCompile(`^#### Scenario:\s*(.+?)\s*$`)
	shallRe    = regexp.MustCompile(`\b(SHALL|MUST)\b`)
	deltaSecs  = map[string]bool{"ADDED": true, "MODIFIED": true, "REMOVED": true, "RENAMED": true}
)

type reqBlock struct {
	name      string
	text      string
	scenarios int
	emptyScn  int
	section   string
}

// parseRequirements splits a spec or delta file into requirement blocks, tagging each with its
// "## X Requirements" section (empty for a main spec).
func parseRequirements(md string) (blocks []*reqBlock, headings map[string]bool) {
	headings = map[string]bool{}
	var cur *reqBlock
	sec := ""
	inScn := false
	scnBody := 0
	closeScn := func() {
		if inScn && cur != nil && scnBody == 0 {
			cur.emptyScn++
		}
		inScn, scnBody = false, 0
	}
	for _, l := range strings.Split(md, "\n") {
		if strings.HasPrefix(l, "## ") {
			closeScn()
			cur = nil
			h := strings.TrimSpace(l[3:])
			headings[h] = true
			sec = ""
			if f := strings.Fields(h); len(f) == 2 && f[1] == "Requirements" && deltaSecs[f[0]] {
				sec = f[0]
			}
			continue
		}
		if m := reqRe.FindStringSubmatch(l); m != nil {
			closeScn()
			cur = &reqBlock{name: m[1], section: sec}
			blocks = append(blocks, cur)
			continue
		}
		if cur == nil {
			continue
		}
		if scenarioRe.MatchString(l) {
			closeScn()
			cur.scenarios++
			inScn = true
			continue
		}
		if strings.HasPrefix(l, "#") {
			closeScn()
			continue
		}
		if inScn {
			if strings.TrimSpace(l) != "" {
				scnBody++
			}
		} else if cur.scenarios == 0 {
			cur.text += l + "\n"
		}
	}
	closeScn()
	return blocks, headings
}

// checkSpecText applies the OpenSpec validator rules (V: docs/process/openspec-reference.md 1.3, 1.5).
func checkSpecText(rel, md string, delta bool) (fs []Finding) {
	add := func(f string, a ...any) { fs = append(fs, Finding{"spec", rel + ": " + fmt.Sprintf(f, a...)}) }
	blocks, heads := parseRequirements(md)
	if !delta {
		if !heads["Purpose"] {
			add("missing ## Purpose")
		} else if p, _ := section(md, "Purpose"); len(p) < 50 || strings.Contains(p, "TBD") {
			add("## Purpose is under 50 characters or TBD")
		}
		if !heads["Requirements"] {
			add("missing ## Requirements")
		}
		if len(blocks) == 0 {
			add("no ### Requirement: block")
		}
	} else {
		found := false
		for h := range heads {
			if f := strings.Fields(h); len(f) == 2 && f[1] == "Requirements" && deltaSecs[f[0]] {
				found = true
			}
		}
		if !found {
			add("delta has no ADDED/MODIFIED/REMOVED/RENAMED Requirements section")
		}
	}
	seen := map[string]bool{}
	for _, b := range blocks {
		if seen[b.name] {
			add("duplicate requirement %q", b.name)
		}
		seen[b.name] = true
		if delta && (b.section == "REMOVED" || b.section == "RENAMED") {
			continue
		}
		if !shallRe.MatchString(b.text) {
			add("requirement %q has no SHALL or MUST", b.name)
		}
		if len(strings.TrimSpace(b.text)) > 500 {
			add("requirement %q text is over 500 characters", b.name)
		}
		if b.scenarios == 0 {
			add("requirement %q has no #### Scenario:", b.name)
		}
		if b.emptyScn > 0 {
			add("requirement %q has an empty scenario", b.name)
		}
	}
	return fs
}

// checkSpecs validates openspec/specs/*/spec.md and every delta under active changes.
func checkSpecs(root string) (fs []Finding) {
	walk := func(dir string, delta bool) {
		_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || d.Name() != "spec.md" {
				return nil
			}
			rel, _ := filepath.Rel(root, p)
			b, rerr := os.ReadFile(p)
			if rerr != nil {
				fs = append(fs, Finding{"spec", rel + ": " + rerr.Error()})
				return nil
			}
			fs = append(fs, checkSpecText(filepath.ToSlash(rel), string(b), delta)...)
			return nil
		})
	}
	walk(filepath.Join(root, "openspec", "specs"), false)
	ents, _ := os.ReadDir(filepath.Join(root, "openspec", "changes"))
	for _, e := range ents {
		if e.IsDir() && e.Name() != "archive" {
			walk(filepath.Join(root, "openspec", "changes", e.Name(), "specs"), true)
		}
	}
	return fs
}

// specSummaries lists capability name and requirement count for the STATUS block.
func specSummaries(root string) (names []string, counts map[string]int) {
	counts = map[string]int{}
	ents, _ := os.ReadDir(filepath.Join(root, "openspec", "specs"))
	for _, e := range ents {
		b, err := os.ReadFile(filepath.Join(root, "openspec", "specs", e.Name(), "spec.md"))
		if err != nil {
			continue
		}
		bl, _ := parseRequirements(string(b))
		names = append(names, e.Name())
		counts[e.Name()] = len(bl)
	}
	return names, counts
}
