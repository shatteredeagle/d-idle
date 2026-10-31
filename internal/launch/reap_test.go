package launch

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"
)

type reapProc struct{ root string }

func (f reapProc) mk(t *testing.T, pid int, comm string, uid int, prefix string) {
	t.Helper()
	d := filepath.Join(f.root, strconv.Itoa(pid))
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	w := func(n, s string) {
		if err := os.WriteFile(filepath.Join(d, n), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	w("comm", comm+"\n")
	w("status", "Name:\t"+comm+"\nUid:\t"+strconv.Itoa(uid)+"\t"+strconv.Itoa(uid)+"\t0\t0\n")
	if prefix != "" {
		w("environ", "A=b\x00WINEPREFIX="+prefix+"\x00")
	}
}

func (f reapProc) rm(pid int) { os.RemoveAll(filepath.Join(f.root, strconv.Itoa(pid))) }

func reapFixture(t *testing.T) (reapProc, ReapOpts, *[]int, *[]string) {
	root := t.TempDir()
	var kills []int
	var logs []string
	o := ReapOpts{
		ProcRoot: root, UID: 1000, GOOS: "linux",
		Kill:  func(pid int) error { kills = append(kills, pid); return nil },
		Log:   func(s string) { logs = append(logs, s) },
		Sleep: func(time.Duration) {},
	}
	return reapProc{root}, o, &kills, &logs
}

const p4, p1 = "/d/prefixes/i4/proton/pfx", "/d/prefixes/i1/proton/pfx"

func TestReapGuards(t *testing.T) {
	f, o, kills, _ := reapFixture(t)
	f.mk(t, 11, "winedevice.exe", 1000, p4+"/") // orphan of i4, trailing slash
	f.mk(t, 12, "winedevice.exe", 1000, p1)     // other instance
	f.mk(t, 13, "winedevice.exe", 999, p4)      // other user
	f.mk(t, 14, "services.exe", 1000, p4)       // other program
	f.mk(t, 15, "winedevice.exe", 1000, "")     // no environ
	f.mk(t, 16, "winedevice.exe", 1000, "/h/u/.wine")
	f.mk(t, 17, "winedevice.exe", 1000, p4+"x") // prefix with same leading text
	f.mk(t, 20, "wineserver", 1000, p1)         // other prefix's server must not block
	got := ReapWinedevices(p4, o)
	if want := []int{11}; !reflect.DeepEqual(got, want) || !reflect.DeepEqual(*kills, want) {
		t.Fatalf("got %v kills %v, want %v", got, *kills, want)
	}
}

func TestReapRefusesWithServer(t *testing.T) {
	for _, name := range []string{"wineserver", "wineserver64"} {
		f, o, kills, logs := reapFixture(t)
		f.mk(t, 11, "winedevice.exe", 1000, p4)
		f.mk(t, 12, "d2input.exe", 1000, p4) // a live client: the server is in use
		f.mk(t, 21, name, 1000, p4)
		n := 0
		now := time.Unix(0, 0)
		o.Now = func() time.Time { return now }
		o.Sleep = func(d time.Duration) { n++; now = now.Add(d) }
		if got := ReapWinedevices(p4, o); got != nil || len(*kills) != 0 {
			t.Fatalf("%s: killed %v", name, *kills)
		}
		if n == 0 || len(*logs) != 1 {
			t.Fatalf("%s: waited %d times, logs %v", name, n, *logs)
		}
	}
}

func TestReapWaitsForServerExit(t *testing.T) {
	f, o, kills, _ := reapFixture(t)
	f.mk(t, 11, "winedevice.exe", 1000, p4)
	f.mk(t, 21, "wineserver", 1000, p4)
	polls := 0
	o.Sleep = func(time.Duration) {
		if polls++; polls == 3 {
			f.rm(21)
		}
	}
	got := ReapWinedevices(p4, o)
	if !reflect.DeepEqual(got, []int{11}) || len(*kills) != 1 {
		t.Fatalf("got %v", got)
	}
}

func TestReapRecheckBeforeEachKill(t *testing.T) {
	f, o, kills, logs := reapFixture(t)
	f.mk(t, 41, "winedevice.exe", 1000, p4)
	f.mk(t, 42, "winedevice.exe", 1000, p4)
	o.Kill = func(pid int) error {
		*kills = append(*kills, pid)
		f.mk(t, 50, "wineserver", 1000, p4) // the first kill makes a server appear
		return nil
	}
	ReapWinedevices(p4, o)
	if len(*kills) != 1 {
		t.Fatalf("kills %v, want exactly 1", *kills)
	}
	if len(*logs) < 1 {
		t.Fatal("no log line")
	}
}

func TestReapNotLinuxOrEmpty(t *testing.T) {
	f, o, kills, _ := reapFixture(t)
	f.mk(t, 11, "winedevice.exe", 1000, p4)
	o.GOOS = "darwin"
	if ReapWinedevices(p4, o) != nil {
		t.Fatal("reaped on darwin")
	}
	o.GOOS = "linux"
	if ReapWinedevices("", o) != nil || len(*kills) != 0 {
		t.Fatal("reaped an empty prefix")
	}
}

func TestReapLogsPids(t *testing.T) {
	f, o, _, logs := reapFixture(t)
	f.mk(t, 11, "winedevice.exe", 1000, p4)
	f.mk(t, 12, "winedevice.exe", 1000, p4)
	ReapWinedevices(p4, o)
	if len(*logs) != 1 || (*logs)[0] != "reap: killed orphaned winedevice.exe pids 11 12 of "+p4 {
		t.Fatalf("logs %q", *logs)
	}
}

func TestReapAfterStopSkips(t *testing.T) {
	f, o, kills, _ := reapFixture(t)
	pfx := "/data/prefixes/i4/proton/pfx"
	f.mk(t, 11, "winedevice.exe", 1000, pfx)
	cfg := Config{Instance: "i4", PrefixDir: "/data/prefixes/i4/proton"}
	t.Setenv("D2BOT_REAP", "0")
	reapAfterStop(cfg, o)
	if len(*kills) != 0 {
		t.Fatal("knob off ignored")
	}
	t.Setenv("D2BOT_REAP", "")
	for _, id := range []string{"", "i0"} {
		c := cfg
		c.Instance = id
		reapAfterStop(c, o)
	}
	if len(*kills) != 0 {
		t.Fatal("i0 or unset instance reaped")
	}
	reapAfterStop(cfg, o)
	if !reflect.DeepEqual(*kills, []int{11}) {
		t.Fatalf("kills %v", *kills)
	}
}
