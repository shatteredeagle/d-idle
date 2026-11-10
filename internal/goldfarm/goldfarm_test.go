package goldfarm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/env"
	"d2lod-bot/internal/resource"
)

func tuning() resource.Tuning { return resource.FromEnv(env.DefaultTuning()) }

func TestWantIsRepairPlusKitPlusMargin(t *testing.T) {
	tu := tuning()
	base := resource.KitCost(tu) + tu.PovertyMargin
	if got := Want(0, tu); got != base {
		t.Errorf("no repair: %d, want %d", got, base)
	}
	if got := Want(120, tu); got != base+120 {
		t.Errorf("repair 120: %d", got)
	}
	if got := Want(-5, tu); got != base {
		t.Errorf("negative repair counts as none: %d", got)
	}
}

func TestOrderPersistsAcrossRestart(t *testing.T) {
	p := filepath.Join(t.TempDir(), StateFile)
	clk := time.Unix(1000, 0)
	c, err := New(p, func() time.Time { return clk })
	if err != nil {
		t.Fatal(err)
	}
	if c.On() {
		t.Fatal("on by default")
	}
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	c.Report(Status{Area: "Blood Moor", Have: 50, Want: 200})
	c2, err := New(p, nil)
	if err != nil || !c2.On() {
		t.Fatalf("restart lost the order: %v on=%v", err, c2.On())
	}
	c2.Report(Status{Area: "Blood Moor", Have: 80, Want: 200})
	if v := c2.View(); v.Earned != 30 || v.Since != 1000 {
		t.Errorf("earned counts from the first report: %+v", v)
	}
	if err := c2.Stop(); err != nil {
		t.Fatal(err)
	}
	if c3, _ := New(p, nil); c3.On() {
		t.Error("stop did not persist")
	}
}

func TestViewTexts(t *testing.T) {
	c, _ := New("", nil)
	if v := c.View(); v.On || v.Text != "Not farming gold." {
		t.Errorf("off: %+v", v)
	}
	c.Report(Status{Area: "X", Have: 1, Want: 2}) // ignored while off
	c.Start()
	if v := c.View(); !strings.Contains(v.Text, "choosing a zone") {
		t.Errorf("no report yet: %q", v.Text)
	}
	c.Report(Status{Area: "Blood Moor", Have: 40, Want: 190})
	if v := c.View(); v.Reached || v.Text != "Farming gold on your order: 40 of 190 in Blood Moor." {
		t.Errorf("short: %+v", v)
	}
	c.Report(Status{Area: "Blood Moor", Have: 190, Want: 190})
	if v := c.View(); !v.Reached || !strings.Contains(v.Text, "still farming in Blood Moor until you stop") {
		t.Errorf("reached keeps farming: %+v", v)
	}
	c.Report(Status{Note: "waiting, every safe zone is cleared or avoided for now", Have: 5, Want: 9})
	if v := c.View(); !strings.Contains(v.Text, "waiting") {
		t.Errorf("hold: %+v", v)
	}
}

func TestHaltEndsTheOrderAndSaysWhy(t *testing.T) {
	p := filepath.Join(t.TempDir(), StateFile)
	c, _ := New(p, nil)
	c.Start()
	c.Halt("level 1 is below the first safe zone")
	if c.On() {
		t.Fatal("halt left the order on")
	}
	c2, _ := New(p, nil)
	if v := c2.View(); v.On || !strings.Contains(v.Text, "below the first safe zone") {
		t.Errorf("reason lost after restart: %+v", v)
	}
	c2.Start()
	if v := c2.View(); strings.Contains(v.Text, "stopped") {
		t.Errorf("a new order shows the old reason: %q", v.Text)
	}
}

func TestDamagedStateFileIsAnError(t *testing.T) {
	p := filepath.Join(t.TempDir(), StateFile)
	if err := writeFile(p, "{nope"); err != nil {
		t.Fatal(err)
	}
	if _, err := New(p, nil); err == nil {
		t.Error("a damaged file must be reported")
	}
}

func writeFile(p, s string) error { return os.WriteFile(p, []byte(s), 0o600) }
