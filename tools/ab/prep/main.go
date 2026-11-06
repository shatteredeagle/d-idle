// Command prep makes the prepared hero of the "late start" A/B variant (docs/ab-design.md, Variants): it starts the
// lab d2d with --lab-verbs, creates a fresh character, applies the declared setup (level, unspent points, purse, merc),
// saves and quits. The A/B arms then load that save with a d2d WITHOUT --lab-verbs, so no setup verb can run after the
// run start: "writes after start: 0" holds by construction (the fork refuses gated debug verbs; docs/lab-verbs.md).
// Every verb is appended to a ledger file with its reason before it is sent.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"d2lod-bot/internal/d2dinput"
)

// Setup is the declared starting state, identical for both arms.
type Setup struct {
	Clvl, Gold, Stash, MercType int
}

// Verbs returns the debug verbs for s in the order they are sent: points follow the level like the build drill
// (5 stat and 1 skill point per level gained, all unspent so the bot's own leveler spends them).
func Verbs(s Setup) []string {
	v := []string{
		fmt.Sprintf("debug stat 12 %d", s.Clvl),
		fmt.Sprintf("debug statpts %d", 5*(s.Clvl-1)),
		fmt.Sprintf("debug skillpts %d", s.Clvl-1),
		fmt.Sprintf("debug stat 14 %d", s.Gold),
		fmt.Sprintf("debug stat 15 %d", s.Stash),
	}
	if s.MercType >= 0 {
		v = append(v, fmt.Sprintf("debug merc hire %d", s.MercType))
	}
	return v
}

type ledgerLine struct {
	Time   string `json:"time"`
	Verb   string `json:"verb"`
	Reason string `json:"reason"`
	Reply  string `json:"reply,omitempty"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "prep:", err)
		os.Exit(1)
	}
}

func run() error {
	bin := flag.String("bin", os.Getenv("D2D_BIN"), "d2d lab binary (must know --lab-verbs)")
	data := flag.String("data", os.Getenv("D2D_DATA"), "d2d --data dir (MPQs)")
	sock := flag.String("sock", os.Getenv("D2D_SOCK"), "control socket path")
	home := flag.String("home", "", "d2d home (XDG config and data go under <home>/config and <home>/data); the arms copy it")
	name := flag.String("name", "", "character name")
	class := flag.Int("class", 4, "d2d class index of the new character (4 = Sorceress)")
	seed := flag.Int64("seed", 0, "map seed")
	ledger := flag.String("ledger", "", "ledger file (JSON lines), default <home>/prep-ledger.jsonl")
	var s Setup
	flag.IntVar(&s.Clvl, "clvl", 11, "start level")
	flag.IntVar(&s.Gold, "gold", 500, "purse gold")
	flag.IntVar(&s.Stash, "stash", 0, "stash gold")
	flag.IntVar(&s.MercType, "merc", 0, "hireling.txt Id to hire for free (the Act 1 Rogue is 0, docs/cheats.md); -1 none")
	flag.Parse()
	if *home == "" || *name == "" || *sock == "" || *bin == "" {
		return fmt.Errorf("needs --home, --name, --sock and --bin (or D2D_BIN, D2D_SOCK)")
	}
	if s.Clvl < 2 || s.Clvl > 99 {
		return fmt.Errorf("clvl %d out of range", s.Clvl)
	}
	if *ledger == "" {
		*ledger = filepath.Join(*home, "prep-ledger.jsonl")
	}
	if err := os.MkdirAll(*home, 0o755); err != nil {
		return err
	}
	lf, err := os.OpenFile(*ledger, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer lf.Close()
	logf, err := os.Create(filepath.Join(*home, "prep-d2d.out"))
	if err != nil {
		return err
	}
	defer logf.Close()

	p := &d2dinput.Process{Bin: *bin, Data: *data, Sock: *sock, ConfigDir: filepath.Join(*home, "config"), DataDir: filepath.Join(*home, "data"),
		Nice: 10, ExtraArgs: []string{"--lockstep", "--lab-verbs", "--toggle", "autoloot=off,autoparty=off"}, Log: logf}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := p.Start(ctx, d2dinput.Spec{Name: *name, Class: *class, Seed: *seed}); err != nil {
		return err
	}
	if err := p.WaitInGame(ctx, 3*time.Minute); err != nil {
		_ = p.Stop()
		return err
	}
	c := p.Controller().C // d2d takes one control client: reuse the one Start made
	enc := json.NewEncoder(lf)
	for _, v := range Verbs(s) {
		l := ledgerLine{Time: time.Now().Format(time.RFC3339), Verb: v, Reason: "SETUP: late-start A/B hero, identical for both arms (docs/ab-design.md Variants)"}
		r, err := c.Do(ctx, v)
		if r != nil {
			l.Reply = strings.Join(append([]string{r.Rest}, r.Lines...), " | ")
		}
		_ = enc.Encode(l)
		if err != nil {
			_ = p.Stop()
			return fmt.Errorf("%s: %w", v, err)
		}
	}
	if _, err := c.Step(ctx, 50); err != nil { // let the level, points and merc settle before the save
		_ = p.Stop()
		return err
	}
	return p.SaveExit(ctx)
}
