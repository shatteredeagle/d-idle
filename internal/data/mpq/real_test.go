package mpq

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const realDir = "/tmp/d2mpq"

func openReal(t *testing.T, name string) *Archive {
	t.Helper()
	p := filepath.Join(realDir, name)
	if _, err := os.Stat(p); err != nil {
		t.Skipf("%s not present", p)
	}
	a, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	return a
}

func TestRealArchives(t *testing.T) {
	for _, tc := range []struct{ arch, file, prefix string }{
		{"d2data.mpq", `data\global\excel\monstats.txt`, "Class\t"},
		{"d2exp.mpq", `data\global\excel\monstats.txt`, "Class\t"},
		{"d2data.mpq", `data\local\lng\eng\string.tbl`, ""},
		{"Patch_D2.mpq", `data\global\excel\skills.txt`, ""},
	} {
		a := openReal(t, tc.arch)
		d, err := a.ReadFile(tc.file)
		if err != nil {
			t.Logf("%s %s: %v", tc.arch, tc.file, err)
			continue
		}
		t.Logf("%s %s: %d bytes", tc.arch, tc.file, len(d))
		if tc.prefix != "" && !strings.HasPrefix(string(d), tc.prefix) {
			t.Errorf("%s: unexpected start %q", tc.file, d[:20])
		}
	}
}

func TestListfile(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	for _, n := range []string{"d2data.mpq", "d2exp.mpq", "Patch_D2.mpq"} {
		a := openReal(t, n)
		l := a.ListFile()
		t.Logf("%s: %d hash, %d block, listfile %d", n, len(a.hashes), len(a.blocks), len(l))
		fails := 0
		for _, f := range l {
			if !strings.HasSuffix(f, ".txt") && !strings.HasSuffix(f, ".tbl") {
				continue
			}
			if _, err := a.ReadFile(f); err != nil {
				fails++
				t.Errorf("%s: %v", f, err)
			}
		}
		_ = fails
	}
}

func TestMaskCensus(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	counts := map[byte]int{}
	maskHook = func(m byte) { counts[m]++ }
	defer func() { maskHook = nil }()
	for _, n := range []string{"d2data.mpq", "d2exp.mpq"} {
		a := openReal(t, n)
		bad := 0
		for _, f := range a.ListFile() {
			if _, err := a.ReadFile(f); err != nil {
				bad++
			}
		}
		t.Logf("%s: %d unreadable (audio etc)", n, bad)
	}
	t.Logf("mask census: %x", counts)
}
