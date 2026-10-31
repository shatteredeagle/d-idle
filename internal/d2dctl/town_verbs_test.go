package d2dctl

import (
	"bufio"
	"context"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// stub answers exact command lines; a handler may change the replies (a purchase lowers the gold).
type stub struct {
	mu   sync.Mutex
	rep  map[string]string
	seen []string
	hook func(line string, s *stub)
}

func startStub(t *testing.T, rep map[string]string, hook func(string, *stub)) (*stub, *Client) {
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
	s := &stub{rep: rep, hook: hook}
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
					line = strings.TrimSpace(line)
					s.mu.Lock()
					s.seen = append(s.seen, line)
					if s.hook != nil {
						s.hook(line, s)
					}
					r, ok := s.rep[line]
					s.mu.Unlock()
					if !ok {
						r = "ok\n"
					}
					c.Write([]byte(r))
				}
			}()
		}
	}()
	t.Cleanup(func() { ln.Close() })
	cl := New(path, Options{})
	t.Cleanup(func() { cl.Close() })
	return s, cl
}

func noPause(t *testing.T) {
	old := Pause
	Pause = func(ctx context.Context, d time.Duration) error { return ctx.Err() }
	t.Cleanup(func() { Pause = old })
}

const vendorJSON = `{"v":1,"vendor":{"open":true,"vendor":0,"npc":4,"tab":1,"gamble":false,"tabs":[[],[{"id":-1,"code":"wnd","x":0,"y":0,"w":1,"h":2,"quality":2,"ilvl":6,"sockets":0,"qty":-1,"price":205}],[],[{"id":-1,"code":"tbk","x":0,"y":3,"w":1,"h":2,"quality":2,"ilvl":6,"sockets":0,"qty":-1,"price":250}]]}}`

func TestParseVendor(t *testing.T) {
	v, err := ParseVendor([]byte(vendorJSON))
	if err != nil {
		t.Fatal(err)
	}
	if !v.Open || v.NPC != 4 || len(v.Items) != 2 {
		t.Fatalf("%+v", v)
	}
	if got := v.Items[1]; got.Tab != 3 || got.Code != "tbk" || got.Price != 250 || got.H != 2 {
		t.Fatalf("%+v", got)
	}
}

func TestVendorStockVerb(t *testing.T) {
	_, c := startStub(t, map[string]string{"dump vendor": vendorJSON + "\nok\n"}, nil)
	v, err := c.Vendor(context.Background())
	if err != nil || len(v.Items) != 2 {
		t.Fatalf("%v %+v", err, v)
	}
	_, c = startStub(t, map[string]string{"dump vendor": "err unknown section vendor\n"}, nil)
	if _, err := c.Vendor(context.Background()); err != ErrNoVendorDump {
		t.Fatalf("err = %v", err)
	}
}

func TestBuyItemClicksCellAndTab(t *testing.T) {
	noPause(t)
	gold := 1000
	s, c := startStub(t, map[string]string{}, func(line string, s *stub) {
		if strings.HasPrefix(line, "rclick") {
			gold -= 250
		}
		s.rep["state"] = "screen=ingame gold=" + itoa(gold) + " store=0 store_tab=1\nok\n"
	})
	paid, err := c.BuyItem(context.Background(), VendorItem{Tab: 3, X: 0, Y: 3, W: 1, H: 2, Code: "tbk"})
	if err != nil || paid != 250 {
		t.Fatalf("paid %d err %v", paid, err)
	}
	seen := strings.Join(s.seen, "|")
	// tab 3: x = 118 + 3*81 = 361; cell (0,3) 1x2: x = 97+14, y = 125+3*29+29
	for _, want := range []string{"click 361 75", "rclick 111 241"} {
		if !strings.Contains(seen, want) {
			t.Errorf("missing %q in %s", want, seen)
		}
	}
}

func TestBuyItemNoGold(t *testing.T) {
	noPause(t)
	_, c := startStub(t, map[string]string{"state": "screen=ingame gold=10 store_tab=2\nok\n"}, nil)
	if _, err := c.BuyItem(context.Background(), VendorItem{Tab: 1, Code: "wnd", W: 1, H: 2}); err == nil {
		t.Fatal("want an error when gold does not change")
	}
}

func TestSellItem(t *testing.T) {
	noPause(t)
	gold := 100
	_, c := startStub(t, map[string]string{"debug itemat 9": "ok 433 373\n"}, func(line string, s *stub) {
		if line == "click 213 299" {
			gold += 60
		}
		s.rep["state"] = "screen=ingame gold=" + itoa(gold) + "\nok\n"
	})
	got, err := c.SellItem(context.Background(), 9)
	if err != nil || got != 60 {
		t.Fatalf("got %d err %v", got, err)
	}
}

func TestOpenNPCMenuAndPick(t *testing.T) {
	noPause(t)
	n := 0
	s, c := startStub(t, map[string]string{}, func(line string, s *stub) {
		if line == "menu" {
			n++
			if n >= 3 {
				s.rep["menu"] = "Akara\t580\t187\ntalk\t580\t205\ntrade\t580\t220\ncancel\t580\t235\nok\n"
			} else {
				s.rep["menu"] = "ok\n"
			}
		}
	})
	m, err := c.OpenNPCMenu(context.Background(), 4, 5*time.Second)
	if err != nil || len(m) != 4 {
		t.Fatalf("%v %v", err, m)
	}
	if l, ok := FindLine(m, "Trade"); !ok || l.Y != 220 {
		t.Fatalf("%v %v", l, ok)
	}
	if _, ok := FindLine(m, "akara"); ok {
		t.Fatal("header must be skipped")
	}
	if err := c.PickMenuLine(context.Background(), "trade"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(s.seen, "|"), "move 577 220|move 580 220|click 580 220") {
		t.Fatalf("hover-click sequence missing: %v", s.seen)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestScreenOfCell(t *testing.T) {
	// V 2026-10-30: player 34.9,17.7, Waypoint cell 35.9,9.9 drew at about 1104,20; portal 34.7,20.7 at about 145,365.
	if x, y := ScreenOfCell(34.9, 17.7, 35.9, 9.9, 0); x < 1090 || x > 1110 || y < 10 || y > 35 {
		t.Fatalf("waypoint %d,%d", x, y)
	}
	if x, y := ScreenOfCell(34.9, 17.7, 34.7, 20.7, 50); x < 135 || x > 155 || y < 340 || y > 380 {
		t.Fatalf("portal %d,%d", x, y)
	}
}

func TestMoveGoldDepositAndWithdraw(t *testing.T) {
	noPause(t)
	purse, stash := 20000, 0
	pj := func() string {
		return `{"v":1,"player":{"gold":` + itoa(purse) + `,"stash_gold":` + itoa(stash) + "}}\nok\n"
	}
	var typed int
	s, c := startStub(t, map[string]string{}, func(line string, s *stub) {
		if strings.HasPrefix(line, "text ") {
			typed, _ = strconv.Atoi(line[5:])
		}
		if line == "key Return" {
			if s.rep["_dep"] == "1" {
				purse, stash = purse-typed, stash+typed
			} else {
				purse, stash = purse+typed, stash-typed
			}
		}
		if line == "click 493 461" {
			s.rep["_dep"] = "1"
		}
		if line == "click 165 92" {
			s.rep["_dep"] = "0"
		}
		s.rep["dump player"] = pj()
	})
	if err := c.MoveGold(context.Background(), 5000, true); err != nil || stash != 5000 {
		t.Fatalf("deposit: %v stash %d", err, stash)
	}
	if err := c.MoveGold(context.Background(), 2000, false); err != nil || purse != 17000 {
		t.Fatalf("withdraw: %v purse %d", err, purse)
	}
	if !strings.Contains(strings.Join(s.seen, "|"), "click 165 92") {
		t.Fatal("withdraw uses the stash coin")
	}
}

func TestRepairAllConfirmsByDurability(t *testing.T) {
	noPause(t)
	cur := 7
	items := func() string {
		return `{"v":1,"items":[{"loc":"body","dur":[` + itoa(cur) + `,15]},{"loc":"belt","dur":[0,0]}]}` + "\nok\n"
	}
	_, c := startStub(t, map[string]string{}, func(line string, s *stub) {
		if line == "click 368 461" {
			cur = 15
		}
		s.rep["dump items"] = items()
	})
	if err := c.RepairAll(context.Background()); err != nil || cur != 15 {
		t.Fatalf("%v %d", err, cur)
	}
	if err := c.RepairAll(context.Background()); err != nil {
		t.Fatalf("full gear is a no-op: %v", err)
	}
	cur = 7
	_, c = startStub(t, map[string]string{"dump items": `{"items":[{"loc":"body","dur":[7,15]}]}` + "\nok\n"}, nil)
	if err := c.RepairAll(context.Background()); err == nil {
		t.Fatal("unchanged durability must fail")
	}
}

func TestApproachNPCSidestepsWhenStalled(t *testing.T) {
	noPause(t)
	x, y := 33.5, 14.1
	opened := false
	s, c := startStub(t, map[string]string{"npcs": "Charsi\t64\t-148\t1\t7\t22.9\t11.5\nok\n"}, func(line string, s *stub) {
		if strings.HasPrefix(line, "cmd move") { // any side step frees the walk
			x, y = 29.4, 11.1
		}
		if line == "cmd interact 7" && x < 30 {
			opened = true
		}
		s.rep["debug level"] = "ok 1 " + strconv.FormatFloat(x, 'f', 2, 64) + " " + strconv.FormatFloat(y, 'f', 2, 64) + " 80 80 0 0\n"
		if opened {
			s.rep["menu"] = "Charsi\t265\t39\ntrade/repair\t265\t72\nok\n"
		}
	})
	if err := c.ApproachNPC(context.Background(), 7, time.Minute); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(s.seen, "|"), "cmd move") {
		t.Fatalf("no side step: %v", s.seen)
	}
}

func TestEnterPortalClicksSprite(t *testing.T) {
	noPause(t)
	level := 2
	s, c := startStub(t, map[string]string{"debug portals": "0\t2\t44.2\t24.8\n1\t1\t34.7\t20.7\nok\n"}, func(line string, s *stub) {
		if strings.HasPrefix(line, "click ") {
			level = 1
		}
		s.rep["debug level"] = "ok " + itoa(level) + " 44.3 24.9 96 56 0 0\n"
	})
	if err := c.EnterPortal(context.Background()); err != nil {
		t.Fatal(err)
	}
	// player 44.3,24.9 portal 44.2,24.8: sprite near the screen middle, above the feet
	if !strings.Contains(strings.Join(s.seen, "|"), "click 400 234") {
		t.Fatalf("click not on the sprite: %v", s.seen)
	}
}

func TestUIReadsInventoryRect(t *testing.T) {
	_, c := startStub(t, map[string]string{"dump ui": `{"ui":{"stash":true,"rects":{"inv_grid":[1,2,3,4]}}}` + "\nok\n"}, nil)
	u, err := c.UI(context.Background())
	if err != nil || !u.Inventory || !u.Stash || u.Vendor {
		t.Fatalf("%v %+v", err, u)
	}
}
