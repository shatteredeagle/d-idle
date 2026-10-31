package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"d2lod-bot/internal/d2"
)

// Env is everything a probe touches, as plain funcs, so tests run without a game. main wires the real ones.
type Env struct {
	Out   io.Writer
	Snap  func() (*d2.Snapshot, error)
	Sleep func(time.Duration)
	Now   func() time.Time

	Progress    func(playerAddr uint32) (*d2.PlayerProgress, error)
	Missiles    func() ([]d2.Unit, bool)
	MissileName func(class int) string // missiles.txt Name, "" when unknown
	StateName   func(id int) string    // states.txt Name, "" when unknown
	ItemCode    func(class int) string // misc/weapons/armor code of an item class row
	MonsterName func(class int) (name string, npc bool)
	CurseIDs    d2.CurseIDs
	Transmute   func(ctx context.Context, inputs []string) ([]string, error)
	Runewords   []RuneRow
	CloseAll    func() error
}

// RuneRow is the part of data.Runeword the probe needs.
type RuneRow struct {
	Name   string
	Ladder bool
	Runes  []string // misc.txt codes in socket order
}

// emit writes one JSON line: {"probe":..,"kind":..,"t":..,...fields}.
func (e *Env) emit(probe, kind string, f map[string]any) {
	m := map[string]any{}
	for k, v := range f {
		m[k] = v
	}
	m["probe"], m["kind"], m["t"] = probe, kind, e.Now().UTC().Format(time.RFC3339Nano) // the envelope wins over a field of the same name
	b, err := json.Marshal(m)
	if err != nil {
		b = []byte(fmt.Sprintf(`{"probe":%q,"kind":"error","error":%q}`, probe, err.Error()))
	}
	fmt.Fprintf(e.Out, "%s\n", b)
}

func (e *Env) fail(probe string, err error) error {
	e.emit(probe, "error", map[string]any{"error": err.Error()})
	return err
}

// poll calls fn every interval until dur has passed (at least once). The loop counts ticks, not wall time, so the
// fake clock of the tests stays deterministic.
func (e *Env) poll(dur, interval time.Duration, fn func(tick int) error) error {
	n := int(dur / interval)
	if n < 1 {
		n = 1
	}
	for i := 0; i < n; i++ {
		if err := fn(i); err != nil {
			return err
		}
		if i < n-1 {
			e.Sleep(interval)
		}
	}
	return nil
}

func hex8(v uint8) string   { return fmt.Sprintf("0x%02x", v) }
func hex32(v uint32) string { return fmt.Sprintf("0x%08x", v) }
