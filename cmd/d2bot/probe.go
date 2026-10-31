package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
)

// probeStates (D2BOT_PROBE_STATES=1) logs every change in the player's extended StatList struct beyond the vectors, once
// per 2 s: how the state bit array was located (cast Frozen Armor, state 10, and the word that gained bit 10 is it).
// Debug aid, off by default.
func probeStates(ctx context.Context, e *env.Env, log *slog.Logger) {
	var last []byte
	var lastStates d2.StateSet
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		p, err := e.Read.Player()
		if err != nil {
			last = nil
			continue
		}
		b, err := e.Read.RawStatList(p.Addr, 0x140)
		if err != nil {
			continue
		}
		if last != nil {
			for off := 0x50; off+4 <= len(b); off += 4 {
				o, n := binary.LittleEndian.Uint32(last[off:]), binary.LittleEndian.Uint32(b[off:])
				if o != n {
					log.Info("probe statlist word", "off", fmt.Sprintf("0x%x", off), "old", fmt.Sprintf("0x%08x", o), "new", fmt.Sprintf("0x%08x", n))
				}
			}
		} else {
			log.Info("probe statlist first", "flags", fmt.Sprintf("0x%08x", binary.LittleEndian.Uint32(b[0x10:])), "states", p.States, "area", p.Area)
		}
		last = b
		if p.States != lastStates {
			var on []int
			for n := 0; n < 192; n++ {
				if p.States.Has(n) {
					on = append(on, n)
				}
			}
			log.Info("probe states changed", "statesOn", on, "area", p.Area, "life", p.Life)
			lastStates = p.States
		}
	}
}
