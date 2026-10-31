package data

import "strings"

// MagicNameLevelReq reads the affixes out of the display name of a MAGIC item ("Sturdy Cap of the Fox": prefix, base
// display name, suffix) and returns the highest levelreq of those rows in magicprefix.txt / magicsuffix.txt, with ok
// true. ok is false when the name does not contain the base's display name or an affix text matches no row, so the
// caller must not trust the 0 then. The item's level requirement is max(base levelreq, this): Str and Dex come from the
// base only (S, d2-sme answer of 2026-10-30; EXP: not checked live).
//
// Several rows can share one display text (different item types or levels); a row whose itype list covers the base type
// is preferred, and among equals the highest levelreq wins (conservative).
func (db *DB) MagicNameLevelReq(b *ItemBase, name string) (levelReq int, ok bool) {
	if b == nil || name == "" {
		return 0, false
	}
	base := b.Display
	if base == "" {
		base = b.Name
	}
	i := strings.Index(strings.ToLower(name), strings.ToLower(base))
	if base == "" || i < 0 {
		return 0, false
	}
	pre := strings.TrimSpace(name[:i])
	suf := strings.TrimSpace(name[i+len(base):])
	if pre == "" && suf == "" {
		return 0, false // a plain name is not magic: nothing was read
	}
	for _, part := range []struct {
		text   string
		suffix bool
	}{{pre, false}, {suf, true}} {
		if part.text == "" {
			continue
		}
		lr, found := db.affixLevelReq(part.text, part.suffix, b.Type)
		if !found {
			return 0, false
		}
		levelReq = max(levelReq, lr)
	}
	return levelReq, true
}

func (db *DB) affixLevelReq(text string, suffix bool, itemType string) (int, bool) {
	list := db.Prefixes
	if suffix {
		list = db.Suffixes
	}
	best, bestTyped := -1, false
	for _, a := range list {
		if !strings.EqualFold(a.Display, text) && !strings.EqualFold(a.Name, text) {
			continue
		}
		typed := false
		for _, t := range a.Include {
			if db.IsA(itemType, t) {
				typed = true
				break
			}
		}
		if typed && !bestTyped {
			best, bestTyped = -1, true // a type match replaces any earlier untyped candidate
		}
		if typed == bestTyped && a.LevelReq > best {
			best = a.LevelReq
		}
	}
	return best, best >= 0
}
