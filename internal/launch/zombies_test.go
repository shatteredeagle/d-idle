package launch

import (
	"os/exec"
	"testing"
	"time"
)

func TestZombieReaperWaitsOneScan(t *testing.T) {
	var waited []int
	z := &ZombieReaper{
		Zombies: func() []int { return []int{7} },
		Wait:    func(pid int) error { waited = append(waited, pid); return nil },
	}
	if got := z.Scan(); len(got) != 0 || len(waited) != 0 {
		t.Fatalf("first scan must only remember: %v", got)
	}
	if got := z.Scan(); len(got) != 1 || got[0] != 7 {
		t.Fatalf("second scan: %v", got)
	}
	if got := z.Scan(); len(got) != 0 { // reaped pid is not listed again by the real scan; here it re-arms
		t.Fatalf("third scan: %v", got)
	}
}

func TestParseStat(t *testing.T) {
	st, pp, ok := parseStat("123 (Game (x).exe) Z 45 1 1 0")
	if !ok || st != 'Z' || pp != 45 {
		t.Fatal(st, pp, ok)
	}
}

// A real orphan zombie: a shell backgrounds a child and exits; with subreaper set the grandchild is ours.
func TestZombieReaperReal(t *testing.T) {
	if err := BecomeSubreaper(); err != nil {
		t.Skip(err)
	}
	if err := exec.Command("sh", "-c", "sleep 0.2 &").Run(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond) // the sleep exits and stays a zombie of ours
	if len(ownZombies()) == 0 {
		t.Skip("no adopted zombie appeared")
	}
	z := &ZombieReaper{}
	z.Scan()
	z.Scan()
	if n := len(ownZombies()); n != 0 {
		t.Fatalf("%d zombies left", n)
	}
}
