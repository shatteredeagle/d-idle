package nav

import (
	"sync"
	"testing"
)

// Two goroutines (the acting one and the guard's Fighter.Safety -> PathTo) plan on one Planner: run with -race.
func TestPlannerConcurrentUse(t *testing.T) {
	g := randomMap(200, 200, 300, 7)
	f := buildField(g, g.Bounds(), 3, 0.75)
	sx, sy, _ := f.nearestWalkable(5003, 5003, 10)
	gx, gy, _ := f.nearestWalkable(5190, 5190, 10)
	var p Planner
	want, wp := p.Plan(f, sx, sy, gx, gy, 1.6, 1)
	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				got, part := p.Plan(f, sx, sy, gx, gy, 1.6, 1)
				if part != wp || len(got) != len(want) {
					t.Errorf("concurrent plan differs: %d/%v vs %d/%v", len(got), part, len(want), wp)
					return
				}
			}
		}()
	}
	wg.Wait()
}
