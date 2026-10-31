package cheat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// godSaneMax is the largest max life (24.8 fixed point, 6000 life) a throwaway character can have from
// level and gear; anything above it is the god buffer (godBuffer, 8191 life) or another bad write. The
// game's own Save and Exit writes the server unit's max life into the .d2s, so an inflated value that is
// still in memory then corrupts the save. VERIFIED LIVE (docs/act2.md "Live results 3": a killed
// `act2live god` job left max life 2097151 in the character). UNVERIFIED: the exact bound for real gear.
const godSaneMax int32 = 6000 * 256

// ErrInflatedLife is returned when god or heal would treat an already inflated max life as the original.
var ErrInflatedLife = errors.New("cheat: max life is above the sane bound with no undo marker; run `d2cheat restore --maxlife N` first")

// godMarker is the "undo pending" record written before god mode changes anything and removed after the
// originals are written back and read back. A SIGKILLed run leaves it behind; the next god start and
// `d2cheat restore` read it. It lives next to the ledger, one file per instance.
type godMarker struct {
	Time      time.Time `json:"time"`
	Instance  string    `json:"instance"`
	Character string    `json:"character"`
	MaxLife   int32     `json:"maxlife"`
	Life      int32     `json:"life"`
	PID       int       `json:"pid"`
}

func (e *Env) markerPath() string {
	return filepath.Join(filepath.Dir(e.Ledger.Path), "god-pending-"+e.Instance+".json")
}

func (e *Env) writeMarker(m godMarker) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	p := e.markerPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func (e *Env) readMarker() (*godMarker, error) {
	b, err := os.ReadFile(e.markerPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var m godMarker
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("cheat: corrupt god marker %s: %w", e.markerPath(), err)
	}
	return &m, nil
}

// PendingGod returns the undo marker of this instance, nil when there is none (for callers that only
// want to warn, e.g. the bot at start).
func (e *Env) PendingGod() (*godMarker, error) { return e.readMarker() }

// restoreLife writes the original max life and a life no higher than it into both units, then reads
// it back. A mismatch is an error and the caller keeps the marker.
func (g *game) restoreLife(maxLife int32) error {
	life, ok := g.get(UnitServer, VecMerged, statLife)
	if !ok || life > maxLife {
		life = maxLife
	}
	if err := g.setBoth(statMaxLife, maxLife); err != nil {
		return err
	}
	if err := g.setBoth(statLife, life); err != nil {
		return err
	}
	for _, un := range []string{UnitServer, UnitClient} {
		if un == UnitServer && g.server == 0 {
			continue
		}
		if v, ok := g.get(un, VecMerged, statMaxLife); ok && v != maxLife {
			return fmt.Errorf("cheat: max life restore not confirmed on %s (%d, want %d)", un, v, maxLife)
		}
	}
	return nil
}

// pendingRestore repairs a leftover god state before anything else touches the character: with a marker
// for this character it restores the originals and deletes it; a marker for another character is kept and
// ignored; an inflated max life without a marker is an error (it must not be taken as the original).
func (e *Env) pendingRestore(g *game) (string, error) {
	m, err := e.readMarker()
	if err != nil {
		return "", err
	}
	if m != nil && m.Character == g.name {
		if cur, ok := g.get(UnitServer, VecMerged, statMaxLife); ok && cur <= godSaneMax {
			// The save was clean (the game died before saving): writing the old value now would lower a character that
			// has levelled since. Drop the marker instead.
			if err := os.Remove(e.markerPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
				return "", err
			}
			return fmt.Sprintf("dropped stale god marker of %s: max life %d is not inflated (marker from %s, pid %d)", m.Character, cur/256, m.Time.Format(time.RFC3339), m.PID), nil
		}
		if err := g.restoreLife(m.MaxLife); err != nil {
			return "", err
		}
		if err := os.Remove(e.markerPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		return fmt.Sprintf("restored leftover god state of %s: max life %d (marker from %s, pid %d)", m.Character, m.MaxLife/256, m.Time.Format(time.RFC3339), m.PID), nil
	}
	if cur, ok := g.get(UnitServer, VecMerged, statMaxLife); ok && cur > godSaneMax {
		return "", ErrInflatedLife
	}
	return "", nil
}

// godRestore is `d2cheat restore`: repair from the marker, or with maxlife=N (display value, e.g. 1000)
// force that max life when max life is above the sane bound and there is no marker.
func (e *Env) godRestore(g *game, req Request) (string, error) {
	msg, err := e.pendingRestore(g)
	if errors.Is(err, ErrInflatedLife) {
		n, ok := req.Int("maxlife")
		if !ok || n < 1 || n*256 > int64(godSaneMax) {
			return "", err
		}
		if err := g.restoreLife(int32(n * 256)); err != nil {
			return "", err
		}
		return fmt.Sprintf("max life forced to %d (no marker)", n), nil
	}
	if err != nil {
		return "", err
	}
	if msg == "" {
		return "nothing to restore", nil
	}
	return msg, nil
}

// GuardSave is for code that is about to make the game save (Save and Exit, a quit, a restart): when
// max life is inflated it restores the marker's originals first. Only works while the game is attached;
// it records a ledger line when it restored something.
func (e *Env) GuardSave(ctx context.Context, reason string) (string, error) {
	res, err := e.Do(ctx, Request{Kind: KindGodRestore, Reason: reason, Actor: "saveguard"})
	if err != nil {
		return "", err
	}
	return res.Message, nil
}

// GuardSave on the gate: a no-op error when cheats are off, because then nothing could have inflated.
func (g Gate) GuardSave(ctx context.Context, reason string) (string, error) {
	if !g.Allow {
		return "", nil
	}
	return g.Env.GuardSave(ctx, reason)
}
