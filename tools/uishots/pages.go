package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
	mountDemos(m)
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
		"schema": 7, "inTown": true, "class": "Sorceress", "gold": 1240, "stashGold": 5300,
		"prompt": "No weapon equipped. Equip one on the Inventory page; the bot will not do it for you.",
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
			item(10, "8ss", "Staff of Charged Bolt", "magic", 0, 0, 1, 3, map[string]any{"canSell": true, "canWear": []string{"rhand"}, "noWearSlot": noSlot("rhand")}),
			item(11, "rin", "Ring", "rare", 1, 0, 1, 1, map[string]any{"canWear": []string{"lring", "rring"}, "noWearSlot": noSlot("lring", "rring")}),
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
		"pins": []any{map[string]any{"slot": "lring", "id": 7, "name": "Raven Grip", "since": 0}},
		"queue": []any{map[string]any{"id": 11, "slot": "rring", "name": "Ring", "state": "queued", "at": 0},
			map[string]any{"id": 12, "kind": "unequip", "slot": "lring", "name": "Raven Grip", "state": "queued", "at": 0, "group": "a1", "groupName": "Fire damage", "seq": 1},
			map[string]any{"id": 10, "slot": "rhand", "name": "Staff of Charged Bolt", "state": "queued", "at": 0, "group": "a1", "groupName": "Fire damage", "seq": 2}},
		"recent": []any{},
		"merc": map[string]any{"state": "ready", "owned": true, "alive": true, "class": 338, "kind": "desert", "level": 14, "str": 60, "dex": 40, "lifePct": 90,
			"slots": []string{"mhead", "mtorso", "mrhand"}, "equipment": map[string]any{"mhead": item(1001, "cap", "Desert Cap", "magic", 0, 0, 2, 2, nil)},
			"actWhy": "the bot cannot change the mercenary's gear yet: the sets you save for him are kept, and you equip him by hand"},
		"sets": map[string]any{"schema": 1, "open": true, "max": 8, "sets": []any{
			map[string]any{"id": "gs-1", "name": "Fire damage", "target": "hero", "saved": 1700000000, "applying": "a1", "slots": []any{
				map[string]any{"slot": "rhand", "name": "Staff of Charged Bolt", "code": "8ss", "quality": "magic"}, map[string]any{"slot": "lring", "name": "Raven Grip", "code": "rin", "quality": "rare"}}},
			map[string]any{"id": "gs-2", "name": "Town and trade", "target": "hero", "saved": 1700000100, "slots": []any{
				map[string]any{"slot": "head", "name": "Cap", "code": "cap", "quality": "normal"}, map[string]any{"slot": "torso", "name": "Sturdy Quilted Armor", "code": "qui", "quality": "magic"},
				map[string]any{"slot": "feet", "name": "Boots", "code": "lbt", "quality": "normal"}}},
			map[string]any{"id": "gs-3", "name": "Desert Guard kit", "target": "merc", "saved": 1700000200, "slots": []any{map[string]any{"slot": "mhead", "name": "Desert Cap", "code": "cap", "quality": "magic"}}}}},
	}
}

// noSlot is the schema 7 noWearSlot map: every slot but the ones the item may go to, with the server's usual reason.
func noSlot(ok ...string) map[string]string {
	m := map[string]string{}
	for _, k := range []string{"head", "neck", "torso", "rhand", "lhand", "rring", "lring", "belt", "feet", "gloves"} {
		m[k] = "does not fit that slot"
	}
	for _, k := range ok {
		delete(m, k)
	}
	return m
}

var errFixture = errors.New("the fixture viewer does not write")

func (fixtureDesk) SetMark(uint32, string) error             { return errFixture }
func (fixtureDesk) RequestEquip(uint32, string) (any, error) { return nil, errFixture }
func (fixtureDesk) Unpin(string) error                       { return errFixture }
func (fixtureDesk) RequestTrash(uint32, bool) (any, error)   { return nil, errFixture }
func (fixtureDesk) Cancel(uint32, string) error              { return errFixture }
func (fixtureDesk) RequestIdentify(uint32) (any, error)      { return nil, errFixture }
func (fixtureDesk) RequestBuyID(int) (any, error)            { return nil, errFixture }
func (fixtureDesk) RequestCain() (any, error)                { return nil, errFixture }

// The schema 7 part: the viewer does not write, but the preview is a fixed, readable example.
func (fixtureDesk) RequestUnequip(string, string) (any, error)          { return nil, errFixture }
func (fixtureDesk) RequestEquipFor(uint32, string, string) (any, error) { return nil, errFixture }
func (fixtureDesk) SaveSet(string, string) (any, error)                 { return nil, errFixture }
func (fixtureDesk) RenameSet(string, string) error                      { return errFixture }
func (fixtureDesk) DeleteSet(string) error                              { return errFixture }
func (fixtureDesk) ApplySet(string) (any, error)                        { return nil, errFixture }
func (fixtureDesk) CancelApply(string) error                            { return errFixture }
func (fixtureDesk) PreviewSet(id string) (any, error) {
	return map[string]any{"setId": id, "name": "Fire damage", "target": "hero", "missing": 1, "ready": true,
		"slots": []any{
			map[string]any{"slot": "rhand", "name": "Staff of Charged Bolt", "now": "Short Staff", "status": "pack", "id": 10},
			map[string]any{"slot": "head", "name": "Great Helm", "status": "missing", "why": "not found in the backpack, stash or worn: skipped, nothing is substituted"},
			map[string]any{"slot": "lring", "name": "Raven Grip", "status": "worn", "id": 7},
			map[string]any{"slot": "rring", "name": "Ring", "status": "stash", "id": 30}},
		"steps": []any{map[string]any{"op": "take", "id": 30, "name": "Ring", "note": "from the stash, in town"},
			map[string]any{"op": "equip", "id": 10, "slot": "rhand", "name": "Staff of Charged Bolt"},
			map[string]any{"op": "equip", "id": 30, "slot": "rring", "name": "Ring"}}}, nil
}

var _ watch.InvGearDesk = fixtureDesk{}

// mountDemos serves /demo/<name>: the inventory page with one more script that acts on it the way a person would (a drag
// held over a slot, a preview opened), so the screenshot suite can picture states that need a pointer. The scripts are
// separate files because the page allows no inline script.
func mountDemos(m *http.ServeMux) {
	for name, script := range demoScripts {
		name, script := name, script
		m.HandleFunc("/demo/"+name+".js", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			io.WriteString(w, demoLib+script)
		})
		m.HandleFunc("/demo/"+name, func(w http.ResponseWriter, r *http.Request) {
			rec := httptest.NewRecorder()
			m.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/watch/inventory", nil))
			for k, v := range rec.Header() {
				w.Header()[k] = v
			}
			page := strings.Replace(rec.Body.String(), "</body>", `<script src="/demo/`+name+`.js"></script></body>`, 1)
			w.Header().Del("Content-Length")
			io.WriteString(w, page)
		})
	}
}

const demoLib = `(function(){window.demo={center:function(n){var r=n.getBoundingClientRect();return[r.left+r.width/2,r.top+r.height/2];},
fire:function(t,x,y,target){target.dispatchEvent(new PointerEvent(t,{bubbles:true,cancelable:true,clientX:x,clientY:y,pointerId:1,pointerType:'mouse',isPrimary:true,button:0}));},
when:function(test,fn){var n=0;(function w(){var v=test();if(v)return fn(v);if(++n<300)setTimeout(w,40);})();}};})();
`

var demoScripts = map[string]string{
	// a staff held over the head slot (red, with the reason) and the slots coloured
	"drag-red": `demo.when(function(){var t=document.querySelector('#pack [data-key="item-10"]'),s=document.querySelector('.d-slot2[data-slot="head"]');return t&&s&&window.d2InvActions&&[t,s];},function(a){
var p=demo.center(a[0]),q=demo.center(a[1]);demo.fire('pointerdown',p[0],p[1],a[0]);demo.fire('pointermove',p[0]+20,p[1]+20,document);demo.fire('pointermove',q[0],q[1],document);});`,
	// the same staff over its own slot (green)
	"drag-green": `demo.when(function(){var t=document.querySelector('#pack [data-key="item-10"]'),s=document.querySelector('.d-slot2[data-slot="rhand"]');return t&&s&&window.d2InvActions&&[t,s];},function(a){
var p=demo.center(a[0]),q=demo.center(a[1]);demo.fire('pointerdown',p[0],p[1],a[0]);demo.fire('pointermove',p[0]+20,p[1]+20,document);demo.fire('pointermove',q[0],q[1],document);});`,
	// the Apply preview of the first set, opened
	"preview": `demo.when(function(){return document.querySelector('[data-act="apply"]');},function(b){b.click();});`,
}
