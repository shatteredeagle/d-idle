package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/pact"
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
	if pact.Off(getenv) || rig == nil || rig.Fun == nil || rig.Pacts == nil || rig.Fun.Policy == nil || stateDir == "" || m == nil || status == nil {
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
							Journal: func(line string) { rig.Hub.Incident(line) },
						})
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
	f.Return.Sections.Register("pacts", pactSection(h))
	return h
}

// pactSection is the return screen's Pact lines: what each chosen Pact does and whether it acts now.
func pactSection(h *pactHolder) watch.ReturnSection {
	return watch.SectionFunc(func(*watch.ReturnData) *watch.SectionView {
		ctl := h.ctl.Load()
		if ctl == nil {
			return nil
		}
		ls := ctl.Lines()
		if len(ls) == 0 {
			return nil
		}
		v := &watch.SectionView{ID: "pacts", Title: "Pacts"}
		for _, l := range ls {
			v.Lines = append(v.Lines, fmt.Sprintf("%s (rank %d): %s [%s]", l.Pact, l.Rank, l.Text, l.Status))
		}
		return v
	})
}
