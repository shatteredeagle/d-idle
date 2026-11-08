package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/pact"
	"d2lod-bot/internal/unlock"
	"d2lod-bot/internal/watch"
)

// Bot-side Pacts on the real bot (docs/pact-botside.md, TASK-167). Off unless a chosen set is in pacts-state.json, and
// never built in a gate or measured run (pact.Off). NOT run against a live bot.

// pactSheet adapts the policy sheet to pact.Sheet.
type pactSheet struct{ s *watch.PolicySheet }

func (p pactSheet) Current(row string) string { return p.s.CurrentOption(row) }
func (p pactSheet) Default(row string) string { return p.s.DefaultOption(row) }
func (p pactSheet) Set(row, opt string) error { return p.s.SetByPact(row, opt) }

// pactRefs checks the shipped catalogue against the real sheet table: a Pact naming a row or option the sheet lacks is a
// start-up error, not a silent no-op.
func pactRefs(sheet *watch.PolicySheet) pact.Refs {
	return pact.Refs{
		Option: func(row, opt string) error {
			if sheet.DefaultOption(row) == "" {
				return fmt.Errorf("the policy sheet has no row %q", row)
			}
			return nil // option ids are checked by the catalogue test against the table; the sheet refuses an unknown one at Set
		},
	}
}

// ladderDifficulty is the ladder's name for a difficulty ("normal" for an unknown or empty one, like difficultyOf).
func ladderDifficulty(s string) string {
	return [...]string{"normal", "nightmare", "hell"}[difficultyOf(s)]
}

func difficultyOf(s string) int {
	switch strings.ToLower(s) {
	case "nightmare":
		return 1
	case "hell":
		return 2
	}
	return 0
}

// startPacts runs the Pact controller beside the goal manager. It returns at once; the work is one goroutine that waits
// for the hero's class, then evaluates every two seconds, counts deaths and resumes after the player takes the hero back.
// A human is present while a viewer request came in the last limits.presence_min minutes or a video stream is open
// (TASK-169); without one a Pact acts only on a typed away-consent. Pacts pause under the life floor.
func startPacts(ctx context.Context, l ladderOptions, rig *ladderRig, m *goals.Manager, status func() goals.Status, stateDir string, getenv func(string) string) {
	if pact.Off(getenv) || rig == nil || rig.Fun == nil || rig.Fun.Unlocks == nil || rig.Pacts == nil || rig.Fun.Policy == nil || stateDir == "" || m == nil || status == nil {
		return
	}
	ch, unsub := m.Subscribe()
	go func() {
		defer unsub()
		var ctl *pact.Controller
		start := time.Now()
		towns := townNames()
		tick := time.NewTicker(2 * time.Second)
		defer tick.Stop()
		wasPaused := false
		lastErr := ""
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-ch:
				if !ok {
					return
				}
				if ctl != nil && ev.Type == goals.EvDeath {
					if err := ctl.OnDeath(); err != nil {
						rig.Hub.Incident("Pact: " + err.Error())
					}
				}
				if ev.Type == goals.EvKill && rig.Fun.Unlocks != nil {
					// The ladder names the bosses (data/unlocks/ladder.json); it reveals what opens.
					rig.Fun.Unlocks.Observe(unlock.Event{Type: unlock.EvBossKill, Name: ev.Monster, Sub: ladderDifficulty(rig.Pacts.desk.heroNow().Difficulty)})
				}
			case <-tick.C:
				st := status()
				hero := st.Character
				if ctl == nil {
					class := strings.ToLower(hero.Class)
					if class == "" {
						continue
					}
					cat, err := pact.Shipped(pactRefs(rig.Fun.Policy))
					if err == nil {
						pres := rig.Pacts.pres
						pres.SetWindow(time.Duration(cat.Limits.PresenceMinutes()) * time.Minute)
						if rig.Video != nil {
							pres.Open = rig.Video.Viewers
						}
						ctl, err = pact.New(pact.Config{
							Cat: cat, Class: class, Path: filepath.Join(stateDir, "pacts-state.json"),
							Knobs: pact.KnobSetter(l.knobs), Sheet: pactSheet{rig.Fun.Policy},
							Pause:   func(reason string) error { rig.Hub.Incident("Pact: " + reason); return m.Pause() },
							Present: pres.Present,
							Rel:     rig.Fun.Policy.Now,
							Journal: func(line string) { rig.Hub.Incident(line) },
							Gate:    rig.Fun.Unlocks.Engine,
							ConsentLog: func(line string) {
								rig.Hub.Incident(line)
								rig.Fun.Away.Add(watch.AwayEvent{Kind: watch.AwayPactConsent, Item: line})
							},
						})
						if err == nil {
							err = adoptLegacyUnlocks(ctl, rig.Fun.Unlocks.Engine)
						}
						if err == nil {
							var hc *pact.Hardcore
							if hc, err = pact.OpenHardcore(filepath.Join(stateDir, "hardcore.json"), nil); err == nil {
								rig.Pacts.desk.ctl, rig.Pacts.desk.hc = ctl, hc
								rig.Pacts.api.Bind(rig.Pacts.desk)
							}
						}
					}
					if err != nil {
						rig.Hub.Incident("Pacts are off: " + err.Error())
						return
					}
					rig.Pacts.ctl.Store(ctl)
				}
				rig.Pacts.desk.setHero(hero)
				paused := st.State == goals.RunnerPaused
				if wasPaused && !paused {
					ctl.Resume() // the player took the hero back
				}
				wasPaused = paused
				w := pact.Where{Level: hero.Level, Difficulty: difficultyOf(hero.Difficulty), InTown: towns[hero.Area], InDen: hero.Area == "Den of Evil", Age: time.Since(start), Life: hero.Life, MaxLife: hero.MaxLife}
				if err := ctl.Evaluate(w); err != nil && err.Error() != lastErr {
					rig.Hub.Incident("Pact: " + err.Error())
					lastErr = err.Error() // said once, not every tick
				}
			}
		}
	}()
}

// adoptLegacyUnlocks moves the unlock triggers an older pacts-state.json recorded into the ladder (silently: the viewer
// saw them open), then drops the old field. Safe to run on every start.
func adoptLegacyUnlocks(ctl *pact.Controller, eng *unlock.Engine) error {
	for _, trig := range ctl.Legacy() {
		if _, err := eng.Adopt(pact.LadderID(trig)); err != nil && !errors.Is(err, unlock.ErrUnknownEntry) {
			return err // an unknown trigger (no ladder entry) stays locked and is dropped with the field
		}
	}
	return ctl.ClearLegacy()
}

// pactHolder is the section's late-bound controller: the section registry is not safe to change once the screen is
// served, so the section is registered before Serve (registerPactSection) and finds its controller here.
type pactHolder struct {
	ctl  atomic.Pointer[pact.Controller]
	pres *pact.Presence
	api  *watch.PactAPI
	desk *pactDesk
}

// pactDesk adapts the controller and the Hardcore flag to watch.PactDesk.
type pactDesk struct {
	ctl *pact.Controller
	hc  *pact.Hardcore

	mu   sync.Mutex
	hero goals.Character
}

func (d *pactDesk) setHero(c goals.Character) { d.mu.Lock(); d.hero = c; d.mu.Unlock() }
func (d *pactDesk) heroNow() goals.Character  { d.mu.Lock(); defer d.mu.Unlock(); return d.hero }

func (d *pactDesk) Offer() any { return d.ctl.Offer(difficultyOf(d.heroNow().Difficulty)) }

func (d *pactDesk) Consent(id string, rank, hours int, typed string) error {
	for _, pk := range d.ctl.Picks() {
		if pk.ID == id {
			if rank != pk.Rank {
				return fmt.Errorf("the chosen rank of %q is %d, not %d: ask again with the rank you see", id, pk.Rank, rank)
			}
			return d.ctl.GrantFor(id, typed, difficultyOf(d.heroNow().Difficulty), hours)
		}
	}
	return fmt.Errorf("%q is not chosen", id)
}

// Catalogue is the offer page's data; propose is "id:rank,..." ("" none, "-" the empty set).
func (d *pactDesk) Catalogue(propose string) (any, error) {
	if propose == "" {
		return d.ctl.View(nil), nil
	}
	picks := []pact.Pick{}
	if propose != "-" {
		var err error
		if picks, err = pact.ParsePicks(propose); err != nil {
			return nil, err
		}
	}
	return d.ctl.View(picks), nil
}

func (d *pactDesk) Choose(picks []watch.PactPick) error {
	ps := make([]pact.Pick, len(picks))
	for i, p := range picks {
		ps[i] = pact.Pick{ID: p.ID, Rank: p.Rank}
	}
	return d.ctl.Choose(ps)
}

func (d *pactDesk) Revoke(id string) error { return d.ctl.Revoke(id) }
func (d *pactDesk) Hardcore() any          { return d.hc.State() }
func (d *pactDesk) HardcoreConfirm(typed string) error {
	return d.hc.Confirm(d.heroNow().Name, typed)
}
func (d *pactDesk) HardcoreRevoke() error { return d.hc.Revoke() }
func (d *pactDesk) HardcoreActivate() error {
	if err := d.hc.Activate(); err != nil {
		return fmt.Errorf("%w: %s", watch.ErrPactInactive, err.Error())
	}
	return nil
}

// registerPactSection adds the "Pacts" return section; nil (nothing registered) when Pacts are off or there is no screen.
func registerPactSection(f *watch.Fun, getenv func(string) string) *pactHolder {
	if pact.Off(getenv) || f == nil {
		return nil
	}
	h := &pactHolder{pres: pact.NewPresence(0, nil), api: watch.NewPactAPI(), desk: &pactDesk{}}
	f.Pacts, f.OnView = h.api, h.pres.Touch
	f.Return.Sections.Register("pacts", pactSection(h, f.Policy, f.Away))
	return h
}

// pactSection is the return screen's Pact lines: what each chosen Pact does and whether it acts now, then what each cost
// and paid, with before/after counters from the policy sheet for a Pact that sets a sheet row (the same counters as the
// policy cards). The tally multiplier is not paid by any code yet, so the "paid" part says what is promised (U).
func pactSection(h *pactHolder, sheet *watch.PolicySheet, away *watch.AwayLog) watch.ReturnSection {
	return watch.SectionFunc(func(d *watch.ReturnData) *watch.SectionView {
		ctl := h.ctl.Load()
		if ctl == nil {
			return nil
		}
		ls := ctl.Lines()
		var consent []string // away-consent changes since the viewer left
		for _, e := range away.Events() {
			if e.Kind == watch.AwayPactConsent && e.T >= d.Since {
				consent = append(consent, e.Item)
			}
		}
		if len(ls) == 0 && len(consent) == 0 {
			return nil
		}
		v := &watch.SectionView{ID: "pacts", Title: "Pacts"}
		for _, l := range ls {
			v.Lines = append(v.Lines, fmt.Sprintf("%s (rank %d): %s [%s]", l.Pact, l.Rank, l.Text, l.Status))
		}
		for _, c := range ctl.Costs() {
			v.Lines = append(v.Lines, costLine(c, sheet))
		}
		v.Lines = append(v.Lines, consent...)
		return v
	})
}

func costLine(c pact.CostLine, sheet *watch.PolicySheet) string {
	head := fmt.Sprintf("%s (rank %d) cost: %s.", c.Name, c.Rank, c.Effect)
	paid := fmt.Sprintf(" Promised: tally +%d%%", c.Reward.TallyPct)
	if c.Reward.CharmPoints > 0 {
		paid += fmt.Sprintf(", charm points +%d", c.Reward.CharmPoints)
	}
	if c.Reward.Title != "" {
		paid += ", title " + c.Reward.Title
	}
	paid += " (nothing pays the tally bonus yet, U)."
	if c.Start < 0 {
		return head + " It has not acted yet." + paid
	}
	end := c.End
	if end < 0 {
		if sheet == nil {
			return head + " It is acting now. No counter is tied to this lever (U)." + paid
		}
		end = sheet.Now()
	}
	acted := end - c.Start
	if acted < 1 {
		return head + " It just started." + paid
	}
	line := fmt.Sprintf("%s It acted for %s.", head, fmtSecs(acted))
	if c.Row == "" || sheet == nil {
		return line + " No counter is tied to this lever (U)." + paid
	}
	after, ok1 := sheet.Counters(c.Row, c.Start, end)
	before, ok2 := sheet.Counters(c.Row, max(c.Start-acted, 0), c.Start)
	if !ok1 || !ok2 {
		return line + " The sheet has no counters for this row." + paid
	}
	return fmt.Sprintf("%s After, %s: %s. Before, %s: %s.%s", line, fmtSecs(acted), after, fmtSecs(acted), before, paid)
}

func fmtSecs(s float64) string {
	if s < 90 {
		return fmt.Sprintf("%.0f s", s)
	}
	return fmt.Sprintf("%.0f min", s/60)
}
