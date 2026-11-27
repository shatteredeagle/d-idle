package watch

// AdviceSectionID is the Return screen section of the runeword and cube advisor (roadmap R14, internal/advisor).
const AdviceSectionID = "advice"

// AdviceSection shows the advice lines. lines is called when the page is built (the bot gathers the stock and the
// class there); no lines hides the section. The section only suggests: it carries no button and moves no item.
func AdviceSection(lines func() []string) ReturnSection {
	return SectionFunc(func(*ReturnData) *SectionView {
		l := lines()
		if len(l) == 0 {
			return nil
		}
		return &SectionView{ID: AdviceSectionID, Title: "Worth making", Lines: l, After: true}
	})
}
