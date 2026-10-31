package d2dreplay

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPool(t *testing.T) {
	t.Setenv("D2D_SLOTS", "")
	dir := t.TempDir()
	p, err := LoadPool(filepath.Join(dir, "none.json"))
	if err != nil || p.Slots != 4 || p.BotAddr(2) != "127.0.0.1:9102" || p.ViewPort(3) != 9203 {
		t.Fatalf("defaults: %+v %v", p, err)
	}
	f := filepath.Join(dir, "pool.json")
	os.WriteFile(f, []byte(`{"slots":8,"free_slots":4,"bot_base":9100,"view_base":9200}`), 0o644)
	if p, _ = LoadPool(f); p.Slots != 8 || !p.NeedsAdmission(4) || p.NeedsAdmission(3) {
		t.Fatalf("file: %+v", p)
	}
	t.Setenv("D2D_SLOTS", "99")
	if p, _ = LoadPool(f); p.Slots != MaxSlots {
		t.Fatalf("clamp: %+v", p)
	}
	t.Setenv("D2D_SLOTS", "x")
	if _, err = LoadPool(f); err == nil {
		t.Fatal("bad env accepted")
	}
	if p.LockFile(0) != LabLock || p.LockFile(5) != "/tmp/d2d-lab-slot-5.lock" {
		t.Fatal("lock files")
	}
}
