package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/scenario"
	"d2lod-bot/internal/scenario/hero"
)

func cmdValidate(args []string, w io.Writer) error {
	fs := newFlags("validate")
	dir := fs.String("dir", defaultDir, "scenario directory")
	board := fs.String("board", "data/questboard", "quest board data directory")
	earned := fs.String("milestones", "tools/provingground/earned/milestones.json", "earned-save milestone catalogue")
	gamedata := fs.String("gamedata", "", "game data directory: the hero check then uses the game's learn rules (default: the plan tables alone)")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}
	tabs := hero.NameTables()
	if *gamedata != "" {
		db, err := data.Open(*gamedata, filepath.Join(os.TempDir(), "d2scenario-tables"))
		if err != nil {
			return fmt.Errorf("game tables: %w", err)
		}
		tabs = hero.DBTables(db)
	}
	scs, err := loadAll(*dir)
	if err != nil {
		return err
	}
	bs, err := loadBehaviours(*dir, *board)
	if err != nil {
		return err
	}
	_, problems := scenario.Coverage(bs, scs)
	if len(problems) > 0 {
		return fmt.Errorf("%s", strings.Join(problems, "; "))
	}
	ms, err := scenario.LoadMilestones(*earned)
	if err != nil {
		return err
	}
	for _, s := range scs {
		if err := s.CheckProvenance(); err != nil {
			return err
		}
		if s.Pre.FromSave != "" {
			if _, ok := ms.Find(s.Pre.FromSave); !ok {
				return fmt.Errorf("scenario %s: pre.from_save %q is not a milestone of %s", s.ID, s.Pre.FromSave, *earned)
			}
		}
		if s.Faults != "" {
			if _, err := faultSet(s.Faults); err != nil {
				return fmt.Errorf("scenario %s: faults: %w", s.ID, err)
			}
		}
		if s.Quest != nil {
			if !hasBehaviour(bs, "quest."+s.Quest.Quest+"."+s.Quest.Chunk) {
				return fmt.Errorf("scenario %s: quest chunk %s/%s is not in %s", s.ID, s.Quest.Quest, s.Quest.Chunk, *board)
			}
		}
		for _, c := range s.Matrix.Classes {
			su, err := scenario.BuildSetup(s, c, s.Matrix.Seeds[0])
			if err != nil {
				return err
			}
			if err := staticCheck(s, c, su, tabs); err != nil {
				return err
			}
		}
	}
	fmt.Fprintf(w, "ok: %d scenarios, %d behaviours\n", len(scs), len(bs))
	return nil
}

// scenarioHero is the hero a scenario's setup declares for one class (design 6.2): the played profile resolves through
// the hero library (points spent as the build table spends them, main attack bound; tabs are the game's tables), the
// banked one leaves the level's points unspent; then the purse and the potions it gives. A raw pre.stats write of the
// level or the points replaces the expectation. Without tables the played hero is banked in the expectation only.
func scenarioHero(s scenario.Scenario, class string, tabs hero.Tables) (hero.Hero, hero.Floors) {
	lvl := max(s.Pre.Level, 1)
	h := hero.Hero{Decl: hero.Decl{Profile: hero.Banked, Class: strings.ToLower(class), Build: s.Trigger.Build, Level: lvl},
		StatPts: 5 * (lvl - 1), SkillPts: lvl - 1, AttackID: -1, AttackIndex: -1}
	if s.HeroProfile() == scenario.HeroPlayed && lvl > 1 && tabs != nil {
		if ph, err := hero.Resolve(hero.Decl{Profile: hero.Played, Class: class, Build: s.Trigger.Build, Level: lvl, Weapon: "none"}, tabs); err == nil {
			h = ph
		}
	}
	for _, st := range s.Pre.Stats {
		switch st.Stat { // S itemstatcost.txt: 4 statpts, 5 newskills, 12 level
		case 4:
			h.StatPts = st.Value
		case 5:
			h.SkillPts = st.Value
		case 12:
			h.Level = st.Value
		}
	}
	f := hero.Floors{Purse: s.Pre.Gold}
	for _, it := range append(append([]scenario.Item(nil), s.Pre.Items...), s.Pre.Pack...) {
		if isPotion(it.Code) && (it.Where == "" || it.Where == "pack") {
			f.Potions += max(it.Count, 1)
		}
	}
	for _, code := range s.Pre.Belt {
		if isPotion(code) {
			f.Potions++
		}
	}
	return h, f
}

func isPotion(code string) bool {
	return len(code) == 3 && (strings.HasPrefix(code, "hp") || strings.HasPrefix(code, "mp") || code == "rvs" || code == "rvl")
}

// staticCheck is the T0 half of the self-check for one scenario and class: the goal is not met at the start, no ground
// item is given before the position verb, the declared pack items fit the pack, and a hero above level 1 resolves with the build it plays and (unless the
// scenario is about points) gets its main attack at that level with quest points 0 (a quest record written by setup
// does not grant its points).
func staticCheck(s scenario.Scenario, class string, su scenario.Setup, t hero.Tables) error {
	at := fmt.Sprintf("scenario %s (%s): selfcheck", s.ID, class)
	if why := goalMet(s); why != "" {
		return fmt.Errorf("%s: goal: %s", at, why)
	}
	lastPos, firstGround := -1, -1
	for i, v := range su.Verbs {
		f := strings.Fields(v.Line)
		switch {
		case len(f) >= 2 && f[0] == "debug" && (f[1] == "goto" || f[1] == "warp"):
			lastPos = i
		case len(f) >= 4 && f[0] == "debug" && f[1] == "give" && isNumber(f[3]) && firstGround < 0:
			firstGround = i // `debug give <code> <quality>` drops at the hero's feet
		}
	}
	if firstGround >= 0 && firstGround < lastPos {
		return fmt.Errorf("%s: order: ground item %q given before the position verb", at, su.Verbs[firstGround].Line)
	}
	if n := packNeed(s); n > packCols*packRows { // the lab d2d answers "no room" and the run is BLOCKED (2026-11-27)
		return fmt.Errorf("%s: pack: the declared pack items need at least %d cells, the pack has %d (give fewer or smaller items)", at, n, packCols*packRows)
	}
	if s.Pre.Level < 2 {
		return nil
	}
	profile := s.HeroProfile() // the declared one (mandatory); a banked hero has no attack to learn up front
	cs := hero.Static(hero.Decl{Profile: profile, Class: class, Build: s.Trigger.Build, Level: s.Pre.Level}, t)
	if bad := hero.Failed(cs); bad != "" {
		for _, c := range cs {
			if !c.OK {
				return fmt.Errorf("%s: %s: %s", at, c.Name, c.Got)
			}
		}
	}
	return nil
}

// packNeed is a lower bound of the pack cells the declared pack items take (packCells: a code it does not know counts
// one cell; the hero's starter items are not counted), so a scenario it refuses cannot be set up.
func packNeed(s scenario.Scenario) int {
	n := 0
	for _, it := range append(append([]scenario.Item(nil), s.Pre.Items...), s.Pre.Pack...) {
		if it.Where == "" || it.Where == "pack" {
			n += packCells(it.Code) * max(it.Count, 1)
		}
	}
	return n
}

// goalMet names why the run would have nothing to do: every --goals entry is a reach goal the start level already
// meets and nothing else drives the bot (no board taps, no API calls, no accepted chunk in the plan store).
func goalMet(s scenario.Scenario) string {
	if s.Trigger.Driver != "" || len(s.Trigger.API) > 0 || s.Pre.Board != nil || strings.TrimSpace(s.Trigger.Goals) == "" {
		return ""
	}
	for _, g := range strings.Split(s.Trigger.Goals, ";") {
		n, ok := strings.CutPrefix(strings.TrimSpace(g), "reach:")
		lvl, err := strconv.Atoi(n)
		if !ok || err != nil || max(s.Pre.Level, 1) < lvl {
			return ""
		}
	}
	return fmt.Sprintf("%s is met at the start (pre.level %d) and nothing else drives the bot", s.Trigger.Goals, s.Pre.Level)
}

func isNumber(s string) bool { _, err := strconv.Atoi(s); return err == nil }

// prepSelfCheck reads the prepared hero from the prep d2d and compares it with the scenario's declaration; the failed
// row names ("" when all pass) end the run as BLOCKED `selfcheck: <names>`. The rows go to the setup ledger as a
// READ-ONLY line.
func prepSelfCheck(ctx context.Context, c *d2dctl.Client, s scenario.Scenario, class, ledger string, tabs hero.Tables) string {
	h, f := scenarioHero(s, class, tabs)
	var o hero.Observed
	if rep, err := c.Do(ctx, "dump player,stats,skills"); err == nil {
		line := strings.TrimSpace(rep.Rest)
		for _, l := range rep.Lines {
			if strings.TrimSpace(l) != "" {
				line = l
				break
			}
		}
		_ = hero.ReadPointDump(&o, line) // an unreadable dump leaves the rows "not read", which fail
	}
	if sv, _, err := c.State(ctx); err == nil {
		l, g := sv.Level, sv.Gold
		o.Level, o.Gold = &l, &g
	}
	if items, _, err := c.Items(ctx); err == nil {
		n := 0
		for _, it := range items {
			if isPotion(it.Code) && it.Loc() != 1 && !strings.Contains(it.Attrs["panel"], "stash") {
				n++
			}
		}
		o.Potions = &n
	}
	cs := h.SelfCheckFloors(o, f)
	failed := hero.Failed(cs)
	if lf, err := os.OpenFile(ledger, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		b, _ := json.Marshal(cs)
		_ = json.NewEncoder(lf).Encode(ledgerLine{Time: time.Now().Format(time.RFC3339), Verb: "selfcheck", Reason: "READ-ONLY hero self-check", Reply: "failed: " + failed + " " + string(b)})
		lf.Close()
	}
	return failed
}

// spendPoints is the played profile's second half: after the level verbs the hero spends its points as the build table
// does and binds the main attack, with player commands (`cmd stat`, `cmd skillpt`, `cmd select`), each ledgered. A banked
// scenario, or a level 1 hero, sends nothing. A hero the library cannot make is an error (BLOCKED), never a banked one.
func spendPoints(ctx context.Context, c *d2dctl.Client, s scenario.Scenario, class string, su scenario.Setup, tabs hero.Tables, ledger string) error {
	if su.HeroProfile != scenario.HeroPlayed || s.Pre.Level < 2 {
		return nil
	}
	h, err := hero.Resolve(hero.Decl{Profile: hero.Played, Class: class, Build: s.Trigger.Build, Level: s.Pre.Level, Weapon: "none"}, tabs)
	if err != nil {
		return err
	}
	lf, err := os.OpenFile(ledger, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer lf.Close()
	enc := json.NewEncoder(lf)
	if _, err := c.Step(ctx, 5); err != nil { // a player command checks the level and the points: let the writes land first
		return err
	}
	for _, v := range h.Verbs {
		if v.Write {
			continue // the level, experience and point counts are the scenario's own verbs
		}
		l := ledgerLine{Time: time.Now().Format(time.RFC3339), Verb: v.Line, Reason: v.Reason}
		rep, err := c.Do(ctx, v.Line)
		if rep != nil {
			l.Reply = strings.Join(append([]string{rep.Rest}, rep.Lines...), " | ")
		}
		_ = enc.Encode(l)
		if err != nil {
			return fmt.Errorf("%s: %w", v.Line, err)
		}
		if _, err := c.Step(ctx, 2); err != nil {
			return err
		}
	}
	return nil
}

// heroTables are the game's tables for the played profile (skills.txt, charstats.txt); the validate command can do
// without them (hero.NameTables).
func heroTables(gamedata, cache string) (hero.Tables, error) {
	db, err := data.Open(gamedata, filepath.Join(cache, "tables"))
	if err != nil {
		return nil, fmt.Errorf("game tables: %w", err)
	}
	return hero.DBTables(db), nil
}
