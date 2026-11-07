package overlay

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const base = "index\tcode\tlvl\r\nA\tcap\t1\r\nB\tcap\t2\r\n"

func frag(table, text string) Fragment {
	return Fragment{File: table + ".txt", Table: strings.ToLower(table), Text: text}
}

func TestMergeReplaceAppend(t *testing.T) {
	m, err := Merge(base, frag("UniqueItems", "index\tcode\tlvl\nA\t\t9\nC\tcap\t3\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := "index\tcode\tlvl\r\nA\tcap\t9\r\nB\tcap\t2\r\nC\tcap\t3\r\n"
	if m.Text != want || m.Replaced != 1 || m.Appended != 1 {
		t.Fatalf("got %q %+v", m.Text, m)
	}
	m, err = Merge(base, frag("UniqueItems", "lvl\tindex\n7\tB\n"))
	if err != nil || m.Text != "index\tcode\tlvl\r\nA\tcap\t1\r\nB\tcap\t7\r\n" {
		t.Fatalf("reordered subset: %q %v", m.Text, err)
	}
}

func TestMergeAppendOnlyTable(t *testing.T) {
	m, err := Merge("description\tnumin\r\nx\t1\r\n", frag("CubeMain", "description\tnumin\nx\t2\n"))
	if err != nil || m.Appended != 1 || m.Replaced != 0 {
		t.Fatalf("%+v %v", m, err)
	}
}

func TestMergeMalformed(t *testing.T) {
	for _, c := range []struct{ frag, line, why string }{
		{"index\tbogus\nA\t1\n", ":1:", "not in the base"},
		{"index\tcode\nA\n", ":2:", "1 cells"},
		{"index\tcode\nA\tcap\textra\n", ":2:", "3 cells"},
		{"code\nA\n", ":1:", "key column"},
		{"index\tcode\n\tcap\n", ":2:", "empty key"},
		{"index\tcode\nA\tx\nA\ty\n", ":3:", "duplicate"},
		{"", ":1:", "empty fragment"},
	} {
		_, err := Merge(base, frag("UniqueItems", c.frag))
		if err == nil || !strings.Contains(err.Error(), "overlay UniqueItems.txt"+c.line) || !strings.Contains(err.Error(), c.why) {
			t.Errorf("%q: %v", c.frag, err)
		}
	}
}

func TestModeAndHash(t *testing.T) {
	if m, err := Mode(""); err != nil || m != "vanilla" {
		t.Fatalf("%q %v", m, err)
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "excel"), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "excel", "UniqueItems.txt"), []byte("index\tcode\nx\tcap\n"), 0o644)
	m1, err := Mode(dir)
	if err != nil || !strings.HasPrefix(m1, "didle:") || len(m1) != len("didle:")+8 {
		t.Fatalf("%q %v", m1, err)
	}
	os.WriteFile(filepath.Join(dir, "excel", "UniqueItems.txt"), []byte("index\tcode\ny\tcap\n"), 0o644)
	if m2, _ := Mode(dir); m2 == m1 {
		t.Fatal("hash did not change with the fragment")
	}
	if _, err := Mode(t.TempDir()); err == nil {
		t.Fatal("a directory without excel/ must be an error")
	}
}

// Pins the hash algorithm against the C++ side (fork tests/test_overlay.cpp): FNV-1a 64 of "a.txt\0X\0".
func TestHashPinned(t *testing.T) {
	if got := Hash8([]Fragment{{File: "A.txt", Text: "X"}}); got != "7af2328b" {
		t.Fatalf("hash8 = %s", got)
	}
}
