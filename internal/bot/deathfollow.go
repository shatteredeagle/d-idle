package bot

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
)

// Death follow-up: two facts are only known after the death (gold on the next minute line, the time the character is
// back in the field). They are written onto the in-memory record (deaths.json, /deaths API) and appended to
// deaths.jsonl as a follow-up line {"followUp":true,"time":<death time>,...}, because the jsonl is append-only.

const routeKeep = 6

// deathFollow tracks one death waiting for its follow-up facts.
type deathFollow struct {
	at    time.Time
	gold  *int
	field *time.Time
	secs  *int
}

func (o *observer) noteRoute(area int) {
	if area == 0 {
		return
	}
	if n := len(o.recent); n > 0 && o.recent[n-1] == area {
		return
	}
	o.recent = append(o.recent, area)
	if len(o.recent) > routeKeep {
		o.recent = o.recent[len(o.recent)-routeKeep:]
	}
}

func (o *observer) routeID() string {
	parts := make([]string, len(o.recent))
	for i, a := range o.recent {
		parts[i] = strconv.Itoa(a)
	}
	return strings.Join(parts, ">")
}

// addFollow registers a fresh death. Called from recordDeath with obMu held.
func (r *Runner) addFollow(at time.Time) {
	o := &r.ob
	o.pendingDeaths = append(o.pendingDeaths, &deathFollow{at: at})
	if len(o.pendingDeaths) > 4 { // never grows: the oldest is flushed with what it has
		r.flushFollow(o.pendingDeaths[0])
		o.pendingDeaths = o.pendingDeaths[1:]
	}
}

// noteGoldNextMinute fills goldNextMinute on every pending death that has none. Caller holds obMu.
func (r *Runner) noteGoldNextMinute(gold int, now time.Time) {
	for _, f := range r.ob.pendingDeaths {
		if f.gold == nil && now.After(f.at) {
			g := gold
			f.gold = &g
		}
	}
	r.settleFollows()
}

// noteBackInField stamps the first living snapshot outside town after a death. Caller holds obMu.
func (r *Runner) noteBackInField(p *d2.Player, now time.Time) {
	if len(r.ob.pendingDeaths) == 0 || p.InTown() {
		return
	}
	for _, f := range r.ob.pendingDeaths {
		if f.field == nil && now.After(f.at) {
			ts := now
			secs := int(now.Sub(f.at).Round(time.Second) / time.Second)
			f.field, f.secs = &ts, &secs
		}
	}
	r.settleFollows()
}

func (r *Runner) settleFollows() {
	o := &r.ob
	keep := o.pendingDeaths[:0]
	for _, f := range o.pendingDeaths {
		if f.gold != nil && f.field != nil {
			r.flushFollow(f)
			continue
		}
		keep = append(keep, f)
	}
	o.pendingDeaths = keep
}

type followLine struct {
	FollowUp       bool       `json:"followUp"`
	Time           time.Time  `json:"time"` // death time, the key of the record
	GoldNextMinute *int       `json:"goldNextMinute"`
	BackInFieldTs  *time.Time `json:"backInFieldTs"`
	SecondsToField *int       `json:"secondsToField"`
}

func (r *Runner) flushFollow(f *deathFollow) {
	for i := range r.dlog.Records {
		if rc := &r.dlog.Records[i]; rc.Time.Equal(f.at) && rc.Autopsy != nil {
			rc.Autopsy.GoldNextMinute, rc.Autopsy.BackInFieldTs, rc.Autopsy.SecondsToField = f.gold, f.field, f.secs
		}
	}
	if err := r.dlog.save(r.cfg.DeathLogPath); err != nil {
		r.e.Log.Warn("death log not saved", "err", err)
	}
	path := r.cfg.DeathLogPath
	if path == "" {
		return
	}
	path = strings.TrimSuffix(path, ".json") + ".jsonl"
	b, err := json.Marshal(followLine{FollowUp: true, Time: f.at, GoldNextMinute: f.gold, BackInFieldTs: f.field, SecondsToField: f.secs})
	if err != nil {
		return
	}
	fh, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	_, _ = fh.Write(append(b, '\n'))
	fh.Close()
}

// noteGoldRecovered records the gold the sweep took after a death: on the record (deaths.json) and as one more
// follow-up line {"followUp":true,"time":<death time>,"goldRecovered":N} in deaths.jsonl (additive, docs/deathautopsy.md).
// Caller holds obMu.
func (r *Runner) noteGoldRecovered(at time.Time, picked int) {
	for i := range r.dlog.Records {
		if rc := &r.dlog.Records[i]; rc.Time.Equal(at) && rc.Autopsy != nil {
			rc.Autopsy.GoldRecovered = &picked
		}
	}
	if err := r.dlog.save(r.cfg.DeathLogPath); err != nil {
		r.e.Log.Warn("death log not saved", "err", err)
	}
	path := r.cfg.DeathLogPath
	if path == "" {
		return
	}
	path = strings.TrimSuffix(path, ".json") + ".jsonl"
	b, err := json.Marshal(struct {
		FollowUp      bool      `json:"followUp"`
		Time          time.Time `json:"time"`
		GoldRecovered int       `json:"goldRecovered"`
	}{true, at, picked})
	if err != nil {
		return
	}
	if fh, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		_, _ = fh.Write(append(b, '\n'))
		fh.Close()
	}
}
