// Command d2dshop is the smoke tool for the town layer on d2d (docs/d2d-town.md). Against a private fork d2d with a
// throwaway character it walks to NPCs, opens their dialogs and shops, buys, sells, stashes gold and claims the free
// Rogue, printing one line per step ("ok", "FAIL" or "BREAK" = an integration break worth a note).
//
//	d2dshop -sock ~/d2d-lab/fork/d2d.sock [-tables DIR] [-steps menus,potions,tome,sell,stash,rogue]
//
// It changes the character (gold via the d2d debug verb, items, gold in the stash): throwaway only, and --no-save.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/d2dtown"
	"d2lod-bot/internal/data"
)

var (
	ctx = context.Background()
	c   *d2dctl.Client
)

func say(tag, f string, a ...any) { fmt.Printf("%-5s %s\n", tag, fmt.Sprintf(f, a...)) }

func main() {
	sock := flag.String("sock", os.Getenv("D2D_SOCK"), "d2d control socket")
	tables := flag.String("tables", "", "extracted game table cache dir (data.Load); optional: runs the adapter's unit view")
	steps := flag.String("steps", "menus,potions,tome,sell,stash,rogue", "comma list of steps")
	flag.Parse()
	if *sock == "" {
		fmt.Fprintln(os.Stderr, "need -sock or D2D_SOCK")
		os.Exit(2)
	}
	c = d2dctl.New(*sock, d2dctl.Options{Timeout: 8 * time.Second})
	defer c.Close()
	if err := c.Ping(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "ping:", err)
		os.Exit(1)
	}
	want := map[string]bool{}
	for _, s := range strings.Split(*steps, ",") {
		want[s] = true
	}
	// A level-20 purse cap lets 20000 gold stay (the cap is clvl*10000, V).
	check(c.SetStat(ctx, 12, 20), "cheat: set level 20 (reason: purse cap clvl*10000 must not clip the shopping gold)")
	check(c.SetStat(ctx, 14, 20000), "cheat: set gold 20000 (reason: throwaway character, shopping test)")
	_ = c.Kick(ctx)
	if *tables != "" {
		adapterView(*tables)
	}
	if want["rogue"] {
		r, err := d2dtown.ClaimRogue(ctx, c)
		say(tag(err), "claim rogue: %+v err=%v", r, err)
	}
	if want["menus"] {
		for _, n := range []string{"Charsi", "Gheed", "Warriv", "Kashya", "Akara"} {
			menu(n)
		}
	}
	if want["potions"] {
		buy("Akara", "hp1", 3, 1) // 3 minor healing potions, misc tab
	}
	if want["tome"] {
		buy("Akara", "tbk", 3, 1)
	}
	if want["sell"] {
		sellFirstJunk("Akara")
	}
	if want["repair"] {
		repair()
	}
	if want["portal"] {
		portal()
	}
	if want["wp"] {
		waypoint()
	}
	if want["stash"] {
		stash()
	}
}

func tag(err error) string {
	if err != nil {
		return "FAIL"
	}
	return "ok"
}

func check(err error, what string) {
	say(tag(err), "%s %v", what, errStr(err))
}

func errStr(err error) string {
	if err != nil {
		return "err=" + err.Error()
	}
	return ""
}

func npcIndex(name string) (int, bool) {
	l, err := c.NPCs(ctx)
	if err != nil {
		return 0, false
	}
	for _, n := range l.Items {
		if strings.EqualFold(n.Name, name) {
			return n.Index, true
		}
	}
	return 0, false
}

func menu(name string) {
	idx, ok := npcIndex(name)
	if !ok {
		say("BREAK", "%s is not in npcs (Cain/Charsi appear only near / after quests, V-d2d probes section 5)", name)
		return
	}
	lines, err := c.OpenNPCMenu(ctx, idx, 25*time.Second)
	if err != nil {
		say("FAIL", "%s: %v", name, err)
		return
	}
	var txt []string
	for _, l := range lines {
		txt = append(txt, l.Text)
	}
	say("ok", "%s menu: %s", name, strings.Join(txt, " | "))
	if _, ok := d2dctl.FindLine(lines, "cancel"); ok {
		check(c.PickMenuLine(ctx, "cancel"), name+": cancel")
	}
}

// openShop opens name's trade window and returns its stock.
func openShop(name string) (d2dctl.Vendor, bool) {
	idx, ok := npcIndex(name)
	if !ok {
		say("BREAK", "%s not in npcs", name)
		return d2dctl.Vendor{}, false
	}
	if _, err := c.OpenNPCMenu(ctx, idx, 25*time.Second); err != nil {
		say("FAIL", "%s: %v", name, err)
		return d2dctl.Vendor{}, false
	}
	if err := c.PickMenuLine(ctx, "trade"); err != nil {
		say("FAIL", "%s trade line: %v", name, err)
		return d2dctl.Vendor{}, false
	}
	_ = d2dctl.Pause(ctx, time.Second)
	v, err := c.Vendor(ctx)
	if err != nil {
		say("BREAK", "%s vendor read: %v", name, err)
		return v, false
	}
	say("ok", "%s shop open=%v items=%d tab=%d", name, v.Open, len(v.Items), v.Tab)
	return v, v.Open
}

func closeShop() {
	// V: Escape closes the vendor window (first press); a second press would open the game menu.
	check(c.Key(ctx, "Escape"), "close shop (Escape)")
	_ = d2dctl.Pause(ctx, 600*time.Millisecond)
	if ui, err := c.UI(ctx); err == nil && (ui.Vendor || ui.GameMenu) {
		say("BREAK", "after Escape: %+v", ui)
	}
}

func buy(npc, code string, n, _ int) {
	v, ok := openShop(npc)
	if !ok {
		return
	}
	for i := 0; i < n; i++ {
		var it *d2dctl.VendorItem
		for k := range v.Items {
			if v.Items[k].Code == code {
				it = &v.Items[k]
				break
			}
		}
		if it == nil {
			say("BREAK", "%s has no %s in stock", npc, code)
			break
		}
		paid, err := c.BuyItem(ctx, *it)
		say(tag(err), "buy %s from %s: paid %d (listed %d) %s", code, npc, paid, it.Price, errStr(err))
		if v, err = c.Vendor(ctx); err != nil {
			break
		}
	}
	closeShop()
}

func sellFirstJunk(npc string) {
	if _, ok := openShop(npc); !ok {
		return
	}
	// the first buy gave a wand or tome in the pack; sell what the cursor-drop path can: the starting scroll of identify
	items, _, err := c.Items(ctx)
	if err != nil {
		say("FAIL", "items: %v", err)
		return
	}
	for _, it := range items {
		if it.Code == "isc" || it.Code == "wnd" {
			id, _ := atoi(it.Attrs["id"])
			g, err := c.SellItem(ctx, id)
			say(tag(err), "sell %s id %d: +%d gold %s", it.Code, id, g, errStr(err))
			break
		}
	}
	closeShop()
}

func atoi(s string) (int, error) { var n int; _, err := fmt.Sscanf(s, "%d", &n); return n, err }

// stash opens the stash chest, deposits and withdraws gold through the amount prompt; dump player confirms.
func stash() {
	check(c.OpenStash(ctx, 25*time.Second), "open stash")
	p0, s0, _ := c.Gold(ctx)
	check(c.MoveGold(ctx, 5000, true), "deposit 5000")
	p1, s1, _ := c.Gold(ctx)
	say("info", "purse %d->%d stash %d->%d", p0, p1, s0, s1)
	check(c.MoveGold(ctx, 2000, false), "withdraw 2000")
	p2, s2, _ := c.Gold(ctx)
	say("info", "purse %d->%d stash %d->%d", p1, p2, s1, s2)
	_ = c.Key(ctx, "Escape")
	_ = d2dctl.Pause(ctx, 600*time.Millisecond)
}

// repair wears the gear down (debug wear), opens Charsi's window and repairs.
func repair() {
	_, werr := c.Debug(ctx, "wear")
	check(werr, "cheat: debug wear (reason: repair test on a throwaway character)")
	c0, m0, _ := c.Durability(ctx)
	if _, ok := openShop("Charsi"); !ok {
		return
	}
	check(c.RepairAll(ctx), "repair all")
	c1, _, _ := c.Durability(ctx)
	say("info", "durability %d/%d -> %d", c0, m0, c1)
	closeShop()
}

// portal casts a town portal from a field level, walks in and back out, level ids confirm.
func portal() {
	items, _, err := c.Items(ctx)
	if err != nil {
		say("FAIL", "items: %v", err)
		return
	}
	id := -1
	for _, it := range items {
		if it.Code == "tsc" || it.Code == "tbk" {
			id, _ = atoi(it.Attrs["id"])
			break
		}
	}
	if id < 0 {
		say("BREAK", "no tp scroll or tome")
		return
	}
	check(c.Goto(ctx, 2), "cheat: debug goto Blood Moor (reason: portal test)")
	_ = d2dctl.Pause(ctx, 3*time.Second)
	check(c.CastTP(ctx, id), "cast town portal")
	check(c.EnterPortal(ctx), "enter portal to town")
	l, _ := c.Level(ctx)
	say("info", "now in level %d", l.ID)
	_ = d2dctl.Pause(ctx, 2*time.Second)
	check(c.EnterPortal(ctx), "enter portal back")
	l, _ = c.Level(ctx)
	say("info", "now in level %d", l.ID)
}

func waypoint() {
	check(c.OpenWaypoint(ctx, 25*time.Second), "open waypoint")
	ui, _ := c.UI(ctx)
	say("info", "waypoint panel %v", ui.Waypoint)
	_ = c.Key(ctx, "Escape")
}

// adapterView prints what the town layer would see through the Adapter (unit table with vendor stock).
func adapterView(dir string) {
	db, err := data.Load(dir)
	if err != nil {
		say("FAIL", "tables: %v", err)
		return
	}
	a := d2dtown.New(c, d2dsrc.New(c, d2dsrc.Options{Tables: d2dsrc.NewDBTables(db), Logf: func(string, ...any) {}}), d2dsrc.NewDBTables(db))
	src := a.Source()
	items, err := src.Items()
	say(tag(err), "adapter items: %d units", len(items))
	snap, err := src.Snapshot()
	if err == nil {
		say("ok", "adapter snapshot: state=%v monsters=%d ui=%v", snap.State, len(snap.Monsters), snap.UI)
	}
}
