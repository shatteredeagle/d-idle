package invdesk

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/status"
)

// fakeStash adds the stash side to fakeSv. roomPack and roomStash say whether a move is possible.
type fakeStash struct {
	*fakeSv
	roomPack, roomStash bool
	visits              []StashPlan
	failID              map[uint32]error
	goldErr             error
}

func (f *fakeStash) CheckStashMove(s *d2.Snapshot, id uint32, toStash bool) (string, error) {
	if toStash && !f.roomStash {
		return "", errors.New("the stash has no free spot for it")
	}
	if !toStash && !f.roomPack {
		return "", errors.New("the backpack has no free spot for it: make room first")
	}
	return "thing", nil
}
func (f *fakeStash) CheckStashIdentify(s *d2.Snapshot, id uint32) (string, error) {
	if f.tome <= 0 && f.scrolls == 0 {
		return "", errors.New(NoToolReason)
	}
	return "yellow sword", nil
}
func (f *fakeStash) StashVisit(ctx context.Context, p StashPlan) StashResult {
	f.visits = append(f.visits, p)
	r := StashResult{Errs: map[uint32]error{}, GoldErr: f.goldErr}
	for _, id := range append(append(append([]uint32{}, p.Take...), p.Stow...), p.Identify...) {
		r.Errs[id] = f.failID[id]
	}
	if p.Gold > 0 && f.goldErr == nil {
		r.Gold = "moved gold"
	}
	return r
}

func stashRig(t *testing.T, class d2.Class, area int) (*Desk, *fakeStash, *d2.Snapshot) {
	t.Helper()
	d, sv, snap := idRig(t, class, area, false)
	snap.Player.Level, snap.Player.Gold, snap.Player.StashGold = 30, 5000, 20000
	fs := &fakeStash{fakeSv: sv, roomPack: true, roomStash: true, failID: map[uint32]error{}}
	d.SetServices(fs)
	d.c.Inventory = func() status.InventoryData {
		return status.InventoryData{Gold: 5000, StashGold: 20000,
			Inventory: []status.Item{{ID: 1, Code: "10", Identified: true}, {ID: 2, Code: "11", Identified: true}, {ID: 4}},
			Stash:     []status.Item{{ID: 3, Code: "12"}}}
	}
	return d, fs, snap
}

func TestStashMovesShareOneVisit(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		d, fs, _ := stashRig(t, class, town1)
		if _, err := d.RequestStash(1); err != nil {
			t.Fatal(err)
		}
		if _, err := d.RequestStash(1); code(err) != "queued" {
			t.Errorf("twice: %v", err)
		}
		if _, err := d.RequestTake(3); err != nil {
			t.Fatal(err)
		}
		if _, err := d.RequestGold(true, 1000); err != nil {
			t.Fatal(err)
		}
		if _, err := d.RequestIdentify(3); code(err) != "queued" {
			t.Errorf("identify of an item with a take queued: %v", err)
		}
		lines := d.RunPending(context.Background())
		if len(fs.visits) != 1 || len(lines) != 3 {
			t.Fatalf("%v: visits %v lines %v", class, fs.visits, lines)
		}
		p := fs.visits[0]
		if len(p.Stow) != 1 || len(p.Take) != 1 || p.Gold != 1000 || !p.Deposit {
			t.Errorf("plan %+v", p)
		}
		v := d.View()
		if len(v.Queue) != 0 || len(v.Recent) != 3 || v.Recent[0].State != StateDone {
			t.Errorf("view %+v", v.Recent)
		}
	}
}

func TestStashRefusals(t *testing.T) {
	d, fs, _ := stashRig(t, d2.Paladin, town1)
	fs.roomStash, fs.roomPack = false, false
	if _, err := d.RequestStash(1); code(err) != "room" {
		t.Errorf("stash full: %v", err)
	}
	if _, err := d.RequestTake(3); code(err) != "room" || err.Error() != "the backpack has no free spot for it: make room first" {
		t.Errorf("pack full: %v", err)
	}
	fs.roomStash, fs.roomPack = true, true
	for _, c := range []struct {
		name string
		err  error
		want string
	}{
		{"take a pack item", second(d.RequestTake(1)), "place"},
		{"stash a stash item", second(d.RequestStash(3)), "place"},
		{"unknown", second(d.RequestStash(77)), "item"},
		{"zero gold", second(d.RequestGold(true, 0)), "amount"},
		{"too much to deposit", second(d.RequestGold(true, 5001)), "amount"},
		{"too much to withdraw", second(d.RequestGold(false, 20001)), "amount"},
	} {
		if code(c.err) != c.want {
			t.Errorf("%s: %v, want %s", c.name, c.err, c.want)
		}
	}
	// the pack takes level x 10000: 5000 held, 300000 cap, the stash holds 20000 -> 20000 is fine
	if _, err := d.RequestGold(false, 20000); err != nil {
		t.Error(err)
	}
	if _, err := d.RequestGold(true, 1); code(err) != "queued" {
		t.Errorf("second gold move: %v", err)
	}
	bare, _, _ := rig(t, d2.Paladin, town1)
	if _, err := bare.RequestStash(1); code(err) != "off" {
		t.Errorf("no backend: %v", err)
	}
}

func second(_ Request, err error) error { return err }

func TestStashFromTheFieldIsAnErrandThenRunsInTown(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		d, fs, snap := stashRig(t, class, field)
		if _, err := d.RequestIdentify(3); err != nil { // the yellow sword: identified IN the stash
			t.Fatal(err)
		}
		if _, err := d.RequestStash(1); err != nil {
			t.Fatal(err)
		}
		if d.Pending() != 0 || d.ErrandsWaiting() != 2 || d.ErrandKind() != "stash" {
			t.Fatalf("field: pending %d errands %d kind %q", d.Pending(), d.ErrandsWaiting(), d.ErrandKind())
		}
		if got := d.RunPending(context.Background()); len(got) != 0 || len(fs.visits) != 0 {
			t.Fatalf("ran in the field: %v", got)
		}
		v := d.View()
		if v.Queue[0].Reason != WaitsForTown || v.Queue[0].Place != PlaceStash || v.Queue[1].Reason != WaitsForTown {
			t.Errorf("queue card %+v", v.Queue)
		}
		if k := d.NextKind(); k != "" {
			t.Errorf("next kind in the field %q", k)
		}
		snap.Player.Area = town1
		if d.Pending() != 2 || d.NextKind() != "stash" {
			t.Fatalf("town: pending %d kind %q", d.Pending(), d.NextKind())
		}
		d.RunPending(context.Background())
		if len(fs.visits) != 1 || len(fs.visits[0].Identify) != 1 || len(fs.visits[0].Stow) != 1 || len(fs.fakeSv.batches) != 0 {
			t.Fatalf("one visit with both jobs, no pack identify batch: %+v batches %v", fs.visits, fs.fakeSv.batches)
		}
	}
}

func TestStashPostponedFailedAndCancelled(t *testing.T) {
	d, fs, _ := stashRig(t, d2.Paladin, town1)
	if _, err := d.RequestStash(1); err != nil {
		t.Fatal(err)
	}
	fs.failID[1] = errors.New("town: a fight is on") // isBusy
	d.RunPending(context.Background())
	if v := d.View(); len(v.Queue) != 1 || v.Queue[0].State != StateQueued {
		t.Fatalf("a fight must requeue: %+v", v.Queue)
	}
	if err := d.Cancel(1, KindStash); err != nil {
		t.Fatal(err)
	}
	if v := d.View(); len(v.Queue) != 0 || v.Recent[0].State != StateCancelled {
		t.Fatalf("cancel: %+v", v)
	}
	if _, err := d.RequestGold(false, 100); err != nil {
		t.Fatal(err)
	}
	fs.goldErr = errors.New("no gold mover")
	d.RunPending(context.Background())
	if v := d.View(); len(v.Queue) != 0 || v.Recent[0].State != StateFailed || v.Recent[0].Reason != "no gold mover" {
		t.Fatalf("gold failure: %+v", v.Recent)
	}
	// the queue survives a restart; a request mid-visit is reported failed
	if _, err := d.RequestStash(2); err != nil {
		t.Fatal(err)
	}
	d2x, err := New(d.c)
	if err != nil || len(d2x.View().Queue) != 1 {
		t.Fatalf("persisted: %v %+v", err, d2x.View().Queue)
	}
}

func TestStashViewFacts(t *testing.T) {
	d, fs, _ := stashRig(t, d2.Sorceress, town1)
	v := d.View()
	if v.StashOps == nil || v.StashOps.DepositMax != 5000 || v.StashOps.WithdrawMax != 20000 {
		t.Fatalf("ops %+v", v.StashOps)
	}
	if v.Backpack[0].StashOp != "stash" || v.Backpack[0].NoStash != "" || v.Stash[0].StashOp != "take" || v.Stash[0].NoIdentify != "" {
		t.Errorf("items %+v %+v", v.Backpack[0], v.Stash[0])
	}
	fs.roomPack = false
	if v = d.View(); v.Stash[0].NoStash == "" {
		t.Error("a full pack must give the take-out a reason")
	}
	fs.tome, fs.scrolls = -1, 0
	if v = d.View(); v.Stash[0].NoIdentify == "" {
		t.Error("no tool: the stash identify needs the reason")
	}
}

// Schema 6 golden: stash ops block, the stash fields of items, a stash job waiting for town and a done one.
func TestViewSchema6StashGolden(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		d, _, _ := stashRig(t, class, field)
		now := time.Unix(1_700_000_000, 0)
		d.c.Now = func() time.Time { return now }
		if _, err := d.RequestIdentify(3); err != nil {
			t.Fatal(err)
		}
		if _, err := d.RequestGold(true, 1000); err != nil {
			t.Fatal(err)
		}
		if err := d.Cancel(0, KindGold); err != nil {
			t.Fatal(err)
		}
		if _, err := d.RequestStash(1); err != nil {
			t.Fatal(err)
		}
		b, err := json.MarshalIndent(asSchema6(d.View()), "", " ")
		if err != nil {
			t.Fatal(err)
		}
		b = append(b, '\n')
		path := filepath.Join("testdata", "view_schema6_stash_"+class.String()+".json")
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
		for _, k := range append(v2Keys, "identify", "stashOps") {
			if _, ok := m[k]; !ok {
				t.Errorf("%v: key %q missing", class, k)
			}
		}
	}
}
