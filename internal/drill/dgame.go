package drill

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// WalkInfo is one navigator walk (P22).
type WalkInfo struct {
	Tx, Ty, Fx, Fy int // target and final position, subtiles
	Cells          float64
	Ticks          int
	Err            string
}

// ExitInfo is one level transition by the navigator (P24).
type ExitInfo struct {
	From, To, Want int
	Ticks          int
	Err            string
}

// SpendInfo is what the leveler did with the points.
type SpendInfo struct {
	StatBefore, StatLeft, SkillBefore, SkillLeft int
	Done, Planned                                int
	Refused                                      []string
	Err                                          string
	Str, Dex, Vit, Ene                           int
	Skills                                       string // "id:level" of the learned skills
}

// GameLab is the instance as the D5 to D9 drills use it: combat's survival decision, the navigator, the leveler and
// the death and lifecycle calls of the bot.
type GameLab interface {
	TownLab
	Field(ctx context.Context, r *Run, level int) error                         // logged goto: a fresh arrival cell in the level
	Invincible(ctx context.Context, r *Run) error                               // stat cheats: a big max life so walking drills survive the field
	ArmBelt(ctx context.Context, r *Run) (string, error)                        // makes sure the belt holds healing and mana potions
	SetMana(ctx context.Context, r *Run, points int) error                      // cheat debug stat 8
	GiveSkills(ctx context.Context, r *Run) error                               // cheat debug points: Fire Bolt 1, Charged Bolt 1
	BindCount(ctx context.Context) (int, error)                                 // d2dcombat Rig.BindSkills
	CastMana(ctx context.Context, skill int) (int, int, error)                  // cast at a cell 10 subtiles away: mana before and after
	SafetyAt(ctx context.Context, r *Run, life, mana int) (string, error)       // sets life and mana, returns combat.Fighter.Safety's action
	ActDrink(ctx context.Context, action string) (before, after int, err error) // Fighter.Act: life before and after, belt potions -1
	Walks(ctx context.Context, n int) []WalkInfo
	Exits(ctx context.Context, levels []int) []ExitInfo
	LevelUp(ctx context.Context, r *Run, clvl int) error // cheats: level, stat points, skill points of a character of that level
	Spend(ctx context.Context) SpendInfo                 // leveler.Do on the snapshot
	EquipCap(ctx context.Context, r *Run) (worn bool, report string, err error)
	DieAndRespawn(ctx context.Context, r *Run) (string, error)
	TakeCorpse(ctx context.Context, r *Run) (string, error)
	WornCodes(ctx context.Context) (string, error)
	Cycles(ctx context.Context, n int) (string, error)
	// D8: merc cheats (debug merc hire|kill|revive) and the bot's own revive and belt drink
	MercCheat(ctx context.Context, r *Run, verb string) error
	MercState(ctx context.Context) (MercInfo, error)
	Revive(ctx context.Context) (paid int, err error) // d2dmerc.Reviver through Kashya's menu
	DrinkMerc(ctx context.Context, col int) error     // the bot's belt verb with the merc flag (controller Belt)
}

// MercInfo is "dump merc".
type MercInfo struct {
	Present, Dead bool
	Code          string
	Life, Max     int
}

func (m MercInfo) String() string {
	return fmt.Sprintf("present=%v dead=%v code=%s life=[%d,%d]", m.Present, m.Dead, m.Code, m.Life, m.Max)
}

// D5 returns the combat group (P26 bind and cast, P28 thresholds); P27 and P29 need monsters.
func D5(l GameLab) []Drill {
	lim := 10 * time.Minute
	do := func(f func(context.Context, *Run, GameLab) Outcome) func(context.Context, *Run) Outcome {
		return func(ctx context.Context, r *Run) Outcome { return f(ctx, r, l) }
	}
	const noMob = "a monster to fight: no debug verb spawns or places one (debug help lists none)"
	return []Drill{
		{ID: "D5.a", Primitive: "P26", Title: "skills learned by cheat: BindSkills counts them, a cast spends mana", Limit: lim, Do: do(d5a)},
		{ID: "D5.b", Primitive: "P28", Title: "life and mana set by cheat: the survival decision follows the knob thresholds, a drink raises life and spends a potion", Limit: lim, Do: do(d5b)},
		{ID: "D5.c", Primitive: "P27", Title: "three mobs, one behind a wall: never the walled one", Needs: []string{noMob}},
		{ID: "D5.d", Primitive: "P29", Title: "pack of 6 at clvl 5: retreat then town portal", Needs: []string{noMob}},
	}
}

func d5a(ctx context.Context, r *Run, l GameLab) Outcome {
	if err := l.GiveSkills(ctx, r); err != nil {
		return setupFail(err)
	}
	if err := l.Field(ctx, r, 2); err != nil {
		return setupFail(err)
	}
	n, err := l.BindCount(ctx)
	if err != nil {
		return setupFail(fmt.Errorf("BindSkills: %w", err))
	}
	m0, m1, cerr := l.CastMana(ctx, 36)
	obs := fmt.Sprintf("BindSkills bound %d skills; Fire Bolt cast: mana %d -> %d err %v", n, m0, m1, cerr)
	o := Outcome{Intended: "2 skills bound (Fire Bolt, Charged Bolt), the cast lowers mana", Observed: obs, Sig: fmt.Sprintf("bound=%d spent=%v err=%v", n, m1 < m0, cerr != nil)}
	switch {
	case n != 2:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("BindSkills bound %d, two skills were learned", n)
	case cerr != nil || m1 >= m0:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("the cast did not spend mana (err %v)", cerr)
	default:
		o.Verdict, o.Reason = Pass, fmt.Sprintf("bound %d, a cast cost %d mana (no target: hit effect not checked)", n, m0-m1)
	}
	return o
}

func d5b(ctx context.Context, r *Run, l GameLab) Outcome {
	if err := l.Field(ctx, r, 2); err != nil {
		return setupFail(err)
	}
	arm, err := l.ArmBelt(ctx, r)
	if err != nil {
		return setupFail(err)
	}
	// knobs (internal/env/tuning_combat.go): drink life below 55 percent, chicken at 25 (in a fight; out of one a potion is
	// drunk instead, combat.Decide); max life 40 on a fresh Sorceress
	type step struct {
		life, mana int
		want       string
	}
	steps := []step{{40, 35, "none"}, {24, 35, "none"}, {20, 35, "drink-life"}, {14, 35, "drink-life"}, {9, 35, "drink-life|chicken"}, {40, 7, "none"}} // mana: resource.PotionPolicy drinks mana only in a fight (Hostiles > 0); no monster here, so none is right and the in-fight half is BLOCKED
	var rows, sig []string
	bad := 0
	for _, s := range steps {
		got, err := l.SafetyAt(ctx, r, s.life, s.mana)
		if err != nil {
			return setupFail(err)
		}
		ok := got == s.want || (strings.Contains(s.want, "|") && strings.Contains("|"+s.want+"|", "|"+got+"|"))
		if !ok {
			bad++
		}
		rows = append(rows, fmt.Sprintf("life %d mana %d -> %s (want %s)", s.life, s.mana, got, s.want))
		sig = append(sig, got)
	}
	o := Outcome{Intended: "none above 55 percent life, drink-life below it (a drink is drunk, not a chicken, out of a fight), no mana potion without a fight", Observed: arm + "; " + strings.Join(rows, "; "),
		Sig: strings.Join(sig, ",")}
	switch {
	case bad > 0:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("%d of %d decisions differ from the knob thresholds", bad, len(steps))
	default:
		o.Verdict, o.Reason = Pass, "life decisions follow the thresholds; BLOCKED here: the mana potion in a fight and the Meter damage totals (no monster, no damage source); the drink itself is D1.P3"
	}
	return o
}

// D6 returns the movement group: P22 walks, P24 exits, P25 (needs a pocket).
func D6(l GameLab) []Drill {
	lim := 20 * time.Minute
	do := func(f func(context.Context, *Run, GameLab) Outcome) func(context.Context, *Run) Outcome {
		return func(ctx context.Context, r *Run) Outcome { return f(ctx, r, l) }
	}
	return []Drill{
		{ID: "D6.a", Primitive: "P22", Title: "10 navigator walks in the Rogue Encampment: arrive within 1.5 cells, 9 of 10", Limit: lim, Do: do(func(ctx context.Context, r *Run, l GameLab) Outcome { return d6walk(ctx, r, l, 1) })},
		{ID: "D6.b", Primitive: "P22", Title: "10 navigator walks in Blood Moor: arrive within 1.5 cells, 9 of 10", Limit: lim, Do: do(func(ctx context.Context, r *Run, l GameLab) Outcome { return d6walk(ctx, r, l, 2) })},
		{ID: "D6.c", Primitive: "P24", Title: "5 level transitions by ToArea (Blood Moor, Cold Plains, camp): the area id changes each time", Limit: lim, Do: do(d6exit)},
		d6pin(l, lim),
	}
}

func d6walk(ctx context.Context, r *Run, l GameLab, level int) Outcome {
	if err := l.Field(ctx, r, level); err != nil {
		return setupFail(err)
	}
	if err := l.Invincible(ctx, r); err != nil {
		return setupFail(err)
	}
	ws := l.Walks(ctx, 10)
	var rows, sig []string
	ok := 0
	for i, w := range ws {
		good := w.Err == "" && w.Cells <= 1.5
		if good {
			ok++
		}
		s := fmt.Sprintf("%d:(%d,%d)->(%d,%d) %.1f cells %d ticks", i+1, w.Tx, w.Ty, w.Fx, w.Fy, w.Cells, w.Ticks)
		if w.Err != "" {
			s += " err " + w.Err
		}
		rows = append(rows, s)
		sig = append(sig, fmt.Sprint(good))
	}
	o := Outcome{Intended: "9 of 10 walks end within 1.5 cells of the target", Observed: fmt.Sprintf("%d of %d arrived: %s", ok, len(ws), strings.Join(rows, " | ")), Sig: strings.Join(sig, "")}
	switch {
	case len(ws) < 10:
		o.Verdict, o.Reason = Blocked, fmt.Sprintf("only %d reachable targets found by PathTo around the arrival cell", len(ws))
	case ok >= 9:
		o.Verdict, o.Reason = Pass, fmt.Sprintf("%d of 10 walks arrived", ok)
	default:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("only %d of 10 walks arrived within 1.5 cells", ok)
	}
	return o
}

func d6exit(ctx context.Context, r *Run, l GameLab) Outcome {
	if err := l.Field(ctx, r, 2); err != nil {
		return setupFail(err)
	}
	if err := l.Invincible(ctx, r); err != nil {
		return setupFail(err)
	}
	es := l.Exits(ctx, []int{3, 2, 1, 2, 3})
	var rows, sig []string
	ok := 0
	for _, e := range es {
		good := e.Err == "" && e.To == e.Want && e.To != e.From
		if good {
			ok++
		}
		s := fmt.Sprintf("%d->%d (wanted %d) %d ticks", e.From, e.To, e.Want, e.Ticks)
		if e.Err != "" {
			s += " err " + e.Err
		}
		rows = append(rows, s)
		sig = append(sig, fmt.Sprint(good))
	}
	o := Outcome{Intended: "5 of 5 transitions change the area id to the wanted one", Observed: fmt.Sprintf("%d of %d: %s", ok, len(es), strings.Join(rows, " | ")), Sig: strings.Join(sig, "")}
	if ok == 5 {
		o.Verdict, o.Reason = Pass, "5 of 5 level transitions"
	} else {
		o.Verdict, o.Reason = Fail, fmt.Sprintf("%d of 5 transitions", ok)
	}
	return o
}

// D7 returns the build group: P30 stat points, P31 skill points, P32 equip.
func D7(l GameLab) []Drill {
	lim := 10 * time.Minute
	do := func(f func(context.Context, *Run, GameLab) Outcome) func(context.Context, *Run) Outcome {
		return func(ctx context.Context, r *Run) Outcome { return f(ctx, r, l) }
	}
	return []Drill{
		{ID: "D7.a", Primitive: "P30", Title: "level 12 by cheat with 55 stat points: the leveler spends them all", Limit: lim, Do: do(func(ctx context.Context, r *Run, l GameLab) Outcome { return d7spend(ctx, r, l, "P30") })},
		{ID: "D7.b", Primitive: "P31", Title: "level 12 by cheat with 11 skill points: the leveler spends them all per the build order", Limit: lim, Do: do(func(ctx context.Context, r *Run, l GameLab) Outcome { return d7spend(ctx, r, l, "P31") })},
		{ID: "D7.c", Primitive: "P32", Title: "a cap in the pack, body slot empty: the leveler wears it, cursor empty", Limit: lim, Do: do(d7equip)},
	}
}

func d7spend(ctx context.Context, r *Run, l GameLab, prim string) Outcome {
	if err := l.LevelUp(ctx, r, 12); err != nil {
		return setupFail(err)
	}
	si := l.Spend(ctx)
	obs := fmt.Sprintf("stat points %d -> %d, skill points %d -> %d, planned %d done %d refused %v err %q; str %d dex %d vit %d ene %d; skills %s", si.StatBefore, si.StatLeft, si.SkillBefore, si.SkillLeft, si.Planned, si.Done, si.Refused, si.Err, si.Str, si.Dex, si.Vit, si.Ene, si.Skills)
	o := Outcome{Intended: "no unspent points after the leveler ran", Observed: obs, Sig: fmt.Sprintf("stat %d->%d skill %d->%d err=%v", si.StatBefore, si.StatLeft, si.SkillBefore, si.SkillLeft, si.Err != "")}
	left := si.StatLeft
	if prim == "P31" {
		left = si.SkillLeft
	}
	switch {
	case si.Err != "":
		o.Verdict, o.Reason = Fail, "leveler error: "+si.Err
	case (prim == "P30" && si.StatBefore == 0) || (prim == "P31" && si.SkillBefore == 0):
		o.Verdict, o.Reason = Blocked, "the points cheat did not give points to spend"
	case left != 0:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("%d points left unspent", left)
	default:
		o.Verdict, o.Reason = Pass, "all points spent"
	}
	return o
}

func d7equip(ctx context.Context, r *Run, l GameLab) Outcome {
	worn, rep, err := l.EquipCap(ctx, r)
	if err != nil {
		return setupFail(err)
	}
	codes, _ := l.WornCodes(ctx)
	o := Outcome{Intended: "the cap is worn, the cursor empty (P35 gate), old items kept", Observed: fmt.Sprintf("worn after: %s; %s", codes, rep), Sig: fmt.Sprintf("worn=%v", worn)}
	if worn {
		o.Verdict, o.Reason = Pass, "equipped"
	} else {
		o.Verdict, o.Reason = Fail, "the cap was not worn"
	}
	return o
}

// D8 returns the NPC group: P18 revive and merc drink with the merc cheats; claim, quest reward, act travel and gamble
// are blocked on named pieces.
func D8(l GameLab) []Drill {
	lim := 10 * time.Minute
	do := func(f func(context.Context, *Run, GameLab) Outcome) func(context.Context, *Run) Outcome {
		return func(ctx context.Context, r *Run) Outcome { return f(ctx, r, l) }
	}
	ds := append([]Drill{d8quest(l, lim)}, D8h(l, lim)...) // D8.c first: a merc owned from an earlier drill forfeits the free Rogue (DRILLS8 F14); D8.h hires, so before D8.a only on a merc-free character
	return append(ds, []Drill{
		{ID: "D8.a", Primitive: "P18", Title: "merc hired and killed by cheat: the bot's revive through Kashya brings it back for gold", Limit: lim, Do: do(d8revive)},
		{ID: "D8.m", Primitive: "P3", Title: "merc at low life: the bot's belt verb with the merc flag heals it and spends one potion", Limit: lim, Do: do(d8drink)},
		d8akara(l, lim),
		d8warriv(l, lim),
		{ID: "D8.g", Primitive: "P19", Title: "gamble never opens the tab when poor", Needs: []string{"the bot has no gamble code (docs/primitives.md P19: class listed only); the drill is a log grep on a soak, not a scripted scenario"}},
	}...)
}

func d8revive(ctx context.Context, r *Run, l GameLab) Outcome {
	if err := l.MercCheat(ctx, r, "debug merc hire 0"); err != nil {
		return setupFail(err)
	}
	if err := l.Tick(ctx, 5); err != nil {
		return setupFail(err)
	}
	if err := l.MercCheat(ctx, r, "debug merc kill"); err != nil {
		return setupFail(err)
	}
	if err := l.Tick(ctx, 140); err != nil {
		return setupFail(err)
	}
	m0, _ := l.MercState(ctx)
	if !m0.Dead {
		return setupFail(fmt.Errorf("the merc is not dead after the kill cheat: %s", m0))
	}
	if err := l.SetGold(ctx, r, 3000); err != nil {
		return setupFail(err)
	}
	g0, _ := l.TView(ctx)
	paid, rerr := l.Revive(ctx)
	_ = l.Tick(ctx, 60)
	m1, _ := l.MercState(ctx)
	g1, _ := l.TView(ctx)
	obs := fmt.Sprintf("merc %s -> %s; gold %d -> %d (revive reports %d); err %v; panels %s", m0, m1, g0.Gold, g1.Gold, paid, rerr, panelsOf(g1.Panels))
	o := Outcome{Intended: "merc alive again, gold down by the reported price", Observed: obs, Sig: fmt.Sprintf("dead=%v->%v paid>0=%v err=%v", m0.Dead, m1.Dead, paid > 0, rerr != nil)}
	switch {
	case rerr != nil:
		o.Verdict, o.Reason = Fail, "revive: "+rerr.Error()
	case !m1.Present || m1.Dead:
		o.Verdict, o.Reason = Fail, "the merc is still dead after the revive"
	case g0.Gold-g1.Gold != paid:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("gold delta %d differs from the reported price %d", g0.Gold-g1.Gold, paid)
	default:
		o.Verdict, o.Reason = Pass, fmt.Sprintf("revived through Kashya's menu; price taken %d gold (a Rogue at clvl 1 is revived free on this build: U whether the price is a rule or a gap)", paid)
	}
	return o
}

func d8drink(ctx context.Context, r *Run, l GameLab) Outcome {
	if err := l.MercCheat(ctx, r, "debug merc hire 0"); err != nil {
		return setupFail(err)
	}
	if err := l.Tick(ctx, 10); err != nil {
		return setupFail(err)
	}
	m0, _ := l.MercState(ctx)
	if bg, ok := l.(beltGiver); ok { // a fresh healing potion in column 0 (a repeat finds it drunk)
		if _, err := r.Cheat(ctx, "debug belt clear", "empty the belt so column 0 holds exactly the healing potion the drill gives (a repeat drinks the same one)"); err != nil {
			return setupFail(err)
		}
		if err := bg.GiveBelt(ctx, r, "hp1", 0); err != nil {
			return setupFail(err)
		}
	}
	if err := l.MercCheat(ctx, r, "debug merc hurt 20"); err != nil {
		return setupFail(err)
	}
	if err := l.Tick(ctx, 10); err != nil {
		return setupFail(err)
	}
	m0, _ = l.MercState(ctx)
	b0, _ := l.TView(ctx)
	derr := l.DrinkMerc(ctx, 0)
	if err := l.Tick(ctx, 80); err != nil {
		return setupFail(err)
	}
	m1, _ := l.MercState(ctx)
	b1, _ := l.TView(ctx)
	obs := fmt.Sprintf("merc %s -> %s; belt potions %d -> %d; err %v", m0, m1, b0.Belt, b1.Belt, derr)
	o := Outcome{Intended: "merc life up, one belt potion used", Observed: obs, Sig: fmt.Sprintf("up=%v err=%v", m1.Life > m0.Life, derr != nil)}
	switch {
	case derr != nil:
		o.Verdict, o.Reason = Fail, "merc drink: "+derr.Error()
	case m0.Life >= 40:
		o = Blocks(fmt.Sprintf("`debug merc hurt 20` left the merc at %s", m0))
		o.Observed, o.Intended = obs, "merc life up, one belt potion used"
	case m1.Life <= m0.Life:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("the belt verb with the merc flag did not raise the merc's life (%d -> %d)", m0.Life, m1.Life)
	case b0.Belt-b1.Belt != 1:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("merc life rose but %d belt potions were used (wanted 1)", b0.Belt-b1.Belt)
	default:
		o.Verdict, o.Reason = Pass, fmt.Sprintf("merc life %d -> %d with one belt potion", m0.Life, m1.Life)
	}
	return o
}

// D9 returns the lifecycle group: P34 death and corpse, P33 leave and enter cycles; P36 is the paired soak.
func D9(l GameLab) []Drill {
	lim := 25 * time.Minute
	do := func(f func(context.Context, *Run, GameLab) Outcome) func(context.Context, *Run) Outcome {
		return func(ctx context.Context, r *Run) Outcome { return f(ctx, r, l) }
	}
	return []Drill{
		{ID: "D9.a", Primitive: "P34", Title: "die in the field, respawn in town (bot's Respawn), walk back, take the corpse: the worn staff returns", Limit: lim, Do: do(d9death)},
		{ID: "D9.b", Primitive: "P33", Title: "10 leave and enter cycles by the bot's lifecycle: in town, UI clean each time", Limit: lim, Do: do(d9cycle)},
		{ID: "D9.c", Primitive: "P36", Title: "paired 8-seed soak, trips and empty-belt minutes", Needs: []string{"a soak, not a drill: tools/d2dsoak runs it on the soak hosts (docs/d2d-soak.md); no drill harness does 8 seeds of hours"}},
	}
}

func d9death(ctx context.Context, r *Run, l GameLab) Outcome {
	w0, _ := l.WornCodes(ctx)
	if err := l.Field(ctx, r, 2); err != nil {
		return setupFail(err)
	}
	died, err := l.DieAndRespawn(ctx, r)
	if err != nil {
		return Failed("die and respawn: "+err.Error(), "alive in town after Respawn", died)
	}
	w1, _ := l.WornCodes(ctx)
	got, cerr := l.TakeCorpse(ctx, r)
	w2, _ := l.WornCodes(ctx)
	obs := fmt.Sprintf("worn before %s; %s; worn after respawn %s; corpse: %s; worn after the corpse %s", w0, died, w1, got, w2)
	o := Outcome{Intended: "worn items gone at the respawn, back after the corpse interact", Observed: obs, Sig: fmt.Sprintf("%s|%s|%s", w0, w1, w2)}
	switch {
	case cerr != nil:
		o.Verdict, o.Reason = Fail, "taking the corpse: "+cerr.Error()
	case w1 == w0:
		o.Verdict, o.Reason = Fail, "the worn items were not lost at death"
	case w2 != w0:
		o.Verdict, o.Reason = Fail, "the worn items did not come back"
	default:
		o.Verdict, o.Reason = Pass, "gear lost at death and recovered from the corpse"
	}
	return o
}

// stripSecs removes the per-cycle wall seconds ("3s") so the determinism check compares the end states.
func stripSecs(s string) string {
	var out []string
	for _, p := range strings.Split(s, "; ") {
		if i := strings.LastIndex(p, ", "); i > 0 && strings.HasSuffix(p, "s") {
			p = p[:i]
		}
		out = append(out, p)
	}
	return strings.Join(out, "; ")
}

func d9cycle(ctx context.Context, r *Run, l GameLab) Outcome {
	s, err := l.Cycles(ctx, 10)
	o := Outcome{Intended: "10 cycles, each ends in town with a clean UI", Observed: s, Sig: stripSecs(s)}
	if err != nil {
		o.Verdict, o.Reason = Fail, err.Error()
		return o
	}
	o.Verdict, o.Reason = Pass, "10 leave and enter cycles"
	return o
}

// QuestLab is the optional part of the lab that drives the quest records (`debug quest set`) through the bot's NPC code.
type QuestLab interface {
	// Claim sets Blood Raven's record to "reward pending" (0x0002) and runs d2dtown.ClaimRogue with the bot's own Pause.
	Claim(ctx context.Context, r *Run) (granted bool, merc string, before, after int, err error)
	// AkaraReward sets the Den of Evil record to "reward pending" and runs the bot's talk path (internal/talk, pick "reward"): skill points and record after.
	AkaraReward(ctx context.Context, r *Run) (skBefore, skAfter int, rec string, err error)
	// Warriv sets Andariel's record done (`debug quest set 6 0x0001`), runs the bot's talk path (internal/talk, pick "go east"): area after (the menu string is the row path).
	Warriv(ctx context.Context, r *Run) (area int, menu string, err error)
}

// MercOwnedBefore prefixes the merc string QuestLab.Claim returns when the character had a merc before the claim.
const MercOwnedBefore = "owned before: "

func d8quest(l GameLab, lim time.Duration) Drill {
	ql, ok := l.(QuestLab)
	if !ok {
		return Drill{ID: "D8.c", Primitive: "P18", Title: "Kashya's free Rogue after Blood Raven", Needs: []string{"the lab has no quest step"}}
	}
	return Drill{ID: "D8.c", Primitive: "P18", Title: "Kashya's free Rogue after Blood Raven (record by `debug quest set`, the bot's ClaimRogue)", Limit: lim, Do: func(ctx context.Context, r *Run) Outcome {
		g, merc, b, a, err := ql.Claim(ctx, r)
		intended := "a fresh character with the Blood Raven reward pending: ClaimRogue opens Kashya's menu and a Rogue appears, the dialog is closed"
		obs := fmt.Sprintf("granted=%v merc=%q record states %d -> %d err %v", g, merc, b, a, err)
		o := Outcome{Intended: intended, Observed: obs, Sig: fmt.Sprintf("granted=%v err=%v", g, err != nil)}
		switch {
		case err != nil:
			o.Verdict, o.Reason = Fail, "claim: "+err.Error()
		case strings.HasPrefix(merc, MercOwnedBefore):
			o.Verdict, o.Reason = Blocked, "the character already owns a merc ("+strings.TrimPrefix(merc, MercOwnedBefore)+"): the free Rogue is claimable once per character, so this drill needs a character that never hired one"
		case !g:
			o.Verdict, o.Reason = Fail, "no merc appeared after the claim (a merc already owned forfeits the reward: run D8.c before D8.a)"
		default:
			o.Verdict, o.Reason = Pass, "the free Rogue was granted by the claim ("+merc+")"
		}
		return o
	}}
}

func d8akara(l GameLab, lim time.Duration) Drill {
	ql, ok := l.(QuestLab)
	if !ok {
		return Drill{ID: "D8.q", Primitive: "P20", Title: "Akara's Den of Evil reward", Needs: []string{"the lab has no quest step"}}
	}
	return Drill{ID: "D8.q", Primitive: "P20", Title: "Akara's Den of Evil reward (record by `debug quest set`, the bot's talk path internal/talk): one skill point, dialog closed", Limit: lim, Do: func(ctx context.Context, r *Run) Outcome {
		b, a, rec, err := ql.AkaraReward(ctx, r)
		intended := "Den of Evil reward pending: opening Akara's menu grants one skill point, the record reads done, the dialog is closed"
		obs := fmt.Sprintf("skill points %d -> %d; quest list: %s; err %v", b, a, rec, err)
		o := Outcome{Intended: intended, Observed: obs, Sig: fmt.Sprintf("gain=%d err=%v", a-b, err != nil)}
		switch {
		case err != nil:
			o.Verdict, o.Reason = Fail, "akara: "+err.Error()
		case a != b+1:
			o.Verdict, o.Reason = Fail, fmt.Sprintf("skill points %d -> %d, wanted +1", b, a)
		default:
			o.Verdict, o.Reason = Pass, "the bot's own talk path (internal/talk, the code cmd/d2bot runs) claimed the reward: one skill point (built by fixtalk, awaiting live proof: a Pass here is the first)"
		}
		return o
	}}
}

// PinInfo is one walk out of a pocket made by `debug pin on`.
type PinInfo struct {
	Err           string
	Moved         float64 // cells between the start and the end position
	Ticks         int
	Area          int // area after the walk (1: the ladder ended in town)
	Rungs         []string
	Pinned, Freed bool // the pocket was built / the cheat took it down again
}

// PinLab is the optional part of the lab for the stuck drill.
type PinLab interface {
	Pin(ctx context.Context, r *Run, level int, within time.Duration) PinInfo
}

func d6pin(l GameLab, lim time.Duration) Drill {
	pl, ok := l.(PinLab)
	if !ok {
		return Drill{ID: "D6.d", Primitive: "P25", Title: "pinned in a pocket: out within 60 s", Needs: []string{"the lab has no pin step"}}
	}
	return Drill{ID: "D6.d", Primitive: "P25", Title: "pinned in a pocket by `debug pin on`: the bot's stall ladder gets the character out within 60 s", Limit: lim, Do: func(ctx context.Context, r *Run) Outcome {
		pi := pl.Pin(ctx, r, 2, 60*time.Second)
		intended := "a 12-cell walk from inside a collision ring: the navigator's stall ladder ends with the character out of the pocket (town portal or waypoint) within 60 s of game time"
		obs := fmt.Sprintf("pinned=%v freed=%v moved %.1f cells in %d ticks (%.0f s), area %d, err %q; ladder lines: %s", pi.Pinned, pi.Freed, pi.Moved, pi.Ticks, float64(pi.Ticks)*0.04, pi.Area, pi.Err, strings.Join(firstN(pi.Rungs, 8), " / "))
		// fixstall: a pinned GoTo now ends with ErrLevelChanged after the town portal escape (area 1)
		escaped := strings.Contains(pi.Err, "level changed")
		o := Outcome{Intended: intended, Observed: obs, Sig: fmt.Sprintf("out=%v", pi.Moved > 6 || pi.Area == 1 || escaped)}
		switch {
		case !pi.Pinned:
			o.Verdict, o.Reason = Blocked, "the pin cheat did not build the pocket: "+pi.Err
		case pi.Moved > 6 || pi.Area == 1 || escaped:
			o.Verdict, o.Reason = Pass, fmt.Sprintf("out of the pocket after %.0f s of game time (area %d)", float64(pi.Ticks)*0.04, pi.Area)
		default:
			o.Verdict, o.Reason = Fail, fmt.Sprintf("still in the pocket after %.0f s of game time: the stall ladder did not leave it (err %q)", float64(pi.Ticks)*0.04, pi.Err)
		}
		return o
	}}
}

func d8warriv(l GameLab, lim time.Duration) Drill {
	ql, ok := l.(QuestLab)
	if !ok {
		return Drill{ID: "D8.w", Primitive: "P21", Title: "Warriv takes the character to Act 2", Needs: []string{"the lab has no quest step"}}
	}
	return Drill{ID: "D8.w", Primitive: "P21", Title: "Warriv to Act 2 (Andariel record by `debug quest set`, the bot's talk path internal/talk): the menu row \"Go East\" moves the character to Lut Gholein", Limit: lim, Do: func(ctx context.Context, r *Run) Outcome {
		area, menu, err := ql.Warriv(ctx, r)
		intended := "Andariel done: Warriv's menu lists Go East, picking it arrives in Lut Gholein (area 40)"
		obs := fmt.Sprintf("menu [%s]; area after %d; err %v", menu, area, err)
		o := Outcome{Intended: intended, Observed: obs, Sig: fmt.Sprintf("area=%d err=%v", area, err != nil)}
		switch {
		case err != nil:
			o.Verdict, o.Reason = Fail, "warriv: "+err.Error()
		case area != 40:
			o.Verdict, o.Reason = Fail, fmt.Sprintf("arrived in area %d, wanted 40", area)
		default:
			o.Verdict, o.Reason = Pass, "the bot's own act hop talk (internal/talk, the code cmd/d2bot runs) took the character to Lut Gholein (built by fixtalk, awaiting live proof: a Pass here is the first)"
		}
		return o
	}}
}
