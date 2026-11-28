package vaultkeep

import (
	"context"
	"strings"
	"testing"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/vault"
)

// mhand is dhand plus the merc rule: a table of what each merc kind may wear (data, not a literal in the keeper).
type mhand struct {
	dhand
	kind  string
	hired bool
	worn  map[int]string // the fake mercenary's slots (MercWearer); nil: the hand cannot show them
}

func (h mhand) MercWorn(context.Context) ([]string, bool) {
	var out []string
	for _, n := range h.worn {
		out = append(out, n)
	}
	return out, h.worn != nil
}

var mercWears = map[string]map[string]string{ // kind -> item code -> merc slot
	"rogue":  {"cap": "mhead", "arm": "mtorso", "bow": "mrhand"},
	"desert": {"cap": "mhead", "arm": "mtorso", "spr": "mrhand"},
}

func (h mhand) MercPreview(rec vault.Rec) MercSide {
	if !h.hired {
		return MercSide{Why: "no mercenary is hired"}
	}
	if s, ok := mercWears[h.kind][rec.Code]; ok {
		return MercSide{Owned: true, Kind: h.kind, Can: true, Slot: s}
	}
	return MercSide{Owned: true, Kind: h.kind, Why: "a " + h.kind + " cannot use that"}
}

// mfake is the fork's merc-equip over the VaultFake: the item leaves the pack, what he wore comes back into it.
type mfake struct {
	*d2dctl.DirectFake
	api   *flaky
	worn  map[int]string // body slot number -> norm hex
	dead  bool
	rule  func(code string) *d2dctl.ItemRefused
	calls int
}

var bodyOf = map[string]int{"cap": 1, "arm": 3, "bow": 4, "spr": 4}

func (m *mfake) MercDirectCaps(context.Context) (bool, error) { return true, nil }

func (m *mfake) MercEquipDirect(ctx context.Context, id uint32, _ string) (d2dctl.ItemResult, error) {
	m.calls++
	if m.dead {
		return d2dctl.ItemResult{}, &d2dctl.ItemRefused{Code: d2dctl.ItemErrDead, Why: "dead"}
	}
	var rec d2dctl.VaultRecord
	for _, c := range m.api.Carried {
		if uint32(c.Rec.UnitID) == id {
			rec = c.Rec
		}
	}
	if rec.Code == "" {
		return d2dctl.ItemResult{}, &d2dctl.ItemRefused{Code: d2dctl.ItemErrItem, Why: "no such item"}
	}
	if m.rule != nil {
		if r := m.rule(rec.Code); r != nil {
			return d2dctl.ItemResult{}, r
		}
	}
	slot := bodyOf[rec.Code]
	if _, err := m.api.ItemVault(ctx, int(id)); err != nil {
		return d2dctl.ItemResult{}, err
	}
	res := d2dctl.ItemResult{Slot: slot, Displaced: -1}
	if old, ok := m.worn[slot]; ok {
		cr, err := m.api.ItemCreate(ctx, old, d2dctl.CreateOpts{Pack: true})
		if err != nil {
			return d2dctl.ItemResult{}, err
		}
		res.Displaced = cr.ID
	}
	m.worn[slot] = rec.Norm
	return res, nil
}

func mercRig(t *testing.T, class, kind string) (*drig, *mfake) {
	d := directRig(t, class)
	mf := &mfake{DirectFake: d.df, api: d.api, worn: map[int]string{}}
	d.k.Direct = mf
	d.k.Hand = mhand{dhand: d.h, kind: kind, hired: true, worn: mf.worn}
	return d, mf
}

func TestEquipMercSwap(t *testing.T) {
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			d, mf := mercRig(t, class, "rogue")
			old := d2dctl.FakeRecord(0, "cap", 6, 20, "oldcap")
			mf.worn[1] = old.Norm
			id := d.add(1, "cap", "new")
			res, ok := d.k.EquipMerc(context.Background(), id)
			if !ok || !res.OK || res.Slot != "mhead" || len(res.Removed) != 1 || res.Where != "vault" {
				t.Fatalf("%v %+v", ok, res)
			}
			if r, _ := d.st.Get(id); r.State == vault.Recorded {
				t.Fatal("the equipped record is still live")
			}
			if got := d.live("previously worn"); got != 1 {
				t.Fatalf("previously worn records: %d", got)
			}
			if !strings.Contains(mf.worn[1], hexOf("new")) {
				t.Fatalf("the mercenary wears %q", mf.worn[1])
			}
			if n := len(d.api.Carried); n != 0 {
				t.Fatalf("%d items left in the game's pack", n)
			}
		})
	}
}

func TestEquipMercRefusals(t *testing.T) {
	cases := []struct {
		name, code, want string
		setup            func(d *drig, mf *mfake)
		item             string
	}{
		{"wrong kind", "merc", "cannot use", func(*drig, *mfake) {}, "wsp"},
		{"no merc", "merc", "no mercenary", func(d *drig, mf *mfake) { d.k.Hand = mhand{dhand: d.h, kind: "rogue"} }, "cap"},
		{"dead", "dead", "dead", func(_ *drig, mf *mfake) { mf.dead = true }, "cap"},
		{"requirements", "requirements", "too low", func(_ *drig, mf *mfake) {
			mf.rule = func(string) *d2dctl.ItemRefused {
				return &d2dctl.ItemRefused{Code: d2dctl.ItemErrRequirements, Why: "level 30"}
			}
		}, "cap"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, mf := mercRig(t, "paladin", "rogue")
			c.setup(d, mf)
			id := d.add(1, c.item, "x")
			res, ok := d.k.EquipMerc(context.Background(), id)
			if !ok || res.OK || res.Code != c.code || !strings.Contains(res.Why, c.want) {
				t.Fatalf("%v %+v", ok, res)
			}
			if r, _ := d.st.Get(id); r.State != vault.Recorded {
				t.Fatalf("the record changed: %s", r.State)
			}
			if n := len(d.api.Carried); n != 0 {
				t.Fatalf("%d stray items in the pack after a refusal", n)
			}
		})
	}
}

func TestEquipMercOffWithoutVerbsOrMeasured(t *testing.T) {
	d, _ := mercRig(t, "sorceress", "desert")
	d.k.Direct = d.df // no merc half
	if _, ok := d.k.EquipMerc(context.Background(), d.add(1, "cap", "x")); ok {
		t.Fatal("offered without the merc verbs")
	}
	if d.k.MercReady(context.Background()) {
		t.Fatal("ready without the merc verbs")
	}
	d, _ = mercRig(t, "sorceress", "desert")
	d.k.Direct = &mfake{DirectFake: d.df, api: d.api, worn: map[int]string{}}
	d.df.FlagOff = true
	if d.k.MercReady(context.Background()) {
		t.Fatal("ready on an older image")
	}
}
