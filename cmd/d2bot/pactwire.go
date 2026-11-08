package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
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
// Presence of a human is not detected yet (TASK-168/169): Pacts act only on a typed away-consent (U).
func startPacts(ctx context.Context, l ladderOptions, rig *ladderRig, m *goals.Manager, status func() goals.Status, stateDir string, getenv func(string) string) {
	if pact.Off(getenv) || rig == nil || rig.Fun == nil || rig.Fun.Policy == nil || stateDir == "" || m == nil || status == nil {
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
						ctl, err = pact.New(pact.Config{
							Cat: cat, Class: class, Path: filepath.Join(stateDir, "pacts-state.json"),
							Knobs: pact.KnobSetter(l.knobs), Sheet: pactSheet{rig.Fun.Policy},
							Pause: func(reason string) error { rig.Hub.Incident("Pact: " + reason); return m.Pause() },
						})
					}
					if err != nil {
						rig.Hub.Incident("Pacts are off: " + err.Error())
						return
					}
					rig.Fun.Return.Sections.Register("pacts", pactSection(ctl))
				}
				paused := st.State == goals.RunnerPaused
				if wasPaused && !paused {
					ctl.Resume() // the player took the hero back
				}
				wasPaused = paused
				w := pact.Where{Level: hero.Level, Difficulty: difficultyOf(hero.Difficulty), InTown: towns[hero.Area], InDen: hero.Area == "Den of Evil", Age: time.Since(start)}
				if err := ctl.Evaluate(w); err != nil && err.Error() != lastErr {
					rig.Hub.Incident("Pact: " + err.Error())
					lastErr = err.Error() // said once, not every tick
				}
			}
		}
	}()
}

// pactSection is the return screen's Pact lines: what each chosen Pact does and whether it acts now.
func pactSection(ctl *pact.Controller) watch.ReturnSection {
	return watch.SectionFunc(func(*watch.ReturnData) *watch.SectionView {
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
