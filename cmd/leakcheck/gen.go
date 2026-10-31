package main

import (
	"encoding/json"
	"io"
	"path"
	"sort"
	"strings"
)

// Replacement classes for scrub-needed files (documented in docs/process/public-split.md).
const (
	ReplEnv         = "env-var"     // code and scripts: read the value from the environment / local.env
	ReplPlaceholder = "placeholder" // prose: generic words (the worker host) or example.lan names
	ReplRedact      = "redact"      // a person's name or schedule: remove
	ReplSplit       = "split"       // AGENTS.md: public rules stay, ops part moves to the ops repo
	ReplReview      = "manual"      // secret or email shapes: a human looks at each
)

func replaceClass(file string, classes map[string]bool) string {
	switch {
	case file == "AGENTS.md":
		return ReplSplit
	case classes["person"]:
		return ReplRedact
	case !classes["domain"] && !classes["ip"] && !classes["homepath"] && !classes["machine"]:
		return ReplReview
	}
	switch strings.ToLower(path.Ext(file)) {
	case ".md", ".txt", ".html", ".json", ".xml", ".yml", ".yaml":
		return ReplPlaceholder
	}
	return ReplEnv
}

// genManifest keeps the hand-kept rules (everything whose note is not "generated") and rewrites the generated
// scrub-needed entries, one per file that currently has hits and is not private-listed.
func genManifest(w io.Writer, old *Manifest, rep *Report) error {
	out := &Manifest{}
	var tail *ManifestRule // the catch-all public rule stays last
	if old != nil {
		for _, r := range old.Rules {
			switch {
			case r.Note == "generated":
			case r.Glob == "**":
				tail = r
			default:
				out.Rules = append(out.Rules, r)
			}
		}
	}
	byFile := map[string]map[string]bool{}
	for _, h := range rep.Hits {
		if byFile[h.Path] == nil {
			byFile[h.Path] = map[string]bool{}
		}
		byFile[h.Path][h.Class] = true
	}
	var files []string
	for f := range byFile {
		files = append(files, f)
	}
	sort.Strings(files)
	for _, f := range files {
		if r := old.Lookup(f); r != nil && r.Class == Priv {
			continue
		}
		out.Rules = append(out.Rules, &ManifestRule{Glob: f, Class: Scrub, Replace: replaceClass(f, byFile[f]), Note: "generated"})
	}
	if tail != nil {
		out.Rules = append(out.Rules, tail)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", " ")
	return enc.Encode(out)
}
