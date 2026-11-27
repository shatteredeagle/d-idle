package questboard

import (
	"encoding/json"
	"fmt"
	"time"

	questboarddata "d2lod-bot/data/questboard"
)

// The pause policy (card questresume, knob quests.resumeAfterDeaths): a chunk that parked because the hero died on it
// too often used to wait for the player's Try again for ever (a Questy hero stood idle for 42 h). With the knob on, a
// park whose code has a row in data/questboard/resume.json lifts itself after a quiet stretch, a bounded number of times;
// a park that follows an automatic resume stays. Everything below is pure except ResumeTick, which writes the plan file.

// ResumeRow is one row of resume.json: how long a park of Code waits, the life the hero must have, and the caps.
type ResumeRow struct {
	Code           string `json:"code"`
	AfterMin       int    `json:"afterMin"`       // quiet minutes since the park (no death) before the chunk may resume
	MinLifePct     int    `json:"minLifePct"`     // life percent the hero needs at that moment
	MaxAutoResumes int    `json:"maxAutoResumes"` // automatic resumes in a row; a park after the last one stays for the player
	DayCap         int    `json:"dayCap"`         // automatic resumes of the whole board in 24 hours
}

// ResumePolicy is the parsed resume.json.
type ResumePolicy struct {
	Rows []ResumeRow `json:"rows"`
}

// ParseResume reads and validates resume.json; a bad row is an error naming it.
func ParseResume(raw []byte) (ResumePolicy, error) {
	var f struct {
		Schema string      `json:"schema"`
		Rows   []ResumeRow `json:"rows"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		return ResumePolicy{}, fmt.Errorf("questboard: resume.json: %w", err)
	}
	if f.Schema != "quests.resume/1" {
		return ResumePolicy{}, fmt.Errorf("questboard: resume.json: schema %q, want quests.resume/1", f.Schema)
	}
	seen := map[string]bool{}
	for _, r := range f.Rows {
		info, ok := ReasonOf(r.Code)
		switch {
		case !ok || info.Kind != KindNeeds:
			return ResumePolicy{}, fmt.Errorf("questboard: resume.json: code %q is not a park code", r.Code)
		case seen[r.Code]:
			return ResumePolicy{}, fmt.Errorf("questboard: resume.json: code %q twice", r.Code)
		case r.AfterMin < 1 || r.MinLifePct < 1 || r.MinLifePct > 100 || r.MaxAutoResumes < 1 || r.DayCap < 1:
			return ResumePolicy{}, fmt.Errorf("questboard: resume.json: row %q needs afterMin, minLifePct (1 to 100), maxAutoResumes and dayCap of at least 1", r.Code)
		}
		seen[r.Code] = true
	}
	return ResumePolicy{Rows: f.Rows}, nil
}

// LoadResume parses the embedded resume.json.
func LoadResume() (ResumePolicy, error) { return ParseResume(questboarddata.Resume) }

func (p ResumePolicy) row(code string) (ResumeRow, bool) {
	for _, r := range p.Rows {
		if r.Code == code {
			return r, true
		}
	}
	return ResumeRow{}, false
}

// ResumeOptions switch the policy on (the knob) and carry its table. The zero value resumes nothing.
type ResumeOptions struct {
	Enabled bool
	Policy  ResumePolicy
}

// ResumeInput is what the bot loop knows at a tick.
type ResumeInput struct {
	Paused    bool      // the player's pause is on
	Calm      bool      // not in a fight
	Recovery  bool      // the purse recovery (broke, kit short) is running
	LifePct   int       // current life, percent of the maximum
	LastDeath time.Time // the hero's last death (zero: none this process)
}

// resumeVerdict is the answer of decideResume: Go says the park lifts now; Wait says what it waits for, in words.
type resumeVerdict struct {
	Go    bool
	Final bool // nothing lifts it but the player (the policy is off, has no row, or its cap is spent)
	Wait  string
}

// decideResume is the whole policy in one pure function. dayUsed is the automatic resumes of the board in the last 24 h.
func decideResume(o ResumeOptions, run Run, in ResumeInput, now time.Time, dayUsed int) resumeVerdict {
	if run.Status != RunParked {
		return resumeVerdict{Final: true}
	}
	if !o.Enabled {
		return resumeVerdict{Final: true, Wait: "Only you can clear it: tap Try again. Nothing resumes it by itself (quests.resumeAfterDeaths is off)."}
	}
	row, ok := o.Policy.row(run.Code)
	if !ok {
		return resumeVerdict{Final: true, Wait: "Only you can clear it: tap Try again."}
	}
	if run.AutoResumes >= row.MaxAutoResumes {
		return resumeVerdict{Final: true, Wait: fmt.Sprintf("It already resumed by itself %d time(s) and failed again, so only you can clear it: tap Try again.", run.AutoResumes)}
	}
	if run.ParkedAt.IsZero() {
		return resumeVerdict{Wait: "Resumes by itself after a quiet stretch (the park time is being recorded), or tap Try again now."}
	}
	// the quiet stretch runs from the park, or from a later death anywhere (a death elsewhere says the hero is not fit yet)
	since := run.ParkedAt
	if in.LastDeath.After(since) {
		since = in.LastDeath
	}
	due := since.Add(mins(row.AfterMin))
	switch {
	case now.Before(due):
		return resumeVerdict{Wait: fmt.Sprintf("Resumes by itself in about %d min if the hero does not die and is at full life, or tap Try again now.", int(due.Sub(now).Minutes())+1)}
	case dayUsed >= row.DayCap:
		return resumeVerdict{Wait: fmt.Sprintf("The limit of %d automatic resumes a day is reached; it tries again tomorrow, or tap Try again now.", row.DayCap)}
	case in.Paused:
		return resumeVerdict{Wait: "The bot is paused; it resumes after you resume the bot."}
	case !in.Calm:
		return resumeVerdict{Wait: "Resumes when the fight is over."}
	case in.Recovery:
		return resumeVerdict{Wait: "Resumes when the hero has refilled the purse and restocked."}
	case in.LifePct < row.MinLifePct:
		return resumeVerdict{Wait: fmt.Sprintf("Resumes when the hero is back to %d%% life.", row.MinLifePct)}
	}
	return resumeVerdict{Go: true}
}

// SetResume follows the knob at run time.
func (e *Exec) SetResume(o ResumeOptions) {
	e.mu.Lock()
	e.o.Resume = o
	e.mu.Unlock()
}

// ResumeNote is the "what clears it" sentence of a parked run, for the API and the Return screen.
func (e *Exec) ResumeNote(run Run) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	w := decideResume(e.o.Resume, run, ResumeInput{Calm: true, LifePct: 100}, e.o.Now(), 0).Wait
	if w == "" {
		w = "Resumes by itself as soon as the hero is calm and at full life, or tap Try again now."
	}
	return w
}

// ResumeTick lifts the parks the policy allows, at most one per tick, and logs it. It reports the run it lifted. It does
// nothing without the machine (no stored park to lift) or with the knob off.
func (e *Exec) ResumeTick(in ResumeInput) (Ref, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.o.Enabled || !e.o.SM || !e.o.Resume.Enabled {
		return Ref{}, false
	}
	now := e.o.Now()
	d := Difficulties[e.env.Diff()]
	m, err := e.st.LoadSM(e.smDone())
	if err != nil {
		return Ref{}, false
	}
	plan, _, err := e.st.Load()
	if err != nil {
		return Ref{}, false
	}
	used := 0
	for _, ev := range plan.AutoLog {
		if ev.Kind == EvAutoResume && now.Sub(ev.At) < 24*time.Hour {
			used++
		}
	}
	var pick *Run
	for _, r := range m.Live(e.b, d) {
		r := r
		if r.Status != RunParked {
			continue
		}
		if r.ParkedAt.IsZero() {
			e.stampPark(r.Quest, r.Chunk, d, now) // a park stored before the policy: its quiet stretch starts now
			continue
		}
		if decideResume(e.o.Resume, r, in, now, used).Go {
			pick = &r
			break
		}
	}
	if pick == nil {
		return Ref{}, false
	}
	prior := pick.AutoResumes
	err = e.st.UpdateSM(e.smDone(), func(m *SM) error {
		if _, _, err := m.Retry(e.b, d, pick.Quest, pick.Chunk, false, now); err != nil {
			return err
		}
		if r := m.find(pick.Quest, pick.Chunk, d); r != nil {
			r.AutoResumes = prior + 1
		}
		return nil
	})
	if err != nil {
		e.o.Log("quest board: auto-resume not saved", "chunk", ChunkKey(pick.Quest, pick.Chunk), "err", err)
		return Ref{}, false
	}
	key := ChunkKey(pick.Quest, pick.Chunk)
	delete(e.rs, key)
	if err := e.st.AddAutoEvent(AutoEvent{At: now.UTC(), Quest: pick.Quest, Chunk: pick.Chunk, Kind: EvAutoResume, Away: e.awayAtLocked(now),
		Note: fmt.Sprintf("after %d deaths, %d min quiet, life %d%%", pick.Deaths, int(now.Sub(pick.ParkedAt).Minutes()), in.LifePct)}); err != nil {
		e.o.Log("quest board: auto-resume event not saved", "err", err)
	}
	e.o.Log("quest board: park lifted by the pause policy", "chunk", key, "code", pick.Code, "autoResumes", prior+1)
	if u, ok := e.act.(unparker); ok {
		if q, err := e.b.Quest(pick.Quest); err == nil {
			if c, err := q.ChunkByID(pick.Chunk); err == nil {
				u.Unpark(q, c)
			}
		}
	}
	return Ref{pick.Quest, pick.Chunk}, true
}

func (e *Exec) stampPark(quest, chunk, diff string, at time.Time) {
	_ = e.st.UpdateSM(e.smDone(), func(m *SM) error {
		if r := m.find(quest, chunk, diff); r != nil && r.ParkedAt.IsZero() {
			r.ParkedAt = at.UTC()
		}
		return nil
	})
}

func (e *Exec) awayAtLocked(now time.Time) bool {
	e.ax.am.Lock()
	defer e.ax.am.Unlock()
	return e.awayAt(now, e.o.Auto)
}
