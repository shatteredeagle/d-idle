// Package overlay is the bot side of the d.idle table overlay (docs/overlay.md): the same fragment merge and
// the same hash as the d2d fork's --overlay-dir (components/mpq/overlay.hpp in the fork), plus the mode stamp
// ("vanilla" or "didle:<hash8>") that saves, run records and probes carry. Our own rows only.
package overlay

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Env is the environment twin of d2bot --overlay-dir; internal/data reads it.
const Env = "D2_OVERLAY_DIR"

// Fragment is one excel/<Table>.txt file of an overlay directory.
type Fragment struct {
	File  string // file name as on disk, e.g. UniqueItems.txt
	Table string // lower-case table name
	Text  string
}

// keyColumn is the key per table (lower-case name). "" marks an append-only table (rows have no unique name);
// an unlisted table keys on its first column. Keep in step with the fork's key_column.
var keyColumn = map[string]string{
	"misc": "code", "armor": "code", "weapons": "code", "gems": "code", "itemtypes": "Code", "properties": "code",
	"uniqueitems": "index", "setitems": "index", "sets": "index",
	"treasureclassex": "Treasure Class", "runes": "*Rune Name",
	"itemstatcost": "Stat", "superuniques": "Superunique", "monstats": "Id", "skills": "skill",
	"cubemain": "", "magicprefix": "", "magicsuffix": "", "automagic": "",
}

// Load reads <dir>/excel/*.txt sorted by lower-case file name. An empty dir means overlay off: (nil, nil).
func Load(dir string) ([]Fragment, error) {
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(filepath.Join(dir, "excel"))
	if err != nil {
		return nil, fmt.Errorf("overlay: %w", err)
	}
	var out []Fragment
	for _, e := range entries {
		if e.IsDir() || strings.ToLower(filepath.Ext(e.Name())) != ".txt" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, "excel", e.Name()))
		if err != nil {
			return nil, fmt.Errorf("overlay: %w", err)
		}
		out = append(out, Fragment{File: e.Name(), Table: strings.ToLower(strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))), Text: string(b)})
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].File) < strings.ToLower(out[j].File) })
	return out, nil
}

// Hash8 is FNV-1a 64 over, per fragment, lower-case file name, 0, bytes, 0; the first 8 hex digits of 16.
// Same value as the fork's overlay=<hash8>. Empty for no fragments dir (use Mode for the stamp).
func Hash8(frags []Fragment) string {
	h := uint64(14695981039346656037)
	feed := func(s string) {
		for i := 0; i < len(s); i++ {
			h ^= uint64(s[i])
			h *= 1099511628211
		}
	}
	for _, f := range frags {
		feed(strings.ToLower(f.File))
		feed("\x00")
		feed(f.Text)
		feed("\x00")
	}
	return fmt.Sprintf("%016x", h)[:8]
}

// Mode is the stamp for saves, run records and probes: "vanilla" without an overlay, else "didle:<hash8>".
func Mode(dir string) (string, error) {
	if dir == "" {
		return "vanilla", nil
	}
	frags, err := Load(dir)
	if err != nil {
		return "", err
	}
	return "didle:" + Hash8(frags), nil
}

// Merged is one merge result.
type Merged struct {
	Text               string
	Replaced, Appended int
}

func split(text string) (lines [][]string, eol string) {
	eol = "\r\n"
	if !strings.Contains(text, "\r\n") && strings.Contains(text, "\n") {
		eol = "\n"
	}
	for len(text) > 0 {
		var line string
		if i := strings.IndexByte(text, '\n'); i >= 0 {
			line, text = text[:i], text[i+1:]
		} else {
			line, text = text, ""
		}
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			lines = append(lines, nil)
			continue
		}
		lines = append(lines, strings.Split(line, "\t"))
	}
	return lines, eol
}

func index(header []string, name string) int {
	for i, h := range header {
		if strings.EqualFold(h, name) {
			return i
		}
	}
	return -1
}

// Merge applies fragment f to the raw base table text: rows with a key present in the base replace its
// non-empty cells, other rows are appended. Errors name the file and the 1-based line.
func Merge(base string, f Fragment) (Merged, error) {
	fail := func(line int, format string, a ...any) (Merged, error) {
		return Merged{}, fmt.Errorf("overlay %s:%d: %s", f.File, line, fmt.Sprintf(format, a...))
	}
	bl, eol := split(base)
	fl, _ := split(f.Text)
	if len(bl) == 0 || len(bl[0]) == 0 {
		return fail(0, "base table %s has no header", f.Table)
	}
	if len(fl) == 0 || len(fl[0]) == 0 {
		return fail(1, "empty fragment (no header line)")
	}
	bh, fh := bl[0], fl[0]
	toBase := make([]int, len(fh))
	for c, h := range fh {
		if toBase[c] = index(bh, h); toBase[c] < 0 {
			return fail(1, "column %q is not in the base table header", h)
		}
	}
	keyName, listed := keyColumn[f.Table]
	appendOnly := listed && keyName == ""
	keyBase, keyFrag := 0, 0
	if !appendOnly {
		if listed {
			if keyBase = index(bh, keyName); keyBase < 0 {
				return fail(1, "key column %q is not in the base table header", keyName)
			}
		}
		keyFrag = -1
		for c, b := range toBase {
			if b == keyBase {
				keyFrag = c
			}
		}
		if keyFrag < 0 {
			return fail(1, "the fragment needs the key column %q", bh[keyBase])
		}
	}
	rowOf := map[string]int{}
	if !appendOnly {
		for i := 1; i < len(bl); i++ {
			if keyBase < len(bl[i]) && bl[i][keyBase] != "" {
				if _, ok := rowOf[bl[i][keyBase]]; !ok {
					rowOf[bl[i][keyBase]] = i
				}
			}
		}
	}
	var out Merged
	seen := map[string]bool{}
	for n := 1; n < len(fl); n++ {
		row := fl[n]
		if row == nil {
			continue
		}
		if len(row) != len(fh) {
			return fail(n+1, "%d cells, header has %d", len(row), len(fh))
		}
		if !appendOnly {
			key := row[keyFrag]
			if key == "" {
				return fail(n+1, "empty key in column %q", bh[keyBase])
			}
			if seen[key] {
				return fail(n+1, "duplicate key %q in this fragment", key)
			}
			seen[key] = true
			if at, ok := rowOf[key]; ok {
				for len(bl[at]) < len(bh) {
					bl[at] = append(bl[at], "")
				}
				for c, v := range row {
					if v != "" {
						bl[at][toBase[c]] = v
					}
				}
				out.Replaced++
				continue
			}
			rowOf[key] = len(bl)
		}
		added := make([]string, len(bh))
		for c, v := range row {
			added[toBase[c]] = v
		}
		bl = append(bl, added)
		out.Appended++
	}
	var sb strings.Builder
	for _, l := range bl {
		sb.WriteString(strings.Join(l, "\t"))
		sb.WriteString(eol)
	}
	out.Text = sb.String()
	return out, nil
}
