package data

import (
	"os"
	"path/filepath"
	"testing"

	"d2lod-bot/internal/overlay"
)

// A fake stackable currency row (our own, not Blizzard text) and a replaced cell of an existing base.
const fakeMisc = "code\tname\ttype\tnamestr\tinvwidth\tinvheight\tstackable\tmaxstack\nzzm1\tFake Mark\thpot\tzzm1\t1\t1\t1\t99\naqv\t\t\t\t\t\t\t500\n"

func writeOverlay(t *testing.T, text string) string {
	t.Helper()
	d := t.TempDir()
	if err := os.MkdirAll(filepath.Join(d, "excel"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "excel", "Misc.txt"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestOverlayAppliedAndStamped(t *testing.T) {
	inst := t.TempDir()
	writeInstall(t, inst, nil)

	vanillaCache := t.TempDir()
	t.Setenv(overlay.Env, "")
	vdb, err := Open(inst, vanillaCache)
	if err != nil {
		t.Fatal(err)
	}
	if vdb.ItemByCode("zzm1") != nil {
		t.Fatal("vanilla DB must not hold the overlay row")
	}
	vst, _ := InstallStamp(inst)
	if vst.OverlayHash8 != "" {
		t.Fatalf("vanilla stamp carries %q", vst.OverlayHash8)
	}

	dir := writeOverlay(t, fakeMisc)
	t.Setenv(overlay.Env, dir)
	if CacheCurrent(inst, vanillaCache) {
		t.Fatal("a vanilla cache must not count as current in d.idle mode")
	}
	cache := t.TempDir()
	db, err := Open(inst, cache)
	if err != nil {
		t.Fatal(err)
	}
	if it := db.ItemByCode("zzm1"); it == nil || !it.Stackable || it.MaxStack != 99 {
		t.Fatalf("appended row: %+v", it)
	}
	if it := db.ItemByCode("aqv"); it == nil || it.MaxStack != 500 || it.Width != 1 || it.Height != 3 {
		t.Fatalf("replaced cell only: %+v", it)
	}
	st, _ := InstallStamp(inst)
	if st.OverlayHash8 == "" {
		t.Fatal("d.idle stamp has no overlay hash")
	}
	// d.idle caches live in their own subdirectory and a changed fragment is a different cache.
	if _, err := os.Stat(filepath.Join(cache, "didle-"+st.OverlayHash8, "stamp.json")); err != nil {
		t.Fatal(err)
	}
	t.Setenv(overlay.Env, writeOverlay(t, "code\tname\nzzm2\tOther Fake\n"))
	if st2, _ := InstallStamp(inst); st2.OverlayHash8 == st.OverlayHash8 {
		t.Fatal("changed fragment, same stamp")
	}
	// Back to vanilla: the first cache is still current.
	t.Setenv(overlay.Env, "")
	if !CacheCurrent(inst, vanillaCache) {
		t.Fatal("vanilla cache lost its stamp")
	}
}

func TestOverlayMalformedIsFatal(t *testing.T) {
	inst := t.TempDir()
	writeInstall(t, inst, nil)
	t.Setenv(overlay.Env, writeOverlay(t, "code\tbogus\nzzm1\t1\n"))
	if _, err := LoadInstall(inst); err == nil {
		t.Fatal("a fragment with an unknown column must fail the load")
	}
}
