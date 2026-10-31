// Package cheat is the single, logged entry point for test cheats on the throwaway characters.
// Everything goes through Do: reason required, character allowlist, offline only, ledger entry for
// every attempt (also refused ones), save backups before any file edit. See docs/cheats.md.
package cheat

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/mem"
)

// Kind names a cheat. Memory kinds act on the running game; save kinds edit a .d2s of a stopped game.
type Kind string

const (
	KindHeal        Kind = "heal"        // refill life, mana, stamina
	KindGod         Kind = "god"         // invincibility on/off (see memory.go for the mechanism)
	KindGodRestore  Kind = "godrestore"  // undo a leftover god state (marker file), see godstate.go
	KindGold        Kind = "gold"        // set carried gold
	KindStashGold   Kind = "stashgold"   // set stash gold
	KindLevel       Kind = "level"       // set level (+ experience for it)
	KindXP          Kind = "xp"          // set experience
	KindStats       Kind = "stats"       // set attributes / unspent points
	KindSetStat     Kind = "setstat"     // research primitive: write one stat of one stat vector
	KindSaveEdit    Kind = "saveedit"    // edit a .d2s header/stats/skills
	KindDataOverlay Kind = "dataoverlay" // per-instance excel .txt override (rates)
)

// Request is one cheat. Reason is mandatory; Actor defaults to $D2BOT_AGENT.
type Request struct {
	Kind   Kind
	Params map[string]any
	Reason string
	Actor  string
}

// Int reads an integer parameter (accepts int, int64, float64 and numeric strings).
func (r Request) Int(key string) (int64, bool) {
	switch v := r.Params[key].(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case int32:
		return int64(v), true
	case float64:
		return int64(v), true
	case string:
		n, err := strconv.ParseInt(v, 0, 64)
		return n, err == nil
	}
	return 0, false
}

func (r Request) Str(key string) string {
	if s, ok := r.Params[key].(string); ok {
		return s
	}
	return ""
}

// Result describes what a successful cheat changed (the same maps go to the ledger).
type Result struct {
	Entry   Entry
	Message string
}

// Env is everything Do needs; DefaultEnv fills it for the running machine. Tests inject fakes.
type Env struct {
	Ledger    *Ledger
	Instance  string // "i5"; from $D2BOT_INSTANCE
	Now       func() time.Time
	Open      func() (mem.Writer, func(), error) // opens the game memory; the func closes it
	SaveDir   string                             // folder with the instance's .d2s files
	BackupDir string                             // save backups (default DefaultBackupDir)
	InGame    func() (bool, error)               // is a character loaded in a running game (save edits need false)
	Rules     func() (Rules, error)              // game tables for level-ups (exp, per-level gains)
	DataDir   string                             // overlay root for dataoverlay
	Sleep     func(time.Duration)
}

// DefaultEnv builds the Env for the current process: ledger path, instance from $D2BOT_INSTANCE,
// the game opened through mem.OpenGame (honors the instance).
func DefaultEnv() *Env {
	inst := os.Getenv(mem.InstanceEnv)
	if inst == "" {
		inst = mem.DefaultInstance
	}
	return &Env{
		Ledger:    NewLedger(DefaultLedgerPath()),
		Instance:  inst,
		Now:       time.Now,
		BackupDir: DefaultBackupDir(),
		SaveDir:   DefaultSaveDir(inst),
		DataDir:   DefaultDataDir(inst),
		Open: func() (mem.Writer, func(), error) {
			p, err := mem.OpenGame()
			if err != nil {
				return nil, nil, err
			}
			return p, func() { p.Close() }, nil
		},
		InGame: func() (bool, error) { return gameInGame(inst) },
		Rules:  defaultRules,
		Sleep:  time.Sleep,
	}
}

// Do runs one cheat on the default environment. This is the only supported entry point.
func Do(ctx context.Context, req Request) (*Result, error) { return DefaultEnv().Do(ctx, req) }

// Do validates, applies and records one cheat. The ledger entry is written for every outcome once the
// request is syntactically valid enough to name a kind and reason; a missing reason is refused before
// anything happens and also recorded (with reason "(none)") so attempts to skip it are visible.
func (e *Env) Do(ctx context.Context, req Request) (*Result, error) {
	ent := Entry{Time: e.now(), Actor: req.Actor, Instance: e.Instance, Kind: req.Kind, Params: req.Params,
		Reason: strings.TrimSpace(req.Reason)}
	if ent.Actor == "" {
		ent.Actor = os.Getenv("D2BOT_AGENT")
	}
	if ent.Actor == "" {
		ent.Actor = "unknown"
	}
	if ent.Reason == "" {
		ent.Reason = "(none)"
		ent.Error = ErrNoReason.Error()
		_ = e.Ledger.Append(ent)
		return nil, ErrNoReason
	}
	if err := e.Ledger.Ensure(); err != nil {
		return nil, fmt.Errorf("cheat: ledger not writable, refusing to cheat: %w", err)
	}
	var msg string
	var err error
	switch req.Kind {
	case KindSaveEdit:
		msg, err = e.doSave(ctx, req, &ent)
	case KindDataOverlay:
		msg, err = e.doOverlay(ctx, req, &ent)
	case KindHeal, KindGod, KindGodRestore, KindGold, KindStashGold, KindLevel, KindXP, KindStats, KindSetStat:
		msg, err = e.doMemory(ctx, req, &ent)
	default:
		err = fmt.Errorf("cheat: unknown kind %q", req.Kind)
	}
	ent.Success = err == nil
	if err != nil {
		ent.Error = err.Error()
	}
	if lerr := e.Ledger.Append(ent); lerr != nil {
		err = errors.Join(err, fmt.Errorf("cheat: LEDGER WRITE FAILED (the cheat may have been applied): %w", lerr))
	}
	if err != nil {
		return nil, err
	}
	return &Result{Entry: ent, Message: msg}, nil
}

func (e *Env) now() time.Time {
	if e.Now != nil {
		return e.Now().UTC()
	}
	return time.Now().UTC()
}
