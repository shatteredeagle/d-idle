package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
)

type fakeGuard struct {
	msg string
	err error
	got []string
}

func (f *fakeGuard) GuardSave(_ context.Context, reason string) (string, error) {
	f.got = append(f.got, reason)
	return f.msg, f.err
}

func TestGuardSaveHook(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	g := &fakeGuard{msg: "restored"}
	if err := guardSave(g, log)(context.Background(), "save and exit"); err != nil || len(g.got) != 1 || g.got[0] != "save and exit" {
		t.Fatalf("%v %v", err, g.got)
	}
	g.err = errors.New("not allowed")
	if err := guardSave(g, log)(context.Background(), "x"); err == nil {
		t.Fatal("error must be passed to the flow (it logs and saves anyway)")
	}
}
