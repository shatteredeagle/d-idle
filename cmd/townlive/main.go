// townlive: live checks for internal/town (run via tools/live.sh town townlive <sub> [args]).
// Read-only subs: state, units. Click subs stay in the Rogue Encampment and end with CloseAll:
// menu CLASS [hold=S], shop CLASS [ENTRY] [hold=S], stash [hold=S], chores.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/cheat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/quest"
	"d2lod-bot/internal/town"
	"d2lod-bot/internal/town/hand"
)

func main() {
	src, err := mem.OpenGame() // honors D2BOT_INSTANCE
	die(err)
	defer src.Close()
	home, _ := os.UserHomeDir()
	db, err := data.Open(home+"/Games/d2lod/proton/pfx/drive_c/Program Files (x86)/Diablo II", "/tmp/d2lod-town-cache")
	die(err)
	rd := d2.New(src)
	r := input.Reader{Src: src}
	game := input.NewGame(input.NewClient(""), r)
	tune := env.DefaultTuning()
	e := &L{}
	e.Env = &env.Env{ID: "town", Log: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})), Mem: src, Read: rd,
		World: d2map.NewTracker(src), In: game, Data: db, Tune: &tune, Clock: env.RealClock{}, Emit: func(goals.Event) {}}
	_ = io.Discard
	nv := nav.New(e.Env)
	t := town.New(e.Env, nv)
	ctx := context.Background()
	sub := "state"
	if len(os.Args) > 1 {
		sub = os.Args[1]
	}
	arg := func(i int) int {
		if len(os.Args) > i {
			n, _ := strconv.Atoi(os.Args[i])
			return n
		}
		return 0
	}
	hold := 0
	for _, a := range os.Args {
		if v, ok := strings.CutPrefix(a, "hold="); ok {
			hold, _ = strconv.Atoi(v)
		}
	}
	switch sub {
	case "state":
		e.state()
	case "units":
		e.units()
	case "menu":
		npc, err := t.OpenForTest(ctx, arg(2))
		fmt.Println("open:", npc.Class, err)
		e.ui()
		if vx, vy, err := r.Viewport(); err == nil {
			ss, _ := e.Snapshot()
			for _, m := range ss.Monsters {
				if m.ID == npc.ID {
					sx, sy := game.X.WorldToScreenVP(vx, vy, float64(m.X)+0.5, float64(m.Y)+0.5)
					fmt.Printf("npc world (%d,%d) screen (%.0f,%.0f); player (%d,%d)\n", m.X, m.Y, sx, sy, ss.Player.X, ss.Player.Y)
				}
			}
		}
		e.wait(hold)
		fmt.Println("closeall:", game.CloseAll())
	case "talk": // talk CLASS [hold=S]: click a loaded NPC from where we stand (no walk), report the dialog (merc probe)
		npc, err := t.OpenDirectForTest(ctx, arg(2))
		fmt.Println("open:", npc.Class, err)
		e.ui()
		if err == nil {
			for _, a := range os.Args[3:] { // IDX:N lines to click after the dialog opened
				var i, n int
				var x, y, w int
				if _, err := fmt.Sscanf(a, "w:%d", &w); err == nil {
					fmt.Println("wait", w)
					e.wait(w)
				} else if _, err := fmt.Sscanf(a, "h:%d:%d", &x, &y); err == nil { // hover only, from 3 px away
					fmt.Println("hover", game.MoveMouse(x-3, y))
					time.Sleep(200 * time.Millisecond)
					fmt.Println("hover", game.MoveMouse(x, y))
				} else if _, err := fmt.Sscanf(a, "k:%d:%d", &x, &y); err == nil { // click where the pointer is
					fmt.Println("click", game.Click(input.Left, x, y))
					time.Sleep(1500 * time.Millisecond)
					e.state()
				} else if _, err := fmt.Sscanf(a, "c:%d:%d", &x, &y); err == nil { // hover, then click a screen point
					fmt.Println("hover", game.MoveMouse(x, y))
					time.Sleep(500 * time.Millisecond)
					fmt.Println("click", game.Click(input.Left, x, y))
					time.Sleep(1500 * time.Millisecond)
					e.state()
				} else if _, err := fmt.Sscanf(a, "%d:%d", &i, &n); err == nil {
					fmt.Println("pickline", i, n, t.PickLine(ctx, npc, i, n))
					time.Sleep(1500 * time.Millisecond)
					e.ui()
				}
			}
		}
		e.wait(hold)
		fmt.Println("closeall:", game.CloseAll())
	case "hire": // hire ROW [AURA]: Town.Hire in the current act (Normal), clicking hire-window row ROW (0 = top)
		aura := "Blessed Aim"
		if len(os.Args) > 3 && !strings.Contains(os.Args[3], "=") {
			aura = os.Args[3]
		}
		ss, _ := e.Snapshot()
		act := 2
		if a := town.ActOfLevel(ss.Player.Area); a != nil {
			act = a.No
		}
		rep, err := t.Hire(ctx, town.MercWant{Act: act, Diff: data.Normal, Aura: aura, Row: arg(2)})
		fmt.Printf("hire: %+v err=%v\n", rep, err)
		e.state()
	case "shop":
		npc, err := t.OpenForTest(ctx, arg(2))
		fmt.Println("open:", npc.Class, err)
		if err == nil {
			fmt.Println("trade:", t.PickEntryForTest(ctx, npc, town.EntryTrade, d2.UIShop))
			time.Sleep(900 * time.Millisecond)
			e.ui()
			e.stock(t, npc)
			for _, a := range os.Args[2:] {
				if v, ok := strings.CutPrefix(a, "tab="); ok { // tab=N: click vendor tab N (0-based), then wait for a screenshot
					n, _ := strconv.Atoi(v)
					fmt.Println("tab click:", game.Click(input.Left, 118+81*n, 75))
				}
			}
			for _, a := range os.Args[2:] {
				if v, ok := strings.CutPrefix(a, "rclick="); ok { // rclick=X,Y raw right click (buy probe)
					xs, ys, _ := strings.Cut(v, ",")
					x, _ := strconv.Atoi(xs)
					y, _ := strconv.Atoi(ys)
					time.Sleep(700 * time.Millisecond)
					fmt.Println("rclick:", game.Click(input.Right, x, y))
				}
			}
			e.wait(hold)
		}
		fmt.Println("closeall:", game.CloseAll())
	case "trade": // trade CLASS: sell the identify scroll, then buy back what the gold allows (cheap potions)
		npc, err := t.OpenForTest(ctx, arg(2))
		fmt.Println("open:", npc.Class, err)
		if err == nil {
			fmt.Println("trade:", t.PickEntryForTest(ctx, npc, town.EntryTrade, d2.UIShop))
			e.state()
			s, _ := e.Snapshot()
			for _, u := range s.Inventory.Container(d2.ContInventory) {
				if e.Data.Items[u.Class].Code == "isc" {
					g, err := t.SellForTest(ctx, u)
					fmt.Println("sold isc: gained", g, "err", err)
				}
			}
			s, _ = e.Snapshot()
			fmt.Println("gold now", s.Player.Gold)
			for _, code := range []string{"hp1", "mp1"} {
				if s.Player.Gold >= 30 {
					p, err := t.BuyForTest(ctx, npc, code)
					fmt.Println("bought", code, "price", p, "err", err)
					s, _ = e.Snapshot()
				}
			}
			e.state()
			e.wait(hold)
		}
		fmt.Println("closeall:", game.CloseAll())
	case "sellprobe": // sellprobe CLASS: click the identify scroll in the open shop and report what memory shows
		npc, err := t.OpenForTest(ctx, arg(2))
		fmt.Println("open:", npc.Class, err)
		if err == nil {
			fmt.Println("trade:", t.PickEntryForTest(ctx, npc, town.EntryTrade, d2.UIShop))
			x, y := 419+9*29+14, 315+2*29+14
			for _, b := range []input.Button{input.Left, input.Right} {
				fmt.Println("click", b, x, y, game.Click(b, x, y))
				time.Sleep(700 * time.Millisecond)
				ss, _ := e.Snapshot()
				fmt.Println(" mouse", ss.Mouse, "cursor", ss.Inventory.Cursor != nil, "gold", ss.Player.Gold, "items", len(ss.Inventory.Items))
				e.ui()
				if ss.Inventory.Cursor != nil {
					fmt.Println(" cursor item", e.name(*ss.Inventory.Cursor))
					c := 96 + 5*29 + 14
					fmt.Println(" drop on vendor", game.Click(input.Left, c, 123+8*29+14))
					time.Sleep(700 * time.Millisecond)
					ss, _ = e.Snapshot()
					fmt.Println(" after drop: gold", ss.Player.Gold, "items", len(ss.Inventory.Items), "cursor", ss.Inventory.Cursor != nil)
				}
			}
		}
		fmt.Println("closeall:", game.CloseAll())
	case "chores": // chores [heal identify sell potions scrolls repair stash gold insure]: forced run; no words = FullPlan
		plan := town.ChorePlan{Force: true}
		if len(os.Args) == 2 {
			plan = town.FullPlan()
			plan.Force = true
		}
		for _, w := range os.Args[2:] {
			switch w {
			case "heal":
				plan.Heal = true
			case "identify":
				plan.Identify = true
			case "sell":
				plan.Sell = true
			case "potions":
				plan.Potions = true
			case "scrolls":
				plan.Scrolls = true
			case "repair":
				plan.Repair = true
			case "stash":
				plan.Stash = true
			case "gold":
				plan.Gold = true
			case "insure":
				plan.Insure = true
			case "ammo":
				plan.Ammo = true
				t.SetAmmoPath("bow")
			case "javelin":
				plan.Ammo = true
				t.SetAmmoPath("javelin")
			case "noforce":
				plan.Force = false
			}
			if v, ok := strings.CutPrefix(w, "keys="); ok {
				e.Tune.Town.KeyTarget, _ = strconv.Atoi(v)
			}
		}
		ctxT, cancel := context.WithTimeout(ctx, 150*time.Second)
		rep, err := t.Chores(ctxT, plan)
		cancel()
		fmt.Printf("chores err=%v gold %d -> %d spent=%d earned=%d bought=%v sold=%d stashed=%d identified=%d healed=%v repaired=%v broke=%v\n",
			err, rep.GoldBefore, rep.GoldAfter, rep.Spent, rep.Earned, rep.Bought, rep.Sold, rep.Stashed, rep.Identified, rep.Healed, rep.Repaired, rep.Broke)
		for _, st := range rep.Steps {
			fmt.Printf("  step %-9s ran=%v err=%q detail=%q\n", st.Name, st.Ran, st.Err, st.Detail)
		}
		e.state()
	case "rebuy":
		rep, err := t.Rebuy(ctx, arg(2))
		fmt.Printf("rebuy err=%v %+v\n", err, rep)
		e.state()
	case "totown":
		fmt.Println("ToTown:", t.ToTown(ctx))
		e.state()
	case "fightportal": // fightportal SECONDS: god mode on (logged cheat), melee the nearest Blood Moor monster, then ToTown mid-fight
		gate := cheat.Gate{Allow: true, Env: cheat.DefaultEnv(), Actor: "town2"}
		stop := gate.God(ctx, "TOWN2: survive a melee fight while testing ToTown with monsters adjacent (throwaway character Towni)")
		defer func() { fmt.Println("god stop:", stop()) }()
		time.Sleep(1500 * time.Millisecond)
		end := time.Now().Add(time.Duration(arg(2)) * time.Second)
		for time.Now().Before(end) {
			ss, err := e.Snapshot()
			if err != nil {
				break
			}
			var tgt *d2.Unit
			bd := 1 << 30
			for i, m := range ss.Monsters {
				if m.Class == 152 || m.Class == 147 || m.Class == 148 || m.Class == 154 || m.Class == 150 || m.Class == 155 || m.Class == 179 || !m.Alive() {
					continue
				}
				if d := int(math.Hypot(float64(m.X-ss.Player.X), float64(m.Y-ss.Player.Y))); d < bd {
					bd, tgt = d, &ss.Monsters[i]
				}
			}
			if tgt == nil {
				fmt.Println("no monster in range")
				break
			}
			if bd > 6 {
				fmt.Println("walk to monster", tgt.Class, "d", bd, nv.GoTo(ctx, input.Point{X: tgt.X, Y: tgt.Y}))
				continue
			}
			fmt.Println("attack", tgt.Class, "d", bd, game.Interact(input.Target{Type: 1, ID: tgt.ID, Pos: input.Point{X: tgt.X, Y: tgt.Y}}))
			time.Sleep(400 * time.Millisecond)
		}
		ss, _ := e.Snapshot()
		n := 0
		for _, m := range ss.Monsters {
			if m.Alive() && m.Class != 152 && math.Hypot(float64(m.X-ss.Player.X), float64(m.Y-ss.Player.Y)) < 12 {
				n++
			}
		}
		fmt.Println("monsters within 12 subtiles:", n, "life", ss.Player.Life, "/", ss.Player.MaxLife)
		start := time.Now()
		fmt.Println("ToTown in fight:", t.ToTown(ctx), "took", time.Since(start).Round(100*time.Millisecond))
		e.state()
	case "groundtest": // drop the first non-start pack item next to the player, then pick it up with loot.PickUp
		h := hand.New(e.Env)
		ss, err := h.OpenInventory(ctx)
		die(err)
		lay, err := h.Layout(ss)
		die(err)
		var it d2.Unit
		for _, u := range ss.Inventory.Container(d2.ContInventory) {
			if e.Data.Items[u.Class].Code == "dgr" {
				it = u
			}
		}
		if it.ID != 0 {
			fmt.Println("take:", h.Take(ctx, it, lay.Inventory))
			fmt.Println("close:", game.CloseAll()) // Esc closes the panel; the item stays on the cursor
			time.Sleep(400 * time.Millisecond)
			fmt.Println("drop click:", game.Click(input.Left, 470, 345))
			time.Sleep(900 * time.Millisecond)
		} else {
			fmt.Println("no dagger in the pack; using what lies on the ground")
			fmt.Println("close:", game.CloseAll())
		}
		ss, _ = e.Snapshot()
		fmt.Printf("ground items: %d cursor %v\n", len(ss.Ground), ss.Inventory.Cursor != nil)
		for _, g := range ss.Ground {
			fmt.Printf("  ground %s id=%d at (%d,%d) player (%d,%d)\n", e.name(g), g.ID, g.X, g.Y, ss.Player.X, ss.Player.Y)
		}
		e.Tune.Loot.PickStrictness = 0
		lt := loot.New(e.Env, nv)
		pk, perr := loot.ParsePickit(strings.NewReader(`{"default":{"action":"drop","tier":"C","reason":"x"},"rules":[{"name":"test","match":{"codes":["dgr"]},"action":"keep","tier":"S","prio":90,"reason":"live pickup test"}]}`))
		die(perr)
		lt.SetPickit(pk)
		rep, err := lt.PickUp(ctx, loot.PickOpts{Radius: 20, Max: 3, StrictSet: true, Strictness: 0})
		fmt.Printf("pickup err=%v picked=%+v skipped=%+v failed=%+v\n", err, rep.Picked, rep.Skipped, rep.Failed)
		e.state()
		fmt.Println("closeall:", game.CloseAll())
	case "pickline": // pickline CLASS IDX:N [IDX:N ...] [hold=S]: open the NPC, click each line (index from the top : line count), hold for screenshots
		npc, err := t.OpenForTest(ctx, arg(2))
		fmt.Println("open:", npc.Class, err)
		if err == nil {
			for _, a := range os.Args[3:] {
				var i, n int
				if _, err := fmt.Sscanf(a, "%d:%d", &i, &n); err == nil {
					fmt.Println("pickline", i, n, t.PickLine(ctx, npc, i, n))
					time.Sleep(1500 * time.Millisecond)
					e.ui()
				}
			}
			e.wait(hold)
		}
		fmt.Println("closeall:", game.CloseAll())
	case "claimden": // claimden [respec=1]: the Akara Den reward claim as the d2bot dialog adapter does it (quest.PickPath "Talk" + PickLine), proof = skill point
		sp := func() int {
			if ss, err := e.Snapshot(); err == nil && ss.Player != nil {
				return ss.Player.SkillPoints
			}
			return -1
		}
		before := sp()
		for try := 0; try < 3; try++ {
			respec := (try == 1) != (len(os.Args) > 2 && os.Args[2] == "respec=1")
			st := func(q string) quest.Status {
				if q == quest.RespecRow && respec {
					return quest.Started
				}
				if q == "den_of_evil" {
					return quest.Started
				}
				return quest.Needed
			}
			d := &liveDialog{t: t, e: e.Env}
			fmt.Println("try", try, "respec rows", respec, "open:", d.Talk(ctx, quest.Akara))
			fmt.Println("pickpath:", quest.PickPath(ctx, d, quest.Akara, st, "Talk"))
			for i := 0; i < 27 && sp() <= before; i++ {
				time.Sleep(300 * time.Millisecond)
			}
			fmt.Println("skill points", before, "->", sp())
			_ = d.Close(ctx)
			if sp() > before {
				break
			}
		}
		e.wait(hold)
	case "openportal":
		fmt.Println("OpenPortal:", t.OpenPortalForTest(ctx))
		e.state()
	case "back":
		fmt.Println("BackToField:", t.BackToField(ctx, arg(2)))
		e.state()
	case "exit":
		fmt.Println("ExitTo:", nv.ExitTo(ctx, arg(2)))
		e.state()
	case "presets":
		_, _ = e.World.Update()
		if lm := e.World.Current(); lm != nil {
			for _, p := range lm.Presets {
				if p.Kind != 5 {
					fmt.Printf("preset kind=%d id=%d at (%d,%d)\n", p.Kind, p.ID, p.Pos.X, p.Pos.Y)
				}
			}
		}
	case "hover": // hover CLASS: sweep the mouse around the NPC's computed screen position and print what is hovered
		ss, _ := e.Snapshot()
		var npc d2.Unit
		for _, m := range ss.Monsters {
			if m.Class == arg(2) {
				npc = m
			}
		}
		vx, vy, verr := r.Viewport()
		sx, sy := game.X.WorldToScreenVP(vx, vy, float64(npc.X)+0.5, float64(npc.Y)+0.5)
		fmt.Printf("player (%d,%d) npc (%d,%d) viewport (%d,%d) err=%v npc screen (%.0f,%.0f)\n", ss.Player.X, ss.Player.Y, npc.X, npc.Y, vx, vy, verr, sx, sy)
		for dy := -90; dy <= 20; dy += 10 {
			line := ""
			for dx := -40; dx <= 40; dx += 10 {
				_ = game.MoveMouse(int(sx)+dx, int(sy)+dy)
				time.Sleep(120 * time.Millisecond)
				typ, id, _ := r.Hovered()
				line += fmt.Sprintf(" %d/%d", typ, id)
			}
			fmt.Println(dy, line)
		}
	case "drink": // drink belt potions 1..4 (hotkeys) to empty the belt
		for _, k := range []string{"1", "2", "3", "4"} {
			fmt.Println("key", k, game.Key(k))
			time.Sleep(600 * time.Millisecond)
		}
		e.state()
	case "tip": // tip CLASS X,Y ...: open the shop and park the mouse on each point (screenshots read the tooltips)
		npc, err := t.OpenForTest(ctx, arg(2))
		fmt.Println("open:", npc.Class, err)
		if err == nil {
			fmt.Println("trade:", t.PickEntryForTest(ctx, npc, town.EntryTrade, d2.UIShop))
			for _, a := range os.Args[3:] {
				var x, y int
				if _, err := fmt.Sscanf(a, "%d,%d", &x, &y); err != nil {
					continue
				}
				fmt.Println("POINT", x, y, game.MoveMouse(x, y))
				e.wait(5)
			}
		}
		fmt.Println("closeall:", game.CloseAll())
	case "strip": // strip CLASS: sell all equipped gear to the vendor of CLASS (Charsi 154 buys weapons and armor)
		npc, err := t.OpenForTest(ctx, arg(2))
		fmt.Println("open:", npc.Class, err)
		if err == nil {
			fmt.Println("trade:", t.PickEntryForTest(ctx, npc, town.EntryTrade, d2.UIShop))
			g, err := t.StripForTest(ctx)
			fmt.Println("stripped: gained", g, "err", err)
			e.state()
		}
		fmt.Println("closeall:", game.CloseAll())
	case "equip": // equip CODE: equip a pack item by pick-up and slot click
		fmt.Println("equip:", t.EquipForTest(ctx, os.Args[2]))
		e.state()
	case "gamble": // gamble CODE: Gheed's gamble window, buy one unidentified item of that base code
		npc, err := t.OpenForTest(ctx, 147)
		fmt.Println("open:", npc.Class, err)
		if err == nil {
			fmt.Println("gamble entry:", t.PickEntryForTest(ctx, npc, town.EntryGamble, d2.UIShop))
			time.Sleep(900 * time.Millisecond)
			e.stock(t, npc)
			e.wait(hold)
			if len(os.Args) > 2 && !strings.HasPrefix(os.Args[2], "hold=") {
				code := os.Args[2]
				if code == "cheapest" {
					st, _ := t.Stock(npc)
					best := 1 << 30
					for _, u := range st {
						if b := e.Data.Items[u.Class]; b.Cost < best {
							best, code = b.Cost, b.Code
						}
					}
				}
				p, err := t.BuyForTest(ctx, npc, code)
				fmt.Println("gambled", os.Args[2], "price", p, "err", err)
				e.ui()
				e.wait(hold)
			}
			e.state()
		}
		fmt.Println("closeall:", game.CloseAll())
	case "gbtn": // gbtn X Y: open the stash, click the point (default: the gold button), hold for screenshots
		err := t.OpenStashForTest(ctx)
		fmt.Println("open:", err)
		if err == nil {
			x, y := arg(2), arg(3)
			e.ui()
			e.wait(8)
			if x == 0 {
				x, y = town.StashGoldButton.X, town.StashGoldButton.Y
			}
			fmt.Println("click:", game.Click(input.Left, x, y))
			time.Sleep(700 * time.Millisecond)
			e.ui()
			e.state()
			e.wait(hold)
		}
		fmt.Println("closeall:", game.CloseAll())
	case "sclicks": // sclicks X,Y ...: open the stash, left-click each point in turn (5 s apart), Esc between only if asked
		err := t.OpenStashForTest(ctx)
		fmt.Println("open:", err)
		if err == nil {
			for _, a := range os.Args[2:] {
				var x, y int
				if _, err := fmt.Sscanf(a, "%d,%d", &x, &y); err != nil {
					continue
				}
				_ = game.MoveMouse(x, y)
				time.Sleep(400 * time.Millisecond)
				fmt.Println("POINT", x, y, game.Click(input.Left, x, y))
				e.ui()
				e.wait(5)
			}
			e.state()
		}
		fmt.Println("closeall:", game.CloseAll())
	case "deposit": // deposit N: open the stash and deposit exactly N gold (typed with WM_CHAR digits)
		err := t.OpenStashForTest(ctx)
		fmt.Println("open:", err)
		if err == nil {
			fmt.Println("deposit:", t.DepositForTest(ctx, arg(2)))
			e.state()
		}
		fmt.Println("closeall:", game.CloseAll())
	case "buy": // buy CLASS CODE [CODE...] [hold=S]: open the shop of the NPC class and buy each code once
		npc, err := t.OpenForTest(ctx, arg(2))
		fmt.Println("open:", npc.Class, err)
		if err == nil {
			fmt.Println("trade:", t.PickEntryForTest(ctx, npc, town.EntryTrade, d2.UIShop))
			for _, code := range os.Args[3:] {
				if strings.HasPrefix(code, "hold=") {
					continue
				}
				p, err := t.BuyForTest(ctx, npc, code)
				fmt.Println("bought", code, "price", p, "err", err)
				if hold > 0 {
					e.wait(hold)
				}
			}
			e.state()
		}
		fmt.Println("closeall:", game.CloseAll())
	case "stash":
		err := t.OpenStashForTest(ctx)
		fmt.Println("open:", err)
		e.ui()
		if err == nil {
			fmt.Println("roundtrip:", t.StashRoundTripForTest(ctx))
			e.Tune.Town.StashGoldPerLvl, e.Tune.Town.StashGoldBase = 0, 0
			fmt.Println("gold deposit:", t.StashGoldForTest(ctx))
		}
		e.state()
		e.wait(hold)
		fmt.Println("closeall:", game.CloseAll())
	default:
		fmt.Println("unknown sub", sub)
	}
	e.ui()
}

type L struct{ *env.Env }

func die(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func (e *L) wait(s int) { time.Sleep(time.Duration(s) * time.Second) }

func (e *L) ui() {
	s, err := e.Snapshot()
	die(err)
	var on []string
	for i, v := range s.UI {
		if v != 0 {
			on = append(on, fmt.Sprintf("0x%x", i))
		}
	}
	fmt.Println("ui flags:", on, "cursor:", s.Inventory != nil && s.Inventory.Cursor != nil)
}

func (e *L) name(u d2.Unit) string {
	if u.Class >= 0 && u.Class < len(e.Data.Items) {
		b := e.Data.Items[u.Class]
		return b.Code + " " + b.Display
	}
	return fmt.Sprint("class", u.Class)
}

func (e *L) state() {
	s, err := e.Snapshot()
	die(err)
	p := s.Player
	fmt.Printf("player %s class=%v clvl=%d area=%d pos=(%d,%d) life %d/%d mana %d/%d gold=%d stash=%d str=%d dex=%d screen=%v\n",
		p.Name, p.Class, p.Level, p.Area, p.X, p.Y, p.Life, p.MaxLife, p.Mana, p.MaxMana, p.Gold, p.StashGold, p.Strength, p.Dexterity, s.Screen)
	for _, u := range s.Inventory.Items {
		fmt.Printf("  item id=%d %-28s where=%v cell=(%d,%d) body=%d q=%d flags=0x%x mode=%d page=%d\n", u.ID, e.name(u), u.Item.Where, u.X, u.Y, u.Item.Body, u.Item.Quality, u.Item.Flags, u.Mode, u.Item.Page)
		if st, err := e.Read.UnitStats(u.Addr); err == nil {
			fmt.Printf("      stats %v\n", st)
		}
	}
	if s.Inventory.Cursor != nil {
		fmt.Println("  cursor:", e.name(*s.Inventory.Cursor))
	}
}

func (e *L) units() {
	s, err := e.Snapshot()
	die(err)
	for _, m := range s.Monsters {
		fmt.Printf("npc class=%d id=%d at (%d,%d) mode=%d\n", m.Class, m.ID, m.X, m.Y, m.Mode)
	}
	for _, o := range s.Objects {
		if o.Class == 267 || o.Class == 119 || o.Class == 59 || math.Hypot(float64(o.X-s.Player.X), float64(o.Y-s.Player.Y)) < 40 {
			fmt.Printf("obj class=%d id=%d at (%d,%d) mode=%d\n", o.Class, o.ID, o.X, o.Y, o.Mode)
		}
	}
	for _, g := range s.Ground {
		fmt.Printf("ground %s id=%d at (%d,%d)\n", e.name(g), g.ID, g.X, g.Y)
	}
}

func (e *L) stock(t *town.Town, npc d2.Unit) {
	st, err := t.Stock(npc)
	fmt.Println("stock:", len(st), err)
	for _, u := range st {
		fmt.Printf("  %-28s cell=(%d,%d) page=%d where=%v mode=%d\n", e.name(u), u.X, u.Y, u.Item.Page, u.Item.Where, u.Mode)
	}
}

// liveDialog is the quest.Dialog over the Townsman (the same shape as cmd/d2bot's townDialog).
type liveDialog struct {
	t   *town.Town
	e   *env.Env
	npc d2.Unit
}

func (d *liveDialog) Talk(ctx context.Context, class int) error {
	n, err := d.t.OpenForTest(ctx, class)
	d.npc = n
	return err
}
func (d *liveDialog) Choose(ctx context.Context, row, of int) error {
	err := d.t.PickLine(ctx, d.npc, row, of)
	d.e.Clock.Sleep(700 * time.Millisecond)
	return err
}
func (d *liveDialog) Close(ctx context.Context) error { return d.e.In.CloseAll() }
