package quest

import "sync"

// ObjectiveAreas is every area a quest handler row works in: the Area of each step that is not plain travel (a kill,
// a clear, a chest, an object, an item placed). These are the boss and chest levels a hero must not enter unsupplied
// (review 14 H3). Derived from the handler rows, so a new quest row needs no second list. Tags follow the rows.
func ObjectiveAreas() map[int]bool { return objectiveAreas() }

var objectiveAreas = sync.OnceValue(func() map[int]bool {
	out := map[int]bool{}
	for _, act := range [][]Handler{Act1, Act2, Act3, Act4, Act5} {
		for _, h := range act {
			for _, s := range h.Steps {
				if s.Area != 0 && s.Kind != KGoArea && s.Kind != KWaitArea {
					out[s.Area] = true
				}
			}
		}
	}
	return out
})
