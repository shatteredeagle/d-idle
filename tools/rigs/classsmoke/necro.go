//go:build rigs

package main

// Necromancer probes (agent CLASSNEC, docs/classes/smoke-necromancer.md). Hooked in through init so main.go stays untouched.
//
//	ntree                    skills.txt rows of the necromancer (nec): id, page/row/col, req level, prereqs, mana, range, left-ok
//	ncheat KIND k=v --reason TEXT   cheat.Do with Necci on the allow list (ledger actor classnec)
//	nunits [R]               every monster-table unit within R subtiles (default 40): class, name, mode, owner, flags
//	nstates                  the player's StatList child lists: state number, owner link, all stat entries
//	nwatch SECS [R]          nunits-style census every second for SECS, only prints changes (count of owned/corpse/hostile)

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
	"d2lod-bot/internal/mem"
)

func init() {
	if len(os.Args) < 2 || !strings.HasPrefix(os.Args[1], "n") {
		return
	}
	switch os.Args[1] {
	case "ntree", "ncheat", "nunits", "nstates", "nwatch":
	default:
		return
	}
	cheat.Allowed = append(cheat.Allowed, "Necci") // the one throwaway Necromancer of this smoke test, this tool only
	necroMain(os.Args[1:])
	os.Exit(0)
}

func necroDB() *data.DB {
	home, _ := os.UserHomeDir()
	dir := home + "/Games/d2lod-inst/" + os.Getenv("D2BOT_INSTANCE") + "/proton/pfx/drive_c/Program Files (x86)/Diablo II"
	db, err := data.Open(dir, home+"/d2lod-live/classnec/data")
	if err != nil {
		fmt.Println("data:", err)
	}
	return db
}

func necroMain(a []string) {
	if a[0] == "ncheat" {
		var reason string
		var rest []string
		for i := 1; i < len(a); i++ {
			if a[i] == "--reason" && i+1 < len(a) {
				reason = a[i+1]
				i++
				continue
			}
			rest = append(rest, a[i])
		}
		req := cheat.Request{Kind: cheat.Kind(rest[0]), Reason: reason, Actor: "classnec", Params: map[string]any{}}
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
		return
	}
	db := necroDB()
	if a[0] == "ntree" {
		var rows []*data.Skill
		for _, sk := range db.Skills {
			if sk.Class == "nec" {
				rows = append(rows, sk)
			}
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
		for _, sk := range rows {
			fmt.Printf("%3d %-20s page %d row %d col %d reqlvl %d left=%v range=%s summon=%q missile=%q req %v mana %d+%d/lvl delay %d\n", sk.ID, sk.Key, sk.Page, sk.Row, sk.Column, sk.ReqLevel, sk.LeftSkill, sk.Range, sk.Summon, sk.Missile, sk.ReqSkills, sk.Mana, sk.LvlMana, sk.Delay)
		}
		return
	}
	src, err := mem.OpenGame()
	die(err)
	defer src.Close()
	rd := d2.New(src)
	radius := 40.0
	arg := func(i int) { // optional numeric argument
		if len(a) > i {
			if f, err := strconv.ParseFloat(a[i], 64); err == nil {
				radius = f
			}
		}
	}
	census := func(verbose bool) (owned, corpses, alive, other int, line string) {
		s, err := rd.Snapshot()
		if err != nil || s.Player == nil {
			return 0, 0, 0, 0, "no snapshot"
		}
		p := s.Player
		var sb strings.Builder
		for _, u := range s.Monsters {
			d := math.Hypot(float64(u.X-p.X), float64(u.Y-p.Y))
			if d > radius {
				continue
			}
			name := "?"
			if m := db.MonsterByClassID(u.Class); m != nil {
				name = m.Key
			}
			own := u.OwnerID == p.ID && u.OwnerType == 0
			kind := "hostile"
			switch {
			case own:
				kind = "OWNED"
				owned++
			case u.OwnerID != 0:
				kind = fmt.Sprintf("owner=%d/%d", u.OwnerType, u.OwnerID)
				other++
			case !u.Alive():
				kind = "corpse"
				corpses++
			default:
				alive++
			}
			if verbose {
				fmt.Fprintf(&sb, "  id %-5d class %-4d %-18s mode %-2d pos (%d,%d) d=%.0f flags %#x owner %d/%d  %s\n", u.ID, u.Class, name, u.Mode, u.X, u.Y, d, u.Flags, u.OwnerType, u.OwnerID, kind)
			}
		}
		return owned, corpses, alive, other, sb.String()
	}
	switch a[0] {
	case "nunits":
		arg(1)
		s, err := rd.Snapshot()
		die(err)
		fmt.Printf("player id %d at (%d,%d) mode %d area %d life %d/%d mana %d/%d\n", s.Player.ID, s.Player.X, s.Player.Y, s.Player.Mode, s.Player.Area, s.Player.Life, s.Player.MaxLife, s.Player.Mana, s.Player.MaxMana)
		o, c, h, ot, l := census(true)
		fmt.Print(l)
		fmt.Printf("owned %d corpses %d hostile-alive %d other-owner %d\n", o, c, h, ot)
		pl, _ := rd.Units(d2.TypePlayer)
		for _, u := range pl {
			fmt.Printf("  player-table unit id %d class %d mode %d (%d,%d) flags %#x\n", u.ID, u.Class, u.Mode, u.X, u.Y, u.Flags)
		}
		for _, m := range s.Missiles {
			fmt.Printf("  missile class %d owner %d/%d (%d,%d)\n", m.Class, m.OwnerType, m.OwnerID, m.X, m.Y)
		}
	case "nstates":
		s, err := rd.Snapshot()
		die(err)
		p := s.Player
		top := rdU32(src, p.Addr+0x5C)
		fmt.Printf("player list %#x\n", top)
		child := rdU32(src, top+0x3c)
		for n := 0; child > 0x10000 && n < 40; n++ {
			own := rdU32(src, child+0x34)
			st := rdU32(src, child+0x14)
			vec, cnt := rdU32(src, child+0x24), rdU32(src, child+0x28)&0xffff
			fmt.Printf("child %#x owner %#x (player=%v) state %d stats:", child, own, own == top, st)
			for i := uint32(0); i < cnt && i < 24; i++ {
				w0, w1 := rdU32(src, vec+i*8), rdU32(src, vec+i*8+4)
				fmt.Printf(" [%d.%d=%d]", w0>>16, w0&0xffff, int32(w1))
			}
			fmt.Println()
			child = rdU32(src, child+0x2c)
		}
	case "nwatch":
		secs := 30
		if len(a) > 1 {
			secs, _ = strconv.Atoi(a[1])
		}
		arg(2)
		var last string
		t0 := time.Now()
		for time.Since(t0) < time.Duration(secs)*time.Second {
			o, c, h, ot, _ := census(false)
			cur := fmt.Sprintf("owned %d corpses %d hostile %d other %d", o, c, h, ot)
			if cur != last {
				fmt.Printf("%5.1fs %s\n", time.Since(t0).Seconds(), cur)
				last = cur
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func rdU32(s mem.Source, a uint32) uint32 {
	v, err := mem.U32(s, a)
	if err != nil {
		return 0
	}
	return v
}
