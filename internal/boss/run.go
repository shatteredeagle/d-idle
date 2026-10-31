package boss

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Opts tunes one run. Zero values take the defaults below.
type Opts struct {
	Poll        time.Duration // loop period (default 250 ms, the prior-art 02 locate poll)
	Slice       time.Duration // one Attack call (default 3 s)
	LocatePolls int           // polls for the boss at each step (default 16)
	Explore     time.Duration // exploration budget when the boss is still unseen (default 90 s)
	CanTeleport bool          // allow StandPoint.NeedsTeleport points
	Exit        *ExitPlan     // overrides Script.Exit
	Hooks       Hooks
	// AndarielMinLevel overrides the Normal clvl floor of the Andariel script (Tuning.Boss.AndarielMinLevel); 0 = the script's own.
	AndarielMinLevel int
	// MaxRetreats overrides Script.MaxRetreats when above 0 (Tuning.Boss.BRMaxRetreats).
	MaxRetreats int
}

func (o Opts) withDefaults() Opts {
	if o.Poll <= 0 {
		o.Poll = 250 * time.Millisecond
	}
	if o.Slice <= 0 {
		o.Slice = 3 * time.Second
	}
	if o.LocatePolls <= 0 {
		o.LocatePolls = 16
	}
	if o.Explore <= 0 {
		o.Explore = 90 * time.Second
	}
	return o
}

// State is the per-run memory hazards and phases read.
type State struct {
	Arrival  Pt // where the player stood when the arena was reached
	Spawn    Pt // where the boss was first seen
	Stand    Pt // resolved anchor stand point (zero when none)
	Last     Mon
	standIdx int
	retreats int // RetreatPct retreats so far in this run
	lastKite time.Time
	cd       map[string]time.Time
}

// Result says what a run did.
type Result struct {
	Killed bool
	// BossMaxLife is the boss's max life in points when first seen (0 = not read), logged as the HP calibration of a script.
	BossMaxLife int
	Boss        string
	Loot        int
	Phases      []string
	Notes       []string
	Elapsed     time.Duration
}

type runner struct {
	s   *Script
	w   World
	a   Actions
	o   Opts
	st  State
	res Result
	t0  time.Time
	cur string
}

// Run executes the script from wherever the character is: in town, on the route, or already in the arena.
func Run(ctx context.Context, s *Script, w World, a Actions, o Opts) (Result, error) {
	r := &runner{s: s, w: w, a: a, o: o.withDefaults(), t0: w.Now()}
	r.st.cd = map[string]time.Time{}
	r.res.Boss = s.Name
	err := r.run(ctx)
	r.res.Elapsed = w.Now().Sub(r.t0)
	return r.res, err
}

func (r *runner) phase(name string) {
	if r.cur == name {
		return
	}
	r.cur = name
	r.res.Phases = append(r.res.Phases, name)
	if r.o.Hooks.OnPhase != nil {
		r.o.Hooks.OnPhase(name)
	}
}

func (r *runner) note(f string, args ...any) {
	r.res.Notes = append(r.res.Notes, fmt.Sprintf(f, args...))
}

func (r *runner) run(ctx context.Context) error {
	v, err := r.w.View(ctx)
	if err != nil {
		return err
	}
	if v.Dead {
		return ErrDead
	}
	if f := r.floor(v.Difficulty); f > 0 && v.Level < f {
		return fmt.Errorf("%w: %s needs clvl %d, have %d", ErrTooWeak, r.s.Name, f, v.Level)
	}
	if err := r.approach(ctx, v); err != nil {
		return err
	}
	v, err = r.w.View(ctx)
	if err != nil {
		return err
	}
	r.st.Arrival = v.Pos
	if err := r.clearOuter(ctx); err != nil {
		return err
	}
	boss, err := r.locate(ctx)
	if err != nil {
		return err
	}
	r.st.Spawn, r.st.Last = boss.Pos, boss
	r.res.BossMaxLife = boss.MaxLife
	if boss.Alive() {
		if err := r.fight(ctx); err != nil {
			return err
		}
	}
	return r.finish(ctx)
}

// clearOuter sweeps the monsters around the arrival point before the boss is looked for (Script.OuterClear). A failed
// sweep is not a failed run; only the combat sentinels pass.
func (r *runner) clearOuter(ctx context.Context) error {
	oc := r.s.OuterClear
	if oc.Radius <= 0 {
		return nil
	}
	budget := oc.Budget
	if budget <= 0 {
		budget = 40 * time.Second
	}
	r.phase("outer-clear")
	if err := r.a.ClearAround(ctx, r.st.Arrival, oc.Radius, budget); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return passFatal(err)
	}
	return nil
}

// retreat walks back to the arrival point and drinks. The MaxRetreats-th call leaves instead (town portal, else chicken):
// a second retreat in one attempt means the fight is lost at this level and a same-level retry would only burn potions.
func (r *runner) retreat(ctx context.Context, v View) error {
	r.st.retreats++
	max := r.s.MaxRetreats
	if r.o.MaxRetreats > 0 {
		max = r.o.MaxRetreats
	}
	if max > 0 && r.st.retreats >= max {
		r.note("retreat %d/%d: leaving the fight", r.st.retreats, max)
		r.phase("retreat-limit")
		return r.escape(ctx, v)
	}
	r.phase("retreat")
	r.note("retreat %d: back to the entry %d,%d", r.st.retreats, r.st.Arrival.X, r.st.Arrival.Y)
	if err := r.a.GoTo(ctx, r.st.Arrival); err != nil {
		r.note("retreat: %v", err)
	}
	return r.drink(ctx, v)
}

func clampDiff(d int) int {
	if d < 0 {
		return 0
	}
	if d > 2 {
		return 2
	}
	return d
}

// approach brings the character into the arena: Travel to the WP when it is on no leg of the chain, then ExitTo
// along the route.
func (r *runner) approach(ctx context.Context, v View) error {
	s := r.s
	if v.Area == s.Arena {
		return nil
	}
	r.phase("approach")
	if need := v.PotionFloor(s.MinPotions); s.MinPotions > 0 && v.Potions < need {
		return fmt.Errorf("%w: %s needs %d potions, have %d", ErrAborted, s.Name, need, v.Potions)
	}
	chain := s.Route
	if s.WP > 0 {
		chain = append([]int{s.WP}, s.Route...)
	}
	idx := -1
	for i, a := range chain {
		if a == v.Area {
			idx = i
		}
	}
	if idx < 0 {
		first := s.Arena
		if s.WP > 0 {
			first = s.WP
		}
		if err := r.a.Travel(ctx, first); err != nil {
			return r.routeFail(ctx, fmt.Errorf("boss: travel to %d: %w", first, err))
		}
		nv, err := r.w.View(ctx)
		if err != nil {
			return err
		}
		if nv.Area == s.Arena {
			return nil
		}
		if nv.Area != first {
			return fmt.Errorf("%w: wanted area %d, in %d", ErrLeftArena, first, nv.Area)
		}
		idx = 0
	}
	for i := idx + 1; i < len(chain); i++ {
		if r.overdue() {
			return r.routeFail(ctx, ErrTimeout)
		}
		if err := r.a.ExitTo(ctx, chain[i]); err != nil {
			return r.routeFail(ctx, fmt.Errorf("boss: exit to %d: %w", chain[i], err))
		}
		nv, err := r.w.View(ctx)
		if err != nil {
			return err
		}
		if nv.Dead {
			return ErrDead
		}
		if nv.Area != chain[i] {
			return fmt.Errorf("%w: wanted area %d, in %d", ErrLeftArena, chain[i], nv.Area)
		}
		if s.RouteMinLifePct > 0 && nv.LifePct() < s.RouteMinLifePct {
			r.note("route life %d%% in area %d", nv.LifePct(), nv.Area)
			return r.escape(ctx, nv)
		}
	}
	return nil
}

// overdue is true once approach and locate together used the script's whole time budget (the fight has its own).
func (r *runner) overdue() bool {
	t := r.s.Timeout
	if t <= 0 {
		t = 10 * time.Minute
	}
	return !r.w.Now().Before(r.t0.Add(t))
}

// routeFail turns a failed hop into an abort to town when the character is inside the dungeon and can leave: a
// stuck walker deep in a level with no waypoint is worse off than a retry from town. The original error is kept.
func (r *runner) routeFail(ctx context.Context, cause error) error {
	if ctx.Err() != nil || errors.Is(cause, ErrDead) {
		return cause
	}
	v, err := r.w.View(ctx)
	if err != nil || v.Dead || v.Area == r.s.WP || v.Area <= 0 || !v.HasTP || r.s.NoTP {
		return cause
	}
	inRoute := v.Area == r.s.Arena
	for _, a := range r.s.Route {
		inRoute = inRoute || a == v.Area
	}
	if !inRoute {
		return cause
	}
	if terr := r.a.ToTown(ctx); terr != nil {
		r.note("abort to town failed: %v", terr)
		return cause
	}
	r.note("route failed in area %d, went to town: %v", v.Area, cause)
	return cause
}

func (r *runner) find(v View) (Mon, bool) {
	var dead *Mon
	for i := range v.Monsters {
		m := v.Monsters[i]
		if !r.s.Match.Is(m) {
			continue
		}
		if m.Alive() {
			return m, true
		}
		dead = &v.Monsters[i]
	}
	if dead != nil {
		return *dead, true
	}
	return Mon{}, false
}

func (r *runner) poll(ctx context.Context, n int) (Mon, bool, error) {
	for i := 0; i < n; i++ {
		v, err := r.w.View(ctx)
		if err != nil {
			return Mon{}, false, err
		}
		if v.Dead {
			return Mon{}, false, ErrDead
		}
		if v.Area != r.s.Arena {
			return Mon{}, false, ErrLeftArena
		}
		if m, ok := r.find(v); ok {
			return m, true, nil
		}
		if i < n-1 {
			if err := r.w.Sleep(ctx, r.o.Poll); err != nil {
				return Mon{}, false, err
			}
		}
	}
	return Mon{}, false, nil
}

// locate finds the boss: look, walk to the preset stand point, look, walk to the hint, look, explore, look.
func (r *runner) locate(ctx context.Context) (Mon, error) {
	r.phase("locate")
	try := func(n int) (Mon, bool, error) { return r.poll(ctx, n) }
	if m, ok, err := try(1); err != nil || ok {
		return m, err
	}
	an := r.s.Anchor
	if an.Preset == nil {
		_, _ = r.w.Presets(ctx) // diagnostics only: the adapter logs the level's monster presets (V live need: where is the Smith)
	}
	if an.Preset != nil {
		ps, err := r.w.Presets(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return Mon{}, ctx.Err()
			}
			r.note("presets: %v", err)
		}
		p, ok := resolvePreset(an.Preset, ps)
		for i := 0; !ok && i < len(an.Also); i++ {
			p, ok = resolvePreset(an.Also[i], ps)
		}
		if ok {
			r.st.Stand = p
			r.checkHint(p)
			if tp, ok := r.a.(Teleporter); ok && an.Teleport {
				if err := tp.TeleportTo(ctx, p); err != nil {
					r.note("teleport to anchor: %v", err)
				}
			}
			if err := r.a.GoTo(ctx, p); err != nil {
				r.note("goto anchor: %v", err)
			}
			if m, ok, err := try(r.o.LocatePolls); err != nil || ok {
				return m, err
			}
		} else {
			r.note("preset %d/%d not found", an.Preset.Kind, an.Preset.ID)
		}
	}
	if r.overdue() {
		return Mon{}, ErrTimeout
	}
	if an.Hint != nil {
		if err := r.a.GoTo(ctx, *an.Hint); err != nil {
			r.note("goto hint: %v", err)
		}
		if m, ok, err := try(r.o.LocatePolls / 2); err != nil || ok {
			return m, err
		}
	}
	if r.overdue() {
		return Mon{}, ErrTimeout
	}
	var found Mon
	seen := false
	stop := func() bool {
		v, err := r.w.View(ctx)
		if err != nil {
			return true
		}
		found, seen = r.find(v)
		return seen
	}
	if err := r.a.Explore(ctx, r.exploreBudget(), stop); err != nil && ctx.Err() != nil {
		return Mon{}, ctx.Err()
	}
	if seen {
		return found, nil
	}
	if an.Preset == nil {
		_, _ = r.w.Presets(ctx) // diagnostics: presets revealed by the exploration
	}
	if m, ok, err := try(r.o.LocatePolls); err != nil || ok {
		return m, err
	}
	return Mon{}, ErrBossNotFound
}

func (r *runner) checkHint(p Pt) {
	h := r.s.Anchor.Hint
	if h == nil {
		return
	}
	slop := r.s.Anchor.HintSlop
	if slop <= 0 {
		slop = 60
	}
	if d := p.Dist(*h); d > slop {
		r.note("preset stand %v is %d from the Kolbot hint %v (layout or origin differs)", p, d, *h)
	}
}

func resolvePreset(ref *PresetRef, ps []Preset) (Pt, bool) {
	for _, p := range ps {
		if p.Kind != ref.Kind || p.ID != ref.ID {
			continue
		}
		for _, v := range ref.Variants {
			if abs(p.Pos.X-v.AtX) <= 2 {
				return v.Stand, true
			}
		}
		return Pt{p.Pos.X + ref.Off.X, p.Pos.Y + ref.Off.Y}, true
	}
	return Pt{}, false
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func (r *runner) cool(name string, d time.Duration) bool {
	now := r.w.Now()
	if t, ok := r.st.cd[name]; ok && now.Before(t) {
		return false
	}
	r.st.cd[name] = now.Add(d)
	return true
}

func (r *runner) escape(ctx context.Context, v View) error {
	if r.s.NoTP || !v.HasTP {
		if err := r.a.Chicken(ctx); err != nil {
			return err
		}
		return ErrChicken
	}
	if err := r.a.ToTown(ctx); err != nil {
		return err
	}
	return ErrEscaped
}

func (r *runner) fight(ctx context.Context) error {
	s := r.s
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	deadline := r.w.Now().Add(timeout)
	var lureEnd time.Time
	if s.Lure != nil {
		lureEnd = r.w.Now().Add(s.Lure.Wait)
	}
	missed, fails, buffed, chases := 0, 0, false, 0
	lured := s.Lure == nil
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !r.w.Now().Before(deadline) {
			return ErrTimeout
		}
		v, err := r.w.View(ctx)
		if err != nil {
			return err
		}
		if v.Dead {
			return ErrDead
		}
		if v.Area != s.Arena {
			return ErrLeftArena
		}
		boss, ok := r.find(v)
		if !ok {
			// A killed boss may disappear from the table at once: treat a boss last seen nearly dead as dead.
			if missed++; missed >= 3 {
				if r.st.Last.LifePct <= 10 {
					return nil
				}
				// Out of the client's unit list is not dead: walk to where she was last seen, a few times.
				if chases < 3 && r.st.Last.Pos != (Pt{}) {
					chases++
					r.note("boss left the unit list; walking to its last position %d,%d (%d)", r.st.Last.Pos.X, r.st.Last.Pos.Y, chases)
					if err := r.a.GoTo(ctx, r.st.Last.Pos); err != nil && ctx.Err() != nil {
						return ctx.Err()
					}
					missed = 0
					continue
				}
				return ErrBossLost
			}
			if err := r.w.Sleep(ctx, r.o.Poll); err != nil {
				return err
			}
			continue
		}
		missed = 0
		r.st.Last = boss
		if !boss.Alive() {
			return nil
		}
		life := v.LifePct()
		if s.ChickenPct > 0 && life < s.ChickenPct {
			r.phase("chicken")
			if err := r.a.Chicken(ctx); err != nil {
				return err
			}
			return ErrChicken
		}
		// With potions left, escape only when life is critical: weak potions do not outheal a boss (V live: Combi died
		// drinking minor potions, i2 2026-10-29).
		if s.EscapePct > 0 && life < s.EscapePct && (v.Potions == 0 || life < s.EscapePct/2) {
			r.phase("escape")
			return r.escape(ctx, v)
		}
		if s.RetreatPct > 0 && life < s.RetreatPct && v.Potions > 0 && r.cool("retreat", 20*time.Second) {
			if err := r.retreat(ctx, v); err != nil {
				return err
			}
			continue
		}
		if s.DrinkPct > 0 && life < s.DrinkPct && v.Potions > 0 && r.cool("drink", 1500*time.Millisecond) {
			if err := r.drink(ctx, v); err != nil {
				return err
			}
			continue
		}
		c := Ctx{V: v, Boss: &boss, St: &r.st, Script: s}
		fired := false
		for _, h := range s.Hazards {
			if !h.Detect(c) || !r.cool("h:"+h.Name, h.Cooldown) {
				continue
			}
			r.phase("hazard:" + h.Name)
			if err := r.respond(ctx, h.Do, v, boss); err != nil {
				return err
			}
			fired = true
			break
		}
		if fired {
			continue
		}
		if !lured {
			r.phase("lure")
			if v.Pos.Dist(r.st.Arrival) > 10 {
				if err := r.a.GoTo(ctx, r.st.Arrival); err != nil {
					r.note("lure goto: %v", err)
				}
			}
			if boss.Pos.Dist(v.Pos) <= s.Lure.Dist || !r.w.Now().Before(lureEnd) {
				lured = true
			} else if err := r.w.Sleep(ctx, r.o.Poll); err != nil {
				return err
			}
			continue
		}
		if k := s.Kite; k != nil && boss.Pos.Dist(v.Pos) < k.Range && r.w.Now().Sub(r.st.lastKite) >= k.Every {
			r.phase("kite")
			r.st.lastKite = r.w.Now()
			if err := r.kite(ctx, v, boss); err != nil {
				r.note("kite: %v", err)
			}
			continue
		}
		r.phase("attack")
		if !buffed {
			buffed = true
			if s.PreAntidoteBelow > 0 && v.PoisonRes < s.PreAntidoteBelow && v.Antidotes > 0 {
				if err := r.a.Antidote(ctx); err != nil {
					r.note("pre antidote: %v", err)
				}
			}
			if f := r.o.Hooks.BeforeAttack; f != nil {
				if err := f(ctx, v, boss); err != nil {
					r.note("before attack: %v", err)
				}
			}
		}
		t := r.w.Now()
		err = r.a.Attack(ctx, boss.ID, r.o.Slice)
		switch {
		case err == nil:
			fails = 0
		case errors.Is(err, ErrEscaped), errors.Is(err, ErrChicken), errors.Is(err, ErrDead):
			return err
		case ctx.Err() != nil:
			return ctx.Err()
		default:
			if fails++; fails >= 5 {
				return fmt.Errorf("boss: attack failed %d times: %w", fails, err)
			}
		}
		if r.w.Now().Sub(t) < r.o.Poll/2 {
			if err := r.w.Sleep(ctx, r.o.Poll); err != nil {
				return err
			}
		}
	}
}

func (r *runner) respond(ctx context.Context, rs Response, v View, boss Mon) error {
	switch rs.Kind {
	case RespAntidote:
		if err := r.a.Antidote(ctx); err != nil {
			r.note("antidote: %v", err)
			return passFatal(err)
		}
	case RespStepAway:
		from := boss.Pos
		if d, ok := nearestDanger(v, DangerPoisonCloud, 1<<20); ok {
			from = d.Pos
		}
		if err := r.a.GoTo(ctx, awayPoint(v.Pos, from, rs.Radius)); err != nil {
			r.note("step away: %v", err)
		}
	case RespClearAdds:
		if err := r.a.ClearAround(ctx, v.Pos, rs.Radius, rs.Budget); err != nil {
			return passFatal(err)
		}
	case RespRetreat:
		if err := r.a.GoTo(ctx, r.st.Arrival); err != nil {
			r.note("retreat: %v", err)
		}
		if v.Potions > 0 {
			return r.drink(ctx, v)
		}
	case RespDrink:
		if v.Potions > 0 {
			return r.drink(ctx, v)
		}
	case RespEscape:
		return r.escape(ctx, v)
	case RespAbort:
		return ErrAborted
	}
	return nil
}

// drink drinks one healing potion. A failed drink (the belt count was stale, the column is empty) is not fatal by
// itself, but when life is already under the escape line it leaves the fight instead of carrying on without healing.
func (r *runner) drink(ctx context.Context, v View) error {
	err := r.a.DrinkHealing(ctx)
	if err == nil {
		return nil
	}
	if fe := passFatal(err); fe != nil {
		return fe
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	r.note("drink failed: %v", err)
	if r.s.EscapePct > 0 && v.LifePct() < r.s.EscapePct {
		r.phase("escape")
		return r.escape(ctx, v)
	}
	return nil
}

// passFatal keeps the sentinel errors of a combat call and drops the rest (a failed clear is not a failed run).
func passFatal(err error) error {
	for _, e := range []error{ErrEscaped, ErrChicken, ErrDead} {
		if errors.Is(err, e) {
			return err
		}
	}
	return nil
}

func (r *runner) kite(ctx context.Context, v View, boss Mon) error {
	k := r.s.Kite
	n := len(r.s.Stand)
	for i := 0; i < n; i++ {
		sp := r.s.Stand[(r.st.standIdx+i)%n]
		if sp.NeedsTeleport && !r.o.CanTeleport {
			continue
		}
		p := sp.Pos
		if sp.Rel {
			p = Pt{r.st.Spawn.X + sp.Pos.X, r.st.Spawn.Y + sp.Pos.Y}
		}
		if p.Dist(boss.Pos) < k.Range || p.Dist(v.Pos) < 3 {
			continue
		}
		r.st.standIdx = (r.st.standIdx + i + 1) % n
		return r.a.GoTo(ctx, p)
	}
	if k.Away > 0 {
		return r.a.GoTo(ctx, awayPoint(v.Pos, boss.Pos, k.Away))
	}
	return nil
}

func (r *runner) finish(ctx context.Context) error {
	s := r.s
	r.res.Killed = true
	r.phase("loot")
	if f := r.o.Hooks.OnKill; f != nil {
		f(r.st.Last)
	}
	if s.PostKillWait > 0 {
		if err := r.w.Sleep(ctx, s.PostKillWait); err != nil {
			return err
		}
	}
	if s.PostClear > 0 {
		if err := r.a.ClearAround(ctx, r.st.Last.Pos, s.PostClear, 25*time.Second); err != nil {
			if fe := passFatal(err); fe != nil {
				return fe
			}
		}
	}
	spec := s.Loot
	spec.Center = r.st.Last.Pos
	if spec.Radius == 0 {
		spec.Radius = 25
	}
	n, err := r.a.Loot(ctx, spec)
	if err == nil && n == 0 && len(spec.Quest) > 0 {
		// Quest drops can land a moment after the death animation: look once more before giving up.
		r.note("quest item %v not picked up; retrying once", spec.Quest)
		if err := r.w.Sleep(ctx, 2*time.Second); err != nil {
			return err
		}
		n, err = r.a.Loot(ctx, spec)
		if err == nil && n == 0 {
			r.note("quest item %v still missing", spec.Quest)
		}
	}
	r.res.Loot = n
	if err != nil {
		return fmt.Errorf("boss: loot: %w", err)
	}
	exit := s.Exit
	if r.o.Exit != nil {
		exit = *r.o.Exit
	}
	if exit == ExitTown {
		r.phase("exit")
		return r.a.ToTown(ctx)
	}
	return nil
}

func (r *runner) exploreBudget() time.Duration {
	if r.s.ExploreBudget > 0 {
		return r.s.ExploreBudget
	}
	return r.o.Explore
}

// floor is the clvl floor of the script in difficulty d; the Andariel knob replaces the Normal one.
func (r *runner) floor(d int) int {
	if r.s.Name == NameAndariel && clampDiff(d) == 0 && r.o.AndarielMinLevel > 0 {
		return r.o.AndarielMinLevel
	}
	return r.s.MinLevel[clampDiff(d)]
}
