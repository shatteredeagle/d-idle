package art

import (
	"bytes"
	"image/png"
	"os"
	"testing"
)

// realInstall points at a real 1.14d install (read-only use); the test skips without it. D2ART_INSTALL, or
// /tmp/d2mpq like internal/data. Nothing it decodes is written anywhere but the temp cache.
func realInstall(t *testing.T) string {
	t.Helper()
	for _, p := range []string{os.Getenv("D2ART_INSTALL"), "/tmp/d2mpq"} {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Skip("no real install (set D2ART_INSTALL)")
	return ""
}

func TestRealArt(t *testing.T) {
	s, err := Open(realInstall(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	dims := func(key string) (int, int) {
		b, err := s.PNG(key)
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		return img.Bounds().Dx(), img.Bounds().Dy()
	}
	if w, h := dims("panel-inv"); w != 320 || h != 432 {
		t.Errorf("panel %dx%d", w, h)
	}
	for name := range slotFiles {
		dims("slot-" + name)
	}
	for _, k := range []string{"item-invgsw", "item-invrel", "item-invhp1", "item-invchm", "item-invamu", "item-invrin"} {
		w, h := dims(k)
		t.Logf("%s %dx%d", k, w, h)
	}
	if w, h := dims("item-invgsw"); w != 28 || h != 28 {
		t.Errorf("invgsw %dx%d, want 28x28", w, h)
	}
}
