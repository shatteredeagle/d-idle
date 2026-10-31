package d2dtown

import (
	"bufio"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town"
)

func serve(t *testing.T, rep map[string]string) *d2dctl.Client {
	t.Helper()
	dir, err := os.MkdirTemp("", "d2dtown")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	path := filepath.Join(dir, "d.sock")
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				rd := bufio.NewReader(c)
				for {
					line, err := rd.ReadString('\n')
					if err != nil {
						return
					}
					r, ok := rep[strings.TrimSpace(line)]
					if !ok {
						r = "ok\n"
					}
					c.Write([]byte(r))
				}
			}()
		}
	}()
	t.Cleanup(func() { ln.Close() })
	cl := d2dctl.New(path, d2dctl.Options{})
	t.Cleanup(func() { cl.Close() })
	return cl
}

type tab struct{}

func (tab) ItemRow(code string) (int, bool) {
	n, ok := map[string]int{"hp1": 587, "tbk": 529}[code]
	return n, ok
}
func (tab) MonsterClass(string) (int, bool) { return 0, false }

const vend = `{"v":1,"vendor":{"open":true,"vendor":0,"npc":4,"tab":3,"gamble":false,"tabs":[[],[],[],[{"id":-1,"code":"hp1","x":1,"y":0,"w":1,"h":1,"quality":2,"ilvl":6,"sockets":0,"qty":-1,"price":30}]]}}`

func TestVendorUnitsAndOwner(t *testing.T) {
	c := serve(t, map[string]string{"dump vendor": vend + "\nok\n"})
	a := New(c, nil, tab{})
	v, err := c.Vendor(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	u := a.vendorUnit(v.Items[0])
	if u.Class != 587 || u.Mode != d2.ItemModeStored || u.Item.Page != 3 || u.X != 1 || u.Y != 0 {
		t.Fatalf("%+v %+v", u, u.Item)
	}
	o, err := a.owner(u)
	if err != nil || o != uint32(d2dsrc.NPCIDBase+4) {
		t.Fatalf("owner %x %v", o, err)
	}
	if o, _ := a.owner(d2.Unit{Addr: 7}); o != playerAddr {
		t.Fatalf("carried item owner %x", o)
	}
}

func TestStatsFromExtras(t *testing.T) {
	a := New(nil, nil, nil)
	if _, err := a.stats(1); err == nil {
		t.Fatal("no snapshot yet must be unsupported")
	}
	a.last = &d2dsrc.Result{Extras: d2dsrc.Extras{Items: map[uint32]d2dsrc.ItemExtra{
		9: {Qty: 5}, 10: {Dur: 12, MaxDur: 24}}}}
	if st, _ := a.stats(9); st.Get(statQuantity) != 5 || st.Has(statMaxDurability) {
		t.Fatalf("%v", st)
	}
	if st, _ := a.stats(10); st.Get(statDurability) != 12 || st.Get(statMaxDurability) != 24 {
		t.Fatalf("%v", st)
	}
}

func TestMenuSeams(t *testing.T) {
	c := serve(t, map[string]string{"menu": "Akara\t580\t187\ntalk\t580\t205\ntrade\t580\t220\ncancel\t580\t235\nok\n"})
	a := New(c, nil, nil)
	if p, ok := a.menuEntry(town.EntryTrade); !ok || p != (input.Point{X: 580, Y: 220}) {
		t.Fatalf("%v %v", p, ok)
	}
	if p, ok := a.menuIndex(0); !ok || p.Y != 205 {
		t.Fatalf("%v %v", p, ok)
	}
	if _, ok := a.menuIndex(3); ok {
		t.Fatal("index past the last line")
	}
	if _, ok := a.menuEntry(town.EntryGamble); ok {
		t.Fatal("no gamble line")
	}
}

type rec struct {
	input.Controller
	ev []string
}

func (r *rec) MoveMouse(x, y int) error { r.ev = append(r.ev, "m"); return nil }
func (r *rec) Click(b input.Button, x, y int, m ...input.Mod) error {
	r.ev = append(r.ev, "c")
	return nil
}

func TestHoverFirst(t *testing.T) {
	r := &rec{}
	in := HoverFirst(r)
	_ = in.Click(input.Left, 10, 10)
	_ = in.Click(input.Left, 10, 10, input.ModShift)
	if got := strings.Join(r.ev, ""); got != "mmcc" {
		t.Fatalf("events %q", got)
	}
}

func TestSourceHasSeams(t *testing.T) {
	s := New(nil, nil, nil).Source()
	if s.Snapshot == nil || s.Items == nil || s.Owner == nil || s.Stats == nil || s.MenuEntry == nil || !s.FixedTabs {
		t.Fatal("source incomplete")
	}
}

func TestClassObjects(t *testing.T) {
	a := New(nil, nil, nil)
	r := &d2dsrc.Result{Snap: &d2.Snapshot{Player: &d2.Player{}}, Extras: d2dsrc.Extras{Names: map[uint32]string{
		1: "Your Private Stash", 2: "Waypoint", 3: "Something"}}}
	r.Snap.Player.Area = 40
	for id := uint32(1); id <= 3; id++ {
		r.Snap.Objects = append(r.Snap.Objects, d2.Unit{ID: id, Class: -1})
	}
	a.classObjects(r)
	if r.Snap.Objects[0].Class != town.ObjStash || r.Snap.Objects[1].Class != town.ObjWPAct2 || r.Snap.Objects[2].Class != -1 {
		t.Fatalf("%+v", r.Snap.Objects)
	}
}

type interactRec struct {
	input.Controller
	called bool
}

func (r *interactRec) Interact(input.Target) error { r.called = true; return nil }

func TestInputRoutesInteract(t *testing.T) {
	c := serve(t, map[string]string{
		"npcs":        "Charsi\t64\t-148\t1\t7\t23.1\t10.3\nok\n",
		"menu":        "Charsi\t265\t39\ntrade/repair\t265\t72\nok\n",
		"debug level": "ok 1 30.0 14.0 80 80 0 0\n",
	})
	r := &interactRec{}
	in := Input(r, c)
	if err := in.Interact(input.Target{ID: d2dsrc.NPCIDBase + 7}); err != nil || r.called {
		t.Fatalf("npc must go through ApproachNPC: %v called=%v", err, r.called)
	}
	if err := in.Interact(input.Target{ID: 99, Type: 4}); err != nil || !r.called {
		t.Fatalf("other targets pass through: %v", err)
	}
}
