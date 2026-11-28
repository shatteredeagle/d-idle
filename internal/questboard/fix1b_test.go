package questboard

import (
	"strings"
	"testing"
	"time"
)

// Card fix1b (review round 2, B2): a park the machine stored and the legacy board (quests.boardSM 0) shows is never
// lifted by the policy, so the page must not promise it, and the player's Reset must clear it. MUST FAIL when the note
// says "by itself" for such a park or Reset leaves it standing.
func TestFix1bLegacyParkNoteAndReset(t *testing.T) {
	for _, class := range smClasses {
		t.Run(class, func(t *testing.T) {
			r, ck := diedRig(t, class, true)
			run := r.x.runOf(t, den, clr)
			if n := r.x.ResumeNote(run); strings.Contains(n, "Only you") {
				t.Fatalf("test setup: with the machine the policy lifts this park: %q", n)
			}
			r.x.SetSM(false)
			if d := r.next(); d.Kind != DecBlocked {
				t.Fatalf("the legacy board must honour the stored park: %+v", d)
			}
			ck.add(48 * time.Hour)
			if _, ok := r.x.ResumeTick(fit()); ok {
				t.Fatal("test setup: the legacy board lifted a park by policy")
			}
			if n := r.x.ResumeNote(run); n != ResumeNoteLegacy || strings.Contains(n, "Resumes") {
				t.Fatalf("the note promises a resume the legacy board never does: %q", n)
			}
			r.x.Reset()
			if d := r.next(); d.Kind != DecRun {
				t.Fatalf("Reset on the legacy board must clear the stored park: %+v", d)
			}
		})
	}
}
