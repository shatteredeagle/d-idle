package town_test

import (
	"context"
	"errors"
	"testing"

	"d2lod-bot/internal/town"
)

// The sixth Duriel drill stopped at "open Tyrael: town: not in a town": Tyrael stands in Duriel's Lair (73), so the
// opener allows the classes of town.fieldNPCAreas there and nowhere else; every other NPC still refuses outside a town.
func TestOpenNPCFieldAreas(t *testing.T) {
	r := newRig(t)
	w := r.W
	w.AddNPC(251, w.Player.X+2, w.Player.Y)

	w.Player.Area = 73
	if _, err := r.T.OpenForTest(context.Background(), 251); err != nil {
		t.Fatalf("Tyrael in the lair: %v", err)
	}
	if _, err := r.T.OpenForTest(context.Background(), 154); !errors.Is(err, town.ErrNotInTown) {
		t.Fatalf("Charsi in the lair: want ErrNotInTown, got %v", err)
	}
	w.Player.Area = 3 // Cold Plains: not the lair, the data row does not apply
	if _, err := r.T.OpenForTest(context.Background(), 251); !errors.Is(err, town.ErrNotInTown) {
		t.Fatalf("Tyrael in the field: want ErrNotInTown, got %v", err)
	}
}
