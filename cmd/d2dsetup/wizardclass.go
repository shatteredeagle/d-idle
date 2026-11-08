package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// The class list comes from the plan tables (internal/plandata build-*.json), not from a literal (decision F11,
// TASK-145). A class is offered when it has a normal30 leveling table (build name contains "normal30"). The Sorceress
// got hers with TASK-144 (sorcnormal30); a class whose only table is a fallback one (build-fallback-*.json) is still
// offered, marked as such. A class with neither is not offered. A table with no "class" key is the Sorceress's, as in
// leveler.ParseBuild.
const (
	tableNormal30 = "normal30"
	tableFallback = "fallback"
)

const fallbackNote = "Plays the older fallback leveling table (Lightning) until a normal30 table exists for this class."

type planHeader struct {
	Build string `json:"build"`
	Class string `json:"class"`
}

// planTables maps class id to its best table kind found in the plan files (normal30 wins over fallback).
func planTables(fsys fs.FS) (map[string]string, error) {
	ents, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, e := range ents {
		n := e.Name()
		if !strings.HasPrefix(n, "build-") || !strings.HasSuffix(n, ".json") {
			continue
		}
		raw, err := fs.ReadFile(fsys, n)
		if err != nil {
			return nil, err
		}
		var h planHeader
		if err := json.Unmarshal(raw, &h); err != nil {
			return nil, fmt.Errorf("plan table %s: %w", n, err)
		}
		class := strings.ToLower(h.Class)
		if class == "" {
			class = "sorceress"
		}
		switch {
		case strings.Contains(h.Build, tableNormal30):
			out[class] = tableNormal30
		case strings.HasPrefix(n, "build-fallback") && out[class] == "":
			out[class] = tableFallback
		}
	}
	return out, nil
}

// withPlan keeps the catalog classes that have a table and fills Supported, Table and Note.
func withPlan(cat []Class, kinds map[string]string) []Class {
	var out []Class
	for _, c := range cat {
		k := kinds[c.ID]
		if k == "" {
			continue
		}
		c.Supported, c.Table, c.Note = true, k, ""
		if k == tableFallback {
			c.Note = fallbackNote
		}
		out = append(out, c)
	}
	return out
}

// planClasses lists the class ids with a table, sorted (used by tests to compare against the catalog).
func planClasses(fsys fs.FS) ([]string, error) {
	k, err := planTables(fsys)
	if err != nil {
		return nil, err
	}
	var ids []string
	for id := range k {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}
