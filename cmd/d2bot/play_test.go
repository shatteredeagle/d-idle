package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/watch"
)

// playArgs fills the tester defaults, prints the URL once and never puts the token anywhere else.
func TestPlayArgsPrintsTokenOnce(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("D2BOT_WATCH_VIDEO_SOURCE", "tap")
	for _, k := range []string{"D2BOT_LADDER", "D2BOT_WATCH", "D2BOT_WATCH_VIDEO", "D2BOT_WATCH_ADDR", "D2BOT_WATCH_ANSWER_ADDR", "D2BOT_WATCH_TOKEN_FILE", "D2BOT_WATCH_ORIGIN", "D2BOT_LADDER_STORE", "D2BOT_WATCH_TOKEN"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	var out bytes.Buffer
	old := playOut
	playOut = &out
	t.Cleanup(func() { playOut = old; playMode = false })
	args, err := playArgs([]string{"-data", dir, "--attach"})
	if err != nil || len(args) != 3 {
		t.Fatalf("%v %v", args, err)
	}
	o, err := parseFlags(args)
	if err != nil {
		t.Fatal(err)
	}
	l := o.ladder
	if !playMode || !l.on || !l.watch || !l.video || l.viewAddr != "0.0.0.0:8891" || l.watchTokenFile != filepath.Join(dir, "watch.token") {
		t.Fatalf("defaults: %+v", l)
	}
	tok, _ := os.ReadFile(l.watchTokenFile)
	if got := out.String(); strings.Count(got, strings.TrimSpace(string(tok))) != 1 || !strings.Contains(got, "/watch?t=") {
		t.Errorf("printed:\n%s", got)
	}
}

// A play session through startLadder: the record exists, the notes box is wired, the token is in no file, the
// summary is written at the stop.
func TestPlaySessionWiring(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} {
		t.Run(class, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			state := t.TempDir()
			tf := filepath.Join(t.TempDir(), "tok")
			const tok = "play-token-0123456789abcdefABCDEF"
			os.WriteFile(tf, []byte(tok), 0o600)
			m := goalManager(t)
			lo := ladderOptions{on: true, watch: true, play: true, viewAddr: "127.0.0.1:0", answerAddr: "127.0.0.1:0", watchTokenFile: tf}
			status := func() goals.Status {
				return goals.Status{State: goals.RunnerRunning, Character: goals.Character{Class: class, Level: 5, Life: 50, MaxLife: 100, XP: 10}}
			}
			rig, err := startLadder(ctx, lo, "", state, fakeEnableFirsts(t, &watch.SimClock{}), m, status, func() string { return "Blood Moor" }, nil)
			if err != nil || rig == nil || rig.Rec == nil {
				t.Fatalf("%v %v", rig, err)
			}
			m.Handle(goals.Event{Type: goals.EvDeath})
			if rig.Fun.Return.Notes == nil {
				t.Error("notes box not wired")
			}
			if err := rig.Fun.Return.Notes.SaveSessionNotes("fun " + tok); err != nil {
				t.Fatal(err)
			}
			time.Sleep(2500 * time.Millisecond) // a few polls and the death event
			rig.StopRec()
			dir := rig.Rec.Dir()
			if filepath.Dir(dir) != filepath.Join(state, "sessions") {
				t.Errorf("dir %s", dir)
			}
			ents, _ := os.ReadDir(dir)
			if len(ents) != 4 {
				t.Errorf("files: %v", ents)
			}
			for _, e := range ents {
				b, _ := os.ReadFile(filepath.Join(dir, e.Name()))
				if bytes.Contains(b, []byte(tok)) {
					t.Errorf("token in %s", e.Name())
				}
			}
			b, _ := os.ReadFile(filepath.Join(dir, "summary.json"))
			if !strings.Contains(string(b), `"class": "`+strings.ToLower(class)+`"`) || !strings.Contains(string(b), `"deaths": 1`) {
				t.Errorf("summary:\n%s", b)
			}
		})
	}
}

func TestSessionsExportCommand(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "20261108T100000Z-alice")
	os.MkdirAll(dir, 0o700)
	os.WriteFile(filepath.Join(dir, "decisions.jsonl"), []byte(`{"v":1,"kind":"header","session":"aa","commit":"abc1234","start":"2026-11-08T10:00:00Z"}`+"\n"+
		`{"v":1,"kind":"fire","id":1,"session":"aa","commit":"abc1234","t_wall":"2026-11-08T10:00:05Z","point":"t1.repair","ctx":{"act":"1"},"class":"sorceress","build":"Zed","options":["a"],"default":"a","answer":"a","by":"default","latency_ms":null}`+"\n"), 0o600)
	var out bytes.Buffer
	if err := sessionsMain([]string{"export", dir, "-name", "Zed"}, &out); err != nil {
		t.Fatal(err)
	}
	if s := out.String(); strings.Contains(s, "Zed") || strings.Contains(s, "alice") || !strings.Contains(s, `"t_off_ms":5000`) {
		t.Errorf("export:\n%s", s)
	}
	if err := sessionsMain([]string{"nonsense"}, &out); err == nil {
		t.Error("usage error expected")
	}
}
