package town

import "d2lod-bot/internal/input"

// pressRepairAll asks the open smith window to repair everything: the repair verb (the message the Repair All button
// sends, input layer I4, docs/design/input-i4.md) when input.planner is on and the connected build lists
// input.repair, else the click. A verb that errors falls back to the click for this try; the caller reads the worn
// durability back either way, so a dropped request is a failed repair, never a silent success.
func (t *Town) pressRepairAll() error {
	if t.e.Planner.On() && t.e.Planner.HasCap(input.CapRepair) {
		if r, ok := input.AsRepairer(t.e.In); ok {
			err := r.Repair(input.RepairAllItems)
			if err == nil {
				t.e.Planner.Count(input.PathVerb)
				return nil
			}
			t.e.Log.Info("repair: verb unavailable, clicking", "err", err)
		}
	}
	return t.e.In.Click(input.Left, RepairAll.X, RepairAll.Y)
}
