package art

import (
	"fmt"
	"image"
	"image/draw"
)

// Panel sheet geometry. VERIFIED by rendering the real invchar.dc6 and invchar6.dc6 and looking at them:
// 8 frames, 0 to 3 are the character-stats sheet and 4 to 7 the inventory sheet; within a sheet the
// frames tile as a 2x2 grid, top-left 256x256, top-right 64x256, bottom-left 256x176, bottom-right 64x176,
// giving 320x432.
const (
	panelW, panelH = 320, 432
	invSheetFirst  = 4
)

var panelQuadrants = [4]image.Point{{0, 0}, {256, 0}, {0, 256}, {256, 256}}

// panelInv composes the inventory panel (equipment frames and backpack grid). invchar.dc6 is the plain
// sheet; invchar6.dc6 (LoD) adds the weapon-swap tabs, which would be dead decoration on the watch page,
// so it is only the fallback (U: which file the game itself draws on LoD).
func (s *Store) panelInv() (image.Image, error) {
	var lastErr error
	for _, name := range []string{"invchar.dc6", "invchar6.dc6"} {
		fr, err := s.frames(panelDir + name)
		if err != nil {
			lastErr = err
			continue
		}
		if len(fr) < invSheetFirst+4 {
			lastErr = fmt.Errorf("art: %s has %d frames, want 8", name, len(fr))
			continue
		}
		dst := image.NewNRGBA(image.Rect(0, 0, panelW, panelH))
		for i, at := range panelQuadrants {
			f := fr[invSheetFirst+i]
			draw.Draw(dst, image.Rectangle{at, at.Add(image.Pt(f.W, f.H))}, f.Image(s.pal), image.Point{}, draw.Src)
		}
		return dst, nil
	}
	return nil, lastErr
}
