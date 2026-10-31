//go:build rigs

// classsmoke: small live probe for the class smoke test (docs/classes/smoke-paladin.md, agent CLASSPAL).
// Run via tools/live.sh classpal classsmoke <sub> [args]. Subcommands:
//
//	state                       class, level, points, skills with names/pages, left/right skill, equipment, UI flags
//	stl [N]                     raw dump of the player's stat list (+0 .. +N, default 0x120) as non-zero words, for aura diffs
//	keys                        press F1..F8 and print left/right skill after each
//	cheat KIND k=v ... --reason TEXT   cheat.Do with Palli added to the allowlist (this tool only)
//	tree PAGE                   skills.txt rows of the character's class (page/row/col)
//	click l|r X Y | key K | hold K MS | rclick-hold X Y MS   raw input
package main

import (
	"context"
	"fmt"
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

func die(err error) {
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: classsmoke state|stl|keys|cheat|tree|click|key|hold")
		return
	}
	cheat.Allowed = append(cheat.Allowed, "Palli") // the one throwaway Paladin of this smoke test, this tool only
	if os.Args[1] == "cheat" {
		doCheat(os.Args[2:])
		return
	}
	src, err := mem.OpenGame()
	die(err)
	defer src.Close()
	rd := d2.New(src)
	g := input.NewGame(input.NewClient(""), input.Reader{Src: src})
	home, _ := os.UserHomeDir()
	dir := home + "/Games/d2lod-inst/" + os.Getenv("D2BOT_INSTANCE") + "/proton/pfx/drive_c/Program Files (x86)/Diablo II"
	db, derr := data.Open(dir, home+"/d2lod-live/classpal/data")
	if derr != nil {
		fmt.Println("data:", derr)
	}
	a := os.Args
	num := func(i, def int) int {
		if len(a) > i {
			if n, err := strconv.ParseInt(a[i], 0, 64); err == nil {
				return int(n)
			}
		}
		return def
	}
	switch a[1] {
	case "state":
		s, err := rd.Snapshot()
		die(err)
		p := s.Player
		if p == nil {
			fmt.Println("not in game", s.State)
			return
		}
		fmt.Printf("%s class %v lvl %d area %d pos (%d,%d) mode %d life %d/%d mana %d/%d stam %d/%d exp %d gold %d\n", p.Name, p.Class, p.Level, p.Area, p.X, p.Y, p.Mode, p.Life, p.MaxLife, p.Mana, p.MaxMana, p.Stamina, p.MaxStamina, p.Exp, p.Gold)
		fmt.Printf("str %d dex %d vit %d ene %d points stat=%d skill=%d left %d right %d\n", p.Strength, p.Dexterity, p.Vitality, p.Energy, p.StatPoints, p.SkillPoints, p.LeftSkill, p.RightSkill)
		for _, sk := range p.Skills {
			if db != nil {
				if d := db.SkillByID(sk.ID); d != nil {
					fmt.Printf("  skill %3d %-18s lvl %d+%d class %q page %d row %d col %d aura=%v leftok=%v range=%s mana=%d\n", sk.ID, d.Key, sk.Level, sk.Bonus, d.Class, d.Page, d.Row, d.Column, d.Aura, d.LeftSkill, d.Range, int(d.ManaCost(max(1, sk.Effective()))))
					continue
				}
			}
			fmt.Printf("  skill %d lvl %d\n", sk.ID, sk.Level)
		}
		if s.Inventory != nil {
			for _, u := range s.Inventory.Items {
				if u.Item == nil {
					continue
				}
				name := ""
				if db != nil {
					for _, k := range []data.ItemKind{} {
						_ = k
					}
				}
				fmt.Printf("  item class %d where %v body %d page %d gx %d gy %d q %d %s\n", u.Class, u.Item.Where, u.Item.Body, u.Item.Page, u.Item.GX, u.Item.GY, u.Item.Quality, name)
			}
		}
		fmt.Println("ui:", s.UI)
	case "stl":
		n := num(2, 0x120)
		pl, err := rd.Snapshot()
		die(err)
		var ptr [4]byte
		die(src.ReadAt(ptr[:], pl.Player.Addr+0x5C))
		sl := uint32(ptr[0]) | uint32(ptr[1])<<8 | uint32(ptr[2])<<16 | uint32(ptr[3])<<24
		buf := make([]byte, n)
		die(src.ReadAt(buf, sl))
		fmt.Printf("statlist %#x\n", sl)
		var out []string
		for i := 0; i+4 <= n; i += 4 {
			v := uint32(buf[i]) | uint32(buf[i+1])<<8 | uint32(buf[i+2])<<16 | uint32(buf[i+3])<<24
			if v != 0 {
				out = append(out, fmt.Sprintf("+%#x=%#x", i, v))
			}
		}
		fmt.Println(strings.Join(out, " "))
		// the owner's linked lists: dump other list heads reachable through the pointers at +0x44 and +0x50..0x5C
	case "chain": // chain: follow the stat list pointer at +0x3c (and +0x44) of the player list and dump what hangs there
		pl, err := rd.Snapshot()
		die(err)
		rd32 := func(at uint32) uint32 { v, _ := mem.U32(src, at); return v }
		top := rd32(pl.Player.Addr + 0x5C)
		for _, off := range []uint32{0x3c, 0x44, 0x58} {
			child := rd32(top + off)
			fmt.Printf("link +%#x -> %#x\n", off, child)
			if child < 0x10000 {
				continue
			}
			var words []string
			for i := uint32(0); i < 0x60; i += 4 {
				if v := rd32(child + i); v != 0 {
					words = append(words, fmt.Sprintf("+%#x=%#x", i, v))
				}
			}
			fmt.Println("  ", strings.Join(words, " "))
			vec, cnt := rd32(child+0x24), rd32(child+0x28)&0xffff
			if vec > 0x10000 && cnt < 64 {
				for i := uint32(0); i < cnt; i++ {
					w0, w1 := rd32(vec+i*8), rd32(vec+i*8+4)
					fmt.Printf("     stat sub %d id %d value %d\n", w0&0xffff, w0>>16, int32(w1))
				}
			}
		}
	case "melee":
		melee(src, g, db, a[2:])
	case "aurakey": // aurakey KEY: press KEY, poll the aura child list (stat 350 = aura skill id) every 20 ms
		auraOf := func() (int32, int32) {
			pl, _ := rd.Snapshot()
			rd32 := func(at uint32) uint32 { v, _ := mem.U32(src, at); return v }
			top := rd32(pl.Player.Addr + 0x5C)
			for l, n := rd32(top+0x3c), 0; l > 0x10000 && n < 8; l, n = rd32(l+0x2c), n+1 {
				if rd32(l+0x34) != top {
					continue
				}
				vec, cnt := rd32(l+0x24), rd32(l+0x28)&0xffff
				for i := uint32(0); vec > 0x10000 && i < cnt && i < 32; i++ {
					if w0 := rd32(vec + i*8); w0>>16 == 350 {
						return int32(rd32(vec + i*8 + 4)), int32(rd32(l + 0x14))
					}
				}
			}
			return 0, 0
		}
		id0, st0 := auraOf()
		pl0, _ := rd.Snapshot()
		fmt.Printf("before: aura skill %d state %d right %d mana %d\n", id0, st0, pl0.Player.RightSkill, pl0.Player.Mana)
		t0 := time.Now()
		die(g.Key(a[2]))
		tKey := time.Since(t0)
		for time.Since(t0) < 2*time.Second {
			if id, st := auraOf(); id != id0 {
				pl1, _ := rd.Snapshot()
				fmt.Printf("after: aura skill %d state %d right %d mana %d; key call took %v, aura visible after %v\n", id, st, pl1.Player.RightSkill, pl1.Player.Mana, tKey.Round(time.Millisecond), time.Since(t0).Round(time.Millisecond))
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		fmt.Println("aura did not change within 2 s")
	case "keys":
		for i := 1; i <= 8; i++ {
			k := fmt.Sprintf("f%d", i)
			die(g.Key(k))
			time.Sleep(250 * time.Millisecond)
			s, _ := rd.Snapshot()
			fmt.Printf("%s -> left %d right %d\n", k, s.Player.LeftSkill, s.Player.RightSkill)
		}
	case "tree":
		var rows []*data.Skill
		for _, sk := range db.Skills {
			if sk.Class == "pal" {
				rows = append(rows, sk)
			}
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
		for _, sk := range rows {
			fmt.Printf("%3d %-18s page %d row %d col %d reqlvl %d aura=%v passive=%v leftskill=%v range=%s req %v mana %d+%d/lvl delay %d etype %s\n", sk.ID, sk.Key, sk.Page, sk.Row, sk.Column, sk.ReqLevel, sk.Aura, sk.Passive, sk.LeftSkill, sk.Range, sk.ReqSkills, sk.Mana, sk.LvlMana, sk.Delay, sk.EType)
		}
	case "click":
		b := input.Left
		if a[2] == "r" {
			b = input.Right
		}
		die(g.Click(b, num(3, 0), num(4, 0)))
	case "key":
		die(g.Key(a[2]))
	case "hold": // hold K MS: key down for MS
		die(g.Hold(a[2]))
		time.Sleep(time.Duration(num(3, 500)) * time.Millisecond)
		die(g.Release(a[2]))
	default:
		fmt.Println("unknown", a[1])
	}
}

func doCheat(args []string) {
	var reason string
	var rest []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--reason" && i+1 < len(args) {
			reason = args[i+1]
			i++
			continue
		}
		rest = append(rest, args[i])
	}
	req := cheat.Request{Kind: cheat.Kind(rest[0]), Reason: reason, Actor: "classpal", Params: map[string]any{}}
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
}
