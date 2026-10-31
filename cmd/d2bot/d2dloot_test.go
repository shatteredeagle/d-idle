package main

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2dctl/fake"
	"d2lod-bot/internal/d2denv"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
)

func TestD2DLootSeam(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "d.sock")
	srv, err := fake.New(sock, fake.Sample(), fake.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	db, err := data.Load(synthTables(t))
	if err != nil {
		t.Fatal(err)
	}
	tune := env.DefaultTuning()
	be, err := d2denv.Build(context.Background(), d2denv.Config{ID: "t", Log: slog.New(slog.DiscardHandler), Tune: &tune, Data: db, Sock: sock})
	if err != nil {
		t.Fatal(err)
	}
	defer be.Close()
	nv := be.Navigator()
	st := buildD2DStack(context.Background(), options{char: "Fake"}, be, nv, db, bot.Config{}, &runnerRef{}, slog.New(slog.DiscardHandler))
	_, _, l := be.Behaviors(st.seams(), nv)
	if _, ok := l.(looterAdapter); !ok {
		t.Fatalf("loot seam is %T, want the looter adapter", l)
	}
}
