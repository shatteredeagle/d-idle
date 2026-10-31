package combat

import (
	"context"
	"fmt"
	"math"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// What a real death looks like. Everything here is VERIFIED LIVE (instance i2, Combi, softcore level 1 Sorceress,
// Normal, 3 deaths on 2026-10-28; fixtures in testdata/death, numbers in docs/combat.md "Death: VERIFIED LIVE").
const (
	// ModeDying is the player mode (UnitAny.dwMode) from the killing blow until the body has fallen: life and gold read 0 at
	// once, equipment still reads as worn. Esc does nothing in this mode.
	ModeDying = 0
	// ModeDead is the mode after the fall, 0.9 to 0.95 s after the first dying sample. Esc respawns only from here.
	ModeDead = 17
	// ModeTownNeutral is the mode of a living character standing in town (5; outdoors 1 stand, 2 walk, 4 hit recovery).
	ModeTownNeutral = 5

	// DyingSeconds is the measured time from mode 0 to mode 17 (0.91 s and 0.95 s, 2 deaths).
	DyingSeconds = 0.95
	// RespawnSeconds is the measured time from the Esc press to the living character in town (0.1 to 0.3 s).
	RespawnSeconds = 0.3
	// EscRetryEvery is the gap between Esc presses while waiting for ModeDead (one press per gap is harmless).
	EscRetryEvery = 400 * time.Millisecond
	// CorpseTouch is the distance in subtiles at which walking over the corpse picks it up (touched at 1.4 to 2).
	CorpseTouch = 2.5
)

// IsDying is true from the killing blow until the body lies.
func IsDying(p *d2.Player) bool {
	return p != nil && (p.Mode == ModeDying || p.Life <= 0 && p.MaxLife > 0 && p.Mode != ModeDead)
}

// IsDead is true while the character is dying or dead: the one test every module should use. It is the MODE only
// (0 at the killing blow, 17 once fallen; VERIFIED LIVE, life reads 0 in the same sample as mode 0). Life 0 in a
// living mode is NOT a death (ZeroLife): VERIFIED LIVE 2026-10-28 on i4, a save written while dead loads as a
// standing level 3 Booti in the Rogue Encampment with life 0 for minutes (mode 5, no regeneration, Esc opens the
// game menu), and treating that as "dead" made the bot wait for a respawn forever.
func IsDead(p *d2.Player) bool {
	return p != nil && (p.Mode == ModeDying || p.Mode == ModeDead)
}

// ZeroLife is a living character with no life (see IsDead): it needs a healer or a potion, not a respawn.
func ZeroLife(p *d2.Player) bool { return p != nil && !IsDead(p) && p.MaxLife > 0 && p.Life <= 0 }

// CanRespawn is true when an Esc press would respawn the character (the body has fallen).
func CanRespawn(p *d2.Player) bool { return p != nil && p.Mode == ModeDead }

// ActTown returns the town area of an act (0-based act number as in Player.Act): where a softcore character respawns.
// Act 1 VERIFIED LIVE (area 1, the same spot as a fresh game: (6343,5413) for Combi); the others are the town level ids.
func ActTown(act int) int {
	towns := [...]int{1, 40, 75, 103, 109}
	if act < 0 || act >= len(towns) {
		return 1
	}
	return towns[act]
}

// Corpse is the body a dead character leaves behind: a second player-type unit (type 0) in mode 17 with a new unit id,
// at the death position, in the area where it died. Walking within CorpseTouch of it picks it up (the worn items
// return at once). Carried gold is not on it: it lies as a separate gold pile (a ground item) at the same place, and
// walking over the pile takes all of it back.
type Corpse struct {
	ID   uint32
	Pos  input.Point
	Area int
}

// FindCorpse picks the body out of the player-type units (Reader.Units(d2.TypePlayer)): a mode 17 unit that is not the
// own unit. While the own character is also dead the own unit has the same mode and its own id, so ownID matters.
func FindCorpse(units []d2.Unit, ownID uint32, area int) (Corpse, bool) {
	for _, u := range units {
		if u.Type == d2.TypePlayer && u.Mode == ModeDead && u.ID != ownID {
			return Corpse{ID: u.ID, Pos: input.Point{X: u.X, Y: u.Y}, Area: area}, true
		}
	}
	return Corpse{}, false
}

// Touched reports whether a character at p has walked onto the corpse.
func (c Corpse) Touched(p input.Point) bool {
	return math.Hypot(float64(p.X-c.Pos.X), float64(p.Y-c.Pos.Y)) <= CorpseTouch
}

// GoldPile finds the gold the dead character dropped: the ground item nearest to the corpse that the catalog calls gold.
// isGold decides by item class (the bot's catalog knows it; class 523 in the worker build, UNVERIFIED elsewhere).
func GoldPile(ground []d2.Unit, near input.Point, isGold func(class int) bool) (input.Point, bool) {
	best, bd, ok := input.Point{}, 1e9, false
	for _, u := range ground {
		if !isGold(u.Class) {
			continue
		}
		if d := math.Hypot(float64(u.X-near.X), float64(u.Y-near.Y)); d < bd {
			best, bd, ok = input.Point{X: u.X, Y: u.Y}, d, true
		}
	}
	return best, ok
}

// Corpse reads the unit tables for the body of the own character (see FindCorpse).
func (f *Fighter) Corpse() (Corpse, bool) {
	s, err := f.snapFn()
	if err != nil || s == nil || s.Player == nil {
		return Corpse{}, false
	}
	units, err := f.e.Read.Units(d2.TypePlayer)
	if err != nil {
		return Corpse{}, false
	}
	return FindCorpse(units, s.Player.ID, s.Player.Area)
}

// Respawn dismisses the death screen: it waits until the body has fallen (ModeDead, Esc is ignored before), presses Esc
// every EscRetryEvery and returns when the character lives again (normally in the act's town, 1 s after the death).
// It never presses Esc while the character is alive (that opens the game menu). timeout 0 = 30 s.
func (f *Fighter) Respawn(ctx context.Context, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	deadline := f.e.Clock.Now().Add(timeout)
	var lastEsc time.Time
	for f.e.Clock.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		s, err := f.snapFn()
		if err == nil && s != nil && s.State == d2.StateInGame && s.Player != nil {
			p := s.Player
			if !IsDead(p) {
				return nil
			}
			if CanRespawn(p) && f.e.Clock.Now().Sub(lastEsc) >= EscRetryEvery {
				lastEsc = f.e.Clock.Now()
				if err := f.e.In.Key("esc"); err != nil {
					return err
				}
			}
		}
		f.e.Clock.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("combat: still dead after %v", timeout)
}
