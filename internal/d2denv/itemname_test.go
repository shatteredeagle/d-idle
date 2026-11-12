package d2denv

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dsrc"
)

// ItemName is what makes town's affixNamer assertion match on d2d (docs/wiring.md).
func TestReaderItemNameFromTheDump(t *testing.T) {
	r := &Reader{last: &d2dsrc.Result{Snap: &d2.Snapshot{}, Extras: d2dsrc.Extras{Items: map[uint32]d2dsrc.ItemExtra{5: {Name: "Sturdy Cap of the Fox"}}}}}
	if got := r.ItemName(5); got != "Sturdy Cap of the Fox" {
		t.Errorf("name = %q", got)
	}
	if got := r.ItemName(6); got != "" {
		t.Errorf("unknown item must be empty, got %q", got)
	}
	if got := (&Reader{}).ItemName(5); got != "" {
		t.Errorf("no snapshot yet must be empty, got %q", got)
	}
	var _ interface{ ItemName(uint32) string } = r
}
