package questboard

import (
	"encoding/json"
	"fmt"

	questboarddata "d2lod-bot/data/questboard"
)

// The refusal park (card refusalpark, knob quests.refusalPark): a guard that refuses the same chunk with the same code
// again and again used to count as StepFailed and park as gave_up, which reads as "the step kept failing". With the knob
// on, parkAfter identical coded refusals in a row park the chunk as refused_repeatedly and keep the code and the text.

// RefusalPolicy is the parsed refusal.json.
type RefusalPolicy struct {
	ParkAfter int `json:"parkAfter"`
}

// ParseRefusal reads and validates refusal.json.
func ParseRefusal(raw []byte) (RefusalPolicy, error) {
	var f struct {
		Schema    string `json:"schema"`
		ParkAfter int    `json:"parkAfter"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		return RefusalPolicy{}, fmt.Errorf("questboard: refusal.json: %w", err)
	}
	if f.Schema != "quests.refusal/1" {
		return RefusalPolicy{}, fmt.Errorf("questboard: refusal.json: schema %q, want quests.refusal/1", f.Schema)
	}
	if f.ParkAfter < 2 || f.ParkAfter > 20 {
		return RefusalPolicy{}, fmt.Errorf("questboard: refusal.json: parkAfter %d, want 2 to 20", f.ParkAfter)
	}
	return RefusalPolicy{ParkAfter: f.ParkAfter}, nil
}

// LoadRefusal parses the embedded refusal.json.
func LoadRefusal() (RefusalPolicy, error) { return ParseRefusal(questboarddata.Refusal) }

// SetRefusal follows the knob at run time; n is the policy's parkAfter, 0 turns the rule off.
func (e *Exec) SetRefusal(n int) {
	e.mu.Lock()
	e.o.RefusalParkAfter = n
	e.mu.Unlock()
}

// refusalCount advances the identical-refusal streak of a chunk and reports whether it reached the limit. Caller holds mu.
func (e *Exec) refusalCount(rs *runState, res StepResult) bool {
	if e.o.RefusalParkAfter == 0 || res.Code == "" {
		rs.refCode, rs.refN = "", 0
		return false
	}
	if rs.refCode == res.Code {
		rs.refN++
	} else {
		rs.refCode, rs.refN = res.Code, 1
	}
	rs.refWhy = res.Why
	return rs.refN >= e.o.RefusalParkAfter
}
