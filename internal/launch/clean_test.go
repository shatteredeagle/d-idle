package launch

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

// fake clock so the waits cost nothing
func fakeClock(o *ReapOpts) {
	now := time.Unix(0, 0)
	o.Now = func() time.Time { return now }
	o.Sleep = func(d time.Duration) { now = now.Add(d) }
}

// The G1 deadlock: orphans keep the wineserver alive, the reaper used to give up.
func TestReapIdleServerKeptByOrphans(t *testing.T) {
	f, o, kills, logs := reapFixture(t)
	fakeClock(&o)
	f.mk(t, 11, "winedevice.exe", 1000, p4)
	f.mk(t, 12, "explorer.exe", 1000, p4)
	f.mk(t, 21, "wineserver", 1000, p4)
	f.mk(t, 30, "winedevice.exe", 1000, p1) // other instance
	f.mk(t, 31, "winedevice.exe", 999, p4)  // other user
	o.Kill = func(pid int) error { *kills = append(*kills, pid); f.rm(pid); return nil }
	got := ReapWinedevices(p4, o)
	sort.Ints(got)
	if !reflect.DeepEqual(got, []int{11, 12}) || !reflect.DeepEqual(*kills, []int{11, 12}) {
		t.Fatalf("got %v kills %v logs %v", got, *kills, *logs)
	}
}

func TestCleanPrefixOrder(t *testing.T) {
	f, o, _, _ := reapFixture(t)
	fakeClock(&o)
	f.mk(t, 11, "winedevice.exe", 1000, p4)
	f.mk(t, 12, "services.exe", 1000, p4)
	f.mk(t, 13, "rpcss.exe", 1000, p4)
	f.mk(t, 14, "winedevice.exe", 1000, p1) // other instance: never
	f.mk(t, 21, "wineserver", 1000, p4)
	var ev []string
	o.ServerKill = func() error { ev = append(ev, "server-k"); f.rm(21); return nil }
	o.Kill = func(pid int) error { ev = append(ev, "term"); return nil } // ignores SIGTERM: stays
	o.Kill9 = func(pid int) error { ev = append(ev, "kill"); f.rm(pid); return nil }
	CleanPrefix(p4, o)
	want := []string{"server-k", "term", "term", "term", "kill", "kill", "kill"}
	if !reflect.DeepEqual(ev, want) {
		t.Fatalf("order %v", ev)
	}
	if _, err := os.Stat(filepath.Join(f.root, "14")); err != nil {
		t.Fatal("other instance's process touched")
	}
}

func TestCleanPrefixRefusals(t *testing.T) {
	f, o, kills, _ := reapFixture(t)
	fakeClock(&o)
	f.mk(t, 11, "winedevice.exe", 1000, p4)
	served := false
	o.ServerKill = func() error { served = true; return nil }
	o.GameRunning = func() bool { return true }
	if CleanPrefix(p4, o) != nil || served || len(*kills) != 0 {
		t.Fatal("cleaned while the game runs")
	}
	o.GameRunning = nil
	if CleanPrefix("", o) != nil || len(*kills) != 0 {
		t.Fatal("empty prefix")
	}
	o.GOOS = "darwin"
	if CleanPrefix(p4, o) != nil || served {
		t.Fatal("non-linux")
	}
}

func TestPrepareLaunchGuards(t *testing.T) {
	f, o, kills, _ := reapFixture(t)
	fakeClock(&o)
	pfx := "/data/prefixes/i4/proton"
	f.mk(t, 11, "winedevice.exe", 1000, pfx+"/pfx")
	run := func(inst string, running bool) error {
		g := &Game{Cfg: Config{Instance: inst, PrefixDir: pfx}, Reap: o, find: func() (Proc, bool) { return Proc{PID: 5}, running }}
		g.Reap.ServerKill = func() error { return nil }
		return g.PrepareLaunch()
	}
	for _, id := range []string{"", "i0", "main"} {
		if err := run(id, false); err != nil || len(*kills) != 0 {
			t.Fatalf("%q: err %v kills %v", id, err, *kills)
		}
	}
	if err := run("i4", true); err == nil || len(*kills) != 0 {
		t.Fatalf("running game: %v %v", err, *kills)
	}
	if err := run("i4", false); err != nil || !reflect.DeepEqual(*kills, []int{11}) {
		t.Fatalf("clean: %v %v", err, *kills)
	}
	t.Setenv("D2BOT_REAP", "0")
	*kills = nil
	if run("i4", false); len(*kills) != 0 {
		t.Fatal("knob ignored")
	}
}

func TestWaitWindow(t *testing.T) {
	ctx := context.Background()
	slept := 0
	sl := func(time.Duration) { slept++ }
	if waitWindow(ctx, nil, time.Minute, sl) != nil {
		t.Fatal("nil check must pass")
	}
	n := 0
	if err := waitWindow(ctx, func() bool { n++; return n == 4 }, time.Minute, sl); err != nil || slept != 3 {
		t.Fatal(err, slept)
	}
	err := waitWindow(ctx, func() bool { return false }, 2*time.Second, sl)
	if err == nil || !strings.Contains(err.Error(), "no Diablo II window") {
		t.Fatal(err)
	}
}

func TestHungMounts(t *testing.T) {
	mf := filepath.Join(t.TempDir(), "mounts")
	os.WriteFile(mf, []byte("proc /proc proc rw 0 0\n//nas.example/media /mnt/media cifs rw 0 0\n//nas.example/x /mnt/with\\040space smb3 rw 0 0\n/dev/sda1 /home ext4 rw 0 0\n"), 0o644)
	if got := NetMounts(mf); !reflect.DeepEqual(got, []string{"/mnt/media", "/mnt/with space"}) {
		t.Fatal(got)
	}
	o := MountOpts{MountsFile: mf, Stat: func(_ context.Context, p string) bool { return p != "/mnt/media" }}
	if got := HungMounts(context.Background(), o); !reflect.DeepEqual(got, []string{"/mnt/media"}) {
		t.Fatal(got)
	}
	if r := hungReason(context.Background(), o); r != "mount /mnt/media is hung (host down): Wine will stall" {
		t.Fatal(r)
	}
	o.Stat = func(context.Context, string) bool { return true }
	if hungReason(context.Background(), o) != "" {
		t.Fatal("reason for healthy mounts")
	}
	if NetMounts(filepath.Join(t.TempDir(), "none")) != nil {
		t.Fatal("missing file")
	}
}
