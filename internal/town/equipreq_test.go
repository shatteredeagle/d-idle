package town_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/town"
)

// fakeReq gives magic items a fixed level requirement, like an affix lookup would.
type fakeReq struct{ level int }

func (f fakeReq) Requirement(u d2.Unit, b *data.ItemBase) town.Req {
	r := town.Req{Level: b.LevelReq, Str: b.ReqStr, Dex: b.ReqDex}
	if u.Item != nil && u.Item.Quality >= d2.QualityMagic {
		r.Level = max(r.Level, f.level)
	}
	return r
}

func packCap(r *rig, quality int, identified bool) d2.Unit {
	u := r.W.Add("cap", d2.ContInventory, 0, 0)
	u.Item.Quality = quality
	if !identified {
		u.Item.Flags &^= d2.FlagIdentified
	}
	return u
}

// A magic cap whose affix needs more level than the character has is skipped without a single click.
func TestEquipSkipsMagicAboveLevel(t *testing.T) {
	r := earlyRig(t, d2.Sorceress, 9)
	r.T.SetRequirementSource(fakeReq{level: 20})
	packCap(r, d2.QualityMagic, true)
	n0 := len(r.W.Clicks)
	done, err := r.T.EquipFound(context.Background())
	if err != nil || len(done) != 0 || len(r.W.Clicks) != n0 {
		t.Fatalf("done %v err %v clicks %d", done, err, len(r.W.Clicks)-n0)
	}
	r.W.Player.Level = 20 // once the level is there it is worn
	if done, _ := r.T.EquipFound(context.Background()); len(done) != 1 {
		t.Fatalf("not worn at level 20: %v", done)
	}
}

func TestEquipSkipsUnidentified(t *testing.T) {
	r := earlyRig(t, d2.Sorceress, 30)
	packCap(r, d2.QualityMagic, false)
	n0 := len(r.W.Clicks)
	if done, _ := r.T.EquipFound(context.Background()); len(done) != 0 || len(r.W.Clicks) != n0 {
		t.Fatalf("unidentified magic cap worn or clicked: %v", done)
	}
}

// Without affix names the default source estimates a magic item at base levelreq + margin; a plain cap is unchanged.
func TestEquipDefaultEstimateForMagic(t *testing.T) {
	r := earlyRig(t, d2.Sorceress, town.MagicLevelMargin-1)
	packCap(r, d2.QualityMagic, true)
	if done, _ := r.T.EquipFound(context.Background()); len(done) != 0 {
		t.Fatalf("magic cap below the margin worn: %v", done)
	}
	r2 := earlyRig(t, d2.Sorceress, 1)
	packCap(r2, d2.QualityNormal, true)
	if done, _ := r2.T.EquipFound(context.Background()); len(done) != 1 {
		t.Fatalf("normal cap not worn: %v", done)
	}
}

// A click the game refuses puts the item back in the pack, blacklists it, and the next scan does not click again.
func TestEquipRefusedPutBackAndBlacklisted(t *testing.T) {
	r := earlyRig(t, d2.Sorceress, 12) // an own level: the refusal list is process-wide and keyed by unit id
	cap := packCap(r, d2.QualityMagic, true)
	r.T.SetRequirementSource(fakeReq{level: 1}) // the tables say wearable, the game says no
	r.W.EquipRefuse[cap.ID] = true
	if _, err := r.T.EquipFound(context.Background()); err == nil {
		t.Fatal("refused equip reported success")
	}
	if r.W.Cursor != nil {
		t.Fatal("item left on the cursor")
	}
	s, _ := r.W.Snapshot()
	if u, ok := findUnit(s, cap.ID); !ok || u.Item.Where != d2.ContInventory {
		t.Fatalf("item not back in the pack: %+v", u)
	}
	n0 := len(r.W.Clicks)
	if done, err := r.T.EquipFound(context.Background()); len(done) != 0 || err != nil || len(r.W.Clicks) != n0 {
		t.Fatalf("blacklisted item retried: %v %v clicks %d", done, err, len(r.W.Clicks)-n0)
	}
}

func findUnit(s *d2.Snapshot, id uint32) (d2.Unit, bool) {
	for _, u := range s.Inventory.Items {
		if u.ID == id {
			return u, true
		}
	}
	return d2.Unit{}, false
}

// The Fresha belt: 3 healing and 1 mana potion fill the 4 slots; pack potions are neither dragged nor shift-clicked.
func TestFullMixedBeltNoDrag(t *testing.T) {
	r := newRig(t)
	for i := 0; i < 3; i++ {
		r.W.Add("hp1", d2.ContBelt, i, 0)
	}
	r.W.Add("mp1", d2.ContBelt, 3, 0)
	addPack(r, "hp1", 4)
	n0 := len(r.W.Clicks)
	rep, err := r.T.FillBelt(context.Background())
	if err != nil || rep.Moved != 0 || rep.Failed != 0 || len(r.W.Clicks) != n0 {
		t.Fatalf("rep %+v err %v clicks %d", rep, err, len(r.W.Clicks)-n0)
	}
}
