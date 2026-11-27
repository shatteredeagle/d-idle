// Command prep makes the prepared hero of the "late start" A/B variant (docs/ab-design.md, Variants): it starts the
// lab d2d with --lab-verbs, creates a fresh character, applies the declared setup (the hero library's profile: level and
// its experience, points spent by the build table or banked, weapon; then purse, merc), runs the hero self-check, writes
// <home>/hero.json, saves and quits. A failed self-check exits non-zero without a save. The A/B arms then load that save with a d2d WITHOUT --lab-verbs, so no setup verb can run after the
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

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dinput"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/scenario"
	"d2lod-bot/internal/scenario/hero"
)

// Setup is the declared starting state, identical for both arms. The hero itself (level, points, weapon) comes from the
// hero library (internal/scenario/hero): profile "played" spends the points as the build table does and binds the main
// attack, "banked" leaves every point unspent for the bot's own leveler (the old prep behaviour, which strands a hero
// whose run has resource.pointsDecisions 0).
type Setup struct {
	Hero                  hero.Decl
	Gold, Stash, MercType int
}

// Verbs returns the hero and the lines for s in the order they are sent: the hero's setup writes and player commands,
// then purse, stash and merc.
func Verbs(s Setup, t hero.Tables) (hero.Hero, []hero.Verb, error) {
	h, err := hero.Resolve(s.Hero, t)
	if err != nil {
		return h, nil, err
	}
	why := "SETUP: late-start A/B hero, identical for both arms (docs/ab-design.md Variants): "
	v := append([]hero.Verb(nil), h.Verbs...)
	v = append(v, hero.Verb{Line: fmt.Sprintf("debug stat 14 %d", s.Gold), Reason: why + "purse", Write: true},
		hero.Verb{Line: fmt.Sprintf("debug stat 15 %d", s.Stash), Reason: why + "stash gold", Write: true})
	if s.MercType >= 0 {
		v = append(v, hero.Verb{Line: fmt.Sprintf("debug merc hire %d", s.MercType), Reason: why + "free merc", Write: true})
	}
	return h, v, nil
}

// WeaponSlot is the right-hand body slot of `debug equip`.
const WeaponSlot = hero.WeaponSlot

// ClassName is the class of a d2d UI class index (the inverse of scenario.ClassIndex).
func ClassName(idx int) (string, error) {
	for _, n := range []string{"Amazon", "Sorceress", "Necromancer", "Paladin", "Barbarian", "Druid", "Assassin"} {
		if i, err := scenario.ClassIndex(n); err == nil && i == idx {
			return n, nil
		}
	}
	return "", fmt.Errorf("no class with d2d index %d", idx)
}

// Record is <home>/hero.json: what the prep made and what the game showed after it. The run id ties the hero to the
// A/B result (abrun.sh writes the HeroLine into the arm log).
type Record struct {
	Run        string       `json:"run"`
	Hero       hero.Hero    `json:"hero"`
	Provenance string       `json:"provenance"`
	Checks     []hero.Check `json:"checks"`
	Failed     string       `json:"failed,omitempty"`
}

// HeroLine is the one-line summary the A/B log carries (abreport reads it).
func (r Record) HeroLine() string {
	sc := "ok"
	if r.Failed != "" {
		sc = r.Failed
	}
	return fmt.Sprintf("hero: run=%s profile=%s class=%s build=%s clvl=%d attack=%s selfcheck=%s tags=%s", r.Run, r.Hero.Profile,
		r.Hero.Class, r.Hero.Build, r.Hero.Level, strings.ReplaceAll(r.Hero.Attack, " ", "_"), sc, strings.Join(r.Hero.Tags, ","))
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
	dataDir := flag.String("data", os.Getenv("D2D_DATA"), "d2d --data dir (MPQs); the game tables for the played profile are read from it")
	sock := flag.String("sock", os.Getenv("D2D_SOCK"), "control socket path")
	home := flag.String("home", "", "d2d home (XDG config and data go under <home>/config and <home>/data); the arms copy it")
	name := flag.String("name", "", "character name")
	class := flag.Int("class", 4, "d2d class index of the new character (4 = Sorceress)")
	seed := flag.Int64("seed", 0, "map seed")
	ledger := flag.String("ledger", "", "ledger file (JSON lines), default <home>/prep-ledger.jsonl")
	runID := flag.String("run", "", "run id written into hero.json and the hero line (default the character name)")
	var s Setup
	flag.StringVar(&s.Hero.Profile, "profile", hero.Played, "hero profile: played (points spent by the build table, main attack bound) or banked (every point unspent; only for points scenarios)")
	flag.StringVar(&s.Hero.Build, "build", "", "build table (default: the table a fresh hero of the class gets)")
	flag.IntVar(&s.Hero.Level, "clvl", 11, "start level")
	flag.IntVar(&s.Hero.QuestSkillPts, "quest-skillpts", 1, "skill points of quests done (the Den of Evil gives 1; the build tables fold it in)")
	flag.IntVar(&s.Gold, "gold", 500, "purse gold")
	flag.IntVar(&s.Stash, "stash", 0, "stash gold")
	flag.IntVar(&s.MercType, "merc", 0, "hireling.txt Id to hire for free (the Act 1 Rogue is 0, docs/cheats.md); -1 none")
	weapon := flag.String("weapon", "", "item code worn in the right hand (SETUP), e.g. hax; empty = none, starter = the class's starting weapon")
	flag.Parse()
	if *home == "" || *name == "" || *sock == "" || *bin == "" {
		return fmt.Errorf("needs --home, --name, --sock and --bin (or D2D_BIN, D2D_SOCK)")
	}
	switch *weapon {
	case "":
		s.Hero.Weapon = "none"
	case "starter":
		s.Hero.Weapon = ""
	default:
		s.Hero.Weapon = *weapon
	}
	if s.Hero.Profile == hero.Banked {
		fmt.Fprintln(os.Stderr, "prep: WARNING profile banked: every point stays unspent; the hero has no class skills unless the bot spends them")
	}
	cn, err := ClassName(*class)
	if err != nil {
		return err
	}
	s.Hero.Class = cn
	if *runID == "" {
		*runID = *name
	}
	var tabs hero.Tables
	if s.Hero.Profile != hero.Banked || s.Hero.Weapon == "" {
		db, err := data.Open(*dataDir, filepath.Join(*home, "tables-cache"))
		if err != nil {
			return fmt.Errorf("game tables: %w", err)
		}
		tabs = hero.DBTables(db)
	}
	h, verbs, err := Verbs(s, tabs)
	if err != nil {
		return err // before any game starts: a hero the library cannot make is never made
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

	p := &d2dinput.Process{Bin: *bin, Data: *dataDir, Sock: *sock, ConfigDir: filepath.Join(*home, "config"), DataDir: filepath.Join(*home, "data"),
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
	wrote := false
	for _, v := range verbs {
		if !v.Write && wrote { // a player command checks the level and the points: let the writes land first
			if _, err := c.Step(ctx, 5); err != nil {
				_ = p.Stop()
				return err
			}
			wrote = false
		}
		l := ledgerLine{Time: time.Now().Format(time.RFC3339), Verb: v.Line, Reason: v.Reason}
		r, err := c.Do(ctx, v.Line)
		if r != nil {
			l.Reply = strings.Join(append([]string{r.Rest}, r.Lines...), " | ")
		}
		_ = enc.Encode(l)
		if err != nil {
			_ = p.Stop()
			return fmt.Errorf("%s: %w", v.Line, err)
		}
		if v.Write {
			wrote = true
		} else if _, err := c.Step(ctx, 2); err != nil { // one command per tick (U whether d2d takes several in one)
			_ = p.Stop()
			return err
		}
	}
	if _, err := c.Step(ctx, 50); err != nil { // let the level, points and merc settle before the save
		_ = p.Stop()
		return err
	}
	rec, err := selfCheck(ctx, c, h, enc)
	if err != nil {
		_ = p.Stop()
		return err
	}
	rec.Run = *runID
	rec.Provenance = h.Provenance(*name)
	if b, err := json.MarshalIndent(rec, "", " "); err == nil {
		_ = os.WriteFile(filepath.Join(*home, "hero.json"), append(b, '\n'), 0o644)
	}
	fmt.Println(rec.HeroLine())
	if rec.Failed != "" {
		_ = p.Stop()
		return fmt.Errorf("selfcheck: %s (hero.json has the rows)", rec.Failed)
	}
	return p.SaveExit(ctx)
}

// selfCheck reads the hero back (read-only verbs) and compares it with h.
func selfCheck(ctx context.Context, c *d2dctl.Client, h hero.Hero, enc *json.Encoder) (Record, error) {
	var o hero.Observed
	// Read the weapon back like the bot does (dump stats "weapons"/"hero"): an equip verb that answered ok but left the
	// right hand empty would otherwise only show up as src=default hours later.
	line, err := firstLine(ctx, c, "dump stats")
	if err != nil {
		return Record{}, err
	}
	rb, rerr := WeaponReadback(line, h.Weapon)
	_ = enc.Encode(ledgerLine{Time: time.Now().Format(time.RFC3339), Verb: "dump stats", Reason: "READ-ONLY readback of the SETUP weapon", Reply: rb})
	if rerr != nil {
		if len(line) > 3000 {
			line = line[:3000]
		}
		fmt.Fprintln(os.Stderr, "prep: raw dump stats:", line)
		return Record{}, rerr
	}
	if h.Weapon != "" {
		o.Weapon = &h.Weapon // WeaponReadback proved this code in the right hand
	}
	if line, err := firstLine(ctx, c, "dump player,stats,skills"); err == nil {
		if err := hero.ReadPointDump(&o, line); err != nil {
			fmt.Fprintln(os.Stderr, "prep:", err)
		}
	}
	if sv, _, err := c.State(ctx); err == nil {
		l := sv.Level
		o.Level = &l
		if v, ok := sv.Int("lskill"); ok {
			o.LeftSkill = &v
		}
		if v, ok := sv.Int("rskill"); ok {
			o.RightSkill = &v
		}
	}
	rec := Record{Hero: h, Checks: h.SelfCheck(o)}
	rec.Failed = hero.Failed(rec.Checks)
	_ = enc.Encode(ledgerLine{Time: time.Now().Format(time.RFC3339), Verb: "selfcheck", Reason: "READ-ONLY hero self-check", Reply: "failed: " + rec.Failed})
	return rec, nil
}

func firstLine(ctx context.Context, c *d2dctl.Client, verb string) (string, error) {
	r, err := c.Do(ctx, verb)
	if err != nil {
		return "", fmt.Errorf("%s: %w", verb, err)
	}
	line := strings.TrimSpace(r.Rest)
	for _, l := range r.Lines {
		if strings.TrimSpace(l) != "" {
			return l, nil
		}
	}
	return line, nil
}

// WeaponReadback checks the "dump stats" reply for the SETUP weapon and returns a one-line summary for the ledger.
// With want == "" it only summarises. It errors when a weapon was asked for and the right hand (slot 4) shows none, or
// when the dump has no "weapons" key at all (a d2d binary older than the lab2-stats branch: the bot could not read it
// either, so the A/B would run on the default offence).
func WeaponReadback(dump, want string) (string, error) {
	var d struct {
		Stats struct {
			Weapons []struct {
				Code   string `json:"code"`
				Slot   int    `json:"slot"`
				MinDam int    `json:"mindam"`
				MaxDam int    `json:"maxdam"`
			} `json:"weapons"`
			Hero struct {
				Hands []struct {
					Slot int `json:"slot"`
					Min  int `json:"min"`
					Max  int `json:"max"`
				} `json:"hands"`
			} `json:"hero"`
		} `json:"stats"`
	}
	if err := json.Unmarshal([]byte(dump), &d); err != nil {
		return "", fmt.Errorf("dump stats readback: %w", err)
	}
	var parts []string
	found := false
	for _, w := range d.Stats.Weapons {
		parts = append(parts, fmt.Sprintf("weapon %s slot %d base %d-%d", w.Code, w.Slot, w.MinDam, w.MaxDam))
		if w.Slot == WeaponSlot && (want == "" || w.Code == want) {
			found = true
		}
	}
	for _, h := range d.Stats.Hero.Hands {
		parts = append(parts, fmt.Sprintf("hero hand slot %d dmg %d-%d", h.Slot, h.Min, h.Max))
	}
	sum := strings.Join(parts, "; ")
	if sum == "" {
		sum = "no weapons in dump"
	}
	if want != "" && !found {
		return sum, fmt.Errorf("SETUP weapon %q is not in the right hand after the equip verb (%s): the binary may predate the lab2-stats dump or the slot is wrong", want, sum)
	}
	return sum, nil
}
