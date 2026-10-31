package saves

import (
	"testing"

	"d2lod-bot/internal/cheat/d2s"
)

func TestCheckFresh(t *testing.T) {
	parse := func(name string, edit func(h []byte)) *d2s.File {
		f, err := Verify(build(name, 0, edit))
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	if err := CheckFresh(parse("Fresha", nil), "fresha", "Sorceress"); err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct {
		edit       func(h []byte)
		fname, cls string
	}{
		"class":    {nil, "Fresha", "paladin"},
		"name":     {nil, "Other", "sorceress"},
		"level":    {func(h []byte) { h[0x2B] = 7 }, "Fresha", "sorceress"},
		"hardcore": {func(h []byte) { h[0x24] |= d2s.FlagHardcore }, "Fresha", "sorceress"},
		"ladder":   {func(h []byte) { h[0x24] |= d2s.FlagLadder }, "Fresha", "sorceress"},
		"classic":  {func(h []byte) { h[0x24] = 0 }, "Fresha", "sorceress"},
	} {
		if CheckFresh(parse("Fresha", c.edit), c.fname, c.cls) == nil {
			t.Errorf("%s accepted", name)
		}
	}
}
