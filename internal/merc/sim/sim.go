// Package sim is a fake merc backend for tests: gold accounting, the hire and revive formulas, the free Blood Raven
// Rogue, gear destroyed by a replacing hire. It implements merc.Backend. The rules are the d2decomped findings (S),
// not verified live.
package sim

import (
	"context"
	"errors"
	"fmt"

	"d2lod-bot/internal/merc"
)

// Sim is one character's merc situation.
type Sim struct {
	Gold      int
	ClvlLevel int // player level, drives offered levels
	Merc      merc.MercState
	Window    []merc.HireOffer // rows of the open hire window; nil = closed
	// FreeRogueLeft: Blood Raven done and the reward not yet taken or forfeited.
	FreeRogueLeft bool
	Pack          []merc.Item // items in the player's pack (taken-off merc gear lands here)
	Potions       int
	Err           error // returned by State/Offers when set
	Log           []string
}

var (
	ErrNoWindow  = errors.New("sim: no hire window open")
	ErrNoMerc    = errors.New("sim: no merc to act on")
	ErrPoor      = errors.New("sim: not enough gold")
	ErrNoPotion  = errors.New("sim: no potion")
	ErrBadChoice = errors.New("sim: bad row")
)

var _ merc.Backend = (*Sim)(nil)

func (s *Sim) State() (merc.MercState, error) {
	if s.Err != nil {
		return merc.MercState{}, s.Err
	}
	m := s.Merc
	m.Known = true
	return m, nil
}

func (s *Sim) Offers() ([]merc.HireOffer, error) {
	if s.Err != nil {
		return nil, s.Err
	}
	if s.Window == nil {
		return nil, ErrNoWindow
	}
	return append([]merc.HireOffer(nil), s.Window...), nil
}

// OpenWindow fills the rows from base prices: class, aura, base gold, base level; the level and price follow the
// formulas with r = row index modulo 5.
func (s *Sim) OpenWindow(rows []Row) {
	s.Window = s.Window[:0]
	for i, r := range rows {
		lvl := merc.OfferedLevel(s.ClvlLevel, i%5)
		s.Window = append(s.Window, merc.HireOffer{Row: i, Class: r.Class, Aura: r.Aura, Level: lvl, Price: merc.HireCost(r.BaseGold, lvl, r.BaseLevel)})
	}
}

// Row is a hire table row.
type Row struct {
	Class     int
	Aura      string
	BaseGold  int
	BaseLevel int
}

func (s *Sim) CloseWindow() { s.Window = nil }

// Hire takes row of the window. Employing any merc forfeits the free Rogue; the old merc and his gear are gone.
func (s *Sim) Hire(_ context.Context, row int) (int, error) {
	if s.Window == nil {
		return 0, ErrNoWindow
	}
	if row < 0 || row >= len(s.Window) {
		return 0, ErrBadChoice
	}
	o := s.Window[row]
	if s.Gold < o.Price {
		return 0, ErrPoor
	}
	s.Gold -= o.Price
	s.Merc = merc.MercState{Owned: true, Class: o.Class, Alive: true, LifePct: 100, Level: o.Level, Aura: o.Aura,
		Str: 60, Dex: 50, StatsKnown: true}
	s.FreeRogueLeft = false
	return o.Price, nil
}

// ClaimFreeRogue is Kashya's reward: a Rogue for 0 gold, only when no merc is owned (even a dead one). Without it
// the reward is lost for good.
func (s *Sim) ClaimFreeRogue() error {
	if !s.FreeRogueLeft {
		return errors.New("sim: no reward")
	}
	s.FreeRogueLeft = false
	if s.Merc.Owned {
		return errors.New("sim: reward forfeited, a merc is owned")
	}
	s.Merc = merc.MercState{Owned: true, Class: merc.ClassRogue, Alive: true, LifePct: 100, Level: merc.OfferedLevel(s.ClvlLevel, 2),
		Aura: "Cold Arrow", Str: 40, Dex: 60, StatsKnown: true}
	return nil
}

// Kill is the merc dying in the field.
func (s *Sim) Kill() { s.Merc.Alive, s.Merc.LifePct = false, 0 }

func (s *Sim) Revive(context.Context) (int, error) {
	if !s.Merc.Owned {
		return 0, ErrNoMerc
	}
	if s.Merc.Alive {
		return 0, errors.New("sim: merc is alive")
	}
	p := merc.ReviveCost(s.Merc.Level)
	if s.Gold < p {
		return 0, ErrPoor
	}
	s.Gold -= p
	s.Merc.Alive, s.Merc.LifePct = true, 100
	return p, nil
}

func (s *Sim) Equip(_ context.Context, slot merc.Slot, it merc.Item) error {
	if !s.Merc.Owned {
		return ErrNoMerc
	}
	if old := s.Merc.Gear[slot]; old != nil {
		s.Pack = append(s.Pack, *old)
	}
	s.Merc.Gear[slot] = &it
	return nil
}

func (s *Sim) Unequip(_ context.Context, slot merc.Slot) error {
	if !s.Merc.Owned || s.Merc.Gear[slot] == nil {
		return fmt.Errorf("sim: nothing in %s", slot)
	}
	s.Pack = append(s.Pack, *s.Merc.Gear[slot])
	s.Merc.Gear[slot] = nil
	return nil
}

func (s *Sim) GiveHealing(context.Context) error {
	if s.Potions < 1 {
		return ErrNoPotion
	}
	s.Potions--
	s.Merc.LifePct = min(100, s.Merc.LifePct+50)
	return nil
}
