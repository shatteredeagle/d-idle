//go:build rigs

package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/cheat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/mem"
)

// The Amazon smoke test (docs/classes/smoke-amazon.md, agent CLASSAMA). `classsmoke ama <sub>`:
//
//	items               every item the character owns with its base name, location and its whole stat list (quantity is stat 70)
//	cheat KIND k=v ... --reason TEXT    cheat.Do as actor classama (Amma only)
//
// It runs from init so main.go stays untouched.
func init() {
	cheat.Allowed = append(cheat.Allowed, "Amma") // the one throwaway Amazon of this smoke test, this tool only
	if len(os.Args) < 3 || os.Args[1] != "ama" {
		return
	}
	switch os.Args[2] {
	case "cheat":
		var reason string
		var rest []string
		for i := 3; i < len(os.Args); i++ {
			if os.Args[i] == "--reason" && i+1 < len(os.Args) {
				reason = os.Args[i+1]
				i++
				continue
			}
			rest = append(rest, os.Args[i])
		}
		req := cheat.Request{Kind: cheat.Kind(rest[0]), Reason: reason, Actor: "classama", Params: map[string]any{}}
		for _, kv := range rest[1:] {
			k, v, _ := strings.Cut(kv, "=")
			if n, err := strconv.ParseInt(v, 0, 64); err == nil {
				req.Params[k] = n
			} else {
				req.Params[k] = v
			}
		}
		res, err := cheat.DefaultEnv().Do(context.Background(), req)
		die(err)
		fmt.Println(res.Message)
	case "tree":
		home, _ := os.UserHomeDir()
		dir := home + "/Games/d2lod-inst/" + os.Getenv("D2BOT_INSTANCE") + "/proton/pfx/drive_c/Program Files (x86)/Diablo II"
		db, err := data.Open(dir, home+"/d2lod-live/classama/data")
		die(err)
		var rows []*data.Skill
		for _, sk := range db.Skills {
			if sk.Class == "ama" {
				rows = append(rows, sk)
			}
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
		for _, sk := range rows {
			fmt.Printf("%3d %-18s page %d row %d col %d reqlvl %d aura=%v passive=%v left=%v range=%s req %v mana %d+%d/lvl delay %d etype %s\n", sk.ID, sk.Key, sk.Page, sk.Row, sk.Column, sk.ReqLevel, sk.Aura, sk.Passive, sk.LeftSkill, sk.Range, sk.ReqSkills, sk.Mana, sk.LvlMana, sk.Delay, sk.EType)
		}
	case "units": // units [R]: monsters within R subtiles (default 40): id, class, mode, owner, life, position
		src, err := mem.OpenGame()
		die(err)
		defer src.Close()
		sn, err := d2.New(src).Snapshot()
		die(err)
		r := 40
		if len(os.Args) > 3 {
			r, _ = strconv.Atoi(os.Args[3])
		}
		p := sn.Player
		fmt.Printf("player id %d at (%d,%d) area %d\n", p.ID, p.X, p.Y, p.Area)
		for _, u := range sn.Monsters {
			dx, dy := u.X-p.X, u.Y-p.Y
			if dx*dx+dy*dy > r*r {
				continue
			}
			life := -1
			if u.Monster != nil {
				life = u.Monster.LifePct
			}
			fmt.Printf("  mon id %d class %d mode %d owner %d/%d life%% %d at (%d,%d) d=%d\n", u.ID, u.Class, u.Mode, u.OwnerType, u.OwnerID, life, u.X, u.Y, int(math.Sqrt(float64(dx*dx+dy*dy))))
		}
		for _, u := range sn.Ground {
			fmt.Printf("  ground class %d at (%d,%d)\n", u.Class, u.X, u.Y)
		}
	case "bases": // bases: bows, crossbows, javelins, quivers and their itemtypes rows (body locations, shoots, quiver)
		home, _ := os.UserHomeDir()
		dir := home + "/Games/d2lod-inst/" + os.Getenv("D2BOT_INSTANCE") + "/proton/pfx/drive_c/Program Files (x86)/Diablo II"
		db, err := data.Open(dir, home+"/d2lod-live/classama/data")
		die(err)
		for _, t := range []string{"bow", "xbow", "bowq", "xboq", "jave", "ajav", "miss", "thro", "amaz", "tkni", "taxe"} {
			if ty := db.ItemType(t); ty != nil {
				fmt.Printf("type %-5s body %v shoots=%q quiver=%q class=%q equiv=%v\n", t, ty.BodyLocs, ty.Shoots, ty.Quiver, ty.Class, ty.Equiv)
			}
		}
		for _, b := range db.Items {
			if db.IsA(b.Type, "miss") || db.IsA(b.Type, "thro") || db.IsA(b.Type, "bowq") || db.IsA(b.Type, "xboq") {
				if b.Level > 25 && b.Name != "" && !(b.Rarity > 0 && b.Level <= 12) {
					continue
				}
				fmt.Printf("base %-22s code=%s type=%s kind=%d lvl=%d reqstr=%d reqdex=%d cost=%d stack=%v max=%d min=%d dura=%d 2h=%v thr=%v dam=%d-%d speed=%d wclass=%s\n", b.Name, b.Code, b.Type, b.Kind, b.Level, b.ReqStr, b.ReqDex, b.Cost, b.Stackable, b.MaxStack, b.MinStack, b.Durability, b.TwoHanded, b.Throwable, b.MinDam, b.MaxDam, b.Speed, b.WClass)
			}
		}
	case "buy":
		amaBuy(os.Args[3:])
	case "seq": // seq l:X:Y r:X:Y k:KEY w:MS ...: mouse moves, clicks, keys and waits in one run
		src, err := mem.OpenGame()
		die(err)
		defer src.Close()
		g := input.NewGame(input.NewClient(""), input.Reader{Src: src})
		for _, st := range os.Args[3:] {
			f := strings.Split(st, ":")
			switch f[0] {
			case "l", "r":
				x, _ := strconv.Atoi(f[1])
				y, _ := strconv.Atoi(f[2])
				b := input.Left
				if f[0] == "r" {
					b = input.Right
				}
				_ = g.MoveMouse(x, y)
				time.Sleep(350 * time.Millisecond)
				die(g.Click(b, x, y))
				time.Sleep(500 * time.Millisecond)
			case "m": // mouse move only
				x, _ := strconv.Atoi(f[1])
				y, _ := strconv.Atoi(f[2])
				die(g.MoveMouse(x, y))
				time.Sleep(600 * time.Millisecond)
			case "k":
				die(g.Key(f[1]))
				time.Sleep(500 * time.Millisecond)
			case "w":
				ms, _ := strconv.Atoi(f[1])
				time.Sleep(time.Duration(ms) * time.Millisecond)
			}
		}
	case "mon": // mon ID...: monstats rows (NPC, Killable) of monster classes
		home, _ := os.UserHomeDir()
		dir := home + "/Games/d2lod-inst/" + os.Getenv("D2BOT_INSTANCE") + "/proton/pfx/drive_c/Program Files (x86)/Diablo II"
		db, err := data.Open(dir, home+"/d2lod-live/classama/data")
		die(err)
		for _, a := range os.Args[3:] {
			id, _ := strconv.Atoi(a)
			if m := db.MonsterByClassID(id); m != nil {
				fmt.Printf("class %d key=%s name=%q npc=%v killable=%v\n", id, m.Key, db.Text(m.NameKey), m.NPC, m.Killable)
			}
		}
	case "items":
		src, err := mem.OpenGame()
		die(err)
		defer src.Close()
		rd := d2.New(src)
		home, _ := os.UserHomeDir()
		dir := home + "/Games/d2lod-inst/" + os.Getenv("D2BOT_INSTANCE") + "/proton/pfx/drive_c/Program Files (x86)/Diablo II"
		db, derr := data.Open(dir, home+"/d2lod-live/classama/data")
		if derr != nil {
			fmt.Println("data:", derr)
		}
		inv, err := rd.Inventory()
		die(err)
		for _, u := range inv.Items {
			if u.Item == nil {
				continue
			}
			name := ""
			if db != nil {
				for _, b := range db.Items {
					if b.Index == u.Class && itemKindMatches(b, u) {
						name += "[" + b.Name + "/" + b.Code + " kind=" + strconv.Itoa(int(b.Kind)) + " type=" + b.Type + " stack=" + strconv.FormatBool(b.Stackable) + " max=" + strconv.Itoa(b.MaxStack) + "]"
					}
				}
			}
			st, _ := rd.UnitStats(u.Addr)
			var keys []string
			for k, v := range st {
				keys = append(keys, fmt.Sprintf("%d=%d", k, v))
			}
			sort.Strings(keys)
			fmt.Printf("item class %d %s where %v body %d gx %d gy %d stats [%s]\n", u.Class, name, u.Item.Where, u.Item.Body, u.Item.GX, u.Item.GY, strings.Join(keys, " "))
		}
	}
	os.Exit(0)
}

func itemKindMatches(b *data.ItemBase, u d2.Unit) bool { return true }
