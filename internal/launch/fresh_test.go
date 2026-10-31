package launch

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

func whoOf(w *fakeWorld, level int) func() (Who, error) {
	return func() (Who, error) { return Who{Name: w.player, Class: w.class, Level: level}, nil }
}

func TestValidCharName(t *testing.T) {
	for _, ok := range []string{"Fresha", "ab", "Soak-y", "Soak_y", "abcdefghijklmno"} {
		if err := ValidCharName(ok); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "a", "abcdefghijklmnop", "Soak1", "-ab", "ab_", "a-b-c", "a b", "../x", "Soaky\n"} {
		if err := ValidCharName(bad); !errors.Is(err, ErrBadName) {
			t.Errorf("%q accepted (%v)", bad, err)
		}
	}
}

func TestAllSevenClassesCreateAndVerify(t *testing.T) {
	if len(Classes()) != 7 {
		t.Fatalf("classes %v", Classes())
	}
	for _, c := range Classes() {
		w := newWorld(ScreenMainMenu)
		w.dropNext = 1
		err := quick(w).CreateVerified(context.Background(), "Fresha", c, FreshOptions{Who: whoOf(w, 1)})
		if err != nil {
			t.Fatalf("%s: %v", c, err)
		}
		if w.screen != ScreenMainMenu || w.typed != "Fresha" {
			t.Fatalf("%s: screen %s typed %q (not saved and exited)", c, w.screen, w.typed)
		}
	}
}

func TestCreateVerifiedRetriesClassOnce(t *testing.T) {
	w := newWorld(ScreenMainMenu)
	w.dropNext = 1
	f := quick(w)
	f.ClassTries = 1 // first attempt ends in ErrWrongClass, the second works
	var lines []string
	err := f.CreateVerified(context.Background(), "Retrya", "druid", FreshOptions{Who: whoOf(w, 1), Log: func(s string, a ...any) { lines = append(lines, s) }})
	if err != nil || len(lines) < 2 {
		t.Fatalf("err=%v log=%v", err, lines)
	}
}

func TestCreateVerifiedGivesUpAfterAttempts(t *testing.T) {
	w := newWorld(ScreenMainMenu)
	w.dropNext = 99
	err := quick(w).CreateVerified(context.Background(), "Retrya", "druid", FreshOptions{Who: whoOf(w, 1), Attempts: 2})
	if !errors.Is(err, ErrWrongClass) {
		t.Fatalf("err=%v", err)
	}
}

func TestCreateVerifiedNameTakenIsFinal(t *testing.T) {
	w := newWorld(ScreenCharSelect, "Taken")
	err := quick(w).CreateVerified(context.Background(), "Taken", "druid", FreshOptions{Who: whoOf(w, 1), Attempts: 3})
	if !errors.Is(err, ErrNameTaken) {
		t.Fatalf("err=%v", err)
	}
	n := 0
	for _, c := range w.clicks {
		if c == [2]int{ClassClick["druid"][0], ClassClick["druid"][1]} {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("name clash retried: %d class clicks", n)
	}
}

func TestCreateVerifiedMismatchAndInputs(t *testing.T) {
	w := newWorld(ScreenMainMenu)
	err := quick(w).CreateVerified(context.Background(), "Wronga", "paladin", FreshOptions{Who: whoOf(w, 5), Leave: -1})
	if !errors.Is(err, ErrVerifyFailed) || !strings.Contains(err.Error(), "level 5") {
		t.Fatalf("err=%v", err)
	}
	if w.screen != "in-game" {
		t.Fatal("Leave<0 must stay in game")
	}
	w = newWorld(ScreenMainMenu)
	if err := quick(w).CreateVerified(context.Background(), "Okname", "wizard", FreshOptions{Who: whoOf(w, 1)}); err == nil || len(w.clicks) != 0 {
		t.Fatalf("unknown class must fail before any click: %v", err)
	}
	if err := quick(w).CreateVerified(context.Background(), "bad1", "druid", FreshOptions{Who: whoOf(w, 1)}); !errors.Is(err, ErrBadName) || len(w.clicks) != 0 {
		t.Fatalf("bad name must fail before any click: %v", err)
	}
}

func TestCreateVerifiedWaitsForPlayer(t *testing.T) {
	w := newWorld(ScreenMainMenu)
	n := 0
	who := func() (Who, error) {
		if n++; n < 4 {
			return Who{}, errors.New("no player yet")
		}
		return Who{Name: w.player, Class: w.class, Level: 1}, nil
	}
	if err := quick(w).CreateVerified(context.Background(), "Slowa", "amazon", FreshOptions{Who: who}); err != nil || n != 4 {
		t.Fatalf("err=%v n=%d", err, n)
	}
}

// The throwaway list of tools/soak/fresh.sh must only hold names the game rules accept.
func TestFreshNamesFileIsValid(t *testing.T) {
	b, err := os.ReadFile("../../tools/soak/fresh-names.txt")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, l := range strings.Split(string(b), "\n") {
		if l = strings.TrimSpace(l); l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		n++
		if err := ValidCharName(l); err != nil {
			t.Error(err)
		}
	}
	if n == 0 {
		t.Fatal("empty list")
	}
}
