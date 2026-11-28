package questboard

import (
	"strings"
	"testing"
)

func readyChunk(v View, quest string) (ChunkRow, bool) {
	for _, q := range v.Quests {
		if q.ID != quest {
			continue
		}
		for _, c := range q.Chunks {
			if c.State == Ready {
				return c, true
			}
		}
	}
	return ChunkRow{}, false
}

// A ready chunk with auto off waits for the Start tap; opted in and safe, AutoTick takes it and nothing waits.
func TestDeskWaitingForStart(t *testing.T) {
	for _, bd := range builds {
		t.Run(bd.class, func(t *testing.T) {
			a := newAutoRig(t, bd.class, 6, opts(20))
			d := a.deskAuto(nil)
			c, ok := readyChunk(d.viewFresh(), den)
			if !ok || c.Auto != AutoOff || c.WaitingFor != WaitStartTap {
				t.Fatalf("auto off: ok=%v chunk %+v", ok, c)
			}
			if l := d.WaitingLines(); len(l) != 1 || !strings.HasPrefix(l[0], "Waiting for you to start: ") {
				t.Fatalf("lines %q", l)
			}
			d.SetAuto(den, true)
			c, _ = readyChunk(d.viewFresh(), den)
			if c.Auto != AutoOn || c.WaitingFor != "" {
				t.Fatalf("auto on: %+v", c)
			}
		})
	}
}

func TestWaitingForNever(t *testing.T) {
	if got := waitingFor(false, false, ReadySafe); got != WaitNeverAuto {
		t.Fatal(got)
	}
	if got := autoWord(true, false); got != AutoNever {
		t.Fatal(got)
	}
	if got := waitingFor(true, true, ReadyTooHard); got != WaitStartTap {
		t.Fatal(got)
	}
}
