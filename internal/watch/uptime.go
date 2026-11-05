package watch

import (
	"fmt"
	"time"
)

// Activity kinds the bot reports. Uptime is the share of counted time spent making progress (ADR 0016 item 22):
// progress over progress + death + recovery + stall. Idle (no order) is not counted at all.
const (
	KindProgress = "progress"
	KindDeath    = "death"
	KindRecovery = "recovery"
	KindStall    = "stall" // town stalls: waiting in town with nothing to do, blocked, looping
	KindIdle     = "idle"
)

var activityKinds = map[string]bool{KindProgress: true, KindDeath: true, KindRecovery: true, KindStall: true, KindIdle: true}

type segment struct {
	from  time.Duration
	kind  string
	label string
}

// uptimeTracker keeps the activity timeline. Not safe for concurrent use: the Hub locks it.
type uptimeTracker struct {
	segs   []segment
	deaths int
}

const maxSegments = 20000

func (u *uptimeTracker) set(now time.Duration, kind, label string) {
	if n := len(u.segs); n > 0 {
		last := u.segs[n-1]
		if last.kind == kind && last.label == label {
			return
		}
		if now < last.from {
			now = last.from
		}
	}
	if kind == KindDeath {
		u.deaths++
	}
	u.segs = append(u.segs, segment{from: now, kind: kind, label: label})
	if len(u.segs) > maxSegments { // forget the oldest half; the totals then cover the kept span
		u.segs = append([]segment(nil), u.segs[len(u.segs)/2:]...)
	}
}

// sums returns the time per kind inside [from, now].
func (u *uptimeTracker) sums(from, now time.Duration) map[string]time.Duration {
	out := map[string]time.Duration{}
	for i, s := range u.segs {
		end := now
		if i+1 < len(u.segs) {
			end = u.segs[i+1].from
		}
		a := max(s.from, from)
		if end > a {
			out[s.kind] += end - a
		}
	}
	return out
}

func pct(s map[string]time.Duration) *float64 {
	lost := s[KindDeath] + s[KindRecovery] + s[KindStall]
	total := s[KindProgress] + lost
	if total <= 0 {
		return nil
	}
	v := float64(int(float64(s[KindProgress])/float64(total)*1000+0.5)) / 10
	return &v
}

// UptimeView is GET /api/uptime.
type UptimeView struct {
	Schema      string   `json:"schema"`
	Seq         uint64   `json:"seq"`
	Now         float64  `json:"now"`
	State       string   `json:"state"`
	Label       string   `json:"label"`
	SinceSec    float64  `json:"sinceSec"`
	UptimePct   *float64 `json:"uptimePct"` // null until some time has been counted
	WindowPct   *float64 `json:"windowPct"` // the last WindowSec
	WindowSec   int      `json:"windowSec"`
	ProgressSec float64  `json:"progressSec"`
	DeathSec    float64  `json:"deathSec"`
	RecoverySec float64  `json:"recoverySec"`
	StallSec    float64  `json:"stallSec"`
	Deaths      int      `json:"deaths"`
	Text        string   `json:"text"`
}

const uptimeSchema = "watch.uptime/1"

func secs(d time.Duration) float64 { return float64(d/time.Millisecond) / 1000 }

func (u *uptimeTracker) view(now time.Duration, window int) UptimeView {
	v := UptimeView{Schema: uptimeSchema, Now: secs(now), WindowSec: window, State: KindIdle, Label: "waiting for an order", Deaths: u.deaths}
	if n := len(u.segs); n > 0 {
		last := u.segs[n-1]
		v.State, v.Label, v.SinceSec = last.kind, last.label, secs(now-last.from)
		all := u.sums(u.segs[0].from, now)
		v.UptimePct = pct(all)
		v.ProgressSec, v.DeathSec, v.RecoverySec, v.StallSec = secs(all[KindProgress]), secs(all[KindDeath]), secs(all[KindRecovery]), secs(all[KindStall])
		v.WindowPct = pct(u.sums(now-time.Duration(window)*time.Second, now))
	}
	switch {
	case v.UptimePct == nil:
		v.Text = "No time counted yet."
	default:
		v.Text = fmt.Sprintf("The hero made progress %.1f%% of the counted time; deaths, recovery and town stalls lower it.", *v.UptimePct)
	}
	return v
}
