package art

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/data/mpq"
)

// synthDC6 builds a DC6 from rows of palette indexes, top row first, using made-up pixels only.
// Scanlines are written bottom-up with the real format's skip and literal runs.
func synthDC6(frames ...[][]byte) []byte {
	var out bytes.Buffer
	le := func(v uint32) { binary.Write(&out, binary.LittleEndian, v) }
	le(6)
	le(1)
	le(0)
	le(0xEEEEEEEE)
	le(1)
	le(uint32(len(frames)))
	var bodies [][]byte
	for _, rows := range frames {
		var d bytes.Buffer
		for y := len(rows) - 1; y >= 0; y-- {
			row := rows[y]
			for x := 0; x < len(row); {
				if row[x] == 0 {
					n := 0
					for x < len(row) && row[x] == 0 && n < 0x7f {
						x++
						n++
					}
					d.WriteByte(0x80 | byte(n))
					continue
				}
				st := x
				for x < len(row) && row[x] != 0 && x-st < 0x7f {
					x++
				}
				d.WriteByte(byte(x - st))
				d.Write(row[st:x])
			}
			d.WriteByte(0x80)
		}
		bodies = append(bodies, d.Bytes())
	}
	off := 24 + 4*len(frames)
	for _, b := range bodies {
		le(uint32(off))
		off += 32 + len(b) + 3
	}
	for i, b := range bodies {
		w, h := len(frames[i][0]), len(frames[i])
		for _, v := range []uint32{0, uint32(w), uint32(h), 0, 0, 0, 0, uint32(len(b))} {
			le(v)
		}
		out.Write(b)
		out.Write([]byte{0xEE, 0xEE, 0xEE})
	}
	return out.Bytes()
}

func synthPal() []byte {
	p := make([]byte, 768)
	for i := 1; i < 256; i++ { // BGR: index i = RGB (i, 2, 3)
		p[3*i], p[3*i+1], p[3*i+2] = 3, 2, byte(i)
	}
	return p
}

var icon = [][]byte{{0, 5, 5, 0}, {7, 7, 0, 9}}

func synthInstall(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	panelFrames := func(n int) []byte {
		var fr [][][]byte
		for i := 0; i < n; i++ {
			fr = append(fr, [][]byte{{byte(i + 1)}})
		}
		return synthDC6(fr...)
	}
	// Panel quadrants of the right sizes so the composition is exercised; all pixels made up.
	quad := func(w, h int, v byte) [][]byte {
		rows := make([][]byte, h)
		for y := range rows {
			rows[y] = bytes.Repeat([]byte{v}, w)
		}
		return rows
	}
	sheet := synthDC6(quad(1, 1, 9), quad(1, 1, 9), quad(1, 1, 9), quad(1, 1, 9),
		quad(256, 256, 11), quad(64, 256, 12), quad(256, 176, 13), quad(64, 176, 14))
	d2data := mpq.Build(map[string][]byte{
		`data\global\palette\ACT1\pal.dat`:        synthPal(),
		`data\global\items\invtst.dc6`:            synthDC6(icon),
		`data\global\ui\PANEL\invchar.dc6`:        sheet,
		`data\global\ui\PANEL\inv_helm_glove.dc6`: panelFrames(2),
	})
	exp := mpq.Build(map[string][]byte{`data\global\items\invexp.dc6`: synthDC6(icon)})
	must := func(name string, b []byte) {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	must("D2Data.mpq", d2data)
	must("d2exp.mpq", exp)
	return dir
}

func TestDecodeDC6(t *testing.T) {
	fr, err := DecodeDC6(synthDC6(icon))
	if err != nil || len(fr) != 1 {
		t.Fatalf("%v %d", err, len(fr))
	}
	want := []byte{0, 5, 5, 0, 7, 7, 0, 9}
	if fr[0].W != 4 || fr[0].H != 2 || !bytes.Equal(fr[0].Pix, want) {
		t.Fatalf("got %dx%d %v", fr[0].W, fr[0].H, fr[0].Pix)
	}
}

func TestDecodeDC6Garbage(t *testing.T) {
	good := synthDC6(icon)
	for _, b := range [][]byte{nil, good[:10], good[:40], append([]byte{7}, good[1:]...)} {
		if _, err := DecodeDC6(b); err == nil {
			t.Errorf("no error for %d bytes", len(b))
		}
	}
	// A literal run that overshoots the data must error, not panic.
	bad := append([]byte(nil), good...)
	bad[24+4*1+32] = 0x7f // first scanline byte of the frame
	DecodeDC6(bad)
}

func TestPalette(t *testing.T) {
	if _, err := ParsePalette(make([]byte, 10)); err == nil {
		t.Fatal("short palette accepted")
	}
	p, _ := ParsePalette(synthPal())
	if p[5] != [3]uint8{5, 2, 3} {
		t.Fatalf("BGR order wrong: %v", p[5])
	}
}

func TestStorePNG(t *testing.T) {
	inst := synthInstall(t)
	cache := t.TempDir()
	s, err := Open(inst, cache)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	b, err := s.PNG("item-invtst")
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if r, g, bl, a := img.At(1, 0).RGBA(); r>>8 != 5 || g>>8 != 2 || bl>>8 != 3 || a>>8 != 255 {
		t.Errorf("pixel (1,0) = %d %d %d %d", r>>8, g>>8, bl>>8, a>>8)
	}
	if _, _, _, a := img.At(0, 0).RGBA(); a != 0 {
		t.Error("index 0 must be transparent")
	}
	// Deterministic bytes (the golden of the synthetic icon).
	b2, _ := s.PNG("item-invtst")
	if !bytes.Equal(b, b2) {
		t.Error("second call differs")
	}
	// Item only in the expansion archive resolves too, with the file in the disk cache.
	if _, err := s.PNG("item-invexp"); err != nil {
		t.Fatal(err)
	}
	var cached int
	filepath.WalkDir(cache, func(p string, d os.DirEntry, _ error) error {
		if strings.HasSuffix(p, ".png") {
			cached++
		}
		return nil
	})
	if cached != 2 {
		t.Errorf("%d cached pngs, want 2", cached)
	}
	// A second store over the same cache serves from disk.
	s2, err := Open(inst, cache)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	if b3, err := s2.PNG("item-invtst"); err != nil || !bytes.Equal(b3, b) {
		t.Errorf("disk cache read: %v", err)
	}
}

func TestStoreKeys(t *testing.T) {
	s, err := Open(synthInstall(t), "")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, k := range []string{"", "item-", "item-../x", `item-..\x`, "item-INV", "item-aaaaaaaaaaaaaaaaa", "item-a/b", "slot-nope", "slot-", "panel-x", "../panel-inv", "item-a.png", "item-a%00"} {
		if ValidKey(k) {
			t.Errorf("ValidKey(%q)", k)
		}
		if _, err := s.PNG(k); !errors.Is(err, ErrBadKey) {
			t.Errorf("PNG(%q): %v", k, err)
		}
	}
	if _, err := s.PNG("item-nosuch"); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing item: %v", err)
	}
}

func TestPanelAndSlots(t *testing.T) {
	s, err := Open(synthInstall(t), "")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	b, err := s.PNG("panel-inv")
	if err != nil {
		t.Fatal(err)
	}
	img, _ := png.Decode(bytes.NewReader(b))
	if img.Bounds().Dx() != 320 || img.Bounds().Dy() != 432 {
		t.Fatalf("panel %v", img.Bounds())
	}
	// Quadrant placement: probe one pixel inside each (synthetic palette index i = red i).
	for _, tc := range []struct{ x, y, red int }{{10, 10, 11}, {300, 10, 12}, {10, 400, 13}, {300, 400, 14}} {
		if r, _, _, _ := img.At(tc.x, tc.y).RGBA(); int(r>>8) != tc.red {
			t.Errorf("pixel %d,%d red %d, want %d", tc.x, tc.y, r>>8, tc.red)
		}
	}
	// Frames 0 and 1 of the helm/glove file are the glove and the helm: helm = frame 1 (index 2).
	hb, err := s.PNG("slot-helm")
	if err != nil {
		t.Fatal(err)
	}
	himg, _ := png.Decode(bytes.NewReader(hb))
	if r, _, _, _ := himg.At(0, 0).RGBA(); r>>8 != 2 {
		t.Errorf("helm red %d, want 2", r>>8)
	}
	if _, err := s.PNG("slot-armor"); !errors.Is(err, ErrNotFound) {
		t.Errorf("absent slot art: %v", err)
	}
}

func TestOpenMissing(t *testing.T) {
	if _, err := Open(t.TempDir(), ""); err == nil {
		t.Fatal("empty install opened")
	}
	if _, err := Open(filepath.Join(t.TempDir(), "none"), ""); err == nil {
		t.Fatal("missing dir opened")
	}
}
