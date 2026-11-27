package questboard

import (
	"strings"
	"testing"
	"time"
)

// The pause policy (card questresume). Every test runs for two classes and on a fake clock.

type clock struct{ t time.Time }

func (c *clock) now() time.Time      { return c.t }
func (c *clock) add(d time.Duration) { c.t = c.t.Add(d) }
func (c *clock) at(m int) time.Time  { return c.t.Add(time.Duration(m) * time.Minute) }

func resumeOpts(t *testing.T, on bool) ResumeOptions {
	t.Helper()
	p, err := LoadResume()
	if err != nil {
		t.Fatal(err)
	}
	return ResumeOptions{Enabled: on, Policy: p}
}

// diedRig parks den_of_evil/clear_den with two deaths (the cap) on a fake clock.
func diedRig(t *testing.T, class string, on bool) (*rig, *clock) {
	t.Helper()
	r := newRigSM(t, class, 6, true)
	ck := &clock{t: time.Date(2026, 11, 25, 22, 0, 0, 0, time.UTC)}
	r.x.o.Now = ck.now
	r.x.SetResume(resumeOpts(t, on))
	if err := r.x.Accept(den, clr); err != nil {
		t.Fatal(err)
	}
	r.f.Script = []StepResult{{StepDied, "Fallen pack", ""}, {StepDied, "Fallen pack", ""}}
	r.tick(2)
	if d := r.next(); d.Kind != DecBlocked || !strings.Contains(d.Why, "died 2 times") {
		t.Fatalf("not parked by deaths: %+v", d)
	}
	return r, ck
}

func fit() ResumeInput { return ResumeInput{Calm: true, LifePct: 100} }

func TestResumeKnobOffNeverLifts(t *testing.T) {
	for _, class := range smClasses {
		t.Run(class, func(t *testing.T) {
			r, ck := diedRig(t, class, false)
			ck.add(48 * time.Hour)
			if _, ok := r.x.ResumeTick(fit()); ok {
				t.Fatal("the knob is off and the park lifted")
			}
			if d := r.next(); d.Kind != DecBlocked {
				t.Fatalf("park gone: %+v", d)
			}
			if n := r.x.ResumeNote(r.x.runOf(t, den, clr)); !strings.Contains(n, "Only you can clear it") || !strings.Contains(n, "off") {
				t.Fatalf("note does not say the park waits for the player: %q", n)
			}
		})
	}
}

func TestResumeAfterQuietStretch(t *testing.T) {
	for _, class := range smClasses {
		t.Run(class, func(t *testing.T) {
			r, ck := diedRig(t, class, true)
			ck.add(10 * time.Minute)
			if _, ok := r.x.ResumeTick(fit()); ok {
				t.Fatal("lifted before afterMin")
			}
			ck.add(40 * time.Minute) // 50 min since the park
			for name, in := range map[string]ResumeInput{
				"low life":    {Calm: true, LifePct: 60},
				"fight":       {LifePct: 100},
				"recovery":    {Calm: true, LifePct: 100, Recovery: true},
				"bot paused":  {Calm: true, LifePct: 100, Paused: true},
				"died lately": {Calm: true, LifePct: 100, LastDeath: ck.at(-5)},
			} {
				if _, ok := r.x.ResumeTick(in); ok {
					t.Fatalf("lifted although the hero is not fit (%s)", name)
				}
			}
			ref, ok := r.x.ResumeTick(fit())
			if !ok || ref.Quest != den || ref.Chunk != clr {
				t.Fatalf("not lifted: %v %v", ref, ok)
			}
			if d := r.next(); d.Kind != DecRun {
				t.Fatalf("chunk does not run after the resume: %+v", d)
			}
			run := r.x.runOf(t, den, clr)
			if run.AutoResumes != 1 || run.Status.Live() == false {
				t.Fatalf("run after resume: %+v", run)
			}
			evs, _, _ := r.x.st.Load()
			if n := len(evs.AutoLog); n != 1 || evs.AutoLog[0].Kind != EvAutoResume {
				t.Fatalf("no ledger event: %+v", evs.AutoLog)
			}
		})
	}
}

// A death anywhere after the park starts the quiet stretch over.
func TestResumeDeathRestartsTheStretch(t *testing.T) {
	for _, class := range smClasses {
		t.Run(class, func(t *testing.T) {
			r, ck := diedRig(t, class, true)
			ck.add(40 * time.Minute)
			death := ck.t
			ck.add(30 * time.Minute) // 70 min since the park, 30 since the death
			in := fit()
			in.LastDeath = death
			if _, ok := r.x.ResumeTick(in); ok {
				t.Fatal("lifted 30 min after a death")
			}
			ck.add(20 * time.Minute)
			if _, ok := r.x.ResumeTick(in); !ok {
				t.Fatal("not lifted 50 min after the death")
			}
		})
	}
}

// Never when the same chunk killed the hero twice in a row: a park that follows an automatic resume stays.
func TestResumeNeverTwiceInARow(t *testing.T) {
	for _, class := range smClasses {
		t.Run(class, func(t *testing.T) {
			r, ck := diedRig(t, class, true)
			ck.add(50 * time.Minute)
			if _, ok := r.x.ResumeTick(fit()); !ok {
				t.Fatal("first resume did not happen")
			}
			r.f.Script = []StepResult{{StepDied, "Fallen pack", ""}, {StepDied, "Fallen pack", ""}}
			r.tick(2)
			if d := r.next(); d.Kind != DecBlocked {
				t.Fatalf("not parked again: %+v", d)
			}
			ck.add(24 * time.Hour)
			if _, ok := r.x.ResumeTick(fit()); ok {
				t.Fatal("resumed a second time in a row")
			}
			n := r.x.ResumeNote(r.x.runOf(t, den, clr))
			if !strings.Contains(n, "already resumed by itself") || !strings.Contains(n, "Try again") {
				t.Fatalf("note: %q", n)
			}
			r.x.Unblock(den, clr) // the player's tap clears the count
			if got := r.x.runOf(t, den, clr).AutoResumes; got != 0 {
				t.Fatalf("a player's retry kept the count: %d", got)
			}
		})
	}
}

func TestResumeDayCap(t *testing.T) {
	o := resumeOpts(t, true)
	row, _ := o.Policy.row("died_too_often")
	run := Run{Status: RunParked, Code: "died_too_often", ParkedAt: time.Unix(0, 0)}
	now := time.Unix(0, 0).Add(time.Hour)
	if v := decideResume(o, run, fit(), now, row.DayCap); v.Go || !strings.Contains(v.Wait, "a day") {
		t.Fatalf("day cap not applied: %+v", v)
	}
	if v := decideResume(o, run, fit(), now, row.DayCap-1); !v.Go {
		t.Fatalf("under the cap: %+v", v)
	}
	if v := decideResume(o, Run{Status: RunParked, Code: "gave_up", ParkedAt: run.ParkedAt}, fit(), now, 0); v.Go || !v.Final {
		t.Fatalf("a code without a row must wait for the player: %+v", v)
	}
}

func TestParseResumeRejects(t *testing.T) {
	for name, raw := range map[string]string{
		"schema": `{"schema":"x","rows":[]}`,
		"code":   `{"schema":"quests.resume/1","rows":[{"code":"nope","afterMin":1,"minLifePct":50,"maxAutoResumes":1,"dayCap":1}]}`,
		"hold":   `{"schema":"quests.resume/1","rows":[{"code":"too_weak","afterMin":1,"minLifePct":50,"maxAutoResumes":1,"dayCap":1}]}`,
		"zero":   `{"schema":"quests.resume/1","rows":[{"code":"died_too_often","afterMin":0,"minLifePct":50,"maxAutoResumes":1,"dayCap":1}]}`,
		"twice":  `{"schema":"quests.resume/1","rows":[{"code":"gave_up","afterMin":1,"minLifePct":50,"maxAutoResumes":1,"dayCap":1},{"code":"gave_up","afterMin":1,"minLifePct":50,"maxAutoResumes":1,"dayCap":1}]}`,
	} {
		if _, err := ParseResume([]byte(raw)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// The API: a death park reads paused_after_deaths with the park time and what clears it; another offer says why Start is held.
func TestDeskShowsPausedAfterDeaths(t *testing.T) {
	for _, class := range smClasses {
		t.Run(class, func(t *testing.T) {
			r, ck := diedRig(t, class, true)
			d := r.desk(nil)
			v := d.viewFresh()
			row, ok := findRow(v, den, clr)
			if !ok || row.Reason == nil || row.Reason.Code != "paused_after_deaths" {
				t.Fatalf("row reason: %+v", row.Reason)
			}
			if row.Reason.Since == nil || !row.Reason.Since.Equal(ck.t) {
				t.Fatalf("since: %v want %v", row.Reason.Since, ck.t)
			}
			if !strings.Contains(row.Reason.Clears, "Resumes by itself in about") {
				t.Fatalf("clears: %q", row.Reason.Clears)
			}
			if v.Blocked == nil || v.Blocked.Reason == nil || v.Blocked.Reason.Code != "paused_after_deaths" {
				t.Fatalf("blocked run: %+v", v.Blocked)
			}
			var needs *NeedRow
			for i := range v.Needs {
				if v.Needs[i].Kind == "quest_blocked" {
					needs = &v.Needs[i]
				}
			}
			if needs == nil || !strings.Contains(needs.Text, "paused") || needs.Sub == "" {
				t.Fatalf("needs row: %+v", needs)
			}
			if len(v.Offers) == 0 {
				t.Fatal("no offers to check")
			}
			for _, o := range v.Offers {
				if o.Startable || !strings.Contains(o.Why, "paused and holds the board") {
					t.Fatalf("offer %s/%s: startable %v why %q", o.Quest, o.Chunk, o.Startable, o.Why)
				}
			}
		})
	}
}

// runOf is the stored run of a chunk (test helper).
func (e *Exec) runOf(t *testing.T, quest, chunk string) Run {
	t.Helper()
	r, ok := e.SMRuns()[ChunkKey(quest, chunk)]
	if !ok {
		t.Fatalf("no run for %s/%s", quest, chunk)
	}
	return r
}

func TestParkLinesForTheReturnScreen(t *testing.T) {
	for _, class := range smClasses {
		t.Run(class, func(t *testing.T) {
			r, ck := diedRig(t, class, true)
			d := r.desk(nil)
			lines := d.ParkLines()
			if len(lines) != 1 || !strings.Contains(lines[0], "paused") || !strings.Contains(lines[0], "Since 2026-11-25 22:00 UTC") || !strings.Contains(lines[0], "Resumes by itself") {
				t.Fatalf("lines: %q", lines)
			}
			ck.add(time.Hour)
			if _, ok := r.x.ResumeTick(fit()); !ok {
				t.Fatal("not lifted")
			}
			d.invalidate()
			if l := d.ParkLines(); len(l) != 0 {
				t.Fatalf("a lifted park still shows: %q", l)
			}
		})
	}
}
