package main

import (
	"encoding/json"
	"strings"
	"testing"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/status"
)

// Made-up rows: a staff (Sorceress gear) and a scepter (Paladin gear), a unique override on the scepter.
func artCat() bot.Catalog {
	return bot.NewDataCatalog(&data.DB{
		Items: []*data.ItemBase{
			{Code: "sst", InvFile: "invSst", UniqueInvFile: "invSstU"},
			{Code: "scp", InvFile: "invScp"},
		},
		Uniques: []*data.UniqueItem{{Row: 5, Code: "scp", InvFile: "invScpX"}},
	})
}

func TestItemArt(t *testing.T) {
	cat := artCat()
	unit := func(class, quality, idx int) d2.Unit {
		return d2.Unit{Class: class, Item: &d2.Item{Quality: quality, FileIndex: idx}}
	}
	for _, tc := range []struct {
		name string
		u    d2.Unit
		want string
	}{
		{"sorceress staff, magic", unit(0, 4, 0), "invsst"},
		{"sorceress staff, unique falls back to uniqueinvfile", unit(0, d2.QualityUnique, 5), "invsstu"},
		{"paladin scepter, rare", unit(1, 6, 0), "invscp"},
		{"paladin scepter, unique override", unit(1, d2.QualityUnique, 5), "invscpx"},
		{"paladin scepter, unique wrong row", unit(1, d2.QualityUnique, 9), "invscp"},
		{"set without setinvfile", unit(1, d2.QualitySet, 5), "invscp"},
		{"unknown class", unit(77, 4, 0), ""},
		{"no item data", d2.Unit{Class: 0}, ""},
	} {
		if got := itemArt(cat, tc.u); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
}

// A catalog without icon names (the test fakes, a bot without tables) leaves art empty, so the page draws cells.
func TestItemArtNoIcons(t *testing.T) {
	if got := itemArt(fakeCat{}, d2.Unit{Item: &d2.Item{}}); got != "" {
		t.Errorf("fakeCat: %q", got)
	}
}

// art is additive: absent from the JSON when empty, so existing consumers and goldens see the same bytes.
func TestStatusItemArtOmitEmpty(t *testing.T) {
	b, _ := json.Marshal(status.Item{Code: "1"})
	if strings.Contains(string(b), "art") {
		t.Errorf("empty art serialised: %s", b)
	}
	b, _ = json.Marshal(status.Item{Code: "1", Art: "invsst"})
	if !strings.Contains(string(b), `"art":"invsst"`) {
		t.Errorf("art missing: %s", b)
	}
}
