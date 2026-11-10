package data

import "strings"

// InvQuality selects which icon columns InvFile consults.
type InvQuality int

const (
	InvPlain  InvQuality = iota // magic, rare, crafted, runeword, normal: the base invfile
	InvUnique                   // uniqueitems invfile, then the base uniqueinvfile
	InvSet                      // setitems invfile, then the base setinvfile
)

// InvFile resolves the inventory icon name (lower case, no extension, as in data\global\items\<name>.dc6)
// of an item. class is the base item row (index into Items), fileIndex the unique or set id of the item.
//
// Order: unique -> uniqueitems.invfile, else base uniqueinvfile, else base invfile; set -> setitems.invfile,
// else base setinvfile, else base invfile; everything else -> base invfile. VERIFIED that every named file
// exists in the 1.14d archives (docs/research/paperdoll-art.md section 2); the exact game rule and the row
// semantics of fileIndex are UNVERIFIED, so an override row whose base code is not the item's own is ignored
// (a wrong index can then only fall back to the base icon, never show another item's).
// ok is false when the class is unknown or the base has no icon name at all.
func (db *DB) InvFile(class int, q InvQuality, fileIndex int) (name string, ok bool) {
	if db == nil || class < 0 || class >= len(db.Items) {
		return "", false
	}
	b := db.Items[class]
	var over, special string
	switch q {
	case InvUnique:
		special = b.UniqueInvFile
		for _, u := range db.Uniques {
			if u.Row == fileIndex {
				if strings.EqualFold(u.Code, b.Code) {
					over = u.InvFile
				}
				break
			}
		}
	case InvSet:
		special = b.SetInvFile
		for _, s := range db.SetItems {
			if s.Row == fileIndex {
				if strings.EqualFold(s.Code, b.Code) {
					over = s.InvFile
				}
				break
			}
		}
	}
	for _, n := range []string{over, special, b.InvFile} {
		if n != "" {
			return strings.ToLower(n), true
		}
	}
	return "", false
}
