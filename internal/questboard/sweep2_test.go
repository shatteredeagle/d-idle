package questboard

import (
	"context"
	"testing"
)

// Review 2 finding 8 (card sweep2): the evidence is read after every hand-in attempt. Before: an attempt that paid the
// reward and then returned an error was retried against an NPC with nothing to say and answered "failed 3 times".
func TestSweep2HandInThatPaidAndErroredIsDone(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} {
		t.Run(class, func(t *testing.T) {
			r := newRig(t, class, 10)
			if err := r.x.Accept(den, clr); err != nil {
				t.Fatal(err)
			}
			r.tick(4)
			r.f.HandInLateErr, r.f.HandInErrs = 1, 9
			res, err := r.x.HandIn(context.Background(), den, clr)
			if err != nil || res.Outcome != HandInDone || res.Attempts != 1 {
				t.Fatalf("want done after one attempt, got %+v %v", res, err)
			}
		})
	}
}

// Review 10 B3 (card sweep2): a hand-in park after a restart keeps the stored death count. Before: HandIn made an empty
// run state and the park stored zeros over the counters.
func TestSweep2HandInParkKeepsStoredDeaths(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} {
		t.Run(class, func(t *testing.T) {
			r := newRigSM(t, class, 10, true)
			if err := r.x.Accept(den, clr); err != nil {
				t.Fatal(err)
			}
			r.x.Report(Decision{Quest: den, Chunk: clr}, StepResult{Outcome: StepDied, Why: "Corpsefire"})
			r.tick(4)
			r.restart()
			r.f.HandInSilent = 9
			if res, err := r.x.HandIn(context.Background(), den, clr); err != nil || res.Outcome != HandInBlocked {
				t.Fatalf("%+v %v", res, err)
			}
			run, ok := r.x.SMRuns()[ChunkKey(den, clr)]
			if !ok || run.Status != RunParked || run.Deaths != 1 {
				t.Fatalf("want the park with the one stored death, got %+v %v", run, ok)
			}
		})
	}
}
