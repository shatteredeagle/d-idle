// Package d2dsrc builds d2.Snapshot values from a running d2d over its control channel (internal/d2dctl),
// with no access to process memory. It is the read side of the d2d backend (docs/d2d-backend.md section 5,
// phase 1): additive, not wired into cmd/d2bot, and it never sends a verb that changes the game.
//
// Honesty rule: whatever d2d cannot tell us is listed in Report.Unknown, counted in Source.UnknownCounts and
// logged once, never presented as a real zero. Where a d2 type has a nil-able or sentinel representation
// (pointers, -1) that is used; otherwise the zero value stays but the Report marks it.
package d2dsrc

import (
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dctl"
)

// Synthetic unit ids for things d2d identifies by something other than a unit id (V that the verbs take
// these indices; the ranges are ours and cannot collide with d2d's small unit ids, U that those stay below
// 0x10000000). The input adapter decodes them.
const (
	NPCIDBase    = 0x10000000 // + "npcs" index (what "cmd interact" takes)
	MercID       = 0x20000000 // the dump has no merc unit id
	PortalIDBase = 0x30000000 // + "debug portals" row
)

// Options tune a Source.
type Options struct {
	Timeout time.Duration // per-snapshot budget (0 = 10 s)
	Tables  Tables        // nil: item rows and monster classes are unknown
	// Logf receives the one-time unknown-field notes and degradation notices. nil = log.Printf, use
	// func(string, ...any) {} to silence.
	Logf func(string, ...any)
	// AddOrigin adds the act-tile origin of "debug level" (x5 to subtiles) to every position. d2d cells are
	// level-relative (V that debug players == debug level position); whether our nav wants absolute act
	// subtiles and whether the origin is in tiles are both U, so this is off by default.
	AddOrigin bool
	// Corpses and Warps fetch the extra lists into Result.Extras (one more round trip each).
	Corpses, Warps bool
	// NoPrefetch turns the one-write batch off (one round trip per verb, the pre-D2DSPEED behaviour).
	NoPrefetch bool
	// NoInvHint leaves the inventory flag unknown when dump ui has no rects instead of tracking the i key.
	NoInvHint bool
}

// Source reads snapshots from one d2d socket.
type Source struct {
	c   *d2dctl.Client
	o   Options
	trk tracker

	screen d2.Point   // cached from "info" (800x600 is fixed in d2d, V)
	cmu    sync.Mutex // cursor tracking (cursor.go)
	prev   []d2.Unit  // carried items of the previous read
	cur    *d2.Unit   // the unit believed to be on the cursor
	wmu    sync.Mutex // warned is touched by every reader goroutine (runner, quest source, merc, town)
	warned map[string]bool
}

// New wraps a client. The Source does not own it (the caller closes it); one client per socket.
func New(c *d2dctl.Client, o Options) *Source {
	if o.Timeout == 0 {
		o.Timeout = 10 * time.Second
	}
	if o.Logf == nil {
		o.Logf = log.Printf
	}
	s := &Source{c: c, o: o, warned: map[string]bool{}}
	s.trk.logf = o.Logf
	return s
}

// UnknownCounts returns how many snapshots reported each unknown field since the Source was created.
func (s *Source) UnknownCounts() map[string]uint64 { return s.trk.snapshot() }

// Snapshot satisfies the read interface the bot uses today (d2.Reader.Snapshot). A transport error is an
// error; a game that is not in a state to read is State loading or menu, like the memory reader.
func (s *Source) Snapshot() (*d2.Snapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.o.Timeout)
	defer cancel()
	r, err := s.Read(ctx)
	if err != nil {
		return nil, err
	}
	return r.Snap, nil
}

// Clock is the dump's time section (lockstep fork only; zero Valid on stock d2d).
type Clock struct {
	Valid         bool
	Frame, GameMS int
	Tick          int
	Lockstep      bool
	Scale         float64
}

// Extras are things d2d provided that the d2.Snapshot types cannot hold, keyed by what they describe.
type Extras struct {
	Clock   Clock
	Items   map[uint32]ItemExtra // by Unit.ID of an item (carried and ground)
	Names   map[uint32]string    // display names of NPCs/objects by Unit.ID
	Menu    []d2dctl.MenuLine    // open NPC menu lines (only fetched while a menu is open)
	Level   *d2dctl.LevelInfo    // only when Options.AddOrigin
	Corpses []d2dctl.Corpse      // Options.Corpses
	Warps   []d2dctl.Warp        // Options.Warps
	Merc    *MercExtra
	Gold    []uint32 // unit ids of gold piles on the ground
	Coords  string   // "relative" or "absolute(origin*5)"
}

// ItemExtra holds item facts without a home in d2.Item.
type ItemExtra struct {
	Code        string
	Name        string // first tooltip line
	Qty         int
	Dur, MaxDur int
	Sockets     int    // socket count, -1 unknown
	Label       string // ground label
	Unique, Set int
}

// MercExtra is merc state beyond the Unit.
type MercExtra struct {
	Code string
	Dead bool
}

// Result is one snapshot with its provenance.
type Result struct {
	Snap   *d2.Snapshot
	Report Report
	Extras Extras
}

func isUnknownVerb(err error) bool {
	var ce *d2dctl.CmdError
	return errors.As(err, &ce) && strings.Contains(ce.Msg, "unknown verb")
}

func isCmdErr(err error) bool {
	var ce *d2dctl.CmdError
	return errors.As(err, &ce)
}

func (s *Source) warnOnce(key, format string, a ...any) {
	s.wmu.Lock()
	seen := s.warned[key]
	s.warned[key] = true
	s.wmu.Unlock()
	if seen {
		return
	}
	s.o.Logf(format, a...)
}

// dumpCall sends "dump <sections>" and parses the JSON line. Returns errNoDump for stock d2d.
var errNoDump = errors.New("d2dsrc: dump verb not available (stock d2d)")

func (s *Source) dump(ctx context.Context, sections string) (obj, error) {
	r, err := s.c.Do(ctx, "dump "+sections)
	if err != nil {
		if isUnknownVerb(err) {
			return nil, errNoDump
		}
		return nil, err
	}
	line := strings.TrimSpace(r.Rest)
	for _, l := range r.Lines {
		if strings.TrimSpace(l) != "" {
			line = l
			break
		}
	}
	return parseObj([]byte(line))
}

const headDump = "player,stats,skills,ui,merc,time"

// readBatch is what every in-game snapshot asks d2d, in one pipelined write. The strings must equal what the typed
// client calls send, they are the keys of the prefetch store.
var readBatch = []string{"state", "debug players", "dump " + headDump, "dump items", "monsters", "npcs", "ground", "debug portals"}

// Read builds one snapshot. A cmd-level "err" from a list verb degrades that list to empty with an unknown
// note; only transport errors fail the call.
func (s *Source) Read(ctx context.Context) (*Result, error) {
	rp := newRep(&s.trk)
	defer s.trk.flush()
	res := &Result{Extras: Extras{Items: map[uint32]ItemExtra{}, Names: map[uint32]string{}, Coords: "relative"}}
	snap := &d2.Snapshot{}
	res.Snap = snap

	if s.screen == (d2.Point{}) {
		if in, err := s.c.Info(ctx); err == nil && in.W > 0 {
			s.screen = d2.Point{X: in.W, Y: in.H}
		} else if err != nil && !isCmdErr(err) {
			return nil, err
		}
	}
	snap.Screen = s.screen
	if snap.Screen == (d2.Point{}) {
		rp.unknown("screen", "info gave no size")
	} else {
		rp.have("screen")
	}
	rp.unknown("mouse", "no mouse readback verb (only the hover/move write)")
	rp.unknown("viewport", "not exposed; pixel clicks need it, semantic input does not")
	rp.unknown("hovered", "state hover= is one int of unknown meaning (U)")

	// One write, one d2d frame: every read this snapshot needs (docs/d2d-pace.md). The typed calls below find their
	// replies in the prefetch store. A failed prefetch only costs the speed-up: they fall back to a round trip each.
	if !s.o.NoPrefetch {
		_ = s.c.Prefetch(ctx, readBatch...)
		defer s.c.DropPrefetch()
	}
	st, _, err := s.c.State(ctx)
	if err != nil {
		if isCmdErr(err) {
			snap.State = d2.StateLoading
			res.Report = rp.Report
			return res, nil
		}
		return nil, err
	}
	snap.State = stateOf(st.Screen)
	rp.have("state")
	if snap.State != d2.StateInGame {
		rp.unknown("ui", "only read in game")
		res.Report = rp.Report
		return res, nil
	}

	players, err := s.c.Players(ctx)
	if err != nil && !isCmdErr(err) {
		return nil, err
	}

	b := &builder{s: s, rp: rp, res: res, snap: snap, st: st}
	if len(players.Items) > 0 {
		b.pl = &players.Items[0]
		for i := range players.Items {
			if players.Items[i].Name == st.Name {
				b.pl = &players.Items[i]
			}
		}
	}

	head, err := s.dump(ctx, headDump)
	nodump := errors.Is(err, errNoDump)
	switch {
	case nodump:
		s.warnOnce("nodump", "d2dsrc: dump verb missing (stock d2d): degraded snapshot from state/players/items verbs")
		rp.Source["player"], rp.Source["ui"], rp.Source["merc"] = "verbs", "verbs", "verbs"
		if err := b.degradedHead(ctx); err != nil {
			return nil, err
		}
	case err != nil:
		if isCmdErr(err) { // "no scene": not readable yet
			*snap = d2.Snapshot{State: d2.StateLoading, Screen: snap.Screen}
			res.Report = rp.Report
			return res, nil
		}
		return nil, err
	default:
		rp.Source["player"], rp.Source["ui"], rp.Source["merc"] = "dump", "dump", "dump"
		b.head(ctx, head)
	}
	if snap.Player == nil {
		*snap = d2.Snapshot{State: d2.StateLoading, Screen: snap.Screen}
		res.Report = rp.Report
		return res, nil
	}
	if b.lvl != nil && s.o.AddOrigin {
		res.Extras.Level = b.lvl
	}

	if err := b.items(ctx, nodump); err != nil {
		return nil, err
	}
	if snap.Inventory != nil {
		b.cursor()
	}
	if err := b.world(ctx); err != nil {
		return nil, err
	}
	if snap.UI.Open(d2.UINPCMenu) {
		if m, err := s.c.Menu(ctx); err == nil {
			res.Extras.Menu = m.Items
		} else if !isCmdErr(err) {
			return nil, err
		}
	}
	if s.o.Corpses {
		if l, err := s.c.Corpses(ctx); err == nil {
			res.Extras.Corpses = l.Items
		} else if !isCmdErr(err) {
			return nil, err
		}
	}
	if s.o.Warps {
		if l, err := s.c.Warps(ctx); err == nil {
			res.Extras.Warps = l.Items
		} else if !isCmdErr(err) {
			return nil, err
		}
	}
	rp.unknown("missiles", "state missiles= is a count only; no list verb")
	rp.unknown("unit.owner", "no owner link on any unit")
	rp.unknown("unit.flags", "no per-unit flags (hostile, friendly, dying)")
	res.Report = rp.Report
	return res, nil
}

func stateOf(screen string) d2.State {
	switch strings.ToLower(screen) {
	case "ingame":
		return d2.StateInGame
	case "", "loading":
		return d2.StateLoading
	}
	return d2.StateMenu
}
