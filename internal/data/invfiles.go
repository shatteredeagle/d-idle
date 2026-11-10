package data

import (
	"sort"
	"strings"
)

// InvFiles lists every inventory icon name (lower case, no extension) the tables can name: the base items'
// invfile, uniqueinvfile and setinvfile plus the unique and set item overrides. It is the set of icon keys the
// watch page can ask for (internal/art pre-extraction); InvFile picks one of them per item.
func (db *DB) InvFiles() []string {
	if db == nil {
		return nil
	}
	seen := map[string]bool{}
	add := func(n string) {
		if n = strings.ToLower(strings.TrimSpace(n)); n != "" {
			seen[n] = true
		}
	}
	for _, b := range db.Items {
		add(b.InvFile)
		add(b.UniqueInvFile)
		add(b.SetInvFile)
	}
	for _, u := range db.Uniques {
		add(u.InvFile)
	}
	for _, s := range db.SetItems {
		add(s.InvFile)
	}
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
