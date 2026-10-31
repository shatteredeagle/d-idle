package d2dreplay

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2dctl/fake"
)

var goodHash = strings.Repeat("ab", 32)

// gz builds a replay stream from raw lines (header line included by the caller).
func gz(lines ...string) []byte {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	for _, l := range lines {
		fmt.Fprintln(w, l)
	}
	w.Close()
	return b.Bytes()
}

const hdr = `{"replay":1,"hash":"detloop-journal-v1","every":100,"d2d":{},"meta":{}}`

func TestReadRejectsCraftedFiles(t *testing.T) {
	cases := map[string][]string{
		"blank verb":         {hdr, `{"v":" "}`},
		"tab verb":           {hdr, `{"v":"\t"}`},
		"newline in verb":    {hdr, `{"v":"click 1 2\nquit"}`},
		"control char":       {hdr, `{"v":"click 1\u0000 2"}`},
		"unicode newline":    {hdr, `{"v":"click 1 quit"}`},
		"screenshot verb":    {hdr, `{"v":"screenshot /tmp/x.png"}`},
		"read verb":          {hdr, `{"v":"state"}`},
		"unknown verb":       {hdr, `{"v":"frobnicate 1"}`},
		"path token":         {hdr, `{"v":"debug goto /etc/passwd"}`},
		"short checkpoint":   {hdr, `{"c":0,"h":"abc"}`},
		"non-hex checkpoint": {hdr, `{"c":0,"h":"` + strings.Repeat("zz", 32) + `"}`},
		"huge step":          {hdr, `{"v":"step 9223372036854775807"}`},
		"step over cap":      {hdr, `{"v":"step 1000001"}`},
		"negative tick":      {hdr, `{"t":-5,"v":"click 1 2"}`},
		"tick over range":    {hdr, `{"t":2000000000,"v":"click 1 2"}`},
		"checkpoint tick":    {hdr, `{"c":-1,"h":"` + goodHash + `"}`},
		"negative wall":      {hdr, `{"w":-1,"v":"click 1 2"}`},
		"garbage mid-file":   {hdr, `{"v":"click 1 2"}`, `not json`, `{"v":"click 3 4"}`},
		"garbage last line":  {hdr, `{"v":"click 1 2"}`, `not json`},
		"short sha":          {`{"replay":1,"hash":"detloop-journal-v1","d2d":{"sha256":"abc"}}`},
		"huge every":         {`{"replay":1,"hash":"detloop-journal-v1","every":99999999999}`},
		"negative every":     {`{"replay":1,"hash":"detloop-journal-v1","every":-3}`},
	}
	for name, lines := range cases {
		if _, err := ReadFrom(bytes.NewReader(gz(lines...))); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestReadAcceptsGoodFilesAndTruncatedTail(t *testing.T) {
	ok := gz(hdr, `{"v":"click 400 500","w":3}`, `{"v":"step 50"}`, `{"t":50,"c":50,"h":"`+goodHash+`"}`, `{"end":true,"ticks":50}`)
	rp, err := ReadFrom(bytes.NewReader(ok))
	if err != nil || rp.Truncated || rp.End == nil {
		t.Fatalf("%v %+v", err, rp)
	}
	// a stream cut inside the last line is a truncated recording, not an error
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	fmt.Fprintln(w, hdr)
	fmt.Fprintln(w, `{"v":"step 10"}`)
	fmt.Fprint(w, `{"v":"cli`)
	w.Flush()
	rp, err = ReadFrom(bytes.NewReader(b.Bytes()))
	if err != nil || !rp.Truncated || len(rp.Entries) != 1 {
		t.Fatalf("truncated: %v %+v", err, rp)
	}
}

func TestReadCapsDecompressedSizeAndEntries(t *testing.T) {
	old := maxDecompressed
	maxDecompressed = 1 << 20
	defer func() { maxDecompressed = old }()
	lines := []string{hdr}
	chunk := strings.Repeat(`{"v":"click 1 2"}`+"\n", 1000)
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	fmt.Fprintln(w, hdr)
	for i := 0; i < 200; i++ { // 3.4 MB decompressed, a few KB compressed
		fmt.Fprint(w, chunk)
	}
	w.Close()
	_ = lines
	if b.Len() > 200<<10 {
		t.Fatalf("test input too big: %d", b.Len())
	}
	if _, err := ReadFrom(bytes.NewReader(b.Bytes())); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("gzip bomb: %v", err)
	}
	maxDecompressed = old
	oe := maxEntries
	maxEntries = 500
	defer func() { maxEntries = oe }()
	if _, err := ReadFrom(bytes.NewReader(b.Bytes())); err == nil || !strings.Contains(err.Error(), "entries") {
		t.Fatalf("entry cap: %v", err)
	}
}

func TestReadLongLineIsAnError(t *testing.T) {
	long := `{"v":"click ` + strings.Repeat("1", 2<<20) + `"}`
	if _, err := ReadFrom(bytes.NewReader(gz(hdr, long))); err == nil {
		t.Fatal("2 MB line accepted")
	}
}

func TestTicksNeverOverflow(t *testing.T) {
	if n, ok := StepCount("step 9223372036854775807"); ok {
		t.Fatalf("huge step counted: %d", n)
	}
	rp := &Replay{Entries: []Entry{{Verb: "step 1000000"}, {Verb: "step 1000000"}}}
	if rp.Ticks() != 2000000 {
		t.Fatal(rp.Ticks())
	}
}

func FuzzReadFrom(f *testing.F) {
	f.Add(gz(hdr, `{"v":"click 1 2"}`, `{"v":"step 5"}`, `{"c":5,"h":"`+goodHash+`"}`, `{"end":true}`))
	f.Add(gz(hdr, `{"v":" "}`))
	f.Add(gz(hdr, `{"v":"step 99999999999999999999"}`))
	f.Add([]byte("not gzip"))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		rp, err := ReadFrom(bytes.NewReader(data))
		if err != nil {
			return
		}
		// whatever was accepted must be safe for every consumer
		_ = rp.Ticks()
		for _, c := range rp.Checkpoints() {
			_ = short(c.H)
		}
		for _, e := range rp.Entries {
			if e.Kind() == "verb" {
				if err := ValidVerb(e.Verb); err != nil {
					t.Fatalf("accepted verb %q: %v", e.Verb, err)
				}
			}
		}
	})
}

func TestLaunchRefusesUnsafeHeaderArgs(t *testing.T) {
	rec := &Recipe{Seed: 1, Class: 1, Name: "Viewy", Screen: "ingame"}
	bad := [][]string{
		{"--devctl", "/some/shared.sock"}, {"--lockstep", "--devctl=/x"}, {"--toggle"}, {"--toggle", "autoloot=maybe"},
		{"--toggle", "a=on;rm"}, {"--data", "/etc"}, {"--toggle", "--devctl"}, {"x"}, {""},
	}
	for _, a := range bad {
		if _, err := (&Launch{Sock: "/s", Build: Build{Args: a}, Recipe: rec}).Args(); err == nil {
			t.Errorf("args %q accepted", a)
		}
	}
	good := []string{"--toggle", "autoloot=off,autoparty=off", "--lockstep", "--no-save"}
	a, err := (&Launch{Sock: "/s", Build: Build{Args: good}, Recipe: rec}).Args()
	if err != nil || !strings.HasSuffix(strings.Join(a, " "), strings.Join(good, " ")) {
		t.Fatalf("%v %q", err, a)
	}
	for _, r := range []*Recipe{{Name: "-x", Class: 1}, {Name: "a b", Class: 1}, {Name: "", Class: 1}, {Name: "ok", Class: 99}, {Name: "ok", Class: 1, Screen: "--x"}, {Name: "../x", Class: 1}} {
		if _, err := (&Launch{Sock: "/s", Recipe: r}).Args(); err == nil {
			t.Errorf("recipe %+v accepted", r)
		}
	}
}

func TestPlayerRefusesBadVerbsAndTickOverflow(t *testing.T) {
	up := startFake(t, fake.Sample(), fake.Options{NoQuirk: true})
	c, _ := Dial(up.Path(), 5*time.Second)
	defer c.Close()
	for _, v := range []string{"click 1 2\nquit", "screenshot /tmp/x.png", " ", "step 9223372036854775807"} {
		rp := &Replay{Header: Header{Hash: HashV1}, Entries: []Entry{{Verb: v}}}
		if _, err := (&Player{Conn: c}).Run(context.Background(), rp); err == nil {
			t.Errorf("player sent %q", v)
		}
	}
	for _, l := range up.Lines() {
		if strings.Contains(l, "screenshot") || strings.Contains(l, "9223") {
			t.Errorf("reached d2d: %q", l)
		}
	}
}

func TestTapRejectsControlCharVerbs(t *testing.T) {
	tp, cancel, done, out := tapRig(t, nil)
	c := dialTap(t, tp.Listen)
	if _, st := c.do(t, "click 1\x00 2"); !strings.HasPrefix(st, "err") {
		t.Fatalf("status %q", st)
	}
	c.do(t, "click 3 4")
	cancel()
	<-done
	rp, err := Read(out)
	if err != nil || len(rp.Entries) != 1 || rp.Entries[0].Verb != "click 3 4" {
		t.Fatalf("%v %+v", err, rp)
	}
}

func TestRecordedFilesStayReadable(t *testing.T) {
	// the writer and reader agree: a normal recording passes the strict reader
	p := filepath.Join(t.TempDir(), "a.gz")
	w, _ := Create(p, header())
	w.Add(Entry{Verb: "debug goto 2"})
	w.Add(Entry{Verb: "cmd move 10 10"})
	w.Add(Entry{Verb: "lockstep on"})
	w.Add(Entry{Verb: "pad a"})
	w.Close(&Entry{})
	if _, err := Read(p); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(p)
}

func TestTakeAnyPicksFreeSlotAndFailsWhenAllHeld(t *testing.T) {
	d := t.TempDir()
	paths := []string{filepath.Join(d, "s0"), filepath.Join(d, "s1")}
	i0, rel0, err := TakeAny(context.Background(), paths, time.Second, nil)
	if err != nil || i0 != 0 {
		t.Fatal(i0, err)
	}
	i1, rel1, err := TakeAny(context.Background(), paths, time.Second, nil)
	if err != nil || i1 != 1 {
		t.Fatalf("second took slot %d: %v", i1, err)
	}
	if _, _, err := TakeAny(context.Background(), paths, 100*time.Millisecond, nil); err == nil {
		t.Fatal("all slots held but a lock was granted")
	}
	rel0()
	if i, rel, err := TakeAny(context.Background(), paths, time.Second, nil); err != nil || i != 0 {
		t.Fatalf("freed slot 0 not reused: %d %v", i, err)
	} else {
		rel()
	}
	rel1()
}
