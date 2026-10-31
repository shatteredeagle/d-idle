package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dctl/fake"
)

func rig(t *testing.T) (*Mux, *fake.Server, string, string) {
	t.Helper()
	up, cleanup, err := fake.TempSocket()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	fs, err := fake.New(up, fake.Sample(), fake.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(fs.Close)
	mux := NewMux(up)
	bot, view := filepath.Join(filepath.Dir(up), "bot.sock"), filepath.Join(filepath.Dir(up), "view.sock")
	for p, isBot := range map[string]bool{bot: true, view: false} {
		ln, err := mux.Serve(p, isBot, "/tmp/shots")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { ln.Close() })
	}
	return mux, fs, bot, view
}

func TestViewerSharesTheSocketReadOnly(t *testing.T) {
	_, fs, bot, view := rig(t)
	ctx := context.Background()
	b := d2dctl.New(bot, d2dctl.Options{})
	v := d2dctl.New(view, d2dctl.Options{})
	defer b.Close()
	defer v.Close()
	if _, err := b.Step(ctx, 3); err != nil {
		t.Fatal(err)
	}
	st, _, err := v.State(ctx) // the bot is connected, the viewer still reads
	if err != nil || st.Name != "Spike" {
		t.Fatalf("viewer state: %v %+v", err, st)
	}
	for _, bad := range []string{"step 5", "cmd move 1 1", "click 1 1", "quit", "debug goto 3", "key Escape", "screenshot /etc/x.png", "screenshot /tmp/shots/../x.png"} {
		if r, err := v.Do(ctx, bad); err == nil || (r != nil && r.Rest == "") && !strings.Contains(err.Error(), "read-only") {
			t.Errorf("viewer %q allowed: %v", bad, err)
		}
	}
	if _, err := v.Do(ctx, "screenshot /tmp/shots/a.png"); err != nil {
		t.Errorf("screenshot in dir refused: %v", err)
	}
	for _, l := range fs.Lines() {
		if strings.HasPrefix(l, "cmd") || l == "quit" || strings.HasPrefix(l, "click") || strings.HasPrefix(l, "debug goto") {
			t.Errorf("a refused line reached d2d: %q", l)
		}
	}
}

func TestWatchModeRendersAndPaces(t *testing.T) {
	mux, fs, bot, _ := rig(t)
	b := d2dctl.New(bot, d2dctl.Options{})
	defer b.Close()
	ctx := context.Background()
	b.Step(ctx, 2)
	mux.SetMode(Mode{Watch: true, Speed: 50})
	t0 := time.Now()
	for i := 0; i < 5; i++ {
		b.Step(ctx, 25) // 1 virtual s each at 50x = 20 ms
	}
	if d := time.Since(t0); d < 90*time.Millisecond {
		t.Errorf("watch pace not applied: %v", d)
	}
	var plain, render int
	for _, l := range fs.Lines() {
		switch l {
		case "step 2":
			plain++
		case "step 25 render":
			render++
		}
	}
	if plain != 1 || render != 5 {
		t.Errorf("plain %d render %d, lines %v", plain, render, fs.Lines())
	}
	if mux.ticks.Load() != 127 {
		t.Errorf("ticks %d", mux.ticks.Load())
	}
}

func TestFeedSnap(t *testing.T) {
	mux, _, _, view := rig(t)
	f := NewFeed(view, mux, 15, "")
	f.poll(context.Background())
	s := f.Snap()
	if !s.OK || s.Level != 1 || s.Area != "Rogue Encampment" || s.Name != "Spike" || s.Life != [2]int{45, 45} {
		t.Fatalf("%+v", s)
	}
	if len(s.Mons) != 2 || s.X < 30 || s.X > 31 || !s.Lockstep {
		t.Fatalf("%+v", s)
	}
	f.poll(context.Background())
	if f.Snap().Seq != 2 {
		t.Fatal("seq")
	}
}

func TestFramesGrab(t *testing.T) {
	// The fake writes no file: the first grab must fail cleanly; with a PNG in place it yields a JPEG.
	mux, _, _, view := rig(t)
	fr, err := NewFrames(view, mux, "/tmp/shots", 5)
	if err != nil {
		t.Fatal(err)
	}
	fr.JPEG = true
	defer os.Remove("/tmp/shots/shot.png")
	os.Remove("/tmp/shots/shot.png")
	if err := fr.grab(context.Background()); err == nil {
		t.Fatal("expected a read error")
	}
	var buf bytes.Buffer
	png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 8, 6)))
	os.WriteFile("/tmp/shots/shot.png", buf.Bytes(), 0o600)
	mux.ticks.Store(40)
	if err := fr.grab(context.Background()); err != nil {
		t.Fatal(err)
	}
	if c := fr.Current(); c.Seq != 1 || c.Ticks != 40 || len(c.Data) < 100 || c.Type != "image/jpeg" {
		t.Fatalf("%+v", c)
	}
}

func BenchmarkStepThroughTap(b *testing.B) {
	up, cleanup, _ := fake.TempSocket()
	defer cleanup()
	fs, _ := fake.New(up, fake.Sample(), fake.Options{})
	defer fs.Close()
	mux := NewMux(up)
	bot := filepath.Join(filepath.Dir(up), "bot.sock")
	ln, _ := mux.Serve(bot, true, "")
	defer ln.Close()
	direct := d2dctl.New(up, d2dctl.Options{})
	tap := d2dctl.New(bot, d2dctl.Options{})
	ctx := context.Background()
	b.Run("direct", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			direct.Step(ctx, 3)
		}
	})
	direct.Close()
	b.Run("tap", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tap.Step(ctx, 3)
		}
	})
}

func TestStreamPageIsStreamSafe(t *testing.T) {
	s := &server{}
	for _, path := range []string{"/stream", "/live"} {
		rec := httptest.NewRecorder()
		s.routes().ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		b := rec.Body.String()
		if rec.Code != 200 || !strings.Contains(b, "d2d (reimplementation)") || !strings.Contains(b, "/stream.mjpg") ||
			!strings.Contains(b, "/api/pub") || !strings.Contains(b, "visibilitychange") || !strings.Contains(b, "id=\"engine\"") || !strings.Contains(b, "tile=1") {
			t.Errorf("%s: code %d, page lacks badge, frames, pub feed or tile variant", path, rec.Code)
		}
		for _, bad := range []string{"/api/state", "/events", "/api/mode", ".lan", "worker.example", "127.0.0.1", "localhost", "/home/", "/tmp/", "Authorization"} {
			if strings.Contains(b, bad) {
				t.Errorf("%s contains %q", path, bad)
			}
		}
	}
}

func TestPubIsSanitized(t *testing.T) {
	bot := `{"totals":{"kills":40,"deaths":2},"runner":{"activity":"Fighting"},"activeGoal":{"title":"Level up in Blood Moor"}}`
	sn := Snap{OK: true, Area: "Blood Moor", Name: "Loopy", Class: "Sorceress", CLvl: 5, Err: "dial unix /tmp/d2dview-1000/x.sock: refused",
		Dead: 3, Speed: 22.5, Bot: json.RawMessage(bot), Mons: []Mon{{Name: "secret@host", X: 1, Y: 2}}}
	p := pubOf(sn)
	b, _ := json.Marshal(p)
	for _, bad := range []string{"/tmp", "sock", "refused", "secret@host", "Name\":\"secret"} {
		if strings.Contains(string(b), bad) {
			t.Errorf("pub json leaks %q: %s", bad, b)
		}
	}
	if p.Engine != "d2d" {
		t.Errorf("engine = %q", p.Engine)
	}
	badge = "d.idle testing stream"
	if q := pubOf(sn); q.Backend != "d.idle testing stream" {
		t.Errorf("badge flag not used: %q", q.Backend)
	}
	badge = Backend
	if p.Backend != "d2d (reimplementation)" || p.Kills != 40 || p.Deaths != 2 || p.Goal != "Level up in Blood Moor: Fighting" {
		t.Errorf("pub = %+v", p)
	}
	for _, in := range []string{"see http://worker.example/x", "go to 192.0.2.5", "/opt/x", "my token abc", "a@b"} {
		if got := safeText(in, 50); got != "" {
			t.Errorf("safeText(%q) = %q", in, got)
		}
	}
	if got := safeText("Walking to Cold Plains!!", 50); got != "Walking to Cold Plains" {
		t.Errorf("safeText = %q", got)
	}
}

func TestReadOnlyRefusesVerbVariants(t *testing.T) {
	bad := []string{
		"step", "step 5", "STEP", "STEP 5", " step", " step 5", "\"step\"", "'step 5'", "step\n", "step\r", "step\t5", "step\x0b5",
		"state;step 5", "state,step 5", "state\nstep 5", "state\x00", "State", "STATE", "\"state\"", "state\tstep\t5", "state step",
		"debug\tgoto\t2", "debug  goto 2", "debug GRID 3", "dump\titems\tx", "screenshot\t/tmp/shots/a.png\tb", "quit", " quit", "quit",
		" step 5", "step  5", "", " ", "\t",
	}
	for _, l := range bad {
		if readOnly(l, "/tmp/shots") {
			t.Errorf("readOnly(%q) = true", l)
		}
	}
	for _, l := range []string{"state", " state ", "state\t", "dump  items", "debug\tgrid 3"} {
		if !readOnly(l, "/tmp/shots") {
			t.Errorf("readOnly(%q) = false (whitespace-normalised read verb)", l)
		}
	}
}

func TestViewerForwardsNormalisedFields(t *testing.T) {
	_, fs, _, view := rig(t)
	v := d2dctl.New(view, d2dctl.Options{})
	defer v.Close()
	v.Do(context.Background(), "dump\t items")
	for _, l := range fs.Lines() {
		if strings.ContainsAny(l, "\t") {
			t.Errorf("raw whitespace reached d2d: %q", l)
		}
	}
}

func TestWatchSpeedClampedAndPaceOutsideLock(t *testing.T) {
	s := &server{mux: NewMux("/nonexistent"), watchSpeed: 1}
	for q, want := range map[string]float64{"0.001": minWatchSpeed, "5000": maxWatchSpeed, "-1": 1, "abc": 1, "0": 0, "8": 8} {
		rq := httptest.NewRequest("POST", "/api/mode?m=watch&speed="+q, nil)
		s.setMode(httptest.NewRecorder(), rq)
		if got := s.mux.Mode().Speed; got != want {
			t.Errorf("speed %s -> %v, want %v", q, got, want)
		}
	}
	// a slow pacing sleep must not hold the mux lock: the viewer reads while the bot waits
	mux, _, bot, view := rig(t)
	mux.SetMode(Mode{Watch: true, Speed: minWatchSpeed})
	b := d2dctl.New(bot, d2dctl.Options{})
	defer b.Close()
	go b.Step(context.Background(), 25) // 1 virtual s at the floor = 2 s of pace
	time.Sleep(300 * time.Millisecond)
	v := d2dctl.New(view, d2dctl.Options{})
	defer v.Close()
	t0 := time.Now()
	if _, _, err := v.State(context.Background()); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(t0); d > 500*time.Millisecond {
		t.Errorf("viewer waited %v behind the pacing sleep", d)
	}
}

func TestPaceSleepIsCappedAndCancelledByMode(t *testing.T) {
	mux := NewMux("/nonexistent")
	mux.SetMode(Mode{Watch: true, Speed: minWatchSpeed})
	done := make(chan time.Duration)
	go func() { t0 := time.Now(); mux.pace(1_000_000); done <- time.Since(t0) }()
	time.Sleep(250 * time.Millisecond)
	mux.SetMode(Mode{}) // back to fast ends the sleep
	select {
	case d := <-done:
		if d > time.Second {
			t.Errorf("sleep took %v after the mode switch", d)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("pace ignored the mode switch")
	}
	if maxPaceSleep > 5*time.Second {
		t.Errorf("maxPaceSleep %v", maxPaceSleep)
	}
}
