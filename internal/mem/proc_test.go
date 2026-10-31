package mem

import "testing"

func TestIsWineExe(t *testing.T) {
	for exe, want := range map[string]bool{
		"/x/wine-preloader": true, "/usr/lib/wine/wine": true, "/usr/lib/wine/wine64": true,
		"/usr/lib/wine/wineserver32": false, "/usr/bin/Xvfb": false, "": false,
	} {
		if got := IsWineExe(exe); got != want {
			t.Errorf("IsWineExe(%q)=%v", exe, got)
		}
	}
}
