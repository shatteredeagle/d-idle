package leveler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town/hand"
)

// Options configure a Leveler. The zero value is usable.
type Options struct {
	PlanDir string // overrides the embedded plan data per file
	// Build selects the table: "fallback" (Lightning, the Sorceress default; needs no respec), "primary" (Blizzard; its
	// rows from the respec level on assume a respec the bot cannot do yet, docs/leveler.md "Respec") or
	// "paladin-normal30". Empty: the default of the character's class (DefaultBuildName). A table for another class than
	// the character's is refused (ErrWrongClass).
	Build string
	// Pause is called when the leveler hit something that must stop the bot (a skill-panel read-back that does not
	// match the plan, no build for the class). The bot should pause; Spend also returns the error. Optional.
	Pause func(reason string)
	// Busy reports a fight in progress; points are never spent then. Nil: nothing hostile within 25 subtiles.
	Busy func(*d2.Snapshot) bool
	// Rebind runs after new skills were learned (the fighter binds new attack skills to F keys). Optional.
	Rebind    func(ctx context.Context) error
	DebugHold int // ms to wait before undoing a failed equip, for screenshots
	// Dry plans and logs but never clicks.
	Dry bool
	// Snapshot replaces env.Snapshot for the hand (tests with the simulated town).
	Snapshot func() (*d2.Snapshot, error)
}

func (l *Leveler) newHand() *hand.Hand {
	h := hand.New(l.e)
	if l.o.Snapshot != nil {
		h.Snap = l.o.Snapshot
	}
	return h
}

// Leveler spends points and manages gear. One per game instance; methods are not safe for concurrent use except
// where noted (the bot loop calls it from one goroutine).
type Leveler struct {
	e      *env.Env
	o      Options
	builds map[string]*Build // loaded tables by selector
	geo    Geometry
	// player reads the player fresh (tests replace it with a fake skill panel).
	player func() (*d2.Player, error)

	mu       sync.Mutex
	rejected map[string]int     // skill -> character level at which the game refused a point
	bad      map[badKey]badMark // (item id, slot) -> its failed equips; slot -1 is a failed purchase
	badSlot  map[int]badMark    // body location -> failed equips of different items into it
	stats    Stats
	need     [numAttr]int // gear requirements reported by NeedStats
}

// Stats counts what the leveler did (for the log and docs).
type Stats struct {
	StatPoints, SkillPoints int
	Refused, Failures       int
	EquipMoves, Purchases   int
}

// New builds a Leveler. Without a loadable build it falls back to no plan (Do reports the error).
func New(e *env.Env, o Options) *Leveler {
	l := &Leveler{e: e, o: o, geo: DefaultGeometry(), rejected: map[string]int{}, builds: map[string]*Build{}}
	l.player = func() (*d2.Player, error) { return e.Read.Player() }
	if o.Build != "" {
		if _, err := l.loadBuild(o.Build); err != nil {
			e.Log.Error("leveler: build not loaded", "err", err)
		}
	}
	return l
}

var (
	// ErrWrongClass: the selected build table is for another class than the character's.
	ErrWrongClass = errors.New("leveler: build is for another class")
	// ErrNoBuild: the class has no default build and none was configured.
	ErrNoBuild = errors.New("leveler: no build for this class")
	// ErrSkillMismatch: the skill panel learned a different skill than the plan asked for (or none). Fatal for the run.
	ErrSkillMismatch = errors.New("leveler: skill panel read-back does not match the plan")
)

func (l *Leveler) loadBuild(name string) (*Build, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if b := l.builds[name]; b != nil {
		return b, nil
	}
	b, err := LoadBuild(name, l.o.PlanDir)
	if err != nil {
		return nil, err
	}
	l.builds[name] = b
	return b, nil
}

// buildFor picks the table for the character: the configured one, else the class default. It never guesses: an
// unknown class or a table for another class is an error (CLASSPAL: the Sorceress table on a Paladin learned
// Smite, Thorns and Resist Fire by icon position).
func (l *Leveler) buildFor(p *d2.Player) (*Build, error) {
	class := strings.ToLower(p.Class.String())
	name := ResolveBuild(l.o.Build, class)
	if name == "" {
		if name = SelectBuild(class, p.Level, OwnedSkills(l.e.Data, p)); name == "" {
			return nil, fmt.Errorf("%w: %s", ErrNoBuild, p.Class)
		}
	}
	b, err := l.loadBuild(name)
	if err != nil {
		return nil, err
	}
	if b.Class != class {
		return nil, fmt.Errorf("%w: build %q is for %s, character is a %s", ErrWrongClass, b.Name, b.Class, class)
	}
	return b, nil
}

func (l *Leveler) pause(reason string) {
	l.e.Log.Error("leveler: "+reason, "pause", l.o.Pause != nil)
	if l.o.Pause != nil {
		l.o.Pause(reason)
	}
}

// Stats returns the counters.
func (l *Leveler) Stats() Stats {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.stats
}

// Report is what one Do call did.
type Report struct {
	Skipped                string // why nothing was attempted ("" when it ran)
	Planned                []Action
	StatBefore, StatLeft   int // unspent stat points before and after
	SkillBefore, SkillLeft int
	Done                   []Action
	Refused                []string // skills the game would not take
	Err                    error
}

var (
	ErrBusy  = errors.New("leveler: a fight is on")
	errPanel = errors.New("leveler: panel did not open")
)

func (l *Leveler) logf(msg string, kv ...any) { l.e.Log.Info("leveler: "+msg, kv...) }

func (l *Leveler) sleep(ms int) { l.e.Clock.Sleep(time.Duration(ms) * time.Millisecond) }

// Spend is the bot.Leveler contract: errors only for real failures.
func (l *Leveler) Spend(ctx context.Context, snap *d2.Snapshot) error {
	r := l.Do(ctx, snap)
	if errors.Is(r.Err, ErrBusy) {
		return nil
	}
	return r.Err
}

// have builds the planner input from a snapshot and memory (base attributes, so item bonuses do not count as spent points).
func (l *Leveler) have(p *d2.Player) Have {
	h := Have{Level: p.Level, StatPts: p.StatPoints, SkillPts: p.SkillPoints, Skills: map[string]int{}}
	h.Attr = [numAttr]int{p.Strength, p.Dexterity, p.Vitality, p.Energy}
	if a, ok := l.BaseAttrs(p); ok {
		h.Attr = a
	}
	h.Floor = l.floor(p)
	h.Start = [numAttr]int{p.Strength, p.Dexterity, p.Vitality, p.Energy}
	if db := l.e.Data; db != nil {
		if c := db.ClassByName(p.Class.String()); c != nil {
			h.Start = [numAttr]int{c.Str, c.Dex, c.Vit, c.Int}
		}
		for _, s := range p.Skills {
			if sk := db.SkillByID(s.ID); sk != nil {
				h.Skills[sk.Key] = s.Level
			}
		}
	}
	return h
}

// classFloor is the base attribute minimum per class that lets the starter gear be worn (Str 12 = Quilted Armor,
// the cheapest body armor; V: charstats.txt Sorceress Str 10, G1 attempt 7 rejected every body armor 'needs Str').
// Every other class starts above what its starter kit needs (U: not checked per class).
var classFloor = map[string][numAttr]int{
	"sorceress": {Str: 12},
}

// maxNeed caps a reported requirement so a mistaken report cannot drain the table.
var maxNeed = [numAttr]int{Str: 40, Dex: 40}

func (l *Leveler) floor(p *d2.Player) [numAttr]int {
	f := classFloor[strings.ToLower(p.Class.String())]
	l.mu.Lock()
	defer l.mu.Unlock()
	for a := range f {
		f[a] = max(f[a], l.need[a])
	}
	return f
}

// NeedStats records the Str and Dex the cheapest affordable gear in a blocked slot needs (the town rebuy reports
// them); the next unspent points go there first. Zero means no change.
func (l *Leveler) NeedStats(str, dex int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for a, v := range map[int]int{Str: str, Dex: dex} {
		if v = min(v, maxNeed[a]); v > l.need[a] {
			l.need[a] = v
			l.logf("stat floor raised for gear", "attr", attrNames[a], "floor", v)
		}
	}
}

// BaseAttrs reads the base (item-free) attribute values from the player's base stat vector.
func (l *Leveler) BaseAttrs(p *d2.Player) ([numAttr]int, bool) {
	var out [numAttr]int
	if l.e.Read == nil || l.e.Mem == nil {
		return out, false
	}
	sl, err := mem32(l.e.Mem, p.Addr+0x5C)
	if err != nil {
		return out, false
	}
	st, err := l.e.Read.ReadBaseStatList(sl)
	if err != nil || !st.Has(d2.StatStrength) {
		return out, false
	}
	return [numAttr]int{int(st.Get(d2.StatStrength)), int(st.Get(d2.StatDexterity)), int(st.Get(d2.StatVitality)), int(st.Get(d2.StatEnergy))}, true
}

// classCode returns the skills.txt class code ("pal") of a character class, "" when unknown.
func (l *Leveler) classCode(c d2.Class) string {
	if l.e.Data != nil {
		if cc := l.e.Data.ClassByName(c.String()); cc != nil {
			return cc.Code
		}
	}
	return ""
}

// ruleFor is the skills.txt prerequisite check for a class. Unknown skills and skills of another class are refused;
// without tables everything is allowed.
func (l *Leveler) ruleFor(code string) Rule {
	return func(name string, level int, lv map[string]int) bool { return l.rule(code, name, level, lv) }
}

func (l *Leveler) rule(code, name string, level int, lv map[string]int) bool {
	db := l.e.Data
	if db == nil {
		return true
	}
	s := db.SkillByName(name)
	if s == nil || s.Page == 0 {
		return false
	}
	if s.Class != "" && code != "" && s.Class != code {
		return false
	}
	if s.ReqLevel > level || (s.MaxLevel > 0 && lv[name] >= s.MaxLevel) {
		return false
	}
	for _, rq := range s.ReqSkills {
		if rq != "" && lv[rq] == 0 {
			return false
		}
	}
	return true
}

func (l *Leveler) skipSet(level int) map[string]bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	m := map[string]bool{}
	for k, lv := range l.rejected {
		if lv == level {
			m[k] = true
		}
	}
	return m
}

// PlanFor returns what Do would spend for a snapshot (no clicks).
func (l *Leveler) PlanFor(snap *d2.Snapshot) []Action {
	if snap == nil || snap.Player == nil {
		return nil
	}
	p := snap.Player
	b, err := l.buildFor(p)
	if err != nil {
		return nil
	}
	return b.Plan(l.have(p), l.ruleFor(l.classCode(p.Class)), l.skipSet(p.Level))
}

// Pending reports whether the snapshot shows unspent points (memory stats, with the HUD button flags 6 and 7 as a cross check).
func Pending(s *d2.Snapshot) bool {
	if s == nil || s.Player == nil {
		return false
	}
	return s.Player.StatPoints > 0 || s.Player.SkillPoints > 0
}

// Do spends every unspent point the plan covers. It is idempotent: it plans from memory each time, verifies each
// point by re-reading memory, and stops at the first thing it cannot prove. Never spends while Busy.
func (l *Leveler) Do(ctx context.Context, snap *d2.Snapshot) (rep Report) {
	if snap == nil || snap.State != d2.StateInGame || snap.Player == nil {
		rep.Skipped = "not in game"
		return
	}
	p := snap.Player
	if Pending(snap) {
		if _, err := l.buildFor(p); err != nil {
			rep.Err = err
			l.pause(err.Error())
			return
		}
	}
	rep.StatBefore, rep.SkillBefore = p.StatPoints, p.SkillPoints
	rep.StatLeft, rep.SkillLeft = p.StatPoints, p.SkillPoints
	if !Pending(snap) {
		rep.Skipped = "no unspent points"
		return
	}
	if l.busy(snap) {
		rep.Skipped, rep.Err = "fight in progress", ErrBusy
		return
	}
	defer func() {
		l.mu.Lock()
		l.stats.Failures += btoi(rep.Err != nil)
		l.mu.Unlock()
	}()
	for round := 0; round < 4; round++ {
		acts := l.PlanFor(snap)
		if round == 0 {
			rep.Planned = acts
		}
		if len(acts) == 0 {
			break
		}
		l.logf("plan", "level", p.Level, "stat", snap.Player.StatPoints, "skill", snap.Player.SkillPoints, "actions", actionList(acts), "dry", l.o.Dry)
		if l.o.Dry {
			rep.Skipped = "dry run"
			return
		}
		refused, err := l.run(ctx, acts, &rep)
		if err != nil {
			rep.Err = err
			break
		}
		if len(refused) == 0 {
			break
		}
		l.mu.Lock()
		for _, k := range refused {
			l.rejected[k] = p.Level
		}
		l.mu.Unlock()
		rep.Refused = append(rep.Refused, refused...)
		s, err := l.e.Snapshot()
		if err != nil || s.Player == nil {
			break
		}
		snap = s
		p = s.Player
	}
	l.closePanels(ctx)
	if s, err := l.e.Snapshot(); err == nil && s.Player != nil {
		rep.StatLeft, rep.SkillLeft = s.Player.StatPoints, s.Player.SkillPoints
	}
	learned := false
	for _, a := range rep.Done {
		learned = learned || a.Kind == KindSkill
	}
	if learned && l.o.Rebind != nil && rep.Err == nil {
		if err := l.o.Rebind(ctx); err != nil {
			l.e.Log.Warn("leveler: rebinding hotkeys failed", "err", err)
		}
	}
	l.logAllocated(rep.Done)
	l.logf("done", "done", actionList(rep.Done), "refused", rep.Refused, "statLeft", rep.StatLeft, "skillLeft", rep.SkillLeft, "err", rep.Err)
	return rep
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func actionList(a []Action) string {
	var s []string
	for _, x := range a {
		s = append(s, x.String())
	}
	return strings.Join(s, ", ")
}

func (l *Leveler) busy(s *d2.Snapshot) bool {
	if l.o.Busy != nil {
		return l.o.Busy(s)
	}
	if s.Player.InTown() {
		return false
	}
	for i := range s.Monsters {
		m := &s.Monsters[i]
		if m.Alive() && max(abs(m.X-s.Player.X), abs(m.Y-s.Player.Y)) <= 25 {
			return true
		}
	}
	return false
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// run performs the actions: all stat points on the character panel, then the skills in the skill tree. It returns
// the skills the game refused (a refused point leaves the unspent count unchanged).
func (l *Leveler) run(ctx context.Context, acts []Action, rep *Report) (refused []string, err error) {
	var stats, skills []Action
	for _, a := range acts {
		if a.Kind == KindStat {
			stats = append(stats, a)
		} else {
			skills = append(skills, a)
		}
	}
	if len(stats) > 0 {
		if err := l.openPanel(ctx, "c", d2.UICharacter); err != nil {
			return nil, err
		}
		for _, a := range stats {
			if err := l.spendStat(ctx, a); err != nil {
				return nil, err
			}
			rep.Done = append(rep.Done, a)
			l.mu.Lock()
			l.stats.StatPoints++
			l.mu.Unlock()
		}
		l.closePanels(ctx)
	}
	if len(skills) > 0 {
		if err := l.openPanel(ctx, "t", d2.UISkillTree); err != nil {
			return nil, err
		}
		for _, a := range skills {
			ok, err := l.spendSkill(ctx, a)
			if err != nil {
				return refused, err
			}
			if !ok {
				refused = append(refused, a.Skill)
				l.mu.Lock()
				l.stats.Refused++
				l.mu.Unlock()
				continue
			}
			rep.Done = append(rep.Done, a)
			l.mu.Lock()
			l.stats.SkillPoints++
			l.mu.Unlock()
		}
		l.closePanels(ctx)
	}
	return refused, nil
}

// poll reads fresh player state until cond holds or the timeout passes.
func (l *Leveler) poll(ctx context.Context, timeout time.Duration, cond func(*d2.Player, *d2.UIState) bool) (*d2.Player, bool, error) {
	deadline := l.e.Clock.Now().Add(timeout)
	for {
		p, err := l.e.Read.Player()
		if err != nil {
			return nil, false, err
		}
		ui, _ := l.e.Read.UI()
		if cond(p, &ui) {
			return p, true, nil
		}
		if !l.e.Clock.Now().Before(deadline) {
			return p, false, nil
		}
		if err := ctx.Err(); err != nil {
			return p, false, err
		}
		l.sleep(100)
	}
}

// openPanel makes sure the panel behind UI flag idx is open: it closes everything else first, then presses the hotkey.
func (l *Leveler) openPanel(ctx context.Context, key string, idx int) error {
	if ui, err := l.e.Read.UI(); err == nil {
		if ui.Open(idx) {
			return nil
		}
		if ui.AnyPanelOpen() {
			if err := l.e.In.CloseAll(); err != nil {
				return err
			}
		}
	}
	for try := 0; try < 3; try++ {
		if err := l.e.In.Key(key); err != nil {
			return err
		}
		if _, ok, err := l.poll(ctx, 1500*time.Millisecond, func(_ *d2.Player, ui *d2.UIState) bool { return ui.Open(idx) }); err != nil {
			return err
		} else if ok {
			l.sleep(150)
			return nil
		}
	}
	return fmt.Errorf("%w (flag 0x%x)", errPanel, idx)
}

func (l *Leveler) closePanels(ctx context.Context) {
	if ui, err := l.e.Read.UI(); err == nil && ui.AnyPanelOpen() {
		if err := l.e.In.CloseAll(); err != nil {
			l.e.Log.Warn("leveler: closing panels failed", "err", err)
		}
	}
}

// spendStat clicks the + button of one attribute and proves the unspent count fell and the attribute rose.
func (l *Leveler) spendStat(ctx context.Context, a Action) error {
	pt := l.geo.StatButton[a.Attr]
	for try := 0; try < 3; try++ {
		before, err := l.e.Read.Player()
		if err != nil {
			return err
		}
		if before.StatPoints == 0 {
			return nil
		}
		if err := l.e.In.Click(input.Left, pt.X, pt.Y); err != nil {
			return err
		}
		_, ok, err := l.poll(ctx, 1200*time.Millisecond, func(p *d2.Player, _ *d2.UIState) bool { return p.StatPoints < before.StatPoints })
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}
	return fmt.Errorf("leveler: %s did not register (3 clicks at %v)", a, pt)
}

// gained lists the skill ids whose base level rose (or that appeared) between two skill lists.
func gained(before, after []d2.Skill) []int {
	was := map[int]int{}
	for _, s := range before {
		was[s.ID] = s.Level
	}
	var out []int
	for _, s := range after {
		if s.Level > was[s.ID] {
			out = append(out, s.ID)
		}
	}
	return out
}

// checkLearned is the skill-ID read-back: exactly the wanted skill id gained a level. Anything else is
// ErrSkillMismatch (CLASSPAL saw Smite, Thorns and Resist Fire learned in place of Sorceress skills).
func checkLearned(db *data.DB, want int, got []int) error {
	if len(got) == 1 && got[0] == want {
		return nil
	}
	name := func(id int) string {
		if db != nil {
			if s := db.SkillByID(id); s != nil {
				return fmt.Sprintf("%s(%d)", s.Key, id)
			}
		}
		return fmt.Sprint(id)
	}
	var names []string
	for _, id := range got {
		names = append(names, name(id))
	}
	return fmt.Errorf("%w: wanted %s, skill list gained %v", ErrSkillMismatch, name(want), names)
}

// spendSkill selects the skill's tab, clicks its icon and proves the unspent count fell AND that the skill with the
// planned id (skills.txt) gained exactly one level in the player's skill list. ok=false: the game refused the point
// (prerequisite, level) although the plan thought it fine. A point that landed on another skill is an error that
// pauses the bot (spent points cannot be undone without a respec).
func (l *Leveler) spendSkill(ctx context.Context, a Action) (ok bool, err error) {
	db := l.e.Data
	if db == nil {
		return false, errors.New("leveler: skill positions need the game tables")
	}
	s := db.SkillByName(a.Skill)
	if s == nil || s.Page < 1 || s.Page > 3 || s.Row < 1 || s.Column < 1 {
		return false, fmt.Errorf("leveler: no tree position for skill %q", a.Skill)
	}
	tab, icon := l.geo.Tab(s.Page), l.geo.Icon(s.Row, s.Column)
	for try := 0; try < 2; try++ {
		before, err := l.player()
		if err != nil {
			return false, err
		}
		if before.SkillPoints == 0 {
			return false, nil
		}
		if err := l.hoverClick(tab); err != nil {
			return false, err
		}
		if err := l.e.In.Click(input.Left, icon.X, icon.Y); err != nil {
			return false, err
		}
		after, done, err := l.pollPlayer(ctx, 1200*time.Millisecond, func(p *d2.Player) bool { return p.SkillPoints < before.SkillPoints })
		if err != nil {
			return false, err
		}
		if !done {
			continue
		}
		// the skill list can lag the point count by a frame or two
		got := gained(before.Skills, after.Skills)
		if len(got) == 0 {
			after, _, err = l.pollPlayer(ctx, 800*time.Millisecond, func(p *d2.Player) bool { return len(gained(before.Skills, p.Skills)) > 0 })
			if err != nil {
				return false, err
			}
			got = gained(before.Skills, after.Skills)
		}
		if err := checkLearned(db, s.ID, got); err != nil {
			l.pause(err.Error())
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// pollPlayer is poll for the player alone (no UI read), through the replaceable reader.
func (l *Leveler) pollPlayer(ctx context.Context, timeout time.Duration, cond func(*d2.Player) bool) (*d2.Player, bool, error) {
	deadline := l.e.Clock.Now().Add(timeout)
	for {
		p, err := l.player()
		if err != nil {
			return nil, false, err
		}
		if cond(p) {
			return p, true, nil
		}
		if !l.e.Clock.Now().Before(deadline) {
			return p, false, nil
		}
		if err := ctx.Err(); err != nil {
			return p, false, err
		}
		l.sleep(100)
	}
}

var _ = data.Normal

// logAllocated writes the "stats: allocated STR+n ..." and "skills: allocated ..." lines.
func (l *Leveler) logAllocated(done []Action) {
	var stat [numAttr]int
	skills := map[string]int{}
	var order []string
	for _, a := range done {
		if a.Kind == KindStat {
			stat[a.Attr]++
			continue
		}
		if skills[a.Skill] == 0 {
			order = append(order, a.Skill)
		}
		skills[a.Skill]++
	}
	var st []string
	for a, n := range stat {
		if n > 0 {
			st = append(st, fmt.Sprintf("%s+%d", strings.ToUpper(attrNames[a][:3]), n))
		}
	}
	if len(st) > 0 {
		l.e.Log.Info("stats: allocated " + strings.Join(st, " "))
	}
	var sk []string
	for _, n := range order {
		sk = append(sk, fmt.Sprintf("%s+%d", n, skills[n]))
	}
	if len(sk) > 0 {
		l.e.Log.Info("skills: allocated " + strings.Join(sk, " "))
	}
}
