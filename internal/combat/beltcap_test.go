package combat

import (
	"testing"

	"d2lod-bot/internal/d2"
)

func TestBeltCapacityNeedsInventoryAndTables(t *testing.T) {
	if BeltCapacity(nil, nil) != 0 || BeltCapacity(&d2.Inventory{}, nil) != 0 {
		t.Fatal("unknown data must give 0 (the floor then stays unchanged)")
	}
}
