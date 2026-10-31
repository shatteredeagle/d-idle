package main

import (
	"context"
	"log/slog"
	"time"

	"d2lod-bot/internal/d2"
)

// runDry is the read-only first run: it attaches, prints what the bot sees every second and sends no input.
// It serves only /healthz.
func runDry(ctx context.Context, o options, log *slog.Logger) error {
	g, err := openGame(ctx, o, log)
	if err != nil {
		return err
	}
	defer g.close()
	log.Info("dry run: read-only, no input is sent, no goals run")
	h := &health{started: time.Now(), mode: "dry", game: g.alive}
	h.ready.Store(true)
	if !o.once {
		go func() { serveHealth(ctx, log, o, h) }()
	}
	rd := d2.New(g.src)
	if o.dryFor > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.dryFor)
		defer cancel()
	}
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		observe(rd, log)
		if o.once {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
		if !g.alive() {
			return errGameGone
		}
	}
}

func observe(rd *d2.Reader, log *slog.Logger) {
	s, err := rd.Snapshot()
	if err != nil {
		log.Warn("snapshot failed", "err", err)
		return
	}
	if s.State != d2.StateInGame || s.Player == nil {
		log.Info("observe", "state", s.State.String())
		return
	}
	p := s.Player
	alive, ground := 0, len(s.Ground)
	for i := range s.Monsters {
		if s.Monsters[i].Alive() {
			alive++
		}
	}
	items, belt := 0, 0
	if s.Inventory != nil {
		items = len(s.Inventory.Container(d2.ContInventory))
		belt = len(s.Inventory.Container(d2.ContBelt))
	}
	log.Info("observe", "state", s.State.String(), "name", p.Name, "class", p.Class.String(), "level", p.Level, "area", p.Area, "town", p.InTown(),
		"x", p.X, "y", p.Y, "life", p.Life, "maxLife", p.MaxLife, "mana", p.Mana, "exp", p.Exp, "gold", p.Gold,
		"monstersAlive", alive, "groundItems", ground, "invItems", items, "beltItems", belt, "uiPanelOpen", s.UI.AnyPanelOpen())
}
