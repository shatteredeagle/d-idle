package bot

import (
	"context"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/questboard"
)

// The live bug of 2026-11-17: the hero farmed Dark Wood under a farm_area goal, the player tapped Start, the accept was
// recorded and nothing ran because boardStep was only reached under a campaign goal. The accept now preempts a farm,
// level or boss goal (two classes; the knob at 0 changes nothing).
func TestQuestBoardStartRunsUnderAFarmGoal(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, goal := range []goals.Goal{farm("Blood Moor"), {ID: "g2", Type: goals.ReachLevel, Params: goals.Params{Level: 30}}} {
			t.Run(class.String()+"/"+string(goal.Type), func(t *testing.T) {
				g, x := boardRig(t, true)
				// The farm must still be running when the accept lands: with an instant explore the farm clears its
				// area and ends in the "no fresh game" error before a loaded test goroutine reaches Accept (flake of
				// 2026-11-18). Explore holds until the accept is recorded (bounded, so a bug fails instead of hanging).
				accepted := make(chan struct{})
				g.mover.exploreFn = func(ctx context.Context, done func() bool) error {
					select {
					case <-accepted:
					case <-ctx.Done():
					case <-time.After(30 * time.Second):
					}
					return nil
				}
				g.w.set(func(w *world) { w.class = class })
				g.run()
				defer g.finish()
				if err := g.r.Start(goal); err != nil {
					t.Fatal(err)
				}
				g.waitFor("the goal runs", func() bool { return g.w.snaps.Load() > 3 })
				if err := x.Accept("den_of_evil", "clear_den"); err != nil {
					t.Fatal(err)
				}
				close(accepted)
				// the log, not x.Next: the test goroutine must not read the board's world while the loop writes it (-race)
				g.waitFor("awaiting hand-in", func() bool {
					l := g.w.log()
					i := strings.Index(l, "exit 8")
					return i >= 0 && strings.Contains(l[i:], "talk Akara")
				})
				if !strings.Contains(g.w.log(), "exit 8") {
					t.Fatalf("the chunk never went to the Den: %s", g.w.log())
				}
				if strings.Contains(g.w.log(), "talk Akara/reward") {
					t.Fatalf("handed in by itself: %s", g.w.log())
				}
			})
		}
	}
}

func TestQuestBoardKnobOffIgnoresAcceptUnderFarm(t *testing.T) {
	g, _ := boardRig(t, false)
	if g.r.boardPreempts(&d2.Snapshot{}, control{spec: goalSpec{kind: goals.FarmArea}}) {
		t.Fatal("preempted with the knob at 0")
	}
}

// A fight in the clear radius delays the start of a chunk and the reason is shown; a campaign goal and a Runs goal are
// not touched; once calm the chunk starts.
func TestQuestBoardQueuedReasonWhileFighting(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		t.Run(class.String(), func(t *testing.T) {
			g, x := boardRig(t, true)
			g.w.set(func(w *world) { w.class = class; w.area = 2 })
			snap, _ := g.w.snapshot()
			g.r.obMu.Lock()
			g.r.ob.lastSnap = snap
			g.r.obMu.Unlock()
			if err := x.Accept("den_of_evil", "clear_den"); err != nil {
				t.Fatal(err)
			}
			farmCtl := control{spec: goalSpec{kind: goals.FarmArea}}
			if !g.r.boardPreempts(snap, farmCtl) {
				t.Fatal("calm field: the chunk should start")
			}
			if got := g.r.QuestQueued(); got != "" {
				t.Fatalf("queued reason with nothing in the way: %q", got)
			}
			if g.r.boardPreempts(snap, control{spec: goalSpec{kind: goals.Runs}}) {
				t.Fatal("a Runs goal was preempted")
			}
			g.w.addMonster(500, 1, 3, 3)
			snap, _ = g.w.snapshot()
			g.r.obMu.Lock()
			g.r.ob.lastSnap = snap
			g.r.obMu.Unlock()
			if g.r.hostileNear(snap, g.r.e.Tune.Bot.ClearRadius) == 0 {
				t.Skip("the fake monster is not hostile here")
			}
			if !g.r.boardPreempts(snap, farmCtl) {
				t.Fatal("a chunk under way was gated by a fight (cainfix3: only a fresh start waits)")
			}
			g.r.qbRT().startedKey = "" // a fresh start
			if g.r.boardPreempts(snap, farmCtl) {
				t.Fatal("started in the middle of a fight")
			}
			if q := g.r.QuestQueued(); !strings.Contains(q, "fight") {
				t.Fatalf("queued reason %q", q)
			}
		})
	}
}

// V lab drill CAIN-L3 (2026-11-19): a hand-in tap that lands while the hero is out in the field stays queued until the
// hero stands in town; it used to be taken off the queue in the field and answered "not in town", three times in a
// second, which parked the chunk.
func TestQuestBoardHandInWaitsForTown(t *testing.T) {
	g, _ := boardRig(t, true)
	rt := g.r.qbRT()
	req := boardHandInReq{ctx: context.Background(), done: make(chan error, 1)}
	rt.reqs <- req
	field := &d2.Snapshot{Player: &d2.Player{Area: 2}} // Blood Moor
	if g.r.boardServeHandIn(context.Background(), field, control{}) {
		t.Fatal("a hand-in was served in the field")
	}
	if len(rt.reqs) != 1 {
		t.Fatal("the hand-in tap was taken off the queue in the field")
	}
	select {
	case err := <-req.done:
		t.Fatalf("the tap was answered in the field: %v", err)
	default:
	}
}

// Proving Ground cain.scroll (2026-11-19): a hero in recovery (empty purse, no kit) defers an accepted chunk for the whole
// recovery and showed no reason. The deferral stays, the reason must be shown; without recovery the chunk starts.
func TestQuestBoardQueuedReasonWhileRecovering(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		t.Run(class.String(), func(t *testing.T) {
			g, x := boardRig(t, true)
			g.w.set(func(w *world) { w.class = class; w.area = 2 })
			snap, _ := g.w.snapshot()
			g.r.obMu.Lock()
			g.r.ob.lastSnap = snap
			g.r.obMu.Unlock()
			if err := x.Accept("den_of_evil", "clear_den"); err != nil {
				t.Fatal(err)
			}
			farmCtl := control{spec: goalSpec{kind: goals.FarmArea}}
			g.r.loop.broke.on = true
			if g.r.boardPreempts(snap, farmCtl) {
				t.Fatal("the chunk started during the recovery")
			}
			if q := g.r.QuestQueued(); !strings.Contains(q, "purse") {
				t.Fatalf("no reason shown while recovering: %q", q)
			}
			g.r.loop.broke.on = false
			if !g.r.boardPreempts(snap, farmCtl) {
				t.Fatal("the chunk did not start after the recovery")
			}
			if q := g.r.QuestQueued(); q != "" {
				t.Fatalf("stale reason %q", q)
			}
		})
	}
}

// cainfix3 F3: a chunk that buys nothing (Chunk.NoGold) starts during the purse recovery; one that may shop (den_of_evil)
// still waits. Two classes.
func TestQuestBoardNoGoldChunkSkipsRecovery(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		t.Run(class.String(), func(t *testing.T) {
			g, x := boardRig(t, true)
			g.w.set(func(w *world) { w.class = class; w.area = 2; w.level = 30 })
			snap, _ := g.w.snapshot()
			g.r.obMu.Lock()
			g.r.ob.lastSnap = snap
			g.r.obMu.Unlock()
			if err := x.Accept("search_for_cain", "scroll"); err != nil {
				t.Fatal(err)
			}
			g.r.loop.broke.on = true
			farmCtl := control{spec: goalSpec{kind: goals.ReachLevel}}
			if !g.r.boardPreempts(snap, farmCtl) {
				t.Fatalf("the scroll chunk waited for the recovery: %q", g.r.QuestQueued())
			}
			if !g.r.boardChunkNeedsNoGold() {
				t.Fatal("campaign path: the recovery still holds a chunk that buys nothing")
			}
		})
	}
}

// cainfix3 F1: the ban note names the deaths and the end time; a player-started chunk (banWaive) is not stopped by a
// death ban, a ban of another producer and the default behaviour stay.
func TestBannedAreaNoteAndWaiver(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		t.Run(class.String(), func(t *testing.T) {
			g, _ := boardRig(t, true)
			g.w.set(func(w *world) { w.class = class; w.area = 2 })
			g.r.obMu.Lock()
			g.r.skipAreaLocked(4, 30*time.Minute, whyDeath)
			g.r.skipAreaLocked(5, 30*time.Minute, whyUnreachable)
			g.r.obMu.Unlock()
			w, err := taskEnv{g.r}.World(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if !w.Banned[4] || !strings.Contains(w.BanNote[4], "until") || !strings.Contains(w.BanNote[4], "deaths") {
				t.Fatalf("death ban %v note %q", w.Banned[4], w.BanNote[4])
			}
			g.r.banWaive.Store(true)
			w, _ = taskEnv{g.r}.World(context.Background())
			if w.Banned[4] {
				t.Fatal("a started chunk is still stopped by the death ban")
			}
			if w.Banned[5] {
				t.Fatal("a started chunk is stopped by a nav-fault skip (cainchain: the skips of a Tristram stay read as bans)")
			}
			if !g.r.questWins(4, whyDeath) || !g.r.questWins(5, whyUnreachable) {
				t.Fatal("route bans: the waiver must lift death and nav-fault skips of a started chunk")
			}
			g.r.banWaive.Store(false)
			if g.r.questWins(4, whyDeath) {
				t.Fatal("no waiver outside a started chunk")
			}
		})
	}
}

// Proving Ground 2026-11-19: a level sweep keeps hostiles in the clear radius, so a chunk queued behind a fight never
// started again. Past boardQueueGrace it starts anyway (two classes).
func TestQuestBoardQueuedChunkStartsAfterGrace(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		t.Run(class.String(), func(t *testing.T) {
			g, x := boardRig(t, true)
			g.w.set(func(w *world) { w.class = class; w.area = 2 })
			snap, _ := g.w.snapshot()
			g.r.obMu.Lock()
			g.r.ob.lastSnap = snap
			g.r.obMu.Unlock()
			if err := x.Accept("den_of_evil", "clear_den"); err != nil {
				t.Fatal(err)
			}
			fc := newFClock()
			g.r.e.Clock = fc
			g.w.addMonster(500, 1, 3, 3)
			snap, _ = g.w.snapshot()
			g.r.obMu.Lock()
			g.r.ob.lastSnap = snap
			g.r.obMu.Unlock()
			if g.r.hostileNear(snap, g.r.e.Tune.Bot.ClearRadius) == 0 {
				t.Skip("the fake monster is not hostile here")
			}
			farmCtl := control{spec: goalSpec{kind: goals.ReachLevel}}
			if g.r.boardPreempts(snap, farmCtl) {
				t.Fatal("started at once in the middle of a fight")
			}
			fc.Sleep(boardQueueGrace + time.Second)
			if !g.r.boardPreempts(snap, farmCtl) {
				t.Fatal("still queued after the grace")
			}
		})
	}
}

// crowdFighter records SetQuestCrowd on top of the rig's fighter.
type crowdFighter struct {
	Fighter
	on bool
}

func (c *crowdFighter) SetQuestCrowd(on bool) { c.on = on }

// cainfix3 F2: the quest crowd line is raised by a chunk that runs and lowered again when the board runs nothing.
func TestQuestBoardCrowdLineFollowsTheChunk(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		t.Run(class.String(), func(t *testing.T) {
			g, x := boardRig(t, true)
			cf := &crowdFighter{Fighter: g.r.d.Fighter}
			g.r.d.Fighter = cf
			g.w.set(func(w *world) { w.class = class; w.area = 2 })
			snap, _ := g.w.snapshot()
			g.r.obMu.Lock()
			g.r.ob.lastSnap = snap
			g.r.obMu.Unlock()
			cf.on = true // left over from an earlier chunk
			if g.r.boardPreempts(snap, control{spec: goalSpec{kind: goals.FarmArea}}) {
				t.Fatal("started with nothing accepted")
			}
			if cf.on {
				t.Fatal("the quest crowd line stayed up with no chunk to run")
			}
			if err := x.Accept("den_of_evil", "clear_den"); err != nil {
				t.Fatal(err)
			}
			g.r.setQuestCrowd(true)
			if !g.r.boardPreempts(snap, control{spec: goalSpec{kind: goals.FarmArea}}) || !cf.on {
				t.Fatalf("a running chunk must keep the line up (on=%v)", cf.on)
			}
		})
	}
}

// V lab 2026-11-23 (necro step scenario stand_click_pickup): the chunk is queued behind the fight (the army fights on), the
// level goal's own area choice then walks the hero out of the chunk's area and the first step parks "ended in 26". While a
// fresh chunk waits for the fight the board says so (boardHoldsArea) and runGoal fights in place; once the chunk starts or
// the field is calm the hold is gone. Two classes. Must fail while nothing holds the area.
func TestQuestBoardHoldsTheAreaWhileQueuedBehindAFight(t *testing.T) {
	for _, class := range []d2.Class{d2.Necromancer, d2.Paladin} {
		t.Run(class.String(), func(t *testing.T) {
			g, x := boardRig(t, true)
			g.w.set(func(w *world) { w.class = class; w.area = 2 })
			snap, _ := g.w.snapshot()
			g.r.obMu.Lock()
			g.r.ob.lastSnap = snap
			g.r.obMu.Unlock()
			if err := x.Accept("den_of_evil", "clear_den"); err != nil {
				t.Fatal(err)
			}
			farmCtl := control{spec: goalSpec{kind: goals.FarmArea}}
			g.w.addMonster(500, 1, 3, 3)
			snap, _ = g.w.snapshot()
			g.r.obMu.Lock()
			g.r.ob.lastSnap = snap
			g.r.obMu.Unlock()
			if g.r.hostileNear(snap, g.r.e.Tune.Bot.ClearRadius) == 0 {
				t.Skip("the fake monster is not hostile here")
			}
			if g.r.boardPreempts(snap, farmCtl) || !g.r.boardHoldsArea() {
				t.Fatal("a fresh chunk queued behind a fight must hold the area")
			}
			g.r.qbRT().queuedSince = g.r.e.Clock.Now().Add(-2 * boardQueueGrace)
			if !g.r.boardPreempts(snap, farmCtl) || g.r.boardHoldsArea() {
				t.Fatal("after the grace the chunk starts and the hold is gone")
			}
		})
	}
}

// TASK akarafix: a tap that timed out must not block the next one (the stale request used to stay in the one-slot mailbox
// and refuse every later tap with "already waiting", parking the chunk).
func TestQuestBoardStaleHandInDoesNotBlockTheNextTap(t *testing.T) {
	g, x := boardRig(t, true)
	rt := g.r.qbRT()
	q, c := boardChunk(t, x, "den_of_evil", "clear_den")
	a := boardActions{rt}
	ctx1, cancel1 := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel1()
	if err := a.HandIn(ctx1, q, c); err == nil {
		t.Fatal("an unserved tap should end with its ctx")
	}
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	res := make(chan error, 1)
	go func() { res <- a.HandIn(ctx2, q, c) }()
	var req boardHandInReq
	select {
	case req = <-rt.reqs:
	case err := <-res:
		t.Fatalf("second tap refused: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("second tap never queued")
	}
	if req.ctx != ctx2 {
		t.Fatal("the mailbox held the stale request")
	}
	req.done <- nil
	if err := <-res; err != nil {
		t.Fatal(err)
	}
}

// A request whose tapper is gone is dropped at serve time, not talked out.
func TestQuestBoardServeDropsCancelledHandIn(t *testing.T) {
	g, x := boardRig(t, true)
	rt := g.r.qbRT()
	q, c := boardChunk(t, x, "den_of_evil", "clear_den")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rt.reqs <- boardHandInReq{ctx: ctx, q: q, c: c, done: make(chan error, 1)}
	town := &d2.Snapshot{Player: &d2.Player{Area: 1}}
	if g.r.boardServeHandIn(context.Background(), town, control{}) {
		t.Fatal("served a cancelled request")
	}
	if len(rt.reqs) != 0 || strings.Contains(g.w.log(), "talk Akara") {
		t.Fatalf("stale request survived or talked: %d %s", len(rt.reqs), g.w.log())
	}
}

// The "why waiting" reason per holder (two classes: the reason is class-free).
func TestQuestBoardHandInWhyPerHolder(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		t.Run(class.String(), func(t *testing.T) {
			g, x := boardRig(t, true)
			rt := g.r.qbRT()
			q, c := boardChunk(t, x, "den_of_evil", "clear_den")
			if g.r.QuestHandInWhy() != "" {
				t.Fatal("a reason without a queued tap")
			}
			rt.reqs <- boardHandInReq{ctx: context.Background(), q: q, c: c, done: make(chan error, 1)}
			field := &d2.Snapshot{Player: &d2.Player{Area: 2}}
			town := &d2.Snapshot{Player: &d2.Player{Area: 1}}
			g.r.noteHandInHold(field, control{})
			if w := g.r.QuestHandInWhy(); !strings.Contains(w, "field") {
				t.Fatalf("field: %q", w)
			}
			g.r.noteHandInHold(town, control{})
			if w := g.r.QuestHandInWhy(); !strings.Contains(w, "another step") {
				t.Fatalf("town: %q", w)
			}
			g.r.loop.broke.on = true
			g.r.noteHandInHold(town, control{})
			if w := g.r.QuestHandInWhy(); !strings.Contains(w, "purse") {
				t.Fatalf("recovery: %q", w)
			}
		})
	}
}

func boardChunk(t *testing.T, _ *questboard.Exec, quest, chunk string) (questboard.Quest, questboard.Chunk) {
	t.Helper()
	b, err := questboard.Load()
	if err != nil {
		t.Fatal(err)
	}
	q, err := b.Quest(quest)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range q.Chunks {
		if c.ID == chunk {
			return q, c
		}
	}
	t.Fatalf("no chunk %s/%s", quest, chunk)
	return q, questboard.Chunk{}
}
