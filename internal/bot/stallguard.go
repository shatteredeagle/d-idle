package bot

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/d2"
)

// Stall diagnostic (Questy death 11, Cave Level 1, 2026-11-26: 14 s with no log line and no survival action while seven hostiles
// killed the hero; the goroutine dump that would say why was never taken). It changes nothing for the hero: when the guard
// sees no log line for guardStallSec while at least guardStallHostiles hostiles are near, it logs ONE `stall: goroutine stack`
// per guardStallGap with the bot goroutines' stacks, counts it and feeds the soft invariant guard-stall (which writes the incident
// file). The numbers are the row of invRows ("guard-stall": Secs, Count) and the gap below.
const guardStallGap = 5 * time.Minute

// logStamp wraps a slog.Handler: every record stamps the time of the last log line (the "no progress" signal of the stall check).
type logStamp struct {
	slog.Handler
	at  *atomic.Int64
	now func() time.Time
}

func (h logStamp) Handle(ctx context.Context, r slog.Record) error {
	h.at.Store(h.now().UnixNano())
	return h.Handler.Handle(ctx, r)
}
func (h logStamp) WithAttrs(a []slog.Attr) slog.Handler {
	return logStamp{h.Handler.WithAttrs(a), h.at, h.now}
}
func (h logStamp) WithGroup(n string) slog.Handler {
	return logStamp{h.Handler.WithGroup(n), h.at, h.now}
}

// stampLog makes the runner's logger stamp lastLogAt on every record.
func (r *Runner) stampLog() {
	r.e.Log = slog.New(logStamp{r.e.Log.Handler(), &r.lastLogAt, func() time.Time { return r.e.Clock.Now() }})
	r.lastLogAt.Store(r.e.Clock.Now().UnixNano())
}

// stallCheck is called by the guard on every look. It returns true when it logged a stall.
func (r *Runner) stallCheck(snap *d2.Snapshot, now time.Time) bool {
	row := invRowOf("guard-stall")
	if r.e.Tune.Bot.Invariants == 0 || snap == nil || snap.Player == nil || snap.Player.InTown() || isDead(snap.Player) {
		return false
	}
	last := time.Unix(0, r.lastLogAt.Load())
	if r.lastLogAt.Load() == 0 || now.Sub(last) < row.dur() {
		return false
	}
	if at := r.lastStallAt.Load(); at != 0 && now.Sub(time.Unix(0, at)) < guardStallGap {
		return false
	}
	if n := r.hostileNear(snap, r.e.Tune.Combat.FightRadius); n < row.Count {
		return false
	}
	n := r.hostileNear(snap, r.e.Tune.Combat.FightRadius)
	r.lastStallAt.Store(now.UnixNano())
	r.stalls.Add(1)
	r.mu.Lock()
	state, activity := string(r.st), r.activity
	r.mu.Unlock()
	age := 0
	if at := r.actionAt.Load(); at != 0 {
		age = int(now.Sub(time.Unix(0, at)).Seconds())
	}
	quiet := int(now.Sub(last).Seconds())
	r.e.Log.Warn("stall: goroutine stack", "quietSec", quiet, "hostiles", n, "state", state, "action", activity, "actionAgeSec", age, "stack", botStacks(60))
	r.inv.noteStall("quiet=" + strconv.Itoa(quiet) + "s hostiles=" + strconv.Itoa(n) + " action=" + activity)
	return true
}

// botStacks is the goroutine profile reduced to the goroutines that run bot code (not the guard's own look), at most lines lines.
func botStacks(lines int) string {
	var b bytes.Buffer
	_ = pprof.Lookup("goroutine").WriteTo(&b, 2)
	var keep []string
	for _, g := range strings.Split(b.String(), "\n\n") {
		if strings.Contains(g, "d2lod-bot/internal/bot.") && !strings.Contains(g, "stallCheck") {
			keep = append(keep, g)
		}
	}
	out := strings.Split(strings.Join(keep, "\n\n"), "\n")
	if len(out) > lines {
		out = out[:lines]
	}
	return strings.Join(out, "\n")
}

// stallKV is the minute line key stalls (only when a stall was logged since the process started).
func (r *Runner) stallKV() []any {
	if n := r.stalls.Load(); n > 0 {
		return []any{"stalls", int(n)}
	}
	return nil
}

// fightProgress remembers, for the running action, the last look at the fight's progress signals.
type fightProgress struct {
	mu     sync.Mutex
	action int64     // actionAt of the action it belongs to
	key    string    // kills, xp and the life of the attacked target at the last change
	since  time.Time // when key last changed
}

// fightStallCheck is the guard's break for a Fighting action that stopped making progress (Doofer 2026-11-26: Fighting in Dark
// Wood for 147 s with 4 hostiles near, xp +260 in 30 minutes). Progress is a kill, an xp change or a drop of the attacked
// target's life. The row fight-stall says how old the action must be (Secs) and how long nothing may change (Count). It returns
// true when the action must be cancelled; the target is blamed in fightTarget and kept out by do() once the action has returned.
// Boss runs and every other state are left alone.
func (r *Runner) fightStallCheck(snap *d2.Snapshot, now time.Time, st State) bool {
	row := invRowOf("fight-stall")
	if r.e.Tune.Bot.Invariants == 0 || st != StateClearing || r.bossRun.Load() || snap == nil || snap.Player == nil || snap.Player.InTown() {
		return false
	}
	r.mu.Lock()
	activity := r.activity
	r.mu.Unlock()
	if !strings.HasPrefix(activity, "Fighting") || r.hostileNear(snap, r.e.Tune.Combat.FightRadius) == 0 {
		return false
	}
	var target uint32
	if a, ok := r.d.Fighter.(interface{ Attacking() uint32 }); ok {
		target = a.Attacking()
	}
	life := -1
	for i := range snap.Monsters {
		if snap.Monsters[i].ID == target && target != 0 && snap.Monsters[i].Monster != nil {
			life = snap.Monsters[i].Monster.LifePct
		}
	}
	key := fmt.Sprintf("%d|%d|%d", r.lifetimeKills(snap.Player.Name), snap.Player.Exp, life)
	at := r.actionAt.Load()
	p := &r.fightProg
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.action != at || p.key != key {
		p.action, p.key, p.since = at, key, now
		return false
	}
	age := now.Sub(time.Unix(0, at))
	if age < row.dur() || now.Sub(p.since) < time.Duration(row.Count)*time.Second {
		return false
	}
	r.fightTarget.Store(target)
	r.e.Log.Warn("fight stall: breaking the action", "actionAgeSec", int(age.Seconds()), "quietSec", int(now.Sub(p.since).Seconds()), "target", target, "activity", activity)
	r.inv.noteFightStall(fmt.Sprintf("age=%ds quiet=%ds target=%d", int(age.Seconds()), int(now.Sub(p.since).Seconds()), target))
	p.action = 0 // the next action starts a new memory
	return true
}

// blameFightTarget keeps the target of a broken fight out of the next fights for the row's Pct minutes (the fighter's own ignore
// list, applied after the action has returned so it never races the fight loop).
func (r *Runner) blameFightTarget() {
	id := r.fightTarget.Swap(0)
	if id == 0 {
		return
	}
	if ig, ok := r.d.Fighter.(interface{ IgnoreFor(uint32, time.Duration) }); ok {
		ig.IgnoreFor(id, time.Duration(invRowOf("fight-stall").Pct)*time.Minute)
	}
}
