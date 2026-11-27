package watch

// QuestSectionID is the Return screen section that says when the quest board is held by a park (card questresume).
const QuestSectionID = "quests"

// QuestSection shows the lines of a parked quest chunk: why the hero has been doing no quest, since when, and what clears
// it. lines is called when the page is built; no lines hides the section.
func QuestSection(lines func() []string) ReturnSection {
	return SectionFunc(func(*ReturnData) *SectionView {
		l := lines()
		if len(l) == 0 {
			return nil
		}
		return &SectionView{ID: QuestSectionID, Title: "Quests paused", Lines: l}
	})
}
