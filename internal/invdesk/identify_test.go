package invdesk

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

// fakeSv is the town side of the identify commands: it records what was asked and answers like the real one would.
type fakeSv struct {
	tome, scrolls                               int // tome -1: none
	identified                                  []uint32
	batches                                     []int // size of each identify call: one call per run, not one per item
	bought                                      []int
	cained                                      int
	idErr, buyErr, cainErr, checkErr, cainCheck error
	buyMsg, cainMsg                             string
	during                                      func()
}

func (f *fakeSv) CheckPlayerIdentify(s *d2.Snapshot, id uint32) (string, error) {
	if f.tome <= 0 && f.scrolls == 0 {
		return "", errors.New(NoToolReason)
	}
	return "thing", nil
}
func (f *fakeSv) PlayerIdentifyBatch(ctx context.Context, ids []uint32) []error {
	f.batches = append(f.batches, len(ids))
	if f.during != nil {
		f.during()
	}
	errs := make([]error, len(ids))
	for i, id := range ids {
		if f.idErr != nil {
			errs[i] = f.idErr
			continue
		}
		f.identified = append(f.identified, id)
	}
	return errs
}
func (f *fakeSv) IdentifyTools(s *d2.Snapshot) (int, int, int) { return f.tome, 20, f.scrolls }
func (f *fakeSv) CheckBuyIdentify(s *d2.Snapshot, n int) (string, error) {
	if f.checkErr != nil {
		return "", f.checkErr
	}
	if n > 0 {
		return "Buy identify scrolls", nil
	}
	return "Fill the Tome of Identify", nil
}
func (f *fakeSv) BuyIdentify(ctx context.Context, n int) (string, error) {
	if f.during != nil {
		f.during()
	}
	f.bought = append(f.bought, n)
	return f.buyMsg, f.buyErr
}
func (f *fakeSv) CheckCainIdentify(s *d2.Snapshot) error { return f.cainCheck }
func (f *fakeSv) CainIdentify(ctx context.Context) (string, error) {
	if f.during != nil {
		f.during()
	}
	f.cained++
	return f.cainMsg, f.cainErr
}

func idRig(t *testing.T, class d2.Class, area int, cainOpen bool) (*Desk, *fakeSv, *d2.Snapshot) {
	t.Helper()
	d, _, snap := rig(t, class, area)
	sv := &fakeSv{tome: 5, buyMsg: "bought 12 for 600 gold", cainMsg: "Deckard Cain identified 3 items for free"}
	d.SetServices(sv)
	d.SetCainOpen(func(*d2.Snapshot) bool { return cainOpen })
	// item 2 is identified, the others (1 pack, 3 stash, 5 worn) are not
	snap.Inventory.Items[1].Item.Flags |= d2.FlagIdentified
	return d, sv, snap
}

func TestIdentifyRefusals(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		t.Run(class.String(), func(t *testing.T) {
			d, sv, _ := idRig(t, class, field, false)
			for _, c := range []struct {
				id   uint32
				want string
			}{{2, "identified"}, {3, "place"}, {77, "item"}} {
				if _, err := d.RequestIdentify(c.id); code(err) != c.want {
					t.Errorf("item %d: %v, want %s", c.id, err, c.want)
				}
			}
			sv.tome, sv.scrolls = -1, 0
			_, err := d.RequestIdentify(1)
			if code(err) != "requirements" || !strings.Contains(err.Error(), "Buy identify scrolls") || !strings.Contains(err.Error(), "Identify all (Cain)") {
				t.Errorf("no tool: %v (the reason must name the two other commands)", err)
			}
			if d.Pending() != 0 {
				t.Error("a refusal queued something")
			}
			bare, _, _ := rig(t, class, field)
			if _, err := bare.RequestIdentify(1); code(err) != "off" {
				t.Errorf("without a town backend: %v", err)
			}
		})
	}
}

func TestIdentifyRunsInTheFieldAndInTown(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		for _, area := range []int{field, town1} {
			d, sv, _ := idRig(t, class, area, false)
			if _, err := d.RequestIdentify(1); err != nil {
				t.Fatal(err)
			}
			if _, err := d.RequestIdentify(1); code(err) != "queued" {
				t.Errorf("twice: %v", err)
			}
			if _, err := d.RequestIdentify(5); err != nil { // a worn item
				t.Fatal(err)
			}
			var seen string
			sv.during = func() { seen = d.View().Queue[0].State }
			lines := d.RunPending(context.Background())
			if len(lines) != 2 || len(sv.identified) != 2 || seen != StateDoing {
				t.Fatalf("%v area %d: lines %v identified %v state %q", class, area, lines, sv.identified, seen)
			}
			v := d.View()
			if len(v.Queue) != 0 || len(v.Recent) != 2 || v.Recent[0].State != StateDone || v.Recent[0].Kind != KindIdentify {
				t.Fatalf("view %+v", v)
			}
		}
	}
}

func TestIdentifyFailureAndFightKeepTheReason(t *testing.T) {
	d, sv, _ := idRig(t, d2.Paladin, field, false)
	if _, err := d.RequestIdentify(1); err != nil {
		t.Fatal(err)
	}
	sv.idErr = errors.New("town: a fight is on")
	if lines := d.RunPending(context.Background()); len(lines) != 1 || !strings.Contains(lines[0], "postponed") {
		t.Fatalf("lines %v", lines)
	}
	if v := d.View(); len(v.Queue) != 1 || v.Queue[0].State != StateQueued {
		t.Fatalf("a fight must put the request back to queued: %+v", v.Queue)
	}
	sv.idErr = errors.New("the scroll ran out")
	d.RunPending(context.Background())
	if v := d.View(); len(v.Queue) != 0 || v.Recent[0].State != StateFailed || v.Recent[0].Reason != "the scroll ran out" {
		t.Fatalf("failure: %+v", v.Recent)
	}
}

func TestBuyScrollsErrandWaitsForTownAndReportsTheResult(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		d, sv, snap := idRig(t, class, field, false)
		r, err := d.RequestBuyID(12)
		if err != nil || r.Kind != KindBuyID || r.Count != 12 || r.ID != 0 {
			t.Fatalf("%v %+v", err, r)
		}
		if _, err := d.RequestBuyID(0); code(err) != "queued" {
			t.Errorf("second errand: %v", err)
		}
		if d.Pending() != 0 || len(d.RunPending(context.Background())) != 0 || len(sv.bought) != 0 {
			t.Fatal("a town errand must not start in the field")
		}
		if v := d.View(); v.Queue[0].Reason != WaitsForTown || v.Queue[0].State != StateQueued {
			t.Fatalf("the queue card must say it waits: %+v", v.Queue[0])
		}
		// a field identify queued behind it still runs
		if _, err := d.RequestIdentify(1); err != nil {
			t.Fatal(err)
		}
		d.RunPending(context.Background())
		if len(sv.identified) != 1 || len(sv.bought) != 0 {
			t.Fatalf("identify %v bought %v", sv.identified, sv.bought)
		}
		snap.Player.Area = town1
		if d.Pending() != 1 {
			t.Fatalf("pending %d in town", d.Pending())
		}
		sv.buyMsg = "bought 12 for 600 gold, short of gold for the other 8"
		d.RunPending(context.Background())
		v := d.View()
		if len(sv.bought) != 1 || sv.bought[0] != 12 || len(v.Queue) != 0 || v.Recent[0].State != StateDone || v.Recent[0].Result != sv.buyMsg {
			t.Fatalf("bought %v view %+v", sv.bought, v.Recent)
		}
	}
}

func TestErrandCancelAndRestart(t *testing.T) {
	d, sv, _ := idRig(t, d2.Sorceress, field, true)
	if _, err := d.RequestBuyID(0); err != nil {
		t.Fatal(err)
	}
	if _, err := d.RequestCain(); err != nil {
		t.Fatal(err)
	}
	// both survive a restart while queued
	d2x, err := New(Config{Path: d.c.Path, Snap: d.c.Snap, CainOpen: d.c.CainOpen})
	if err != nil {
		t.Fatal(err)
	}
	if got := d2x.View().Queue; len(got) != 2 || got[0].Kind != KindBuyID || got[1].Kind != KindCain {
		t.Fatalf("reloaded queue %+v", got)
	}
	if err := d.Cancel(0, KindBuyID); err != nil {
		t.Fatal(err)
	}
	if err := d.Cancel(0, KindBuyID); code(err) != "item" {
		t.Errorf("second cancel: %v", err)
	}
	if v := d.View(); len(v.Queue) != 1 || v.Recent[0].State != StateCancelled || v.Recent[0].Kind != KindBuyID {
		t.Fatalf("view %+v", v)
	}
	// a request found "doing" at load is failed, never retried
	d.d2set(KindCain, StateDoing)
	re, err := New(Config{Path: d.c.Path})
	if err != nil {
		t.Fatal(err)
	}
	if v := re.View(); len(v.Queue) != 0 || v.Recent[0].State != StateFailed || !strings.Contains(v.Recent[0].Reason, "restart") {
		t.Fatalf("restart: %+v", v)
	}
	_ = sv
}

// d2set marks the queued request of kind as in the given state (test helper: a crash while the bot was doing it).
func (d *Desk) d2set(kind, state string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i := range d.st.Queue {
		if d.st.Queue[i].kind() == kind {
			d.st.Queue[i].State = state
		}
	}
	_ = d.saveLocked()
}

func TestCainLockedUntilRescuedThenRuns(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		rescued := false
		d, sv, _ := idRig(t, class, town1, false)
		d.SetCainOpen(func(*d2.Snapshot) bool { return rescued })
		v := d.View()
		if v.Identify == nil || v.Identify.Cain != "locked" || v.Identify.CainWhy != CainTeaser {
			t.Fatalf("locked view: %+v", v.Identify)
		}
		if _, err := d.RequestCain(); code(err) != "locked" || err.Error() != CainTeaser {
			t.Fatalf("locked request: %v", err)
		}
		rescued = true
		if v := d.View(); v.Identify.Cain != "open" || v.Identify.CainWhy != "" {
			t.Fatalf("open view: %+v", v.Identify)
		}
		sv.cainCheck = errors.New("nothing to identify")
		if _, err := d.RequestCain(); code(err) != "requirements" {
			t.Fatalf("nothing to identify: %v", err)
		}
		sv.cainCheck = nil
		if _, err := d.RequestCain(); err != nil {
			t.Fatal(err)
		}
		if _, err := d.RequestCain(); code(err) != "queued" {
			t.Errorf("twice: %v", err)
		}
		lines := d.RunPending(context.Background())
		v = d.View()
		if sv.cained != 1 || len(lines) != 1 || len(v.Queue) != 0 || v.Recent[0].Kind != KindCain || v.Recent[0].Result != sv.cainMsg {
			t.Fatalf("cained %d lines %v view %+v", sv.cained, lines, v.Recent)
		}
		// a failure keeps its reason
		sv.cainErr = errors.New("Deckard Cain is not in this town")
		if _, err := d.RequestCain(); err != nil {
			t.Fatal(err)
		}
		d.RunPending(context.Background())
		if v := d.View(); v.Recent[0].State != StateFailed || !strings.Contains(v.Recent[0].Reason, "not in this town") {
			t.Fatalf("failure: %+v", v.Recent[0])
		}
	}
	// no quest reader wired: the command is not offered at all
	d, _, _ := idRig(t, d2.Paladin, town1, false)
	d.SetCainOpen(nil)
	if v := d.View(); v.Identify.Cain != "" {
		t.Errorf("cain %q without a check", v.Identify.Cain)
	}
	if _, err := d.RequestCain(); code(err) != "off" {
		t.Errorf("without a check: %v", err)
	}
}

func TestIdentifyViewFacts(t *testing.T) {
	d, sv, _ := idRig(t, d2.Paladin, field, true)
	v := d.View()
	id := v.Identify
	if id == nil || id.Tool != "tome" || id.Tome != 5 || id.TomeCap != 20 || id.BuyDefault != 15 || id.BuyMax != 20 || id.Unid != 3 || id.Cain != "open" {
		t.Fatalf("%+v", id)
	}
	sv.tome, sv.scrolls = 20, 0
	if id := d.View().Identify; id.BuyWhy == "" {
		t.Errorf("a full tome must refuse the shopping errand: %+v", id)
	}
	sv.tome, sv.scrolls = -1, 0
	v = d.View()
	if v.Identify.Tool != "none" || v.Identify.BuyDefault != 0 {
		t.Errorf("no tome: %+v", v.Identify)
	}
	// the item menu data: unidentified pack item without a tool carries the reason; identified item and stash item behave
	for _, it := range v.Backpack {
		if it.ID == 1 && it.NoIdentify != NoToolReason {
			t.Errorf("item 1 noIdentify %q", it.NoIdentify)
		}
	}
	sv.scrolls = 1
	for _, it := range d.View().Backpack {
		if it.NoIdentify != "" {
			t.Errorf("item %d noIdentify %q with a scroll in the pack", it.ID, it.NoIdentify)
		}
	}
}

// Schema 5 golden: the view with the identify block, a queued errand that waits for town and a done identify, for a
// Paladin (Cain open) and a Sorceress (Cain locked). UPDATE=1 rewrites the files.
func TestViewSchema5IdentifyGolden(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		d, sv, _ := idRig(t, class, field, class == d2.Paladin)
		now := time.Unix(1_800_000_000, 0)
		d.c.Now = func() time.Time { return now }
		if _, err := d.RequestIdentify(1); err != nil {
			t.Fatal(err)
		}
		d.RunPending(context.Background())
		if _, err := d.RequestBuyID(12); err != nil {
			t.Fatal(err)
		}
		if class == d2.Paladin {
			if _, err := d.RequestCain(); err != nil {
				t.Fatal(err)
			}
		}
		_ = sv
		b, err := json.MarshalIndent(asSchema5(d.View()), "", " ")
		if err != nil {
			t.Fatal(err)
		}
		b = append(b, '\n')
		path := filepath.Join("testdata", "view_schema5_identify_"+class.String()+".json")
		if os.Getenv("UPDATE") != "" {
			if err := os.WriteFile(path, b, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(want) != string(b) {
			t.Errorf("%v: view differs from %s\n got: %s\nwant: %s", class, path, b, want)
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}
		for _, k := range append(v2Keys, "identify") { // every schema 2 key is still there next to the new block
			if _, ok := m[k]; !ok {
				t.Errorf("%v: key %q missing", class, k)
			}
		}
	}
}

// Priority: a queued identify runs before an equip and a trash that were asked earlier, and the errands come last; all
// queued identifies share ONE call (one inventory-open).
func TestIdentifyPriorityAndBatch(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		d, sv, snap := idRig(t, class, town1, true)
		d.SetTrasher(&fakeTr{snap: snap})
		var order []string
		sv.during = func() { order = append(order, "identify") }
		good := map[d2.Class]uint32{d2.Paladin: 1, d2.Sorceress: 2}[class]
		snap.Inventory.Items[0].Item.Flags |= d2.FlagIdentified // item 1 equippable for the Paladin
		snap.Inventory.Items[1].Item.Flags |= d2.FlagIdentified
		if _, err := d.RequestEquip(good, "rhand"); err != nil {
			t.Fatal(err)
		}
		if _, err := d.RequestCain(); code(err) != "requirements" && err != nil {
			t.Fatal(err)
		}
		if _, err := d.RequestBuyID(3); err != nil {
			t.Fatal(err)
		}
		// two unidentified items: a worn one (5) and a fresh backpack one (9)
		snap.Inventory.Items = append(snap.Inventory.Items, d2.Unit{ID: 9, Class: 14, Item: &d2.Item{Where: d2.ContInventory, Quality: 4}})
		for _, id := range []uint32{5, 9} {
			if _, err := d.RequestIdentify(id); err != nil {
				t.Fatalf("%v: %v", class, err)
			}
		}
		if k := d.NextKind(); k != KindIdentify {
			t.Fatalf("%v: next kind %q, want identify first", class, k)
		}
		lines := d.RunPending(context.Background())
		if len(sv.batches) != 1 || sv.batches[0] != 2 {
			t.Fatalf("%v: batches %v, want one call with two items", class, sv.batches)
		}
		if len(sv.bought) != 1 || len(lines) < 4 {
			t.Fatalf("%v: lines %v", class, lines)
		}
		// identify lines come before the equip line and the errand line
		pos := map[string]int{}
		for i, l := range lines {
			for _, k := range []string{"identify", "equip", "buyid"} {
				if strings.HasPrefix(l, k) && pos[k] == 0 {
					pos[k] = i + 1
				}
			}
		}
		if !(pos["identify"] < pos["equip"] && pos["equip"] < pos["buyid"]) {
			t.Fatalf("%v: order %v (%v)", class, lines, pos)
		}
	}
}

// A fight puts the whole batch back to queued; after the bound with no safe moment the request fails with a reason;
// a missing tool fails fast.
func TestIdentifyFightBoundAndNoTool(t *testing.T) {
	d, sv, _ := idRig(t, d2.Paladin, field, false)
	now := time.Unix(1_800_000_000, 0)
	d.c.Now = func() time.Time { return now }
	if _, err := d.RequestIdentify(1); err != nil {
		t.Fatal(err)
	}
	sv.idErr = errors.New("town: a fight is on")
	d.RunPending(context.Background())
	if v := d.View(); len(v.Queue) != 1 || v.Queue[0].State != StateQueued {
		t.Fatalf("fight must postpone: %+v", v.Queue)
	}
	now = now.Add(IdentifySafeWait + time.Second)
	if n := d.Pending(); n != 0 {
		t.Fatalf("pending %d after the bound", n)
	}
	if v := d.View(); len(v.Queue) != 0 || v.Recent[0].State != StateFailed || !strings.Contains(v.Recent[0].Reason, "no safe moment") {
		t.Fatalf("bound: %+v", v.Recent)
	}
	sv.idErr = errors.New(NoToolReason)
	if _, err := d.RequestIdentify(1); err != nil {
		t.Fatal(err)
	}
	d.RunPending(context.Background())
	if v := d.View(); v.Recent[0].State != StateFailed || !strings.Contains(v.Recent[0].Reason, "Buy identify scrolls") {
		t.Fatalf("no tool: %+v", v.Recent[0])
	}
	if d.ErrandsWaiting() != 0 {
		t.Error("no errand queued")
	}
	if _, err := d.RequestBuyID(2); err != nil {
		t.Fatal(err)
	}
	if d.ErrandsWaiting() != 1 {
		t.Error("errand not counted")
	}
}
