package bot

import (
	"context"
	"d2lod-bot/internal/buildinfo"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/questboard"
	"d2lod-bot/internal/task"
)

// The quest board executor hand-off (slice S3, docs/questboard.md "Executor"). With knob quests.board = 1 the planner no
// longer starts quests: an accepted plan record runs the existing chunk handler (internal/task act*.go) up to, but not
// including, its hand-in step; the hero stops in town next to the giver (Awaiting) and the player's HandIn tap runs the
// hand-in step. Between chunks the hero acts on the player's mode (boardmode.go, slice S6). Knob 0: none of this runs.

// questBoardOn is the knob quests.board.
func (r *Runner) questBoardOn() bool { return r.e.Tune.Quests.Board != 0 }

// boardFailGap spaces the failures one chunk counts (a refused step repeats every tick; three in a row park the chunk,
// so without a gap a gear gate would park it in three seconds). U: judgment.
const boardFailGap = 15 * time.Second

type boardHandInReq struct {
	ctx  context.Context // the tapper's bound; a request whose ctx ended is dropped, never served
	q    questboard.Quest
	c    questboard.Chunk
	done chan error
}

// boardRT is the bot side of the board: the evidence cache, the hand-in mailbox and per-process notes. None of it
// survives a restart; the chunk is re-derived from the evidence.
type boardRT struct {
	r     *Runner
	board *questboard.Board
	exec  *questboard.Exec
	st    *questboard.Store
	hero  string // the hero the store belongs to ("" when built by NewQuestBoard)
	reqs  chan boardHandInReq
	// handWhy is what holds a queued hand-in tap, in words ("" when none is queued or the loop serves it); guarded by mu.
	handWhy string

	// yieldKey is the chunk a goal travel already yielded to once (boardYield).
	yieldKey string

	mu       sync.Mutex
	snap     *d2.Snapshot // the snapshot the cached world was built from
	w        task.World
	class    string
	ready    map[string]questboard.Word
	explain  map[string]questboard.Readiness // the full answers behind ready, when the margin query ran
	assessed time.Time                       // when ready was last computed (the margin query is throttled)
	assessOn string                          // the inputs it was computed for (level, difficulty)
	th       *questboard.Thresholds
	handed   map[string]bool      // hand-in dialogue ran this process (T3 proxy until the save or memory agrees)
	reached  map[string]bool      // an auto or item chunk read done once this process (its evidence can vanish: key used, Tristram left)
	near     map[string]bool      // walked to the giver once this process
	lastFail map[string]time.Time // chunk key: last failure that was counted
	noted    map[string]string    // visible-reason log lines already written
	unpark   map[string]bool      // task groups the executor must lift the block of (Retry, knob quests.boardSM): the loop goroutine applies it
	applied  string               // the between-chunks mode the hero acts on (boardmode.go); "" before the first tick
	farming  bool                 // inside a farm-mode slice: the area choice farms
	queued   string               // why an accepted chunk is not running yet ("" when it runs or none is accepted)
	// queuedSince is when the chunk first met a fight in the clear radius (zero when it is not waiting): past
	// boardQueueGrace the chunk starts anyway, see boardPreemptsWhy.
	holdFight   bool // the chunk is queued behind a fight in the clear radius: the level goal stays and fights here
	queuedSince time.Time
	// startedKey is the chunk the board has already started this run of decisions: a fight gates only a fresh start, never
	// the next step of a chunk under way (cainfix3: after "Go to Stony Field" the level plan took the hero away again).
	startedKey string
}

// NewQuestBoard builds the board executor of this runner and its hero's plan store and keeps it for boardStep. The S4
// API calls the returned Exec (Offers, Accept, HandIn, State). Safe to call with the knob at 0: the Exec then answers
// ErrOff and boardStep never runs.
func (r *Runner) NewQuestBoard(b *questboard.Board, st *questboard.Store) *questboard.Exec {
	return r.newQuestBoard(b, st, "")
}

func (r *Runner) newQuestBoard(b *questboard.Board, st *questboard.Store, hero string) *questboard.Exec {
	rt := &boardRT{r: r, board: b, st: st, hero: hero, reqs: make(chan boardHandInReq, 1), handed: map[string]bool{}, reached: map[string]bool{}, near: map[string]bool{},
		lastFail: map[string]time.Time{}, noted: map[string]string{}, unpark: map[string]bool{}}
	rt.exec = questboard.NewExec(b, st, boardEnv{rt}, boardActions{rt}, questboard.Options{
		Enabled: r.questBoardOn(),
		SM:      r.e.Tune.Quests.BoardSM != 0,
		Release: botReleaseID(),
		Log:     func(msg string, kv ...any) { r.e.Log.Info(msg, kv...) },
		Now:     r.e.Clock.Now,
		Auto:    r.questAutoOptions(),
		Ledger: func() questboard.Ledger {
			r.qbMu.Lock()
			f := r.qbLedger
			r.qbMu.Unlock()
			if f == nil {
				return nil
			}
			return f()
		},
	})
	if m, err := st.Mode(); err == nil {
		rt.exec.SetMode(m)
	}
	r.qbMu.Lock()
	r.qb = rt
	r.qbMu.Unlock()
	return rt.exec
}

// QuestBoardMaker opens the board and the plan store of one hero (cmd/d2bot, per-hero file under the state directory).
type QuestBoardMaker func(hero string) (*questboard.Board, *questboard.Store, error)

// SetQuestLedger gives the board the account wallet (slice S10); the function may return nil while there is none.
func (r *Runner) SetQuestLedger(f func() questboard.Ledger) {
	r.qbMu.Lock()
	r.qbLedger = f
	r.qbMu.Unlock()
}

// SetQuestBoardMaker lets the runner build the board of the loaded hero by itself: the hero's name is only known from the
// first snapshot, and a Character Select switch loads another hero (its plan file is <hero>.quests.json).
func (r *Runner) SetQuestBoardMaker(f QuestBoardMaker) {
	r.qbMu.Lock()
	r.qbMake = f
	r.qbMu.Unlock()
}

func (r *Runner) qbRT() *boardRT { r.qbMu.Lock(); defer r.qbMu.Unlock(); return r.qb }

// boardRTFor returns the board of the loaded hero, building (or, after a hero switch, rebuilding) it through the maker.
// It follows the knob on every call. nil while there is neither a maker nor a board.
func (r *Runner) boardRTFor() *boardRT {
	r.qbMu.Lock()
	rt, mk := r.qb, r.qbMake
	r.qbMu.Unlock()
	if mk != nil {
		name := ""
		if s := r.latest(); s != nil && s.Player != nil {
			name = s.Player.Name
		}
		if name != "" && (rt == nil || rt.hero != name) {
			b, st, err := mk(name)
			if err != nil {
				r.boardNote("make", "quest board: no plan store for "+name+": "+err.Error())
			} else {
				r.newQuestBoard(b, st, name)
				rt = r.qbRT()
			}
		}
	}
	if rt != nil {
		rt.exec.SetEnabled(r.questBoardOn())
		rt.exec.SetAutoOptions(r.questAutoOptions())
	}
	return rt
}

// SetQuestMargin installs the margin query before the loop starts (cmd/d2bot builds the survival model after bot.New).
func (r *Runner) SetQuestMargin(f func(s *d2.Snapshot, boss, diff string, potions int) (questboard.Hero, *questboard.Encounter, bool)) {
	r.qbMu.Lock()
	r.d.QuestMargin = f
	r.qbMu.Unlock()
}

// QuestBoard returns the executor and plan store of the loaded hero for the API desk; nil before a hero is loaded.
func (r *Runner) QuestBoard() (*questboard.Exec, *questboard.Store) {
	rt := r.boardRTFor()
	if rt == nil {
		return nil, nil
	}
	return rt.exec, rt.st
}

// QuestReadiness is the full readiness of a chunk (reasons, legacy floor, sentence) for the API; ok is false when no margin
// query ran for it.
func (r *Runner) QuestReadiness(q questboard.Quest, c questboard.Chunk) (questboard.Readiness, bool) {
	rt := r.qbRT()
	if rt == nil {
		return questboard.Readiness{}, false
	}
	rt.world()
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rd, ok := rt.explain[questboard.ChunkKey(q.ID, c.ID)]
	return rd, ok
}

// QuestClass is the loaded hero's class in lower case ("" before a snapshot).
func (r *Runner) QuestClass() string {
	if rt := r.qbRT(); rt != nil {
		_, class, _ := rt.world()
		return class
	}
	return ""
}

// ---- Env: evidence ------------------------------------------------------------------------------------

func taskID(q questboard.Quest) string {
	if q.TaskID != "" {
		return q.TaskID
	}
	return q.ID
}

// world returns the task world of the latest snapshot, built once per snapshot (the executor asks for every chunk).
//
// taskEnv.World reads and mutates loop-owned maps (cleared areas, touched waypoints), so an API goroutine (Accept, readiness)
// may build it only between loop steps (stepMu free); during a step it gets the loop's last world, at most one tick old (the
// loop refreshes it through worldLoop). Building it mid-step raced with the loop (-race, TestQuestBoardStartRunsUnderAFarmGoal).
func (rt *boardRT) world() (task.World, string, map[string]questboard.Word) {
	if rt.r.stepMu.TryLock() {
		defer rt.r.stepMu.Unlock()
		return rt.worldLoop()
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.w, rt.class, rt.ready
}

// worldLoop is world() for the loop goroutine: it rebuilds the cache when the snapshot changed.
func (rt *boardRT) worldLoop() (task.World, string, map[string]questboard.Word) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	s := rt.r.latest()
	if s == nil || s.Player == nil {
		return task.World{}, "", nil
	}
	if s != rt.snap {
		w, err := (taskEnv{rt.r}).World(context.Background())
		if err != nil {
			return task.World{}, "", nil
		}
		rt.snap, rt.w = s, w
		rt.class = strings.ToLower(s.Player.Class.String())
		rt.assess(s)
	}
	return rt.w, rt.class, rt.ready
}

// assessEvery spaces the margin queries: the readiness words change with the level, the kit and the merc, not per snapshot.
// U: judgment.
const assessEvery = 30 * time.Second

// assess fills ready and explain for the snapshot. Callers hold rt.mu. With the margin query (Deps.QuestMargin, the survival
// model, read only) each chunk is judged by questboard.Assess against its boss; without it, or where Assess has no data (the
// word is Unknown), the transitional legacy word stands, as before S4 (a quest without a floor is safe).
func (rt *boardRT) assess(s *d2.Snapshot) {
	diff := difficultyName(s.Player.Difficulty)
	on := fmt.Sprintf("%d/%s/%d", s.Player.Level, diff, int(s.Player.Difficulty))
	now := rt.r.e.Clock.Now()
	if rt.ready != nil && on == rt.assessOn && now.Sub(rt.assessed) < assessEvery {
		return
	}
	legacy, err := rt.board.LegacyReadiness(rt.class, s.Player.Level)
	if err != nil {
		rt.r.e.Log.Warn("quest board: readiness unknown", "err", err)
		rt.ready, rt.explain = nil, nil
		return
	}
	rt.ready, rt.explain, rt.assessOn, rt.assessed = legacy, nil, on, now
	hook := rt.r.d.QuestMargin
	if hook == nil {
		return
	}
	if rt.th == nil {
		th, err := questboard.LoadThresholds()
		if err != nil {
			rt.r.e.Log.Warn("quest board: thresholds not loaded, legacy readiness stays", "err", err)
			return
		}
		rt.th = th
	}
	potions := -1
	if healing, _, known := rt.r.beltAndInv(s); known {
		potions = healing
	}
	hero, _, ok := hook(s, "", diff, potions)
	if !ok {
		return
	}
	if potions < 0 {
		hero.Potions = 1 << 20 // the belt was not read: never claim a shortage
	}
	bosses := map[string]*questboard.Encounter{}
	rt.explain = map[string]questboard.Readiness{}
	for _, q := range rt.board.Quests() {
		floor, hasFloor, _ := rt.board.Floor(q, rt.class)
		for _, c := range q.Chunks {
			boss := c.Boss
			if boss == "" {
				boss = q.Boss
			}
			in := questboard.AssessInput{Dial: rt.r.zoneDial(), Diff: int(s.Player.Difficulty), Hero: hero, Floor: floor, HasFloor: hasFloor, LegacyWall: rt.r.e.Tune.Quests.LegacyWall}
			if boss != "" {
				enc, seen := bosses[boss]
				if !seen {
					_, enc, _ = hook(s, boss, diff, potions)
					bosses[boss] = enc
				}
				if enc != nil {
					in.Encounters = []questboard.Encounter{*enc}
				}
			}
			rd, err := rt.th.Assess(in)
			if err != nil {
				continue
			}
			k := questboard.ChunkKey(q.ID, c.ID)
			rt.explain[k] = rd
			if !rd.Unknown {
				rt.ready[k] = rd.Word
			}
		}
	}
}

type boardEnv struct{ rt *boardRT }

func (e boardEnv) Diff() int {
	if s := e.rt.r.latest(); s != nil && s.Player != nil {
		return int(s.Player.Difficulty)
	}
	return 0
}

func (e boardEnv) Facts() questboard.Facts { return boardFacts{e.rt} }

type boardFacts struct{ rt *boardRT }

func evTier(ev task.Evidence) questboard.Tier {
	switch ev {
	case task.EvProxy:
		return questboard.TierProxy
	case task.EvSave:
		return questboard.TierSave
	case task.EvMemory:
		return questboard.TierMemory
	}
	return questboard.TierNone
}

// hopFact is the quest fact of an act hop (kind unlock): the hop has no quest word of its own for every act (Act 3 and 4
// have none the reader maps), and a fact that is only a proxy never unlocks the next act's quests (lockedBy wants exact
// evidence), so every Act 4 and 5 row stayed locked (V lab drill.act5.siege: "locked until Take Tyrael's portal to
// Act 5"). The world's act is a live read: done when the hero stands in the next act.
func hopFact(q questboard.Quest, w task.World) (questboard.Fact, bool) {
	if q.Kind != "unlock" {
		return questboard.Fact{}, false
	}
	if w.Act == 0 {
		return questboard.Fact{}, true
	}
	return questboard.Fact{Known: true, Tier: questboard.TierMemory, Done: w.Act >= q.Act+1}, true
}

func (f boardFacts) Quest(diff int, id string) questboard.Fact {
	if diff != (boardEnv{f.rt}).Diff() {
		return questboard.Fact{}
	}
	q, err := f.rt.board.Quest(id)
	if err != nil {
		return questboard.Fact{}
	}
	w, _, _ := f.rt.world()
	if hf, ok := hopFact(q, w); ok {
		return hf
	}
	st := w.Quest(taskID(q))
	t := evTier(st.Ev)
	return questboard.Fact{Known: st.Ev != task.EvNone || st.Stage != task.StageNone, Tier: t,
		Done: st.Stage >= task.StageDone, Started: st.Stage >= task.StageStarted,
		Pending: st.Stage == task.StageCleared && t.Exact(), Cleared: st.Stage >= task.StageCleared}
}

// legacyChunk is the single-chunk quest that runs its own task group as a whole (slice S3): its done and objective
// facts come from the quest word. Every other chunk (a part of a multi-chunk quest, or a chunk with its own group or
// node list) reads its facts from the handler steps (slice S7).
func legacyChunk(q questboard.Quest, c questboard.Chunk) bool {
	return len(q.Chunks) == 1 && c.Handler == "" && len(c.Nodes) == 0 && c.HandInStep == ""
}

// stepSuffix is the step id without "quest.<group>/".
func stepSuffix(group string, s task.Step) string {
	return strings.TrimPrefix(s.ID, "quest."+group+"/")
}

// handlerNodes are the field nodes of a chunk: the steps of its task group without the hand-in. ok is false for a
// chunk no handler can run (unsupported, no group, or none of its steps exist).
func handlerNodes(q questboard.Quest, c questboard.Chunk) (nodes []task.Node, ok bool) {
	if c.Unsupported != "" {
		return nil, false
	}
	ref := q.HandlerOf(c)
	all := task.Handler(ref.Group)
	if len(all) == 0 {
		return nil, false
	}
	for _, n := range all {
		suf := stepSuffix(ref.Group, n.Step)
		switch {
		case ref.All:
			if !isHandIn(n.Step, suf, ref.HandIn) {
				nodes = append(nodes, n)
			}
		case contains(ref.Field, suf):
			nodes = append(nodes, n)
		}
	}
	if len(nodes) == 0 {
		if _, has := handInNode(q, c); !has {
			return nil, false // none of the chunk's steps exist in the group
		}
	}
	return nodes, true
}

func contains(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

// isHandIn: the hand-in step is a turn-in whose suffix is the chunk's handInStep ("reward" by default: V by grep of
// internal/task/act1.go to act5.go, ten handlers use id(q, "reward")).
func isHandIn(s task.Step, suffix, want string) bool { return s.Phase == "turn-in" && suffix == want }

func handInNode(q questboard.Quest, c questboard.Chunk) (task.Node, bool) {
	ref := q.HandlerOf(c)
	for _, n := range task.Handler(ref.Group) {
		if isHandIn(n.Step, stepSuffix(ref.Group, n.Step), ref.HandIn) {
			return n, true
		}
	}
	return task.Node{}, false
}

// nodesDone is the handler's own evidence for a part: every required field step satisfied (T3: items, areas), the
// hand-in step satisfied when it has one, and every chunk it comes after done in the same way (a hand-in step reads
// "done" before its item exists, so a part never counts before its predecessors).
//
// kept, when given, names the chunks already seen done in this process (boardRT.keptDone): they count without their
// evidence, which can have vanished (the key used up by the Tristram portal).
func nodesDone(q questboard.Quest, c questboard.Chunk, w task.World, depth int, kept ...func(chunk string) bool) bool {
	if depth > len(q.Chunks) {
		return false
	}
	nodes, ok := handlerNodes(q, c)
	if !ok {
		return false
	}
	total := 0
	for _, n := range nodes {
		if n.Step.Optional {
			continue
		}
		total++
		if !n.Step.Done(w) {
			return false
		}
	}
	hand, has := handInNode(q, c)
	if has && c.Ends == "handin" {
		if !hand.Step.Done(w) {
			return false
		}
	} else if total == 0 {
		return false
	}
	for _, a := range c.After {
		pc, err := q.ChunkByID(a)
		if len(kept) > 0 && kept[0](a) {
			continue
		}
		if err != nil || !nodesDone(q, pc, w, depth+1, kept...) {
			return false
		}
	}
	return true
}

func (e boardEnv) Chunk(q questboard.Quest, c questboard.Chunk) questboard.ChunkFact {
	rt := e.rt
	if _, ok := handlerNodes(q, c); !ok {
		return questboard.ChunkFact{}
	}
	w, _, _ := rt.world()
	return rt.chunkFact(q, c, w)
}

// keptDone reports the chunks of q that were seen done in this process (hand-in ran, or an item or auto chunk read done
// once) for the hero's difficulty in w.
func (rt *boardRT) keptDone(q questboard.Quest, w task.World) func(chunk string) bool {
	return func(chunk string) bool {
		k := questboard.ChunkKey(q.ID, chunk)
		rt.mu.Lock()
		defer rt.mu.Unlock()
		return rt.handed[k] || rt.reached[k+"/"+w.Difficulty]
	}
}

// chunkFact is Chunk over a given world (the evidence of one derive).
func (rt *boardRT) chunkFact(q questboard.Quest, c questboard.Chunk, w task.World) questboard.ChunkFact {
	nodes, ok := handlerNodes(q, c)
	if !ok {
		return questboard.ChunkFact{}
	}
	if w.Quests == nil && w.Level == 0 {
		return questboard.ChunkFact{} // no snapshot yet
	}
	cf := questboard.ChunkFact{Steps: make([]bool, len(c.Steps))}
	need, total, done := 0, 0, 0
	for _, n := range nodes {
		if n.Step.Optional {
			continue
		}
		total++
		if n.Step.Done(w) {
			done++
		} else {
			need++
			// A Ready fault without prerequisites to splice in front is a readiness fact ("needs clvl 8"), not a failure.
			if cf.Missing == "" && n.Step.Ready != nil {
				if f := n.Step.Ready(w); f != nil && len(f.Insert) == 0 {
					cf.Missing = f.Why
				}
			}
		}
	}
	// The questboard step names are descriptive and do not match the handler's step ids, so the per-step marks are the
	// share of satisfied handler steps spread over the named steps (I: progress for the strip, never a decision input).
	if total > 0 {
		for i := range cf.Steps {
			cf.Steps[i] = i < len(cf.Steps)*done/total
		}
	}
	rt.mu.Lock()
	key := questboard.ChunkKey(q.ID, c.ID)
	handed := rt.handed[key]
	reached := rt.reached[key+"/"+w.Difficulty]
	rt.mu.Unlock()
	if q.Kind == "unlock" {
		// An act hop is done when the world is in the next act (a live read), never because the dialogue ran: a talk
		// that did not take the boat latched "Sail east: DONE" on a hero still in Act 1 (user screenshot 2026-11-21).
		handed = false
	}
	if legacyChunk(q, c) {
		st := w.Quest(taskID(q))
		cf.Objective = need == 0 && st.Stage >= task.StageCleared
		if cf.Objective {
			cf.Missing = ""
		}
		if handed {
			cf.Done, cf.Tier = true, questboard.TierProxy // the dialogue ran; the quest word has the last say
		}
		return cf
	}
	// A part: the steps are the evidence. The group's own word (T1/T2) raises the tier when it says done.
	if nodesDone(q, c, w, 0, rt.keptDone(q, w)) || handed || reached {
		if c.Ends != "handin" && !reached {
			// V lab drill CAIN-L3 (2026-11-19): the stones chunk reads done only while the hero stands in Tristram
			// (or holds a portal) and the scroll chunk only while the key is in the pack. After the first town trip
			// from Tristram both read open again and the board walked back to the Dark Wood for a second scroll.
			// Once seen done, an item or auto chunk stays done for this hero and difficulty until the process ends.
			rt.mu.Lock()
			rt.reached[key+"/"+w.Difficulty] = true
			rt.mu.Unlock()
		}
		cf.Done, cf.Tier = true, questboard.TierProxy
		// The quest word of the handler group, or of the quest itself when the group is the board's own (S7b).
		for _, id := range []string{q.HandlerOf(c).Group, taskID(q)} {
			if st := w.Quest(id); st.Stage >= task.StageDone && evTier(st.Ev).Exact() {
				cf.Tier = evTier(st.Ev)
				break
			}
		}
		return cf
	}
	if _, has := handInNode(q, c); has && c.Ends == "handin" && need == 0 && priorDone(q, c, w, rt.keptDone(q, w)) {
		cf.Objective = true
		cf.Missing = ""
	}
	return cf
}

func priorDone(q questboard.Quest, c questboard.Chunk, w task.World, kept ...func(chunk string) bool) bool {
	for _, a := range c.After {
		pc, err := q.ChunkByID(a)
		if len(kept) > 0 && kept[0](a) {
			continue
		}
		if err != nil || !nodesDone(q, pc, w, 1, kept...) {
			return false
		}
	}
	return true
}

func (e boardEnv) Ready(q questboard.Quest, c questboard.Chunk) questboard.Word {
	if _, ok := handlerNodes(q, c); !ok {
		return questboard.ReadyUnknown // no handler for this chunk (unsupported)
	}
	_, _, ready := e.rt.world()
	return ready[questboard.ChunkKey(q.ID, c.ID)]
}

// ---- Actions: the hand-in tap and stop ----------------------------------------------------------------

type boardActions struct{ rt *boardRT }

// HandIn hands the tap to the runner loop (the dialogue must run on the loop's goroutine, inside its guard) and
// waits for the result. The caller bounds the wait with ctx (a town trip or a death delays the pick-up); S4 sets it.
func (a boardActions) HandIn(ctx context.Context, q questboard.Quest, c questboard.Chunk) error {
	req := boardHandInReq{ctx: ctx, q: q, c: c, done: make(chan error, 1)}
	k := questboard.ChunkKey(q.ID, c.ID)
	if err := a.rt.enqueue(req); err != nil {
		return err
	}
	a.rt.r.e.Log.Info("quest board: hand-in queued " + k)
	select {
	case err := <-req.done:
		return err
	case <-ctx.Done():
		a.rt.dropOwn(req)
		a.rt.r.e.Log.Info("quest board: hand-in dropped "+k+" (not served in time)", "err", ctx.Err())
		return ctx.Err()
	}
}

// enqueue puts the tap in the one-slot mailbox. A request whose ctx has ended is stale and makes room (before this
// it stayed for ever and refused every later tap with "already waiting").
func (rt *boardRT) enqueue(req boardHandInReq) error {
	for {
		select {
		case rt.reqs <- req:
			return nil
		case <-req.ctx.Done():
			return req.ctx.Err()
		default:
		}
		select {
		case old := <-rt.reqs:
			if old.ctx.Err() == nil {
				select {
				case rt.reqs <- old: // still live: put it back
				default:
				}
				return errors.New("a hand-in is already waiting for the bot")
			}
		default:
		}
	}
}

// dropOwn takes the caller's own request out of the mailbox when it timed out unserved.
func (rt *boardRT) dropOwn(req boardHandInReq) {
	select {
	case old := <-rt.reqs:
		if old.done != req.done {
			select {
			case rt.reqs <- old:
			default:
			}
		}
	default:
	}
	rt.mu.Lock()
	rt.handWhy = ""
	rt.mu.Unlock()
}

// QuestHandInWhy is DeskConfig.HandInWhy: why a queued hand-in tap waits ("" when none waits or it is being served).
func (r *Runner) QuestHandInWhy() string {
	rt := r.qbRT()
	if rt == nil {
		return ""
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if len(rt.reqs) == 0 {
		return ""
	}
	return rt.handWhy
}

// noteHandInHold records what keeps a queued tap from being served this tick (reason.code handin_waiting).
func (r *Runner) noteHandInHold(snap *d2.Snapshot, c control) {
	rt := r.qbRT()
	if rt == nil || len(rt.reqs) == 0 {
		return
	}
	why := "the hero is finishing another step first."
	switch {
	case snap == nil || snap.Player == nil || !snap.Player.InTown():
		why = "the hero is out in the field; it hands in at the next town visit."
	case r.recoveryActive():
		why = "the hero is refilling the purse and restocking first."
	case !r.questBoardOn():
		why = "the quest board is off."
	}
	rt.mu.Lock()
	rt.handWhy = why
	rt.mu.Unlock()
}

// Stop forgets the per-process notes of the chunk. A slice that is running ends at its own bound (the guard cancels it
// on safety and trip interrupts only): the cancel takes effect at the next slice boundary (U for a mid-walk cancel).
func (a boardActions) Stop(q questboard.Quest, c questboard.Chunk) {
	k := questboard.ChunkKey(q.ID, c.ID)
	a.rt.mu.Lock()
	delete(a.rt.handed, k)
	delete(a.rt.near, k)
	delete(a.rt.lastFail, k)
	a.rt.mu.Unlock()
}

// Unpark is the Retry of a parked chunk: the task executor's own block of the chunk's group has to go too, or the chunk
// is parked again at the next slice. The executor belongs to the loop goroutine, so the request is queued for it.
func (a boardActions) Unpark(q questboard.Quest, c questboard.Chunk) {
	a.rt.mu.Lock()
	a.rt.unpark[q.HandlerOf(c).Group] = true
	a.rt.mu.Unlock()
}

// applyUnparks lifts the requested blocks; it runs on the loop goroutine, the owner of ex.
func (rt *boardRT) applyUnparks(ex *task.Executor) {
	rt.mu.Lock()
	groups := rt.unpark
	rt.unpark = map[string]bool{}
	rt.mu.Unlock()
	for g := range groups {
		ex.Unpark(g)
	}
}

// ---- the loop step ------------------------------------------------------------------------------------

// boardPlan is the task.Planner of the board: the accepted chunk's handler without its hand-in, or the train slice.
type boardPlan struct {
	q     questboard.Quest
	c     questboard.Chunk
	train bool
}

func (p boardPlan) Plan(w task.World, s task.Session) task.Plan {
	if p.train {
		n := w.Level + 1
		return task.Plan{ID: fmt.Sprintf("level.%d", n), Why: "quest board: training", Nodes: []task.Node{{Step: task.StepLevel(n)}}}
	}
	tid := p.q.HandlerOf(p.c).Group
	pl := task.Plan{ID: "quest." + tid, Group: tid, Why: "quest board: accepted chunk"}
	if why, blocked := s.Blocked(tid); blocked {
		pl.Stuck, pl.Why = true, why
		return pl
	}
	pl.Nodes, _ = handlerNodes(p.q, p.c)
	return pl
}

// boardStep is one loop tick of a campaign goal with the board on.
func (r *Runner) boardStep(ctx context.Context, snap *d2.Snapshot, c control) bool {
	rt := r.qbRT()
	if rt == nil {
		r.boardNote("unwired", "quests.board is on but no board is wired: the hero only trains")
		return r.boardTrain(ctx, snap, c)
	}
	rt.worldLoop()
	rt.exec.SetEnabled(true)
	rt.exec.SetAutoOptions(r.questAutoOptions())
	r.parkedBoard.Store(false)
	if r.boardServeHandIn(ctx, snap, c) {
		return true
	}
	// Opt-in auto-start (slice S8, questauto.go): counts a safe offer down and starts it, or hands in what it started.
	if act, err := rt.exec.AutoTick(questboard.AutoInput{Paused: r.PlayerPauseView().State != PauseRunning, Calm: r.safeToPause(snap)}); err != nil {
		r.boardNote("auto", "quest board: auto-start: "+err.Error())
	} else if act.Kind == questboard.AutoStarted {
		r.boardNote("auto start "+act.Quest+"/"+act.Chunk, fmt.Sprintf("quest board: %s/%s started by itself", act.Quest, act.Chunk))
	}
	d, err := rt.exec.Next()
	r.syncTry(rt)
	if err != nil {
		r.boardNote("next", "quest board: "+err.Error())
		return r.boardBetween(ctx, snap, c)
	}
	if d.Kind != questboard.DecRun {
		r.setQuestCrowd(false)
	}
	switch d.Kind {
	case questboard.DecRun:
		rt.setQueued("")
		r.boardNote("run "+questboard.ChunkKey(d.Quest, d.Chunk), "quest board: starting chunk "+questboard.ChunkKey(d.Quest, d.Chunk))
		return r.boardRun(ctx, snap, c, d)
	case questboard.DecAwait:
		rt.setQueued("")
		k := questboard.ChunkKey(d.Quest, d.Chunk)
		r.boardNote("awaiting "+k, "quest board: awaiting hand-in of "+k)
		return r.boardAwait(ctx, snap, c, d)
	case questboard.DecPaused, questboard.DecBusy:
		return r.rest(ctx, snap, c)
	case questboard.DecBlocked:
		r.boardNote("blocked "+d.Quest, fmt.Sprintf("quest board: %s/%s parked: %s", d.Quest, d.Chunk, d.Why))
	}
	return r.boardBetween(ctx, snap, c) // DecIdle, or a parked chunk: the mode decides
}

// boardNote logs a visible reason once per distinct text.
func (r *Runner) boardNote(key, line string) {
	rt := r.qbRT()
	if rt == nil {
		return
	}
	rt.mu.Lock()
	seen := rt.noted[key] == line
	rt.noted[key] = line
	rt.mu.Unlock()
	if !seen {
		r.e.Log.Info(line)
	}
}

func (r *Runner) boardTrain(ctx context.Context, snap *d2.Snapshot, c control) bool {
	r.noteTrain(snap)
	_, ok := r.boardSlice(ctx, snap, c, boardPlan{train: true}, "Quest board: training")
	return ok
}

// boardRun runs one slice of the accepted chunk and reports it.
func (r *Runner) boardRun(ctx context.Context, snap *d2.Snapshot, c control, d questboard.Decision) bool {
	rt := r.qbRT()
	q, err := rt.board.Quest(d.Quest)
	if err != nil {
		r.boardNote("quest", err.Error())
		return r.boardBetween(ctx, snap, c)
	}
	ch, err := rt.board.Chunk(d.Quest, d.Chunk)
	if err != nil {
		r.boardNote("chunk", err.Error())
		return r.boardBetween(ctx, snap, c)
	}
	// F1/F2 (cainfix3): the player started this chunk; unless the readiness model calls it too hard, the farm areas' death
	// bans and the crowd-abandon rule of walks do not stop it (a refused start shows the ban's deaths and end time instead).
	rt.mu.Lock()
	word := rt.ready[questboard.ChunkKey(d.Quest, d.Chunk)]
	rt.mu.Unlock()
	if word != questboard.ReadyTooHard {
		r.banWaive.Store(true)
		defer r.banWaive.Store(false)
	}
	// The crowd line stays raised between slices too (a potion trip or a fight of the guard runs outside boardSlice, and
	// the Proving Ground chain run abandoned its walk in Tristram that way); boardStep and boardPreemptsWhy lower it when
	// the board is not running a chunk.
	r.setQuestCrowd(word != questboard.ReadyTooHard)
	res, ok := r.boardSlice(ctx, snap, c, boardPlan{q: q, c: ch}, "Quest: "+q.Title)
	out := questboard.StepResult{Outcome: questboard.StepRan}
	idle := false // nothing ran this tick: the loop must still spend time (below)
	held := false // a hold: the hero trains instead (below)
	switch {
	case r.deadNow() || res.Outcome == task.OutDeath:
		out = questboard.StepResult{Outcome: questboard.StepDied, Why: nonEmptyStr(whyOf(res), "died on the way")}
	case res.Outcome == task.OutInterrupted || res.Outcome == task.OutWait:
		out.Outcome = questboard.StepWaiting
		if w := whyOf(res); w != "" {
			r.boardNote("wait "+questboard.ChunkKey(d.Quest, d.Chunk), "quest board: waiting: "+w)
		}
	case res.Outcome == task.OutRetry && res.Hold && res.Code != "":
		// quests.boardSM: a guard refused for a reason that is no failure (potions, ban, too weak, closed road): a hold.
		// Nothing is counted and the hero trains meanwhile; the board parks it with hold_timeout when it outlives its budget.
		out = questboard.StepResult{Outcome: questboard.StepHeld, Code: res.Code, Why: nonEmptyStr(whyOf(res), "the step was refused")}
		held = true
		r.boardNote("hold "+questboard.ChunkKey(d.Quest, d.Chunk)+" "+res.Code, "quest board: hold "+res.Code+": "+out.Why)
	case res.Outcome == task.OutRetry && res.Kind == task.KindExploring:
		// still walking toward an object that is not built yet: progress, so the failure counter stays at zero
		r.boardNote("explore "+questboard.ChunkKey(d.Quest, d.Chunk), "quest board: "+whyOf(res))
	case res.Outcome == task.OutRetry || res.Outcome == task.OutBlocked:
		out = questboard.StepResult{Outcome: questboard.StepFailed, Why: nonEmptyStr(whyOf(res), "the step was refused")}
		// A refusal by the area ban or a guard ran nothing: under a lockstep clock no time passes, so the failure gap never
		// elapses and the chunk spins (Proving Ground cain.chain, 2026-11-19: virtual time frozen 20 minutes wall after the
		// re-entry step met a banned Stony Field). Going to town spends time; three counted failures then park the chunk.
		idle = res.Kind == task.KindDanger
	case res.Outcome == task.OutRan && res.StepID == "":
		// Nothing left to run, yet the quest does not read as finished: never spin on it silently.
		out = questboard.StepResult{Outcome: questboard.StepFailed, Why: "all steps done but the quest does not read as finished (waiting for the save or memory)"}
		idle = true
	case !ok:
		out.Outcome = questboard.StepWaiting
	}
	if out.Outcome == questboard.StepRan && res.Via != "" {
		// quests.boardSM: a coded refusal (a potion shortage) sent the hero to the chores trip that just ran. The board shows
		// the hold for the trip; nothing is counted, and the next slice clears it when the guard lets the step through.
		out = questboard.StepResult{Outcome: questboard.StepHeld, Code: res.Via, Why: nonEmptyStr(res.ViaWhy, "a town trip fixes it")}
		r.boardNote("hold "+questboard.ChunkKey(d.Quest, d.Chunk)+" "+res.Via, "quest board: hold "+res.Via+": "+out.Why)
	}
	failed := out.Outcome == questboard.StepFailed
	if out.Outcome == questboard.StepFailed {
		k := questboard.ChunkKey(d.Quest, d.Chunk)
		rt.mu.Lock()
		now := r.e.Clock.Now()
		if last, seen := rt.lastFail[k]; seen && now.Sub(last) < boardFailGap {
			out.Outcome = questboard.StepWaiting // same refusal inside the gap: not a new failure
		} else {
			rt.lastFail[k] = now
		}
		rt.mu.Unlock()
		r.boardNote("fail "+k, fmt.Sprintf("quest board: %s: %s", k, out.Why))
	}
	rt.exec.Report(d, out)
	if d := failSpend(failed, idle); d > 0 {
		// A refused step that ran nothing (a path or exit error, Proving Ground necro.act1-stage1 2026-11-23: the hero stood in
		// the Barracks and the tick loop spun 49 minutes wall with the lockstep clock stopped) spends no game time, so the
		// failure gap never elapses and the three counted failures that park the chunk never come. Spend a third of the gap.
		r.e.Clock.Sleep(d)
	}
	if held {
		return r.boardTrain(ctx, snap, c)
	}
	if idle {
		// Proving Ground cain.cage (2026-11-19): a free_cain start outside Tristram has no step to run. The tick did nothing,
		// so under a lockstep clock no time passed, the failure gap above never elapsed and the chunk spun for 17 minutes
		// wall in a field. Going to town spends time, and three counted failures then park the chunk with its reason.
		if snap.Player.InTown() {
			// rest does nothing in town unless chores are due: spend the time here so the failure gap can elapse
			// (Proving Ground cain.chain 2026-11-19: virtual clock frozen with the hero in camp, 30 minutes wall).
			r.e.Clock.Sleep(boardFailGap / 3)
		}
		return r.rest(ctx, snap, c)
	}
	return ok
}

// failSpend is the game time one board tick spends after a failed step that did not already wait (idle ticks spend theirs below).
func failSpend(failed, idle bool) time.Duration {
	if failed && !idle {
		return boardFailGap / 3
	}
	return 0
}

func whyOf(res task.Result) string {
	if res.Err != nil {
		return res.Err.Error()
	}
	return res.Attention
}

func nonEmptyStr(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

// boardSlice runs one executor tick of pl inside the runner's guard (the same r.do reactions as taskStep).
func (r *Runner) boardSlice(ctx context.Context, snap *d2.Snapshot, c control, pl task.Planner, activity string) (task.Result, bool) {
	l := &r.loop
	r.ensureTaskExecutor(c)
	ex := l.task.ex
	if rt := r.qbRT(); rt != nil {
		rt.applyUnparks(ex)
	}
	env := taskEnv{r}
	w, err := env.World(ctx)
	if err != nil {
		return task.Result{Outcome: task.OutWait}, false
	}
	r.setTargetArea(0)
	r.unparkDue(ex)
	plan := pl.Plan(w, ex)
	st, act := StateTraveling, activity
	if step, _, has := ex.Peek(plan, w); has {
		st, act = phaseState(step.Phase)
		act += " (" + step.Title + ")"
		l.task.stepTitle = step.Title
		if bp, ok := pl.(boardPlan); ok && !bp.train {
			r.boardNote("step", "quest board: step "+step.Title)
		}
	}
	var res task.Result
	l.task.err = nil
	r.setWhy(WhyQuestStep, "step", activity, "level", itoa(snap.Player.Level))
	ok := r.do(ctx, c, st, act, 40*time.Minute, "quest board step", func(ctx context.Context) error {
		res, _ = ex.Tick(ctx, pl)
		if l.task.err != nil {
			return l.task.err
		}
		if res.Outcome == task.OutInterrupted {
			var si safetyInterrupt
			if cause := context.Cause(ctx); errors.As(cause, &si) {
				return cause
			}
		}
		r.noteStepRefused(res)
		if r.e.Tune.Bot.TypedOutcome != 0 {
			if o := classify(res); o.Kind == OutRefusedKind {
				return o.asError() // handled in do (handleRefused): no strike, one line per change
			}
		}
		if res.Outcome == task.OutRetry && res.Kind != task.KindExploring || res.Outcome == task.OutBlocked {
			return res.Err
		}
		return nil
	})
	return res, ok
}

// boardAwait stands the hero in town next to the giver and waits for the player's tap. The walk to the giver is one
// open-and-close of the dialog (the live path of the hand-in), done once per chunk and process.
func (r *Runner) boardAwait(ctx context.Context, snap *d2.Snapshot, c control, d questboard.Decision) bool {
	rt := r.qbRT()
	k := questboard.ChunkKey(d.Quest, d.Chunk)
	rt.mu.Lock()
	seen := rt.near[k]
	rt.mu.Unlock()
	if !seen && snap.Player.InTown() {
		if ch, err := rt.board.Chunk(d.Quest, d.Chunk); err == nil && ch.Giver != nil && r.d.Task != nil && r.d.Task.Dialog != nil {
			rt.mu.Lock()
			rt.near[k] = true
			rt.mu.Unlock()
			r.boardNote("await "+k, fmt.Sprintf("quest board: %s is ready to hand in to %s", k, ch.Giver.NPC))
			r.do(ctx, c, StateResupplying, "Quest: walking to "+ch.Giver.NPC, 3*time.Minute, "quest board approach", func(ctx context.Context) error {
				return r.d.Task.Dialog.Talk(ctx, ch.Giver.NPC, "")
			})
			return true
		}
	}
	return r.rest(ctx, snap, c)
}

// boardServeHandIn runs a waiting HandIn tap on the loop's goroutine: the hand-in step of the quest's handler, nothing
// else. It reports true when it used the tick.
func (r *Runner) boardServeHandIn(ctx context.Context, snap *d2.Snapshot, c control) bool {
	rt := r.qbRT()
	if rt == nil {
		return false
	}
	// V lab drill CAIN-L3 (2026-11-19): a tap that landed while the hero was out in the Dark Wood was taken off the
	// queue in the field, answered "not in town" three times in one second and parked the chunk (docs/questboard.md
	// promises the hand-in waits for the next town visit). The request stays queued until the hero stands in town.
	if snap == nil || snap.Player == nil || !snap.Player.InTown() {
		return false
	}
	select {
	case req := <-rt.reqs:
		if req.ctx.Err() != nil { // the tapper gave up: nobody waits for this talk
			r.e.Log.Info("quest board: hand-in dropped " + questboard.ChunkKey(req.q.ID, req.c.ID) + " (tapper gone)")
			return false
		}
		rt.mu.Lock()
		rt.handWhy = ""
		rt.mu.Unlock()
		r.e.Log.Info("quest board: hand-in served " + questboard.ChunkKey(req.q.ID, req.c.ID))
		req.done <- r.boardHandIn(ctx, snap, c, req)
		return true
	default:
		return false
	}
}

func (r *Runner) boardHandIn(ctx context.Context, snap *d2.Snapshot, c control, req boardHandInReq) error {
	rt := r.qbRT()
	n, ok := handInNode(req.q, req.c)
	if !ok {
		return fmt.Errorf("quest %s has no hand-in step in the task layer", req.q.ID)
	}
	if !snap.Player.InTown() {
		return errors.New("not in town")
	}
	var derr error
	okDo := r.do(ctx, c, StateResupplying, "Quest: handing in "+req.q.Title, 3*time.Minute, "quest hand-in", func(ctx context.Context) error {
		derr = n.Step.Do(ctx, taskActs{r, c})
		return derr
	})
	if !okDo || derr != nil {
		return nonNilErr(derr, "the hand-in dialogue did not finish")
	}
	rt.mu.Lock()
	rt.handed[questboard.ChunkKey(req.q.ID, req.c.ID)] = true
	rt.mu.Unlock()
	return nil
}

func nonNilErr(err error, msg string) error {
	if err != nil {
		return err
	}
	return errors.New(msg)
}

// chunkNeedsNoGold says the decided chunk buys nothing (questboard Chunk.NoGold): the purse recovery does not hold it
// (cainfix3 F3, Proving Ground cain.stones: the stones chunk waited behind a recovery it did not need).
func (r *Runner) chunkNeedsNoGold(rt *boardRT, d questboard.Decision) bool {
	if d.Kind != questboard.DecRun {
		return false
	}
	ch, err := rt.board.Chunk(d.Quest, d.Chunk)
	return err == nil && ch.NoGold
}

// boardChunkNeedsNoGold is chunkNeedsNoGold for the chunk the board would run next (campaign goals).
func (r *Runner) boardChunkNeedsNoGold() bool {
	if !r.questBoardOn() {
		return false
	}
	rt := r.boardRTFor()
	if rt == nil {
		return false
	}
	d, err := rt.exec.Next()
	return err == nil && r.chunkNeedsNoGold(rt, d)
}

// boardQueueGrace is how long an accepted chunk waits for a calm moment before it starts amid hostiles (virtual clock).
const boardQueueGrace = 20 * time.Second

func (rt *boardRT) setQueued(why string) {
	rt.mu.Lock()
	rt.queued = why
	rt.mu.Unlock()
}

// QuestQueued is DeskConfig.Queued: why an accepted chunk is not running yet, in words ("" when it runs).
func (r *Runner) QuestQueued() string {
	rt := r.qbRT()
	if rt == nil {
		return ""
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.queued
}

// boardPreempts is true when the Start tap of the player must run now although the goal is not a campaign: a farm area,
// level or boss goal is the hero's standing order, but an accepted chunk (or a hand-in tap) is the newer one. Before this
// the board only ran under a campaign goal, so the accept was recorded and nothing started (live 2026-11-17, goal
// farm_area on Dark Wood, 13 minutes). A fight in the clear radius delays only the start of a chunk, never a hand-in; the
// reason is shown as "queued". The Farm gold order and a recovery that has to refill the purse still come first.
// boardYield tells a goal travel (the level plan walking to its area) to stop at the next hop because a chunk the player
// started is runnable: a long walk is one blocking action, so the start gate only saw the chunk after the hero had crossed
// several areas (Proving Ground necro.act1-stage1 2026-11-23: Blood Raven accepted in the Stony Field, started 30 s later in
// the Barracks, after the plan had walked the hero along the Monastery road). Once per chunk, so a chunk that cannot start
// (recovery, gates) never starves the travel.
func (r *Runner) boardYield() bool {
	if !r.questBoardOn() {
		return false
	}
	rt := r.boardRTFor()
	if rt == nil {
		return false
	}
	d, err := rt.exec.Next()
	if err != nil || d.Kind != questboard.DecRun {
		return false
	}
	k := questboard.ChunkKey(d.Quest, d.Chunk)
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.startedKey == k || rt.yieldKey == k {
		return false
	}
	rt.yieldKey = k
	return true
}

// syncTry follows the executor's try-anyway state into the task layer (task.SetTry): on while the owner's one-shot override
// runs its chunk, off the moment the executor ends it (park, death, finish, stop). Called right after every Next.
func (r *Runner) syncTry(rt *boardRT) {
	on := rt.exec.TryKey() != ""
	if on != task.TryActive() {
		r.e.Log.Info("override: try anyway", "active", on)
	}
	task.SetTry(on)
}

// boardHoldsArea says an accepted chunk waits for the fight the hero is in: the level goal must not walk the hero out of
// the chunk's area meanwhile (V lab 2026-11-23, necro step scenario: the plan walked the hero out of the Barracks while the
// army fought, then the first step of the chunk parked "ended in 26").
func (r *Runner) boardHoldsArea() bool {
	rt := r.boardRTFor()
	if rt == nil {
		return false
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.holdFight
}

func (r *Runner) boardPreempts(snap *d2.Snapshot, c control) bool {
	ok, why := r.boardPreemptsWhy(snap, c)
	if why != "" {
		// once per distinct reason: the Proving Ground trace showed a chunk deferred for 20 virtual minutes with no line saying why
		r.boardNote("preempt", "quest board: start gate: "+why)
	}
	return ok
}

func (r *Runner) boardPreemptsWhy(snap *d2.Snapshot, c control) (bool, string) {
	if !r.questBoardOn() || c.spec.camp != nil {
		return false, ""
	}
	switch c.spec.kind {
	case goals.ReachLevel, goals.FarmArea, goals.KillBoss:
	default:
		return false, ""
	}
	if r.goldFarmOn() {
		return false, "the Farm gold order comes first"
	}
	rt := r.boardRTFor()
	if rt == nil {
		return false, ""
	}
	rt.worldLoop()
	rt.mu.Lock()
	rt.holdFight = false
	rt.mu.Unlock()
	d, err := rt.exec.Next()
	r.syncTry(rt)
	if err != nil {
		return false, ""
	}
	k := questboard.ChunkKey(d.Quest, d.Chunk)
	if d.Kind != questboard.DecRun {
		r.setQuestCrowd(false)
	}
	if (d.Kind == questboard.DecRun || d.Kind == questboard.DecAwait || d.Kind == questboard.DecBusy) &&
		r.recoveryActive() && !r.gearVerdict(snap).Gated && !r.chunkNeedsNoGold(rt, d) {
		// The recovery (empty purse, no kit) stays ahead of a chunk, but the player must see it: before, the accepted chunk
		// showed no reason at all while the hero farmed gold for many minutes (Proving Ground cain.scroll, 2026-11-19).
		rt.setQueued("Starts after the hero has refilled the purse and restocked.")
		return false, "waiting for the recovery (purse and kit) to finish"
	}
	switch d.Kind {
	case questboard.DecRun:
		rt.mu.Lock()
		under := rt.startedKey == k
		rt.mu.Unlock()
		if !under && !r.safeToPause(snap) {
			now := r.e.Clock.Now()
			rt.mu.Lock()
			if rt.queuedSince.IsZero() {
				rt.queuedSince = now
			}
			late := now.Sub(rt.queuedSince) >= boardQueueGrace
			rt.holdFight = !late
			rt.mu.Unlock()
			if !late {
				rt.setQueued("Starts when the fight is over.")
				r.boardNote("queued "+k, "quest board: "+k+" queued: waiting for the fight to end")
				return false, "waiting for the fight to end"
			}
			// Proving Ground 2026-11-19 (cain.scroll Sorceress, cain.stones): the level sweep keeps a hostile in the clear
			// radius at nearly every decision, so a chunk that waits for calm never starts. The step fights what stands in
			// its way, so after the grace the player's Start wins over the level goal's area choice.
			r.boardNote("queued-late "+k, "quest board: "+k+" starts although hostiles are near (queued over "+boardQueueGrace.String()+")")
		}
		rt.mu.Lock()
		rt.queuedSince = time.Time{}
		rt.startedKey = k
		rt.mu.Unlock()
		rt.setQueued("")
		return true, ""
	case questboard.DecAwait, questboard.DecBusy:
		return true, ""
	case questboard.DecBlocked:
		rt.mu.Lock()
		rt.startedKey = ""
		rt.mu.Unlock()
		rt.setQueued("Parked: " + d.Why)
		r.boardNote("blocked "+d.Quest, fmt.Sprintf("quest board: %s parked: %s", k, d.Why))
	case questboard.DecPaused:
		rt.setQueued("Paused: the hero starts it when you resume.")
		rt.mu.Lock()
		rt.startedKey = ""
		rt.mu.Unlock()
	default:
		rt.setQueued("")
		rt.mu.Lock()
		rt.startedKey = ""
		rt.mu.Unlock()
	}
	return false, ""
}

// botReleaseID is the release this bot was built as (buildinfo, the ldflags stamp). D2BOT_RELEASE_ID overrides it so
// a Proving Ground scenario can fake a new release across a restart.
func botReleaseID() string {
	if v := os.Getenv("D2BOT_RELEASE_ID"); v != "" {
		return v
	}
	return buildinfo.Current()
}
