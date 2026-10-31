package bot

import (
	"context"
	"sort"
	"testing"

	"d2lod-bot/internal/task"
)

// recActs records the object clicks and quest items a handler asks for; every call succeeds.
type recActs struct {
	objects map[string]bool
	items   map[string]bool
}

func (r *recActs) Travel(context.Context, task.Area) error { return nil }
func (r *recActs) ToTown(context.Context) error            { return nil }
func (r *recActs) Clear(context.Context, task.Area) error  { return nil }
func (r *recActs) KillBoss(context.Context, string) error  { return nil }
func (r *recActs) Talk(context.Context, string, string) error {
	return nil
}
func (r *recActs) Interact(_ context.Context, o string) error { r.objects[o] = true; return nil }
func (r *recActs) PickUp(_ context.Context, i string) error   { r.items[i] = true; return nil }
func (r *recActs) ActivateWaypoint(context.Context, task.Area) error {
	return nil
}
func (r *recActs) Chores(context.Context) error              { return nil }
func (r *recActs) LevelTo(context.Context, int) error        { return nil }
func (r *recActs) UseItem(_ context.Context, i string) error { r.objects["item:"+i] = true; return nil }

// Every object a quest handler clicks is either one the adapter proves a pseudo item for (interactPseudo, Anya, the
// tree) or a plain click whose effect the World sees by itself (a chest that drops a pick-up, a cage, a portal).
// A new object that is in neither list fails here, so whoever adds a handler decides which it is.
func TestHandlerObjectsAreKnownToTheAdapter(t *testing.T) {
	rec := &recActs{objects: map[string]bool{}, items: map[string]bool{}}
	var qs []string
	for _, c := range []*task.Campaign{task.NightmareAll(true), task.Template("normal_all", true)} {
		for _, e := range c.Entries {
			qs = append(qs, e.Quest)
		}
	}
	for _, q := range append(qs, task.QHellforge, task.QPrison, task.QBetrayal, task.QRescue, task.QArcane, task.QSummoner) {
		for _, n := range task.Handler(q) {
			_ = n.Step.Do(context.Background(), rec) // a failing optional capability is fine: only the clicks matter
		}
	}
	known := map[string]bool{AnyaObject: true, TreeObject: true, OrbObject: true, ForgePlaceObject: true, ForgeSmashObject: true}
	for o := range interactPseudo {
		known[o] = true
	}
	plain := map[string]string{}
	for _, o := range []string{"Anya's portal", "Arcane Sanctuary portal", "Cain's Cage", "cage", "duriel-door", "red portal", "Summoner's journal"} {
		plain[o] = "door, portal or cage: the area or quest word shows the effect"
	}
	for _, o := range []string{"Horadric Cube chest", "Horadric Scroll chest", "Khalim's Brain chest", "Khalim's Eye chest", "Khalim's Heart chest", "Staff of Kings chest", "taintedsunaltar"} {
		plain[o] = "drops an item the handler picks up next"
	}
	var unknown []string
	for o := range rec.objects {
		if !known[o] && plain[o] == "" {
			unknown = append(unknown, o)
		}
	}
	sort.Strings(unknown)
	if len(unknown) > 0 {
		t.Fatalf("object clicks the adapter test does not classify (pseudo item needed?): %q", unknown)
	}
}
