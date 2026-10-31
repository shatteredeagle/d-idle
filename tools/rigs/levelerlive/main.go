//go:build rigs

// levelerlive: live probes and checks for internal/leveler (tools/live.sh bot levelerlive <sub> [args]).
// Subs: state, ui, click l|r X Y, key NAME, spend (stat and skill points), skills, items, equip, shop.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/leveler/townvendor"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/town"
	"d2lod-bot/internal/town/hand"
)

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func main() {
	src, err := mem.OpenGame()
	die(err)
	defer src.Close()
	inst := os.Getenv("D2BOT_INSTANCE")
	home, _ := os.UserHomeDir()
	dir := home + "/Games/d2lod/proton/pfx/drive_c/Program Files (x86)/Diablo II"
	if inst != "" && inst != "i0" {
		dir = home + "/Games/d2lod-inst/" + inst + "/proton/pfx/drive_c/Program Files (x86)/Diablo II"
	}
	db, err := data.Open(dir, "/tmp/d2lod-leveler-cache")
	die(err)
	cl := input.NewClient("")
	rd := d2.New(src)
	tune := env.DefaultTuning()
	e := &env.Env{ID: "leveler", Log: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})),
		Mem: src, Read: rd, World: d2map.NewTracker(src), In: input.NewGame(cl, input.Reader{Src: src}), Data: db, Tune: &tune, Clock: env.RealClock{}, Emit: func(goals.Event) {}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
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
	lv := leveler.New(e, leveler.Options{DebugHold: 9000})
	switch sub {
	case "state":
		s, err := e.Snapshot()
		die(err)
		p := s.Player
		fmt.Printf("lvl %d exp %d gold %d stash %d pts stat=%d skill=%d area %d diff %d\n", p.Level, p.Exp, p.Gold, p.StashGold, p.StatPoints, p.SkillPoints, p.Area, p.Difficulty)
		fmt.Printf("str %d dex %d vit %d ene %d life %d/%d mana %d/%d\n", p.Strength, p.Dexterity, p.Vitality, p.Energy, p.Life, p.MaxLife, p.Mana, p.MaxMana)
		fmt.Println("skills:", p.Skills, "left", p.LeftSkill, "right", p.RightSkill)
		fmt.Println("ui:", s.UI)
		fmt.Println("pos:", p.X, p.Y, "mode", p.Mode)
		for _, m := range s.Monsters {
			fmt.Printf("  mon class=%d id=%d (%d,%d) mode=%d\n", m.Class, m.ID, m.X, m.Y, m.Mode)
		}
		b, ok := lv.BaseAttrs(p)
		fmt.Println("base attrs:", b, ok)
		for _, sk := range p.Skills {
			if d := db.SkillByID(sk.ID); d != nil {
				fmt.Printf("  skill %d %s lvl %d page %d row %d col %d\n", sk.ID, d.Key, sk.Level, d.Page, d.Row, d.Column)
			}
		}
	case "tree":
		code := classCode(d2.Sorceress)
		if len(os.Args) > 2 { // tree [ama|sor|nec|pal|bar|dru|ass]
			code = os.Args[2]
		} else if s, err := e.Snapshot(); err == nil && s.Player != nil {
			code = classCode(s.Player.Class) // the logged-in character's class, not a hard-coded one
		}
		for _, sk := range db.Skills {
			if sk.Class == code {
				fmt.Printf("%3d %-18s page %d row %d col %d reqlvl %d req %v\n", sk.ID, sk.Key, sk.Page, sk.Row, sk.Column, sk.ReqLevel, sk.ReqSkills)
			}
		}
	case "ui":
		s, err := e.Snapshot()
		die(err)
		fmt.Println(s.UI)
	case "click": // click l|r X Y
		btn := input.Left
		if os.Args[2] == "r" {
			btn = input.Right
		}
		die(e.In.Click(btn, arg(3), arg(4)))
	case "move":
		die(e.In.MoveMouse(arg(2), arg(3)))
	case "key":
		die(e.In.Key(os.Args[2]))
	case "spend":
		s, err := e.Snapshot()
		die(err)
		rep := lv.Do(ctx, s)
		fmt.Printf("%+v\n", rep)
	case "plan":
		s, err := e.Snapshot()
		die(err)
		fmt.Printf("%+v\n", lv.PlanFor(s))
	case "items":
		s, err := e.Snapshot()
		die(err)
		lv.DumpItems(os.Stdout, s)
		for _, sw := range lv.SwapsFor(s) {
			fmt.Println("swap:", sw)
		}
	case "shop":
		nv := nav.New(e)
		tw := town.New(e, nv)
		if arg(3) != 0 { // this game's town sits at another world offset than the table in internal/town: walk near the NPC first
			fmt.Println("pre-walk:", nv.GoTo(ctx, input.Point{X: arg(3), Y: arg(4)}))
		}
		rep := lv.Shop(ctx, townvendor.Vendor{T: tw}, arg(2))
		fmt.Printf("%+v\n", rep)
	case "opentrade": // opentrade CLASS HOLDSECONDS: open a vendor and keep it open for a screenshot
		nv := nav.New(e)
		tv := townvendor.Vendor{T: town.New(e, nv)}
		npc, err := tv.OpenTrade(ctx, arg(2))
		fmt.Println("open:", npc.Class, err)
		if err == nil {
			st, err := tv.Stock(npc)
			fmt.Println("stock:", len(st), err)
			s, _ := e.Snapshot()
			for _, u := range st {
				c := lv.Candidate(s.Player, u)
				fmt.Printf("  page=%d cell=(%d,%d) %q q=%d score=%.1f price~%d usable=%v %s\n", u.Item.Page, u.X, u.Y, c.Name, u.Item.Quality, c.Score, c.Price, c.Usable, c.Why)
			}
		}
		time.Sleep(time.Duration(arg(3)) * time.Second)
		fmt.Println("closeall:", e.In.CloseAll())
	case "trybuy": // trybuy CLASS PAGE X Y: open vendor, click the tab of PAGE, right-click the grid cell, screenshot windows in between
		nv := nav.New(e)
		tv := townvendor.Vendor{T: town.New(e, nv)}
		npc, err := tv.OpenTrade(ctx, arg(2))
		fmt.Println("open:", npc.Class, err)
		s, _ := e.Snapshot()
		g0 := s.Player.Gold
		if arg(3) >= 0 {
			tab := leveler.ShopTabs[arg(3)]
			fmt.Println("tab click", tab, e.In.Click(input.Left, tab.X, tab.Y))
			time.Sleep(8 * time.Second)
		}
		h := hand.New(e)
		lay, err := h.Layout(s)
		fmt.Println("layout err", err, "vendor grid", lay.Vendor)
		pt := lay.Vendor.ItemCenter(arg(4), arg(5), 1, arg(6))
		fmt.Println("right click", pt, e.In.Click(input.Right, pt.X, pt.Y))
		time.Sleep(8 * time.Second)
		s, _ = e.Snapshot()
		fmt.Println("gold", g0, "->", s.Player.Gold)
		fmt.Println("closeall:", e.In.CloseAll())
	case "buyone": // buyone CLASS ITEMCLASS QUALITY: open vendor, buy the first stock item of that class/quality, report, hold 15 s
		nv := nav.New(e)
		tv := townvendor.Vendor{T: town.New(e, nv)}
		npc, err := tv.OpenTrade(ctx, arg(2))
		fmt.Println("open:", npc.Class, err)
		st, _ := tv.Stock(npc)
		left := max(arg(5), 1)
		for _, u := range st {
			if lv.Candidate(func() *d2.Player { s, _ := e.Snapshot(); return s.Player }(), u).Name == os.Args[3] && int(u.Item.Quality) == arg(4) {
				s0, _ := e.Snapshot()
				fmt.Println("pack before", len(s0.Inventory.Container(d2.ContInventory)), "gold", s0.Player.Gold)
				paid, err := lv.BuyUnit(ctx, tv, arg(2), npc, lv.Candidate(s0.Player, u))
				fmt.Println("buy:", paid, err)
				if left--; left <= 0 {
					break
				}
			}
		}
		time.Sleep(15 * time.Second)
		fmt.Println("closeall:", e.In.CloseAll())
	case "clickat": // clickat CLASS X Y: open vendor, click one point, report the stock pages visible
		nv := nav.New(e)
		tv := townvendor.Vendor{T: town.New(e, nv)}
		npc, err := tv.OpenTrade(ctx, arg(2))
		fmt.Println("open:", npc.Class, err)
		fmt.Println("click", arg(3), arg(4), e.In.Click(input.Left, arg(3), arg(4)))
		time.Sleep(12 * time.Second)
		fmt.Println("closeall:", e.In.CloseAll())
	case "tabprobe": // tabprobe CLASS X1 Y1 X2 Y2 ...: open vendor, click each point with 5 s between, print markers
		nv := nav.New(e)
		tv := townvendor.Vendor{T: town.New(e, nv)}
		npc, err := tv.OpenTrade(ctx, arg(2))
		fmt.Println("open:", npc.Class, err)
		time.Sleep(4 * time.Second)
		fmt.Println("ready")
		for i := 3; i+1 < len(os.Args); i += 2 {
			x, y := arg(i), arg(i+1)
			fmt.Println("hover click", x, y, lv.HoverClick(x, y))
			time.Sleep(5 * time.Second)
		}
		fmt.Println("done")
		fmt.Println("closeall:", e.In.CloseAll())
	case "exitto":
		nv := nav.New(e)
		fmt.Println("exit:", nv.ExitTo(ctx, arg(2)))
	case "goto":
		nv := nav.New(e)
		fmt.Println("goto:", nv.GoTo(ctx, input.Point{X: arg(2), Y: arg(3)}))
	case "slots":
		s, err := e.Snapshot()
		die(err)
		lay, err := hand.New(e).Layout(s)
		die(err)
		for n, r := range lay.Slots {
			fmt.Println(n, r, r.Center())
		}
		fmt.Println("inventory grid", lay.Inventory)
	case "equip":
		fmt.Printf("%+v\n", lv.Equip(ctx))
	default:
		fmt.Println("unknown subcommand")
	}
}

// classCode is the skills.txt class column value of a character class.
func classCode(c d2.Class) string {
	codes := [...]string{"ama", "sor", "nec", "pal", "bar", "dru", "ass"}
	if c >= 0 && int(c) < len(codes) {
		return codes[c]
	}
	return "sor"
}
