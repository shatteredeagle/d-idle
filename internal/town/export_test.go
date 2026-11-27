package town

// ExcursionForTest is the supply profile the stage 4 pickup reads ("" before any visit or for the field profile).
func (t *Town) ExcursionForTest() string {
	if p := t.excursion.Load(); p != nil {
		return *p
	}
	return ""
}
