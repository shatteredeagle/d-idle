package art

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestProbeDump(t *testing.T) {
	inst, dump := os.Getenv("D2ART_INSTALL"), os.Getenv("D2ART_DUMP")
	if inst == "" || dump == "" {
		t.Skip()
	}
	s, err := Open(inst, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"invchar.dc6", "invchar6.dc6", "inv_ring_amulet.dc6", "inv_helm_glove.dc6", "inv_belt.dc6", "inv_boots.dc6", "inv_armor.dc6", "inv_weapons.dc6"} {
		fr, err := s.frames(panelDir + n)
		if err != nil {
			t.Log(n, err)
			continue
		}
		for i, f := range fr {
			t.Logf("%s frame %d: %dx%d off %d,%d", n, i, f.W, f.H, f.OffX, f.OffY)
			o, _ := os.Create(filepath.Join(dump, fmt.Sprintf("%s_%d.png", n, i)))
			png.Encode(o, f.Image(s.pal))
			o.Close()
		}
	}
}
