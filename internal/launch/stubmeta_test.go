package launch

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestStubMetaRoundTrip(t *testing.T) {
	dir := t.TempDir()
	a, b := []byte("hello "), []byte("world, this is installer data")
	os.MkdirAll(filepath.Join(dir, "D"), 0o755)
	os.WriteFile(filepath.Join(dir, "D", "a.bin"), a, 0o644)
	os.WriteFile(filepath.Join(dir, "b.bin"), b, 0o644)
	all := append(append([]byte{}, a...), b...)
	var pieces []byte
	for i := 0; i < len(all); i += 8 {
		e := min(i+8, len(all))
		s := sha1.Sum(all[i:e])
		pieces = append(pieces, s[:]...)
	}
	torrent := fmt.Sprintf("d8:announce3:foo15:direct download4:http4:infod5:filesld6:lengthi%de4:pathl1:D5:a.bineed6:lengthi%de4:pathl5:b.bineee4:name3:T-112:piece lengthi8e6:pieces%d:%see", len(a), len(b), len(pieces), pieces)
	stub := filepath.Join(dir, "stub.exe")
	os.WriteFile(stub, append([]byte("MZjunk"), []byte(torrent)...), 0o644)
	m, err := ReadStubMeta(stub)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "T-1" || m.DirectDownload != "http" || len(m.Files) != 2 || m.TotalSize() != int64(len(all)) {
		t.Fatalf("%+v", m)
	}
	if err := m.VerifyDir(dir, true); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "b.bin"), []byte("world, this is installer dXta"), 0o644)
	if err := m.VerifyDir(dir, true); err == nil {
		t.Fatal("corruption not detected")
	}
	if err := m.VerifyDir(dir, false); err != nil {
		t.Fatal("size-only check should pass")
	}
}
