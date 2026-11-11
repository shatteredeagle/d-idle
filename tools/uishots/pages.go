package main

import (
	"errors"
	"net/http"
	"time"

	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/watch"
)

// mountPages adds the inventory page (a fixture desk), the policy sheet (the shipped table on a fresh store) and the
// return screen (the frozen sim) to the fixture viewer, so the suite can render them at every width.
func mountPages(m *http.ServeMux, sim *watch.Sim, class string) {
	inv := watch.NewInvAPI()
	inv.Bind(fixtureDesk{})
	ih := inv.Handler()
	for _, p := range watch.InvRoutes {
		m.Handle(p, ih)
	}
	if cat, err := decide.Shipped(); err == nil {
		if tbl, err := decide.ShippedPolicy(cat); err == nil {
			if st, err := decide.New(cat, "fixture", &decide.ManualClock{}); err == nil {
				if ps, err := watch.NewPolicySheet(st, cat, tbl, func() string { return class }, func() float64 { return 30 }, watch.NewAwayLog(nil), nil); err == nil {
					ph := ps.PolicyHandler()
					for _, p := range watch.PolicyRoutes {
						m.Handle(p, ph)
					}
				}
			}
		}
	}
	rc := watch.ReturnConfig{Hub: sim.Hub, Events: watch.NewAwayLog(nil), Visit: watch.NewVisit("", func() time.Time { return time.Unix(1700000000, 0) }), Speed: 1}
	rh := rc.ReturnHandler(nil)
	for _, p := range watch.ReturnRoutes {
		m.Handle(p, rh)
	}
}

// fixtureDesk is an /api/inventory view in the desk's shape (internal/invdesk View): a broken weapon whose "dur" is
// absent (omitempty at 0, the bug the page once showed as "undefined"), a pinned ring, marks, long names in one-cell
// items next to each other, and a queue.
type fixtureDesk struct{}

func item(id int, code, name, q string, x, y, w, h int, extra map[string]any) map[string]any {
	it := map[string]any{"id": id, "code": code, "name": name, "quality": q, "identified": true, "x": x, "y": y, "w": w, "h": h}
	for k, v := range extra {
		it[k] = v
	}
	return it
}

func (fixtureDesk) View() any {
	return map[string]any{
		"schema": 1, "inTown": true, "class": "Sorceress", "gold": 1240, "stashGold": 5300,
		"equipment": map[string]any{
			"rhand":  item(1, "sst", "Short Staff", "normal", 0, 0, 1, 3, map[string]any{"maxDur": 18}),
			"head":   item(2, "cap", "Cap", "normal", 0, 0, 2, 2, map[string]any{"dur": 8, "maxDur": 12}),
			"torso":  item(3, "qui", "Sturdy Quilted Armor", "magic", 0, 0, 2, 3, map[string]any{"dur": 5, "maxDur": 24, "why": "you put it on at 12:50"}),
			"gloves": item(4, "lgl", "Leather Gloves", "normal", 0, 0, 2, 2, map[string]any{"dur": 11, "maxDur": 12}),
			"belt":   item(5, "lbl", "Sash", "normal", 0, 0, 2, 1, map[string]any{"dur": 12, "maxDur": 12}),
			"feet":   item(6, "lbt", "Boots", "normal", 0, 0, 2, 2, map[string]any{"dur": 10, "maxDur": 12}),
			"lring":  item(7, "rin", "Raven Grip", "rare", 0, 0, 1, 1, nil),
		},
		"backpack": []any{
			item(10, "8ss", "Staff of Charged Bolt", "magic", 0, 0, 1, 3, map[string]any{"canSell": true, "canWear": []string{"rhand"}}),
			item(11, "rin", "Ring", "rare", 1, 0, 1, 1, map[string]any{"canWear": []string{"lring", "rring"}}),
			item(12, "ssd", "Short Sword", "normal", 2, 0, 1, 3, map[string]any{"mark": "sell", "canSell": true, "maxDur": 24, "dur": 3}),
			item(13, "tsc", "Scroll of Town Portal", "normal", 3, 0, 1, 1, nil),
			item(14, "isc", "Scroll of Identify", "normal", 4, 0, 1, 1, nil),
			item(15, "tbk", "Tome of Town Portal", "normal", 3, 1, 1, 2, nil),
			item(16, "ltp", "Light Plate", "normal", 0, 3, 2, 1, map[string]any{"mark": "keep", "maxDur": 60, "dur": 60}),
			item(17, "box", "Horadric Cube", "unique", 8, 2, 2, 2, map[string]any{"mark": "keep"}),
		},
		"belt": []any{
			item(20, "hp1", "Minor Healing Potion", "normal", 0, 0, 1, 1, nil),
			item(21, "hp1", "Minor Healing Potion", "normal", 1, 0, 1, 1, nil),
			item(22, "mp1", "Minor Mana Potion", "normal", 2, 0, 1, 1, nil),
		},
		"stash": []any{
			item(30, "rin", "Ring", "magic", 0, 0, 1, 1, nil),
			item(31, "ltp", "Light Plate", "magic", 1, 0, 2, 3, nil),
			item(32, "tsc", "Scroll of Town Portal", "normal", 3, 0, 1, 1, nil),
		},
		"pins":   []any{map[string]any{"slot": "lring", "id": 7, "name": "Raven Grip", "since": 0}},
		"queue":  []any{map[string]any{"id": 11, "slot": "rring", "name": "Ring", "state": "queued", "at": 0}},
		"recent": []any{},
	}
}

var errFixture = errors.New("the fixture viewer does not write")

func (fixtureDesk) SetMark(uint32, string) error             { return errFixture }
func (fixtureDesk) RequestEquip(uint32, string) (any, error) { return nil, errFixture }
func (fixtureDesk) Unpin(string) error                       { return errFixture }
func (fixtureDesk) RequestTrash(uint32, bool) (any, error)   { return nil, errFixture }
func (fixtureDesk) Cancel(uint32, string) error              { return errFixture }
