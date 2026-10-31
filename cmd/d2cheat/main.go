// d2cheat: logged test cheats for the throwaway characters (docs/cheats.md).
//
//	tools/live.sh multi d2cheat --reason "why" [--actor NAME] heal
//	tools/live.sh multi d2cheat --reason "why" level 25
//	tools/live.sh multi d2cheat ledger --since 2h
//
// Every subcommand except show, ledger and probe writes, needs --reason and is recorded in the ledger.
package main

import (
	"context"
	"d2lod-bot/internal/buildinfo"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"d2lod-bot/internal/cheat"
	"d2lod-bot/internal/cheat/d2s"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/mem"
)

func usage() {
	fmt.Fprint(os.Stderr, `usage: d2cheat [--reason TEXT] [--actor NAME] [--ledger FILE] <command>
  heal                      refill life, mana, stamina
  god on|off [--for 60s]    invincibility (keeps running for the duration; restores max life on exit/TERM/INT)
  restore [--maxlife N]     undo a leftover god state (killed run): from the marker, or force max life N
  gold N | stashgold N      set gold
  level N | xp N            set level / experience
  stats str=N dex=N vit=N ene=N points=N skills=N
  setstat unit=server|client|both vec=merged|base|both stat=ID value=RAW   (research primitive)
  overlay xprate=N | nodrop=0 | exppct=P   per-instance excel override (needs -direct -txt launch)
  save FILE key=value...    edit a .d2s (game must be stopped)
  quest set FILE [D/]ACT.QUEST[:STATE]...   quest words (STATE done|started|pending|none|number); wINDEX = raw word
  quest clear FILE [D/]ACT.QUEST...         zero those words;  quest reset FILE [D]  zero a whole difficulty (default all)
  waypoints FILE [D/]all|none|ID+ID...      waypoints of difficulty D (default normal); ID = bit 0-38 or name (cold_plains)
  unlock-difficulty FILE nightmare|hell     raise the progression byte
  act FILE [D/]N            start in the town of act N (1-5): difficulty byte, town waypoint, earlier act-done words
  (FILE = character name or path; D = normal|nightmare|hell|0-2; all edits need the game stopped, a backup is made)
  show                      read-only: both player copies (no ledger)
  ledger [--since 2h] [--char NAME] [--kind K] [--json]   list / summarize the ledger
  probe                     read-only research dump
`)
	os.Exit(2)
}

func main() {
	buildinfo.Exit("d2cheat")
	args := os.Args[1:]
	var reason, actor, ledgerPath string
	for len(args) > 0 && strings.HasPrefix(args[0], "--") {
		k := args[0]
		if k == "--reason" || k == "--actor" || k == "--ledger" {
			if len(args) < 2 {
				usage()
			}
			switch k {
			case "--reason":
				reason = args[1]
			case "--actor":
				actor = args[1]
			default:
				ledgerPath = args[1]
			}
			args = args[2:]
			continue
		}
		break
	}
	if len(args) == 0 {
		usage()
	}
	cmd, rest := args[0], args[1:]
	env := cheat.DefaultEnv()
	if ledgerPath != "" {
		env.Ledger = cheat.NewLedger(ledgerPath)
	}
	switch cmd {
	case "ledger":
		cmdLedger(env.Ledger, rest)
		return
	case "show":
		_, st, err := env.Inspect()
		if err != nil {
			fatal(err)
		}
		keys := make([]string, 0, len(st))
		for k := range st {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("%-20s %v\n", k, st[k])
		}
		if len(rest) > 0 && rest[0] == "-v" {
			withGame(show)
		}
		return
	case "probe":
		withGame(probe)
		return
	case "savedump":
		b, err := os.ReadFile(rest[0])
		if err != nil {
			fatal(err)
		}
		f, err := d2s.Parse(b)
		if err != nil {
			fatal(err)
		}
		fmt.Printf("%s class %d level %d progression %d difficulty %x %x %x\n", f.Name(), f.Class(), f.Level(), f.Progression(), f.Difficulty(0), f.Difficulty(1), f.Difficulty(2))
		for _, st := range f.Stats {
			fmt.Printf("  stat %2d = %d\n", st.ID, st.Value)
		}
		fmt.Printf("  waypoints %x %x %x\n", f.WaypointMask(0), f.WaypointMask(1), f.WaypointMask(2))
		return
	case "qdump":
		qdump(rest[0])
		return
	case "speedtest":
		withGame(func(p *mem.Proc) { speedtest(p, rest) })
		return
	case "seq":
		withGame(func(p *mem.Proc) { seqCmd(p, rest) })
		return
	case "refs":
		withGame(func(p *mem.Proc) { refsCmd(p, rest) })
		return
	}
	req := cheat.Request{Reason: reason, Actor: actor, Params: map[string]any{}}
	switch cmd {
	case "heal":
		req.Kind = cheat.KindHeal
	case "god":
		req.Kind = cheat.KindGod
		on := len(rest) > 0 && rest[0] == "on"
		if len(rest) == 0 || (rest[0] != "on" && rest[0] != "off") {
			usage()
		}
		req.Params["on"] = on
		for i, a := range rest {
			if a == "--for" && i+1 < len(rest) {
				req.Params["for"] = rest[i+1]
			}
			if a == "--simhit" && i+1 < len(rest) {
				n, _ := strconv.ParseInt(rest[i+1], 0, 64)
				req.Params["simhit"] = n
			}
		}
	case "restore":
		req.Kind = cheat.KindGodRestore
		kv(req.Params, rest)
	case "gold", "stashgold", "level", "xp":
		req.Kind = cheat.Kind(cmd)
		if len(rest) != 1 {
			usage()
		}
		n, err := strconv.ParseInt(rest[0], 0, 64)
		if err != nil {
			fatal(err)
		}
		req.Params["value"] = n
	case "stats", "setstat":
		req.Kind = cheat.Kind(cmd)
		kv(req.Params, rest)
	case "overlay":
		req.Kind = cheat.KindDataOverlay
		kv(req.Params, rest)
	case "save":
		req.Kind = cheat.KindSaveEdit
		if len(rest) < 2 {
			usage()
		}
		req.Params["file"] = rest[0]
		kv(req.Params, rest[1:])
	case "quest", "waypoints", "unlock-difficulty", "act":
		req.Kind = cheat.KindSaveEdit
		progressParams(req.Params, cmd, rest)
	default:
		usage()
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	res, err := env.Do(ctx, req)
	if err != nil {
		fatal(err)
	}
	fmt.Println(res.Message)
	printDiff(res.Entry.Before, res.Entry.After)
}

func kv(m map[string]any, args []string) {
	for _, a := range args {
		k, v, ok := strings.Cut(a, "=")
		if !ok {
			fatal(fmt.Errorf("want key=value, got %q", a))
		}
		if n, err := strconv.ParseInt(v, 0, 64); err == nil {
			m[k] = n
		} else {
			m[k] = v
		}
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "d2cheat:", err)
	os.Exit(1)
}

func printDiff(before, after map[string]any) {
	keys := make([]string, 0, len(after))
	for k := range after {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		a := after[k]
		b, had := before[k]
		if !had || fmt.Sprint(a) != fmt.Sprint(b) {
			fmt.Printf("  %-18s %v -> %v\n", k, b, a)
		}
	}
}

func withGame(f func(*mem.Proc)) {
	p, err := mem.OpenGame()
	if err != nil {
		fatal(err)
	}
	defer p.Close()
	f(p)
}

func show(p *mem.Proc) {
	pl, err := d2.New(p).Player()
	if err != nil {
		fatal(err)
	}
	fmt.Printf("%s %v level %d life %d/%d mana %d/%d exp %d gold %d stash %d points %d/%d\n", pl.Name, pl.Class, pl.Level, pl.Life, pl.MaxLife,
		pl.Mana, pl.MaxMana, pl.Exp, pl.Gold, pl.StashGold, pl.StatPoints, pl.SkillPoints)
	server, account, realm, err := cheat.OfflineInfo(p)
	fmt.Printf("online markers: server=%q account=%q realm=%q err=%v\n", server, account, realm, err)
	probe(p)
}

func cmdLedger(l *cheat.Ledger, args []string) {
	var f cheat.Filter
	asJSON := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--since":
			i++
			d, err := time.ParseDuration(args[i])
			if err != nil {
				t, err2 := time.Parse(time.RFC3339, args[i])
				if err2 != nil {
					fatal(err)
				}
				f.Since = t
			} else {
				f.Since = time.Now().Add(-d)
			}
		case "--char":
			i++
			f.Character = args[i]
		case "--kind":
			i++
			f.Kind = cheat.Kind(args[i])
		case "--actor":
			i++
			f.Actor = args[i]
		case "--json":
			asJSON = true
		}
	}
	es, err := l.List(f)
	if err != nil {
		fatal(err)
	}
	for _, e := range es {
		if asJSON {
			fmt.Println(cheat.FormatJSON(e))
			continue
		}
		ok := "ok  "
		if !e.Success {
			ok = "FAIL"
		}
		fmt.Printf("%s %s %-6s %-7s %-9s %-9s %v  [%s]", e.Time.Local().Format("01-02 15:04:05"), ok, e.Actor, e.Character, e.Kind, "", e.Params, e.Reason)
		if e.Error != "" {
			fmt.Printf(" err=%s", e.Error)
		}
		fmt.Println()
	}
	fmt.Println(cheat.Summarize(es))
}

// progressParams turns the progress subcommands into saveedit parameters (see internal/cheat/save_progress.go).
func progressParams(m map[string]any, cmd string, rest []string) {
	switch cmd {
	case "quest":
		if len(rest) < 2 {
			usage()
		}
		sub, file, refs := rest[0], rest[1], rest[2:]
		m["file"] = file
		switch sub {
		case "set":
			if len(refs) == 0 {
				usage()
			}
			m["quest"] = strings.Join(refs, ",")
		case "clear":
			if len(refs) == 0 {
				usage()
			}
			for i, r := range refs {
				if !strings.Contains(r, ":") {
					refs[i] = r + ":none"
				}
			}
			m["quest"] = strings.Join(refs, ",")
		case "reset":
			mask := int64(7)
			if len(refs) == 1 {
				d, ok := map[string]int64{"normal": 0, "nightmare": 1, "hell": 2, "0": 0, "1": 1, "2": 2}[refs[0]]
				if !ok {
					usage()
				}
				mask = 1 << d
			}
			m["questclear"] = mask
		default:
			usage()
		}
	case "waypoints":
		if len(rest) < 2 {
			usage()
		}
		m["file"] = rest[0]
		m["wp"] = strings.Join(rest[1:], "+")
	case "unlock-difficulty":
		if len(rest) != 2 {
			usage()
		}
		m["file"] = rest[0]
		d, ok := map[string]int64{"nightmare": 1, "hell": 2, "1": 1, "2": 2}[rest[1]]
		if !ok {
			usage()
		}
		m["unlock"] = d
	case "act":
		if len(rest) != 2 {
			usage()
		}
		m["file"] = rest[0]
		m["startat"] = rest[1]
	}
}
