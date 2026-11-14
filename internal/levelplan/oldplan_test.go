package levelplan

// oldPlanRows are the farm rows of progression-normal.json for a melee Paladin in file order (plan v2, as the AREAMON
// "plan rows" route of docs/research/areamon-trace.md section 5 used them), for the model calibration test.
func oldPlanRows() *ClassPlan {
	f := func(id string, area, entry, exit int) Slice {
		return Slice{ID: id, Kind: KindFarm, Areas: []int{area}, Entry: entry, Exit: exit, WayIn: "walk"}
	}
	return &ClassPlan{Class: "paladin", Build: "v2", Slices: []Slice{
		f("blood_moor", 2, 1, 4), f("den", 8, 2, 6), f("cold_plains", 3, 4, 8), f("burial", 17, 8, 10), f("cave1", 9, 6, 9),
		f("stony", 4, 7, 12), f("dark_wood", 5, 9, 13), f("black_marsh", 6, 10, 14), f("cave2", 13, 10, 15), f("crypt", 18, 10, 17),
	}}
}
