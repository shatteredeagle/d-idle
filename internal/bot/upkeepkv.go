package bot

// upkeepComparer is the optional Fighter part that counts the upkeep drivers' shadow comparisons
// (combat.Fighter.UpkeepCompareField, knob combat.deliveryUpkeep).
type upkeepComparer interface {
	UpkeepCompareField() string
}

// upkeepKV is the minute line field "upkeep" = agree/cmp per driver kind; nothing while the knob is 0, so the default
// line is unchanged.
func (r *Runner) upkeepKV() []any {
	u, ok := r.d.Fighter.(upkeepComparer)
	if !ok {
		return nil
	}
	if v := u.UpkeepCompareField(); v != "" {
		return []any{"upkeep", v}
	}
	return nil
}
