package main

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"d2lod-bot/internal/selfupdate"
)

// The boot shim (TASK-123; docs/self-update-drill.md, docs/research/self-update-design.md 3.1, 3.8, 3.9). It is the
// part of the app that an update can NOT change: it picks the release to run, probes its health, and goes back to the
// previous release when the new one is not healthy. A broken release is never trusted to roll itself back.
// Everything that touches the machine (clock, child process, probes) is an interface so the logic is tested with a
// fake clock and fake probes.

// Release is one runnable bundle.
type Release struct {
	Version string
	Base    bool
	BinDir  string
	D2DDir  string
	Entry   string // the launcher that holds the mode logic (entrypoint.sh)
}

// Image describes what is baked into the image.
type Image struct {
	ABI         int
	BaseVersion string
	Base        Release
	Local       selfupdate.Compat
	Keys        []ed25519.PublicKey
	WatchWanted bool // the tester profile: the watch screen must answer too
}

// Policy is the health rule set (design 3.8).
type Policy struct {
	Step         time.Duration // poll interval (stop and child exit are seen within one step)
	ProbeEvery   time.Duration // probes run this often (a probe may start a process)
	BotAPIWithin time.Duration // the bot API answers
	TickWithin   time.Duration // d2d's counter advances
	Stable       time.Duration // stays up without exiting, then the release is good
	MaxBoots     int           // the Nth boot of a pending release rolls it back before it starts
	TermGrace    time.Duration // after TERM, wait this long before KILL
}

// DefaultPolicy: 60 s, 3 min, 10 min, third boot, 45 s (the compose stop_grace_period).
func DefaultPolicy() Policy {
	return Policy{Step: time.Second, ProbeEvery: 5 * time.Second, BotAPIWithin: time.Minute, TickWithin: 3 * time.Minute, Stable: 10 * time.Minute, MaxBoots: 3, TermGrace: 45 * time.Second}
}

type Clock interface {
	Now() time.Time
	Sleep(time.Duration)
}

type Probes interface {
	BotAPI() bool
	Tick() (uint64, bool) // d2d's frame/tick counter; ok false: could not be read
	Watch() bool
}

type Child interface {
	Exited() (code int, done bool)
	Terminate()
	Kill()
}

type Runner interface {
	Start(r Release) (Child, error)
}

// Boot is the shim.
type Boot struct {
	Layout selfupdate.Layout
	Image  Image
	Policy Policy
	Clock  Clock
	Probe  Probes
	Runner Runner
	Stop   func() bool // true once the shim itself was asked to stop (container stop)
	Log    func(format string, a ...any)
}

func (b *Boot) logf(f string, a ...any) {
	if b.Log != nil {
		b.Log(f, a...)
	}
}

// Skipped is a stored release that could not be used, and why.
type Skipped struct {
	Version string `json:"version"`
	Reason  string `json:"reason"`
}

// ImageStatus is /data/didle/image.json: what the shim ran on and what it had to leave out.
type ImageStatus struct {
	ABI           int       `json:"abi"`
	BaseVersion   string    `json:"base_version"`
	Keys          []string  `json:"keys"`
	Running       string    `json:"running"`
	Changed       []string  `json:"changed,omitempty"` // abi, base or keys differ from the previous start
	Skipped       []Skipped `json:"skipped,omitempty"`
	RebuildNeeded bool      `json:"rebuild_needed"`
	Reason        string    `json:"reason,omitempty"`
}

func (b *Boot) loadStored(v string) (Release, error) {
	dir := b.Layout.Release(v)
	raw, err := os.ReadFile(filepath.Join(dir, selfupdate.ManifestName))
	if err != nil {
		return Release{}, err
	}
	sig, err := os.ReadFile(filepath.Join(dir, selfupdate.ManifestName+".sig"))
	if err != nil {
		return Release{}, err
	}
	if err := selfupdate.Verify(b.Image.Keys, raw, sig); err != nil {
		return Release{}, fmt.Errorf("manifest signature: %w", err)
	}
	m, err := selfupdate.ParseManifest(raw)
	if err != nil {
		return Release{}, err
	}
	if m.Version != v {
		return Release{}, fmt.Errorf("directory %s holds manifest %s", v, m.Version)
	}
	if err := selfupdate.Gate(m, b.Image.ABI, b.Image.Local); err != nil {
		return Release{}, err
	}
	r := Release{Version: v, BinDir: filepath.Join(dir, "bin"), D2DDir: filepath.Join(dir, "d2d")}
	if _, err := os.Stat(filepath.Join(r.BinDir, "d2bot")); err != nil {
		return Release{}, fmt.Errorf("release %s has no bin/d2bot", v)
	}
	// the release carries the mode logic; an older release without it uses the baked launcher
	r.Entry = filepath.Join(r.BinDir, "entrypoint.sh")
	if _, err := os.Stat(r.Entry); err != nil {
		r.Entry = b.Image.Base.Entry
	}
	return r, nil
}

// Select returns the highest version among {current, previous, base} that is usable: not on the bad list, signed by
// a baked key, inside the compatibility gate. The base wins a tie and is the last resort (design 3.1).
func (b *Boot) Select(st selfupdate.State) (Release, []Skipped) {
	best := b.Image.Base
	best.Version, best.Base = b.Image.BaseVersion, true
	var skipped []Skipped
	seen := map[string]bool{}
	for _, v := range []string{st.Current, st.Previous} {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		if st.IsBad(v) {
			skipped = append(skipped, Skipped{v, "on the bad list"})
			continue
		}
		r, err := b.loadStored(v)
		if err != nil {
			skipped = append(skipped, Skipped{v, err.Error()})
			continue
		}
		if selfupdate.CompareVersions(r.Version, best.Version) > 0 {
			best = r
		}
	}
	return best, skipped
}

func (b *Boot) writeImageStatus(running string, skipped []Skipped) {
	path := filepath.Join(b.Layout.Root, "image.json")
	cur := ImageStatus{ABI: b.Image.ABI, BaseVersion: b.Image.BaseVersion, Running: running, Skipped: skipped}
	for _, k := range b.Image.Keys {
		cur.Keys = append(cur.Keys, selfupdate.Fingerprint(k))
	}
	var prev ImageStatus
	if raw, err := os.ReadFile(path); err == nil && json.Unmarshal(raw, &prev) == nil {
		if prev.ABI != cur.ABI {
			cur.Changed = append(cur.Changed, "abi")
		}
		if prev.BaseVersion != cur.BaseVersion {
			cur.Changed = append(cur.Changed, "base")
		}
		if fmt.Sprint(prev.Keys) != fmt.Sprint(cur.Keys) {
			cur.Changed = append(cur.Changed, "keys")
		}
	}
	for _, s := range skipped {
		if s.Reason != "on the bad list" {
			cur.RebuildNeeded, cur.Reason = true, fmt.Sprintf("release %s cannot run on this image: %s", s.Version, s.Reason)
			break
		}
	}
	raw, _ := json.MarshalIndent(cur, "", "  ")
	os.MkdirAll(b.Layout.Root, 0o755)
	os.WriteFile(path, append(raw, '\n'), 0o644)
}

type outcome int

const (
	outStopped outcome = iota // the shim was asked to stop
	outExited                 // the child ended on its own
	outUnhealthy
)

// Run supervises until the child ends on its own (exit code returned, the container restart policy takes over) or
// the shim is asked to stop. It restarts the previous release in-process after a rollback.
func (b *Boot) Run() int {
	for round := 0; ; round++ {
		st, err := b.Layout.LoadState()
		if err != nil {
			b.logf("state unreadable (%v); running the base release", err)
			st = selfupdate.State{}
		}
		rel, skipped := b.Select(st)
		b.writeImageStatus(rel.Version, skipped)
		pending := !rel.Base && st.Pending == rel.Version && st.Pending != ""
		if st.Pending != "" && !pending { // the pending release cannot run here (newer base, gate): do not judge a later one by it
			st.Pending, st.Boots = "", 0
			b.Layout.SaveState(st)
		}
		if pending {
			st.Boots++
			b.Layout.SaveState(st)
			if st.Boots >= b.Policy.MaxBoots {
				b.rollback(rel.Version, st.Previous, fmt.Sprintf("crash loop: boot %d of an unconfirmed release", st.Boots))
				continue
			}
		}
		b.logf("starting release %s (base=%v pending=%v)", rel.Version, rel.Base, pending)
		child, err := b.Runner.Start(rel)
		if err != nil {
			if pending {
				b.rollback(rel.Version, st.Previous, "start failed: "+err.Error())
				continue
			}
			b.logf("start failed: %v", err)
			return 1
		}
		out, code, why, confirmed := b.watch(child, rel, pending)
		switch out {
		case outStopped:
			return code
		case outExited:
			if pending && !confirmed {
				b.rollback(rel.Version, st.Previous, fmt.Sprintf("exited with code %d before it was confirmed healthy", code))
				continue
			}
			return code
		case outUnhealthy:
			b.stopChild(child)
			b.rollback(rel.Version, st.Previous, why)
		}
		if round > 8 {
			b.logf("too many rollbacks in one start; giving up")
			return 1
		}
	}
}

func (b *Boot) rollback(from, to, reason string) {
	b.logf("rolling back %s to %q: %s", from, to, reason)
	if err := b.Layout.Rollback(from, to, reason, b.Clock.Now()); err != nil {
		b.logf("rollback bookkeeping failed: %v", err)
	}
}

// stopChild asks the child to stop (TERM), waits for the grace period, then kills it.
func (b *Boot) stopChild(c Child) int {
	c.Terminate()
	waited := time.Duration(0)
	for waited < b.Policy.TermGrace {
		if code, ok := c.Exited(); ok {
			return code
		}
		b.Clock.Sleep(b.Policy.Step)
		waited += b.Policy.Step
	}
	c.Kill()
	for i := 0; i < 20; i++ {
		if code, ok := c.Exited(); ok {
			return code
		}
		b.Clock.Sleep(100 * time.Millisecond)
	}
	return 137
}

// watch polls the child. For a pending release it also runs the health rules; after Stable the release becomes good.
func (b *Boot) watch(c Child, rel Release, probation bool) (out outcome, code int, why string, confirmed bool) {
	p := b.Policy
	start := b.Clock.Now()
	var apiOK, tickOK, watchOK, haveTick bool
	var lastTick uint64
	var lastProbe time.Time
	for {
		if b.Stop != nil && b.Stop() {
			return outStopped, b.stopChild(c), "", confirmed
		}
		if code, ok := c.Exited(); ok {
			return outExited, code, "", confirmed
		}
		if probation {
			now := b.Clock.Now()
			el := now.Sub(start)
			if lastProbe.IsZero() || now.Sub(lastProbe) >= p.ProbeEvery {
				lastProbe = now
				if !apiOK && b.Probe.BotAPI() {
					apiOK = true
				}
				if !tickOK {
					if t, ok := b.Probe.Tick(); ok {
						if haveTick && t > lastTick {
							tickOK = true
						}
						haveTick, lastTick = true, t
					}
				}
				if !watchOK && (!b.Image.WatchWanted || b.Probe.Watch()) {
					watchOK = true
				}
			}
			switch {
			case !apiOK && el > p.BotAPIWithin:
				return outUnhealthy, 0, fmt.Sprintf("the bot API did not answer within %s", p.BotAPIWithin), false
			case !tickOK && el > p.TickWithin:
				return outUnhealthy, 0, fmt.Sprintf("d2d did not advance its counter within %s", p.TickWithin), false
			case !watchOK && el > p.TickWithin:
				return outUnhealthy, 0, fmt.Sprintf("the watch screen did not answer within %s", p.TickWithin), false
			case el >= p.Stable && apiOK && tickOK && watchOK:
				b.markGood(rel.Version)
				probation, confirmed = false, true
			}
		}
		b.Clock.Sleep(p.Step)
	}
}

func (b *Boot) markGood(v string) {
	st, err := b.Layout.LoadState()
	if err != nil {
		return
	}
	st.Good, st.Pending, st.Boots = v, "", 0
	b.Layout.SaveState(st)
	b.Layout.Prune(st.Current, st.Previous)
	b.logf("release %s confirmed healthy", v)
}
