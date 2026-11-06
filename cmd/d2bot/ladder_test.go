package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/api"
	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/goals/sim"
	"d2lod-bot/internal/ladder"
	"d2lod-bot/internal/watch"
)

// fakeEnable mirrors bot.(*Runner).EnableLadderWith on a virtual clock the test steps.
func fakeEnable(t *testing.T, clk *watch.SimClock, called *int) enableFunc {
	return func(dir, profile string, wire bot.LadderWire) (*ladder.Ladder, error) {
		*called++
		cat, err := decide.Shipped()
		if err != nil {
			return nil, err
		}
		st, err := decide.New(cat, profile, clk)
		if err != nil {
			return nil, err
		}
		sink, src, err := wire(st, cat, clk)
		if err != nil {
			return nil, err
		}
		l, err := ladder.New(ladder.Config{Cat: cat, Store: st, Clock: clk, Sink: sink, Answers: src})
		if err != nil {
			return nil, err
		}
		ladder.Install(l)
		t.Cleanup(func() { ladder.Install(nil) })
		return l, nil
	}
}

func goalManager(t *testing.T) *goals.Manager {
	t.Helper()
	m, err := goals.New(goals.Options{Runner: sim.New(sim.Options{Seed: 1}), Path: filepath.Join(t.TempDir(), "goals.json")})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func httpDo(t *testing.T, method, url string, body any, out any) int {
	t.Helper()
	return httpDoH(t, method, url, body, nil, out)
}

func httpDoH(t *testing.T, method, url string, body any, hdr map[string]string, outs ...any) int {
	t.Helper()
	var out any
	if len(outs) > 0 {
		out = outs[0]
	}
	var rd *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, url, rd)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if out != nil {
		json.NewDecoder(resp.Body).Decode(out)
	}
	return resp.StatusCode
}

var testBuilds = []struct {
	name string
	hero ladder.Hero
	life float64
}{
	{"sorceress", ladder.Hero{Class: d2.Sorceress, Gold: 600, AreaClass: "field", Durability: 25}, 30},
	{"paladin", ladder.Hero{Class: d2.Paladin, Hardcore: true, Gold: 600, AreaClass: "field", Durability: 25}, 45},
}

func askRetreat(t *testing.T, l *ladder.Ladder, h ladder.Hero, life float64) {
	t.Helper()
	extra := decide.Obs{Num: map[string]float64{"life": life}, Cat: map[string]string{"packdanger": "2", "kit": "OK", "held": "hp"}}
	if _, err := l.Decide("t0.retreat.drink", h, extra, ""); err != nil {
		t.Fatal(err)
	}
	l.Pump()
}

// With -ladder -watch the Hub receives the bot's thoughts; a person answers on the separate answer route and the order
// box reaches the goal manager; a scripted profile answers by itself. Two builds.
func TestLadderWatchOnTheBot(t *testing.T) {
	for _, b := range testBuilds {
		for _, scripted := range []string{"", "cautious"} {
			t.Run(b.name+"/"+scripted, func(t *testing.T) {
				clk := &watch.SimClock{}
				called := 0
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				m := goalManager(t)
				lo := ladderOptions{on: true, profile: scripted, watch: true, viewAddr: "127.0.0.1:0", answerAddr: "127.0.0.1:0"}
				rig, err := startLadder(ctx, lo, "", "", fakeEnable(t, clk, &called), m, func() goals.Status { return goals.Status{State: goals.RunnerRunning, Activity: "Fighting"} }, func() string { return "Cold Plains" },
					func(id int) string {
						if id == 20 {
							return "no merc, clvl 6 < 16"
						}
						return ""
					})
				if err != nil || rig == nil || rig.Hub == nil || called != 1 {
					t.Fatalf("%v %v %d", rig, err, called)
				}
				view, ans := "http://"+rig.Served.View.String(), "http://"+rig.Served.Answer.String()
				askRetreat(t, rig.Ladder, b.hero, b.life)
				var tv watch.ThoughtsView
				httpDo(t, "GET", view+"/api/thoughts", nil, &tv)
				if len(tv.Thoughts) != 1 {
					t.Fatalf("the Hub did not receive the thought: %+v", tv)
				}
				if scripted == "" {
					choice := tv.Thoughts[0].Choices[0].ID
					if code := httpDo(t, "POST", ans+"/api/answer", map[string]any{"thought": tv.Thoughts[0].ID, "answer": choice}, nil); code != 200 {
						t.Fatalf("answer route: %d", code)
					}
					if code := httpDo(t, "POST", view+"/api/answer", map[string]any{"thought": tv.Thoughts[0].ID, "answer": choice}, nil); code == 200 {
						t.Fatal("the viewer accepted a write")
					}
				} else {
					clk.Advance(40 * time.Second)
					rig.Ladder.Pump()
				}
				if n := rig.Ladder.Stats().Answered; scripted != "" && n == 0 {
					t.Errorf("the scripted profile did not answer: %+v", rig.Ladder.Stats())
				}
				if scripted == "" {
					httpDo(t, "GET", view+"/api/thoughts", nil, &tv)
					if len(tv.Thoughts) != 0 {
						t.Errorf("answered thought still open: %+v", tv.Thoughts)
					}
				}
				// An order becomes an area goal for its duration; cancelling removes it.
				var rep struct {
					Order *watch.Order `json:"order"`
				}
				if code := httpDo(t, "POST", ans+"/api/orders", map[string]any{"kind": "go", "area": "Dark Wood", "minutes": 15}, &rep); code != 200 || rep.Order == nil {
					t.Fatalf("order: %d %+v", code, rep)
				}
				var found bool
				for _, g := range m.Goals() {
					found = found || (g.Type == goals.FarmArea && g.Params.Area == "Dark Wood" && g.Params.UntilMins == 15)
				}
				if !found {
					t.Fatalf("no farm_area goal: %+v", m.Goals())
				}
				if code := httpDo(t, "POST", ans+"/api/orders/cancel", map[string]any{"id": rep.Order.ID}, nil); code != 200 {
					t.Fatalf("cancel: %d", code)
				}
				for _, g := range m.Goals() {
					if g.Params.Area == "Dark Wood" {
						t.Errorf("the cancelled order's goal is still queued: %+v", g)
					}
				}
				// "Go now" with no time limit: an endless area goal that replaces the open order; the map shows the bot's guard.
				httpDo(t, "POST", ans+"/api/orders", map[string]any{"kind": "go", "area": "Dark Wood", "until": "forever", "replace": true}, nil)
				httpDo(t, "POST", ans+"/api/orders", map[string]any{"kind": "go", "area": "Cold Plains", "minutes": 20, "replace": true}, nil)
				var farm []goals.Goal
				for _, g := range m.Goals() {
					if g.Type == goals.FarmArea {
						farm = append(farm, g)
					}
				}
				if len(farm) != 1 || farm[0].Params.Area != "Cold Plains" || farm[0].Params.UntilMins != 20 {
					t.Errorf("replace left %+v", farm)
				}
				var mv watch.MapView
				if code := httpDo(t, "GET", view+"/api/orders/map", nil, &mv); code != 200 {
					t.Fatalf("map: %d", code)
				}
				for _, n := range mv.Nodes {
					if (n.ID == "ftw") != n.Status.Hard || n.Status.HardWhy != map[bool]string{true: "no merc, clvl 6 < 16"}[n.ID == "ftw"] {
						t.Errorf("node %s: %+v", n.ID, n.Status)
					}
				}
			})
		}
	}
}

// Without the flags nothing is made: no enable call, no state directory, no listener.
func TestLadderOffDoesNothing(t *testing.T) {
	dir := t.TempDir()
	called := 0
	rig, err := startLadder(context.Background(), ladderOptions{viewAddr: "127.0.0.1:0", answerAddr: "127.0.0.1:0"}, "", filepath.Join(dir, "state"),
		fakeEnable(t, &watch.SimClock{}, &called), nil, nil, nil, nil)
	if rig != nil || err != nil || called != 0 || ladder.Active() != nil {
		t.Fatalf("%v %v %d", rig, err, called)
	}
	if ents, _ := os.ReadDir(dir); len(ents) != 0 {
		t.Fatalf("files made: %v", ents)
	}
	// -ladder alone with a scripted player: a store but no listener.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	rig, err = startLadder(context.Background(), ladderOptions{on: true, profile: "bold", viewAddr: l.Addr().String(), answerAddr: l.Addr().String()}, "", "",
		fakeEnable(t, &watch.SimClock{}, &called), nil, nil, nil, nil)
	if err != nil || rig == nil || rig.Hub != nil || rig.Served.View != nil {
		t.Fatalf("scripted without -watch must open no listener: %+v %v", rig, err)
	}
}

func TestLadderFlagRules(t *testing.T) {
	base := []string{"--sim"}
	for _, c := range []struct {
		args []string
		bad  string
	}{
		{[]string{"--watch"}, "needs --ladder"},
		{[]string{"--ladder-profile", "bold"}, "needs --ladder"},
		{[]string{"--ladder", "--watch"}, "--attach or --managed"},
	} {
		if _, err := parseFlags(append(append([]string{}, base...), c.args...)); err == nil || !strings.Contains(err.Error(), c.bad) {
			t.Errorf("%v: %v", c.args, err)
		}
	}
	for _, c := range []struct {
		args []string
		bad  string
	}{
		{[]string{"--attach", "--ladder"}, "someone to answer"},
		{[]string{"--attach", "--ladder", "--ladder-profile", "holdout:x"}, "whole number"},
		{[]string{"--attach", "--ladder", "--ladder-profile", "nobody"}, "unknown answer profile"},
	} {
		if _, err := parseFlags(c.args); err == nil || !strings.Contains(err.Error(), c.bad) {
			t.Errorf("%v: %v", c.args, err)
		}
	}
	for _, args := range [][]string{{"--attach", "--ladder", "--watch"}, {"--attach", "--ladder", "--ladder-profile", "holdout:101"}, {"--attach"}} {
		if _, err := parseFlags(args); err != nil {
			t.Errorf("%v: %v", args, err)
		}
	}
	if o, _ := parseFlags([]string{"--attach"}); o.ladder.on || o.ladder.watch {
		t.Error("ladder and watch must default to off")
	}
}

// The token rule of -watch: a non-loopback listener needs one, and nothing is bound when it is refused.
func TestWatchServeTokenRule(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	called := 0
	lo := ladderOptions{on: true, watch: true, viewAddr: "0.0.0.0:0", answerAddr: "127.0.0.1:0"}
	if _, err := startLadder(ctx, lo, "", "", fakeEnable(t, &watch.SimClock{}, &called), goalManager(t), nil, nil, nil); err == nil || !strings.Contains(err.Error(), "token") {
		t.Fatalf("want a token error, got %v", err)
	}
	if ladder.Active() != nil {
		t.Error("the ladder stayed installed after the screen was refused")
	}
	lo.watchToken = "s3cret-watch-token"
	if _, err := startLadder(ctx, lo, "s3cret-watch-token", "", fakeEnable(t, &watch.SimClock{}, &called), goalManager(t), nil, nil, nil); err == nil || !strings.Contains(err.Error(), "differ") {
		t.Fatalf("watch token equal to the API token must be refused, got %v", err)
	}
	rig, err := startLadder(ctx, lo, "api-token-of-the-bot", "", fakeEnable(t, &watch.SimClock{}, &called), goalManager(t), nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, port, _ := net.SplitHostPort(rig.Served.View.String())
	url := "http://127.0.0.1:" + port + "/api/thoughts"
	if code := httpDo(t, "GET", url, nil, nil); code != 401 {
		t.Errorf("no token: %d", code)
	}
	if code := httpDo(t, "GET", url+"?t=api-token-of-the-bot", nil, nil); code != 401 {
		t.Errorf("the bot API token must not open the viewer: %d", code)
	}
	if code := httpDo(t, "GET", url+"?t=s3cret-watch-token", nil, nil); code != 200 {
		t.Errorf("watch token: %d", code)
	}
	_, aport, _ := net.SplitHostPort(rig.Served.Answer.String())
	body := map[string]any{"thought": "none", "answer": "x"}
	post := "http://127.0.0.1:" + aport + "/api/answer"
	hdr := func(tok string) map[string]string { return map[string]string{"Authorization": "Bearer " + tok} }
	if code := httpDoH(t, "POST", post, body, hdr("api-token-of-the-bot")); code != 401 {
		t.Errorf("the bot API token must not open the answer route: %d", code)
	}
	if code := httpDoH(t, "POST", post, body, hdr("s3cret-watch-token")); code == 401 {
		t.Errorf("the watch token must pass the answer gate: %d", code)
	}
}

func TestLadderPostKeepsTheFlagAcrossATuningFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tuning.json")
	first := env.DefaultTuning()
	tn0, err := bot.NewTuner(&first, bot.Config{}, path)
	if err != nil {
		t.Fatal(err)
	}
	f := 20.0
	if err := tn0.Edit(api.TuneEdit{Overrides: map[string]*float64{"combat.fightRadius": &f}}); err != nil { // writes tuning.json
		t.Fatal(err)
	}
	for _, c := range []struct {
		on   bool
		want int
	}{{true, 1}, {false, 0}} {
		tune := env.DefaultTuning()
		tn, err := bot.NewTuner(&tune, bot.Config{}, path, ladderPost(ladderOptions{on: c.on}))
		if err != nil || tn == nil {
			t.Fatal(err)
		}
		if tune.Ladder.Enabled != c.want {
			t.Fatalf("flag %v: ladder.enabled %d, want %d", c.on, tune.Ladder.Enabled, c.want)
		}
	}
}

// Drills name their run and get their own store; an unnamed run keeps the shared one; "shared" is the explicit
// continuation (t0drill Result 2, defect 7).
func TestLadderStorePerRun(t *testing.T) {
	st := filepath.Join("state", "i5")
	shared := filepath.Join(st, "ladder")
	cases := []struct {
		name, run, flag string
		on              bool
		want            string
		bad             bool
	}{
		{"unnamed run", "", "", true, shared, false},
		{"named run", "t0live-b", "", true, filepath.Join(shared, "runs", "t0live-b"), false},
		{"flag wins", "t0live-b", "mine", true, filepath.Join(shared, "runs", "mine"), false},
		{"continue", "t0live-b", "shared", true, shared, false},
		{"off ignores both", "a/b", "", false, shared, false},
		{"bad name", "../x", "", true, "", true},
	}
	for _, c := range cases {
		t.Setenv("D2BOT_RUN", c.run)
		got, err := ladderDir(st, ladderOptions{on: c.on, store: c.flag})
		if (err != nil) != c.bad || got != c.want {
			t.Errorf("%s: got %q, %v want %q", c.name, got, err, c.want)
		}
	}
}

func TestWatchVideoOptions(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ffmpeg"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	ok := ladderOptions{on: true, watch: true, video: true, videoFPS: 8, videoQuality: 5, instance: "i5", viewAddr: "127.0.0.1:0", answerAddr: "127.0.0.1:0"}
	if err := ok.validate(false); err != nil {
		t.Fatalf("valid: %v", err)
	}
	if v, err := ok.newVideo(); err != nil || v == nil || v.Running() {
		t.Fatalf("a video must exist but not run before a viewer: %v %v", v, err)
	}
	if v, _ := (ladderOptions{on: true, watch: true}).newVideo(); v != nil {
		t.Error("video without the flag")
	}
	for name, mod := range map[string]func(*ladderOptions){
		"i0 has no private display": func(l *ladderOptions) { l.instance = "i0" },
		"odd instance id":           func(l *ladderOptions) { l.instance = "main" },
		"needs --watch":             func(l *ladderOptions) { l.watch = false; l.profile = "bold" },
		"fps too high":              func(l *ladderOptions) { l.videoFPS = 60 },
		"quality out of range":      func(l *ladderOptions) { l.videoQuality = 1 },
	} {
		l := ok
		mod(&l)
		if err := l.validate(false); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if err := (ladderOptions{video: true, instance: "i5"}).validate(false); err == nil {
		t.Error("--watch-video without --ladder accepted")
	}
	t.Setenv("PATH", t.TempDir())
	if err := ok.validate(false); err == nil || !strings.Contains(err.Error(), "ffmpeg") {
		t.Errorf("no tool: %v", err)
	}
}

func TestWatchVideoSourceTap(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no ffmpeg, no import: the tap source needs neither, and no X display
	ok := ladderOptions{on: true, watch: true, video: true, videoSource: "tap", tapURL: "http://127.0.0.1:8893", videoQuality: 5, instance: "i0"}
	if err := ok.validate(false); err != nil {
		t.Fatalf("tap source: %v", err)
	}
	if v, err := ok.newVideo(); err != nil || v == nil || v.Running() {
		t.Fatalf("a video must exist but not run before a viewer: %v %v", v, err)
	}
	for name, mod := range map[string]func(*ladderOptions){
		"fps above the tap rate": func(l *ladderOptions) { l.videoFPS = 7 },
		"unknown source":         func(l *ladderOptions) { l.videoSource = "vnc" },
		"bad url":                func(l *ladderOptions) { l.tapURL = "127.0.0.1:8893" },
	} {
		l := ok
		mod(&l)
		if err := l.validate(false); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	// x11 stays the default and keeps its own limits; the tap flags do nothing there
	x := ladderOptions{on: true, watch: true, video: true, videoFPS: 10, videoQuality: 5, instance: "i0"}
	if err := x.validate(false); err == nil {
		t.Error("x11 on i0 accepted: lost the private-display rule")
	}
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	var l ladderOptions
	ladderFlags(fs, &l)
	if err := fs.Parse(nil); err != nil || l.videoSource != "x11" || l.tapURL != "http://127.0.0.1:8893" {
		t.Errorf("defaults: %q %q %v", l.videoSource, l.tapURL, err)
	}
}

// The token file: created once with a random 0600 token, reused on restart, printed only as the one-time URL line.
func TestWatchTokenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "watch.token")
	var out bytes.Buffer
	old := watchTokenOut
	watchTokenOut = &out
	defer func() { watchTokenOut = old }()
	lo := ladderOptions{on: true, watch: true, watchTokenFile: path, viewAddr: "0.0.0.0:8891", answerAddr: "0.0.0.0:8892", origins: "https://tester.example/"}
	if err := lo.validate(false); err != nil {
		t.Fatal(err)
	}
	tok, err := lo.resolveWatchToken("api")
	if err != nil || len(tok) < 32 {
		t.Fatalf("%q %v", tok, err)
	}
	if st, err := os.Stat(path); err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("token file mode: %v %v", st, err)
	}
	if !strings.Contains(out.String(), "https://tester.example/watch?t="+tok) || strings.Count(out.String(), "\n") != 1 {
		t.Errorf("one-time URL line: %q", out.String())
	}
	out.Reset()
	again, err := lo.resolveWatchToken("api")
	if err != nil || again != tok || out.Len() != 0 {
		t.Errorf("restart must reuse the token silently: %q %v %q", again, err, out.String())
	}
	other, _, _ := loadOrCreateToken(filepath.Join(filepath.Dir(path), "b"))
	if other == tok {
		t.Error("tokens must be random")
	}
	os.Chmod(path, 0o644)
	if _, err := lo.resolveWatchToken("api"); err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Stat(path); st.Mode().Perm() != 0o600 {
		t.Error("a loose token file must be tightened")
	}
	os.WriteFile(path, []byte("\n"), 0o600)
	if _, err := lo.resolveWatchToken("api"); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("an empty token file must be refused, got %v", err)
	}
	os.WriteFile(path, []byte("short\n"), 0o600)
	if _, err := lo.resolveWatchToken("api"); err == nil {
		t.Error("a short token must be refused")
	}
}

// Flag/env rules of the watch token, and env-only enablement as the container entrypoint sets it.
func TestWatchTokenFlagRules(t *testing.T) {
	loop := ladderOptions{on: true, watch: true, viewAddr: "127.0.0.1:8891", answerAddr: "127.0.0.1:8892"}
	if err := loop.validate(false); err != nil {
		t.Errorf("loopback needs no token: %v", err)
	}
	open := loop
	open.viewAddr = ":8891"
	if err := open.validate(false); err == nil || !strings.Contains(err.Error(), "--watch-token") {
		t.Errorf("non-loopback without a watch token: %v", err)
	}
	open.watchToken = "0123456789abcdef0123"
	if err := open.validate(false); err != nil {
		t.Error(err)
	}
	open.watchTokenFile = "x"
	if err := open.validate(false); err == nil {
		t.Error("token and token file together must be refused")
	}
	off := ladderOptions{watchToken: "0123456789abcdef0123"}
	if err := off.validate(false); err == nil {
		t.Error("a watch token without --ladder must be refused")
	}
	t.Setenv("D2BOT_LADDER", "1")
	t.Setenv("D2BOT_WATCH", "1")
	t.Setenv("D2BOT_WATCH_ADDR", ":8891")
	t.Setenv("D2BOT_WATCH_ANSWER_ADDR", ":8892")
	t.Setenv("D2BOT_WATCH_ORIGIN", "https://tester.example")
	t.Setenv("D2BOT_WATCH_TOKEN_FILE", "/data/watch.token")
	t.Setenv("D2BOT_LADDER_STORE", "shared")
	t.Setenv("D2BOT_TOKEN", "api-token-of-the-bot")
	o, err := parseFlags([]string{"--attach"})
	if err != nil {
		t.Fatal(err)
	}
	l := o.ladder
	if !l.on || !l.watch || l.viewAddr != ":8891" || l.answerAddr != ":8892" || l.origins != "https://tester.example" || l.watchTokenFile != "/data/watch.token" || l.store != "shared" {
		t.Errorf("env-only enablement: %+v", l)
	}
	if o.token != "api-token-of-the-bot" || l.watchToken != "" {
		t.Error("the bot API token and the watch token must stay independent")
	}
}
