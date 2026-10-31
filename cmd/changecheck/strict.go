package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Strict holds openspec/.strict.yaml: per check group the date from which its findings are fatal in -cli mode, and the
// old merge commits that name no change and are therefore never reported. Parsed by hand (stdlib only, a tiny subset:
// "key: value" lines and one "- value" list).
type Strict struct {
	CardsFrom, MergesFrom time.Time
	AllowMerges           []string
	HistoryBaseline       string // commits up to and including this one are not checked (a fresh-history cut)
}

// strictGroup maps a finding's check name to its date key; checks not listed are never fatal by date.
func strictGroup(check string) string {
	switch check {
	case "card", "overlap", "spec", "owners", "status":
		return "cards_overlap_fatal_from"
	case "merge", "diff":
		return "merges_diff_fatal_from"
	}
	return ""
}

func loadStrict(root string) Strict {
	var s Strict
	b, err := readFile(filepath.Join(root, "openspec", ".strict.yaml"))
	if err != nil {
		return s
	}
	list := false
	for _, l := range strings.Split(b, "\n") {
		if i := strings.Index(l, "#"); i >= 0 {
			l = l[:i]
		}
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "- ") && list {
			s.AllowMerges = append(s.AllowMerges, strings.TrimSpace(t[2:]))
			continue
		}
		list = false
		k, v, ok := strings.Cut(t, ":")
		v = strings.TrimSpace(v)
		switch {
		case !ok:
		case k == "allow_merges":
			list = true
		case k == "history_baseline":
			s.HistoryBaseline = v
		case k == "cards_overlap_fatal_from":
			s.CardsFrom, _ = time.Parse("2006-01-02", v)
		case k == "merges_diff_fatal_from":
			s.MergesFrom, _ = time.Parse("2006-01-02", v)
		}
	}
	return s
}

// allowed reports whether merge hash h is on the allowlist (prefix match either way, hashes are abbreviated).
func (s Strict) allowed(h string) bool {
	for _, a := range s.AllowMerges {
		if a != "" && h != "" && (strings.HasPrefix(a, h) || strings.HasPrefix(h, a)) {
			return true
		}
	}
	return false
}

// fatal returns the findings that are fatal on day now, and one note per group that has gone strict.
func (s Strict) fatal(fs []Finding, now time.Time) (bad []Finding, notes []string) {
	on := map[string]bool{}
	for k, d := range map[string]time.Time{"cards_overlap_fatal_from": s.CardsFrom, "merges_diff_fatal_from": s.MergesFrom} {
		if !d.IsZero() && !now.Before(d) {
			on[k] = true
			notes = append(notes, fmt.Sprintf("%s checks are strict since %s (openspec/.strict.yaml)", map[string]string{"cards_overlap_fatal_from": "card/overlap/spec/owners/status", "merges_diff_fatal_from": "merge/diff"}[k], d.Format("2006-01-02")))
		}
	}
	for _, f := range fs {
		if g := strictGroup(f.Check); g != "" && on[g] {
			bad = append(bad, f)
		}
	}
	return bad, notes
}
