package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"
)

// diffDirs compares every PNG of got with the same name in golden. A pixel counts as changed when a channel moved by
// more than 16 of 255 (antialiasing noise stays under that). A missing golden is reported as new, not as a failure.
func diffDirs(golden, got string, max float64) ([]string, error) {
	names, err := filepath.Glob(filepath.Join(got, "*.png"))
	if err != nil || len(names) == 0 {
		return nil, fmt.Errorf("no PNGs in %q", got)
	}
	sort.Strings(names)
	var out []string
	failed := 0
	for _, n := range names {
		base := filepath.Base(n)
		g := filepath.Join(golden, base)
		if _, err := os.Stat(g); err != nil {
			out = append(out, fmt.Sprintf("NEW   %s", base))
			continue
		}
		r, err := changed(g, n)
		switch {
		case err != nil:
			out = append(out, fmt.Sprintf("FAIL  %s: %v", base, err))
			failed++
		case r > max:
			out = append(out, fmt.Sprintf("FAIL  %s: %.3f%% of pixels changed (max %.3f%%)", base, 100*r, 100*max))
			failed++
		default:
			out = append(out, fmt.Sprintf("ok    %s: %.3f%%", base, 100*r))
		}
	}
	if failed > 0 {
		return out, fmt.Errorf("%d image(s) changed", failed)
	}
	return out, nil
}

func load(p string) (image.Image, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

// changed returns the share of pixels that differ between two PNGs of the same size.
func changed(a, b string) (float64, error) {
	ia, err := load(a)
	if err != nil {
		return 0, err
	}
	ib, err := load(b)
	if err != nil {
		return 0, err
	}
	ra, rb := ia.Bounds(), ib.Bounds()
	if ra.Dx() != rb.Dx() || ra.Dy() != rb.Dy() {
		return 1, fmt.Errorf("size %dx%d, golden %dx%d", rb.Dx(), rb.Dy(), ra.Dx(), ra.Dy())
	}
	n := 0
	for y := 0; y < ra.Dy(); y++ {
		for x := 0; x < ra.Dx(); x++ {
			r1, g1, b1, _ := ia.At(ra.Min.X+x, ra.Min.Y+y).RGBA()
			r2, g2, b2, _ := ib.At(rb.Min.X+x, rb.Min.Y+y).RGBA()
			if far(r1, r2) || far(g1, g2) || far(b1, b2) {
				n++
			}
		}
	}
	return float64(n) / float64(ra.Dx()*ra.Dy()), nil
}

func far(a, b uint32) bool {
	d := int(a>>8) - int(b>>8)
	return d > 16 || d < -16
}
