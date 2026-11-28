package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// smokeGame is a d2d control socket with just enough of a hero for the prep half of a run (`d2scenario smoke`): the
// level, the unspent points, the hard skill points, the two hands, the purse and a pack. It is a MODEL, not the game:
//   - reply shapes follow the client's parsers and internal/d2dsrc/testdata/dump_head_necro.json (S);
//   - `cmd stat` and `cmd skillpt` spend a banked point and fail without one (S docs/forkcheats.md), `cmd select` puts a
//     skill on a hand without checking that it is learned (U);
//   - `debug points` is accepted and changes nothing (the fork sets the hard points, V lab run 20261128-000130
//     skill_levels; the model cannot map an id to a class index without the game tables);
//   - `cmd skill` is a cast: the model counts every cast as one standing summon and `debug minions` lists them (U: no
//     corpse, mana or one-golem rule), `debug players` puts the hero at a fixed cell;
//   - the pack is packCols x packRows cells, an item needs packCells(code) of them (S), `debug give` answers
//     "err no room" when they are not free.
//
// An unknown verb is an error, so a prep that starts using a verb the model does not know fails the smoke instead of
// passing unseen. What the model cannot catch is the fork changing a reply: that needs a real run.
type smokeGame struct {
	// NoHands leaves lskill and rskill out of the dump: the 2026-11-27 defect, kept as the must-fail arm.
	NoHands bool
	// DeafSelect answers `cmd select` ok and changes no hand; NoSummon answers `cmd skill` ok and summons nothing: the
	// must-fail arms of pre.bind and pre.summon.
	DeafSelect, NoSummon bool

	mu       sync.Mutex
	class    string
	level    int
	exp      int
	gold     int
	statPts  int
	skillPts int
	levels   [30]int
	hands    [2]int // left, right
	minions  int
	pack     []string
	belt     []string
	used     int
	ln       net.Listener
	dir      string
	lines    []string
}

const packCols, packRows = 10, 4 // S: the hero's pack

// packCells is the pack area of an item code; a code the table does not know counts as one cell, so a sum is a lower
// bound (S: cap is 2x2, potions and scrolls 1x1).
func packCells(code string) int {
	if n, ok := map[string]int{"cap": 4, "skp": 4, "buc": 4, "lea": 6}[code]; ok {
		return n
	}
	return 1
}

// startSmokeGame listens on a private unix socket (short path: unix paths are limited to about 100 bytes).
func startSmokeGame(class string) (*smokeGame, error) {
	dir, err := os.MkdirTemp("", "pgsmoke")
	if err != nil {
		return nil, err
	}
	ln, err := net.Listen("unix", filepath.Join(dir, "d.sock"))
	if err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	g := &smokeGame{class: class, level: 1, ln: ln, dir: dir}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go g.serve(conn)
		}
	}()
	return g, nil
}

func (g *smokeGame) Path() string { return filepath.Join(g.dir, "d.sock") }

func (g *smokeGame) Close() {
	g.ln.Close()
	os.RemoveAll(g.dir)
}

func (g *smokeGame) serve(conn net.Conn) {
	defer conn.Close()
	sc := bufio.NewScanner(conn)
	for sc.Scan() {
		if _, err := conn.Write([]byte(g.handle(sc.Text()))); err != nil {
			return
		}
	}
}

func (g *smokeGame) handle(line string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.lines = append(g.lines, line)
	f := strings.Fields(line)
	num := func(i int) int {
		if i >= len(f) {
			return 0
		}
		n, _ := strconv.Atoi(f[i])
		return n
	}
	switch {
	case len(f) == 0:
		return "err empty command\n"
	case f[0] == "ping", f[0] == "save", f[0] == "click":
		return "ok\n"
	case f[0] == "caps":
		return "caps=lockstep,lab-verbs\nok\n"
	case f[0] == "step":
		return "frame=1 game_ms=40 tick=1\nok\n"
	case f[0] == "state":
		return fmt.Sprintf("screen=ingame class=%s name=Smoke difficulty=0 cam=30.70,13.70 walking=0 gold=%d str=15 life=45/45 mana=25/25 stamina=100/100 level=%d exp=%d pmode=NU gamemenu=0:0 merc=-\nok\n",
			g.class, g.gold, g.level, g.exp)
	case f[0] == "items":
		var b strings.Builder
		for i, code := range g.pack {
			fmt.Fprintf(&b, "[%s loc=0 q=2 panel=0 id=%d size=1x1 at=%d,0]\n  item\n", code, 100+i, i)
		}
		for i, code := range g.belt { // U: the loc value of a belt item; the prep only asks "not worn (1), not in the stash"
			fmt.Fprintf(&b, "[%s loc=2 q=2 panel=0 id=%d size=1x1 at=%d,0]\n  item\n", code, 200+i, i)
		}
		return b.String() + "ok\n"
	case f[0] == "dump":
		player := map[string]any{"class": strings.ToLower(g.class), "level": g.level, "exp": g.exp, "gold": g.gold, "stat_pts": g.statPts, "skill_pts": g.skillPts}
		if !g.NoHands {
			player["lskill"], player["rskill"] = g.hands[0], g.hands[1]
		}
		b, _ := json.Marshal(map[string]any{"v": 1, "player": player, "skills": map[string]any{"base_levels": g.levels[:]}})
		return string(b) + "\nok\n"
	case f[0] == "cmd" && len(f) >= 3:
		switch f[1] {
		case "stat":
			if n := max(num(3), 1); n <= g.statPts {
				g.statPts -= n
				return "ok\n"
			}
			return "err no stat points\n"
		case "skillpt":
			if i := num(2); g.skillPts > 0 && i >= 0 && i < len(g.levels) {
				g.skillPts--
				g.levels[i]++
				return "ok\n"
			}
			return "err no skill point\n"
		case "select":
			hand := 1
			if num(3) != 0 {
				hand = 0
			}
			if !g.DeafSelect {
				g.hands[hand] = num(2)
			}
			return "ok\n"
		case "skill":
			if !g.NoSummon {
				g.minions++
			}
			return "ok\n"
		}
	case f[0] == "debug" && len(f) >= 2:
		switch f[1] {
		case "stat":
			switch num(2) { // S itemstatcost.txt: 4 statpts, 5 newskills, 12 level, 13 experience, 14 gold
			case 4:
				g.statPts = num(3)
			case 5:
				g.skillPts = num(3)
			case 12:
				g.level = num(3)
			case 13:
				g.exp = num(3)
			case 14:
				g.gold = num(3)
			}
			return "ok\n"
		case "statpts":
			g.statPts = num(2)
			return "ok\n"
		case "skillpts":
			g.skillPts = num(2)
			return "ok\n"
		case "give":
			if len(f) < 3 {
				return "err usage: debug give <code> ...\n"
			}
			if len(f) >= 4 && f[3] == "pack" {
				if g.used+packCells(f[2]) > packCols*packRows {
					return "err no room\n"
				}
				g.used += packCells(f[2])
				g.pack = append(g.pack, f[2])
			}
			if len(f) >= 4 && f[3] == "belt" {
				g.belt = append(g.belt, f[2])
			}
			return "ok\n"
		case "belt":
			g.belt = nil
			return "ok\n"
		case "players":
			return "1\tSmoke\t30.5\t20.5\t-1\t1\t0\nok\n"
		case "minions":
			ms := make([]map[string]any, g.minions)
			for i := range ms {
				ms[i] = map[string]any{"id": 10 + i, "name": "summon", "owner": 1}
			}
			b, _ := json.Marshal(map[string]any{"tick": 1, "hero": 1, "minions": ms})
			return string(b) + "\nok\n"
		case "goto":
			return "ok 30.5 20.5\n"
		case "points", "wp", "warp", "spawn", "quest", "difficulty":
			return "ok\n"
		}
	}
	return fmt.Sprintf("err unknown verb '%s'\n", line)
}
