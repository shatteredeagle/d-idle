package watch_test

import (
	"strings"
	"testing"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/watch"
)

// Every node with an Order must resolve through the resolver the bot uses for a go-to order (Plan.Resolve with the
// plan table and the goals catalog; the game-table catalog is nil here, so only names the offline tables know pass).
// A resolved id must be one of the levels the node stands for.
func TestEveryOrderNameResolvesLikeTheBot(t *testing.T) {
	plan, err := bot.LoadPlan("")
	if err != nil {
		t.Fatal(err)
	}
	fallback := func(name string) int {
		for _, a := range goals.Catalog().Areas {
			if a.ID > 0 && strings.EqualFold(a.Name, name) {
				return a.ID
			}
		}
		return 0
	}
	for _, n := range watch.Maps[1].Nodes {
		if n.Order == "" {
			continue
		}
		id, err := plan.Resolve(n.Order, 0, nil, fallback)
		if err != nil {
			// Names only the game tables know (levels.txt "Hole Level 1") cannot resolve offline; those need the real catalog.
			if strings.Contains(n.Order, "Level") {
				continue
			}
			t.Errorf("node %s (%s): order %q does not resolve: %v", n.ID, n.Name, n.Order, err)
			continue
		}
		in := false
		for _, a := range n.IDs {
			in = in || a == id
		}
		if !in {
			t.Errorf("node %s: order %q resolves to area %d, not one of %v", n.ID, n.Order, id, n.IDs)
		}
	}
}

func TestUndergroundPassIsOrderable(t *testing.T) {
	found := false
	for _, a := range watch.OrderableAreas() {
		found = found || a == "Underground Passage 1"
	}
	if !found {
		t.Fatal("Underground Passage 1 is not in OrderableAreas")
	}
}
