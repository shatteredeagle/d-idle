package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePNG(t *testing.T, p string, w, h, dots int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < dots; i++ {
		img.Set(i%w, i/w, color.RGBA{255, 255, 255, 255})
	}
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	png.Encode(f, img)
}

func TestDiffDirs(t *testing.T) {
	g, n := t.TempDir(), t.TempDir()
	writePNG(t, filepath.Join(g, "a.png"), 100, 100, 0)
	writePNG(t, filepath.Join(n, "a.png"), 100, 100, 30) // 0.3 percent changed
	writePNG(t, filepath.Join(n, "b.png"), 10, 10, 0)    // no golden: new
	if res, err := diffDirs(g, n, 0.005); err != nil || len(res) != 2 || !strings.HasPrefix(res[1], "NEW") {
		t.Fatalf("%v %v", res, err)
	}
	if _, err := diffDirs(g, n, 0.001); err == nil {
		t.Fatal("0.3 percent passed a 0.1 percent limit")
	}
	writePNG(t, filepath.Join(n, "a.png"), 90, 100, 0)
	if _, err := diffDirs(g, n, 0.5); err == nil {
		t.Fatal("a size change passed")
	}
}

func TestCheckDump(t *testing.T) {
	saved := pagePanels
	defer func() { pagePanels = saved }()
	pagePanels = map[string][]string{"watch-": {"hero-status", "why", "video", "map"}}
	good := `<div class="d-shell"><div data-panel="hero-status" data-state="ready"></div><div data-state="ready" data-panel="why"></div>` +
		`<section class="d-card d-panel" data-panel="video" data-size="m" data-state="ready"></section><section data-panel="map" data-state="ready"></section></div>`
	if bad := checkDump("watch-1280-dark.html", good); len(bad) != 0 {
		t.Fatal(bad)
	}
	bad := checkDump("watch-1280-dark.html", strings.Replace(strings.Replace(good, `"map" data-state="ready"`, `"map" data-state="loading"`, 1), `data-panel="video"`, `data-panel="video2"`, 1))
	if len(bad) != 2 {
		t.Fatalf("want a loading map and a missing video, got %v", bad)
	}
	if bad := checkDump("watch-x.html", strings.Replace(good, `"whyline"`, `"whyline" data-x`, 1)+`<p data-panel="gear" data-state="error"></p>`); len(bad) != 1 {
		t.Fatalf("error state: %v", bad)
	}
}

func TestFixtures(t *testing.T) {
	s := sensesFixture()
	m := s["map"].(map[string]any)
	n := 0
	rle := m["rle"].([]int)
	for i := 1; i < len(rle); i += 2 {
		n += rle[i]
	}
	if n != m["w"].(int)*m["h"].(int) {
		t.Fatalf("rle covers %d cells", n)
	}
	if b := frame(); len(b) < 1000 || b[0] != 0xff || b[1] != 0xd8 {
		t.Fatal("frame is not a JPEG")
	}
}

func TestCheckConsole(t *testing.T) {
	log := `[1:2:INFO:CONSOLE(12)] "watch layout: no panel for risk (ignored)", source: http://x/watch/shell.js (12)
[1:2:INFO:CONSOLE:7] "Uncaught (in promise) ReferenceError: x is not defined", source: http://x/watch/shell.js (7)
[1:2:INFO:CONSOLE(40)] "Uncaught TypeError: Cannot read properties of null (reading 'x')", source: http://x/watch/panels/gear/panel.js (40)
[1:2:INFO:CONSOLE(0)] "Failed to load resource: the server responded with a status of 404 (Not Found)", source: http://x/api/events (0)`
	bad := checkConsole("watch-1440-dark.html", log)
	if len(bad) != 2 || !strings.Contains(bad[0], "ReferenceError") || !strings.Contains(bad[1], "Uncaught TypeError") {
		t.Fatalf("got %v", bad)
	}
}
