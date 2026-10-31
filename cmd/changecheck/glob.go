package main

import (
	"regexp"
	"strings"
)

// prefix is the literal part of a glob before its first wildcard.
func prefix(g string) string {
	if i := strings.IndexAny(g, "*?["); i >= 0 {
		return g[:i]
	}
	return g
}

// globsOverlap is conservative: two globs overlap when one literal prefix is a prefix of the other,
// which can report a false clash for patterns like a/*b and a/*c but never misses a real one.
func globsOverlap(a, b string) bool {
	pa, pb := prefix(a), prefix(b)
	return strings.HasPrefix(pa, pb) || strings.HasPrefix(pb, pa)
}

// globRegexp turns a path glob (** across directories, * within one, trailing / = whole tree) into a regexp.
func globRegexp(g string) *regexp.Regexp {
	if strings.HasSuffix(g, "/") {
		g += "**"
	}
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(g); i++ {
		switch {
		case strings.HasPrefix(g[i:], "**"):
			b.WriteString(".*")
			i++
		case g[i] == '*':
			b.WriteString("[^/]*")
		case g[i] == '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(g[i])))
		}
	}
	b.WriteString("$")
	return regexp.MustCompile(b.String())
}

func matchAny(globs []string, path string) bool {
	for _, g := range globs {
		if globRegexp(g).MatchString(path) {
			return true
		}
	}
	return false
}
