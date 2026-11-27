package logx

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type failW struct{}

func (failW) Write([]byte) (int, error) { return 0, errors.New("broken pipe") }

func TestTeeSurvivesBrokenWriter(t *testing.T) {
	p := filepath.Join(t.TempDir(), "logs", "bot.log")
	r, err := OpenRotating(p, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	w := Tee(failW{}, r) // stdout first: a closed pipe must not stop the file
	for i := 0; i < 3; i++ {
		if _, err := w.Write([]byte("line\n")); err != nil {
			t.Fatal(err)
		}
	}
	b, _ := os.ReadFile(p)
	if string(b) != "line\nline\nline\n" {
		t.Fatalf("file = %q", b)
	}
}

func TestRotateKeepsCount(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bot.log")
	r, err := OpenRotating(p, 20, 2)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		r.Write([]byte(strings.Repeat("x", 9) + string(rune('a'+i)) + "\n")) // 11 bytes: one per file
	}
	for _, n := range []string{"bot.log", "bot.log.1", "bot.log.2"} {
		if _, err := os.Stat(filepath.Join(filepath.Dir(p), n)); err != nil {
			t.Fatalf("%s: %v", n, err)
		}
	}
	if _, err := os.Stat(p + ".3"); err == nil {
		t.Fatal("kept more than keep rotations")
	}
	got, err := Tail(p, 0, nil)
	if err != nil || len(got) != 3 || !strings.HasSuffix(got[2], "h") || !strings.HasSuffix(got[0], "f") {
		t.Fatalf("tail = %v, %v", got, err)
	}
}

func TestTailFilterAndNul(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bot.log")
	os.WriteFile(p, []byte("a tick 1\nb other\n\x00\x00c tick 2\nd tick 3\n"), 0o644)
	got, err := Tail(p, 2, regexp.MustCompile("tick"))
	if err != nil || len(got) != 2 || got[0] != "c tick 2" || got[1] != "d tick 3" {
		t.Fatalf("got %v, %v", got, err)
	}
	if _, err := Tail(p+".missing", 5, nil); err == nil {
		t.Fatal("missing file must be an error")
	}
}
