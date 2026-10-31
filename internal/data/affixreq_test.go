package data

import "testing"

func TestMagicNameLevelReq(t *testing.T) {
	db := &DB{
		Prefixes: []*Affix{{Display: "Sturdy", LevelReq: 3}, {Display: "Jagged", LevelReq: 12}, {Display: "Jagged", LevelReq: 20}},
		Suffixes: []*Affix{{Display: "of the Fox", LevelReq: 18, Suffix: true}, {Display: "of Health", LevelReq: 1, Suffix: true}},
	}
	cap := &ItemBase{Display: "Cap", Name: "Cap", Type: "helm"}
	for _, c := range []struct {
		name string
		want int
		ok   bool
	}{
		{"Sturdy Cap", 3, true},
		{"Cap of Health", 1, true},
		{"Sturdy Cap of the Fox", 18, true}, // the highest of prefix and suffix
		{"Jagged Cap", 20, true},            // equal display text: the conservative row
		{"Cap", 0, false},                   // no affix read: not trusted
		{"Mystic Cap", 0, false},            // unknown prefix: not trusted
		{"Sturdy Helm", 0, false},           // base name absent
		{"", 0, false},
	} {
		got, ok := db.MagicNameLevelReq(cap, c.name)
		if got != c.want || ok != c.ok {
			t.Errorf("%q: got %d,%v want %d,%v", c.name, got, ok, c.want, c.ok)
		}
	}
	if _, ok := db.MagicNameLevelReq(nil, "Sturdy Cap"); ok {
		t.Error("nil base trusted")
	}
}
