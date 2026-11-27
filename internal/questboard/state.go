package questboard

import (
	"fmt"
	"strings"
)

// State is one of the nine states of design 1.3, per quest and per chunk.
type State string

const (
	Locked     State = "locked"
	Available  State = "available"
	Ready      State = "ready"
	Accepted   State = "accepted"
	InProgress State = "in_progress"
	Awaiting   State = "awaiting_handin"
	Done       State = "done"
	Skipped    State = "skipped"
	Blocked    State = "blocked"
)

// AllStates lists the nine states.
var AllStates = []State{Locked, Available, Ready, Accepted, InProgress, Awaiting, Done, Skipped, Blocked}

// Tier is the evidence tier of docs/progression/quests.md section 1; the order matches quest.Evidence so S3
// converts by value. A step may SKIP work, and a prerequisite may UNLOCK, only on TierSave or above.
type Tier int

const (
	TierNone   Tier = iota
	TierProxy       // T3: reward item, kill event, dialog change (may lie)
	TierSave        // T1: the .d2s quest block (stale until the game writes the file)
	TierMemory      // T2: PlayerData quest pointers (immediate, offsets U)
)

func (t Tier) String() string { return [...]string{"none", "T3", "T1", "T2"}[t] }

// Exact reports whether the tier may skip work or unlock a prerequisite (T1 or T2).
func (t Tier) Exact() bool { return t >= TierSave }

// Word is the readiness word (slice S2, readiness.go). Unknown (the model is absent) never makes an
// offer: the quest stays available (I: no offer without the model).
type Word string

const (
	ReadyUnknown Word = ""
	ReadySafe    Word = "safe"
	ReadyRisky   Word = "risky"
	ReadyTooHard Word = "too_hard"
)

// Fact is what the quest reader says about one quest in one difficulty. S3 builds it from quest.Reader
// (Info.Status, Pending, Evidence); Failed is the game-failed signal of Rescue on Mount Arreat (U). Cleared is the
// quest's objective reached at any tier (a proxy such as the made item in the pack counts): what Quest.RequiresCleared
// asks for.
type Fact struct {
	Known                                   bool
	Done, Started, Pending, Failed, Cleared bool
	Tier                                    Tier
}

// Facts answers quest facts per difficulty index.
type Facts interface {
	Quest(diff int, id string) Fact
}

// ChunkFact is what the handler layer derives for one chunk each tick (never persisted): the per-step
// checks (each step's own "already done" test, true = satisfied), whether the field objective is met,
// whether the executor is running it, and why it cannot start or has failed.
type ChunkFact struct {
	Done      bool   // the chunk's own done check (a proxy such as the item in the pack, or a sub-bit)
	Tier      Tier   // tier of that Done
	Objective bool   // field work finished; the hero only needs the NPC
	Steps     []bool // per-step checks, in Chunk.Steps order
	Running   bool   // the executor is on this chunk now
	Missing   string // a readiness fact that is not a failure ("needs 3 antidotes"), keeps the chunk available
	Denied    bool   // exact evidence (T1 or T2) says "not done" although a latch says done (card bsm4, R3); the bot does not set it yet (U)
	Blocked   string // the executor gave up (3 failures, lost item, retry cap); words for the Needs-you row
}

// ChunkKey is the key of Input.Chunks and Input.Ready.
func ChunkKey(quest, chunk string) string { return quest + "/" + chunk }

// Input is everything Derive reads. Nothing in it is handler progress that was saved: Facts and Chunks are
// re-read each tick, Plan holds only the player's records.
type Input struct {
	Diff   int
	Facts  Facts
	Chunks map[string]ChunkFact
	Ready  map[string]Word // by ChunkKey
	Plan   Plan
}

// ChunkView is the derived state of one chunk.
type ChunkView struct {
	ID        string
	State     State
	Tier      Tier   // the strongest evidence behind the state
	Confirmed bool   // for Done: T1/T2 says so; false means "confirming" (T3 only)
	Step      int    // InProgress: index of the first unsatisfied step, else -1
	StepName  string // Chunk.Steps[Step] when known
	Why       string
	Reason    Word
}

// QuestView is the derived state of one quest.
type QuestView struct {
	ID        string
	Title     string
	State     State
	Tier      Tier
	Confirmed bool
	Why       string
	Chunks    []ChunkView
	// Offered lists chunks that raise an offer now (Ready), several when their order does not matter.
	Offered []string
	// Awards are meta keys earned by evidence now. Empty for a quest the player did by hand (no backfill,
	// user decision 2026-11-16) and for T3-only completion (confirming).
	Awards []Award
}

// Award is one meta currency credit; the key makes it idempotent in the account ledger (design section 7).
type Award struct {
	Key    string
	Amount int
}

// AwardKey is the account-wide ledger key: quest, chunk ("-" for the whole quest), difficulty. Never the
// hero, so a quest earns its meta once per account per difficulty (user decision 2026-11-16).
func AwardKey(quest, chunk string, diff int) string {
	if chunk == "" {
		chunk = "-"
	}
	return fmt.Sprintf("quest/%s/%s/%s", quest, chunk, Difficulties[diff])
}

// Derive computes the view of every quest for one difficulty from evidence and the plan records. It is a pure
// function of its input: derive twice, same result; a restart changes nothing but the live strip position.
func Derive(b *Board, in Input) ([]QuestView, error) {
	if in.Diff < 0 || in.Diff > 2 {
		return nil, fmt.Errorf("questboard: difficulty index %d out of range (0 to 2)", in.Diff)
	}
	if in.Facts == nil {
		return nil, fmt.Errorf("questboard: no facts source")
	}
	if err := in.Plan.check(b); err != nil {
		return nil, err
	}
	for k := range in.Chunks {
		q, c, ok := strings.Cut(k, "/")
		if !ok {
			return nil, fmt.Errorf("questboard: chunk fact key %q is not quest/chunk", k)
		}
		if _, err := b.Chunk(q, c); err != nil {
			return nil, err
		}
	}
	out := make([]QuestView, 0, len(b.quests))
	for _, q := range b.quests {
		out = append(out, deriveQuest(b, q, in))
	}
	return out, nil
}

func deriveQuest(b *Board, q Quest, in Input) QuestView {
	v := QuestView{ID: q.ID, Title: q.Title, Tier: TierNone}
	fact := in.Facts.Quest(in.Diff, q.ID)
	byHand := !in.Plan.touched(in.Diff, q.ID)

	if !q.Exists(in.Diff) {
		v.State, v.Why = Skipped, "skipped in this difficulty"
		for _, c := range q.Chunks {
			v.Chunks = append(v.Chunks, ChunkView{ID: c.ID, State: Skipped, Step: -1, Why: v.Why})
		}
		return v
	}

	for _, c := range q.Chunks {
		v.Chunks = append(v.Chunks, deriveChunk(b, q, c, fact, in))
	}
	v.State, v.Tier, v.Confirmed, v.Why = aggregate(v.Chunks, fact)
	for _, cv := range v.Chunks {
		if cv.State == Ready {
			v.Offered = append(v.Offered, cv.ID)
		}
	}
	if fact.Known && fact.Done {
		// Quiet done: the board never ran it, so it pays nothing.
		if byHand {
			v.Why = "already done"
		}
	}
	v.Awards = awards(q, v, in)
	return v
}

func deriveChunk(b *Board, q Quest, c Chunk, fact Fact, in Input) ChunkView {
	cf := in.Chunks[ChunkKey(q.ID, c.ID)]
	cv := ChunkView{ID: c.ID, Step: -1}

	// Done beats everything: the quest word (T1/T2/T3) for the whole quest, the chunk's own check for a part.
	switch {
	case fact.Known && fact.Done:
		cv.State, cv.Tier, cv.Confirmed = Done, fact.Tier, fact.Tier.Exact()
		return cv
	// A proxy read never beats an unmet prerequisite: a step that is "not needed" only because its ready test is false
	// (Duriel's door without the Staff, the Hellforge without a Soulstone) reads done, and an Act 1 hero showed Duriel and
	// the Hellforge done (sandbox 2026-11-23). Exact evidence (T1/T2) of a later part still counts.
	case cf.Done && !memoryDenies(q, fact, cf) && (cf.Tier.Exact() || lockedBy(b, q, c, in) == ""):
		cv.State, cv.Tier, cv.Confirmed = Done, cf.Tier, cf.Tier.Exact()
		return cv
	}
	if fact.Known && fact.Failed {
		cv.State, cv.Why = Blocked, "failed in this game"
		cv.Tier = fact.Tier
		return cv
	}
	if in.Plan.skipped(in.Diff, q.ID, c.ID) {
		cv.State, cv.Why = Skipped, "skipped by you"
		return cv
	}
	rec := in.Plan.accepted(in.Diff, q.ID, c.ID)
	if rec && cf.Blocked != "" {
		cv.State, cv.Why = Blocked, cf.Blocked
		return cv
	}
	if why := lockedBy(b, q, c, in); why != "" {
		cv.State, cv.Why = Locked, why
		return cv
	}
	// No handler can run it yet: shown with the reason, never startable (slice S7).
	if c.Unsupported != "" {
		cv.State, cv.Why = Available, "not supported yet: "+c.Unsupported
		return cv
	}
	// Awaiting hand-in: the pending bit on exact evidence, or the objective met on a chunk the player started.
	// T3 is enough to STOP and ask for the tap, never to skip work (design 1.4).
	if c.Ends == "handin" && ((fact.Known && fact.Pending && fact.Tier.Exact()) || (rec && cf.Objective)) {
		cv.State, cv.Tier = Awaiting, TierProxy
		if fact.Known && fact.Pending && fact.Tier.Exact() {
			cv.Tier = fact.Tier
		}
		return cv
	}
	if rec {
		cv.State, cv.Tier = Accepted, TierNone
		if cf.Running {
			cv.State = InProgress
			cv.Step, cv.StepName = firstOpenStep(c, cf)
		}
		return cv
	}
	r := in.Ready[ChunkKey(q.ID, c.ID)]
	cv.Reason = r
	switch {
	case cf.Missing != "":
		cv.State, cv.Why = Available, cf.Missing
	case r == ReadySafe || r == ReadyRisky:
		cv.State = Ready
	case r == ReadyTooHard:
		cv.State, cv.Why = Available, "too hard for now"
	default:
		cv.State, cv.Why = Available, "readiness not known"
	}
	return cv
}

// lockedBy returns why the chunk cannot be offered, or "". Quest prerequisites unlock on T1/T2 only (T3 never
// unlocks); chunk order accepts any evidence of the earlier chunk, because an offer skips no work and the handler
// re-derives (I, documented in docs/questboard.md).
// memoryDenies is the evidence conflict rule: for a single-chunk quest, a fresh T2 read that says "not done"
// beats a proxy that says "done" (a reward item that was dropped, a kill by the merc). A multi-chunk part has
// no quest bit of its own, so its proxy stands.
func memoryDenies(q Quest, f Fact, cf ChunkFact) bool {
	return len(q.Chunks) == 1 && f.Known && !f.Done && f.Tier == TierMemory && cf.Tier < TierSave
}

func lockedBy(b *Board, q Quest, c Chunk, in Input) string {
	for _, id := range q.RequiresCleared {
		if f := in.Facts.Quest(in.Diff, id); !(f.Known && (f.Done || f.Cleared)) {
			return lockedUntil(b, id)
		}
	}
	for _, id := range append(append([]string{}, q.Requires...), c.Requires...) {
		f := in.Facts.Quest(in.Diff, id)
		// An act hop (kind unlock) waits for the boss's objective, not for its hand-in: the Warriv talk that sets the
		// quest's done bit is the hop itself (V docs/d2d-probes.md: Andariel dead is 0x200a, the talk makes it 0x2009 and
		// "go east" lands in Lut Gholein), so asking for done made Sail east unreachable.
		hop := q.Kind == "unlock" && f.Pending
		if !(f.Known && (f.Done || hop) && f.Tier.Exact()) {
			return lockedUntil(b, id)
		}
	}
	for _, a := range c.After {
		if !chunkDone(q, a, in, in.Facts.Quest(in.Diff, q.ID)) {
			return "after " + chunkTitle(q, a)
		}
	}
	return ""
}

func lockedUntil(b *Board, id string) string {
	if p, err := b.Quest(id); err == nil {
		return "locked until " + p.Title
	}
	return "locked until " + id
}

func chunkDone(q Quest, id string, in Input, qf Fact) bool {
	if qf.Known && qf.Done {
		return true
	}
	return in.Chunks[ChunkKey(q.ID, id)].Done
}

func chunkTitle(q Quest, id string) string {
	for _, c := range q.Chunks {
		if c.ID == id {
			return c.Title
		}
	}
	return id
}

func firstOpenStep(c Chunk, cf ChunkFact) (int, string) {
	for i, name := range c.Steps {
		if i >= len(cf.Steps) || !cf.Steps[i] {
			return i, name
		}
	}
	return -1, ""
}

// priority orders chunk states for the quest's headline state (highest wins). Done chunks are excluded.
var priority = map[State]int{Skipped: 1, Locked: 2, Available: 3, Ready: 4, Accepted: 5, InProgress: 6, Awaiting: 7, Blocked: 8}

func aggregate(cs []ChunkView, fact Fact) (st State, tier Tier, confirmed bool, why string) {
	open, done := 0, 0
	var best *ChunkView
	for i := range cs {
		c := &cs[i]
		if c.State == Done {
			done++
			if c.Tier > tier {
				tier = c.Tier
			}
			continue
		}
		open++
		if best == nil || priority[c.State] > priority[best.State] {
			best = c
		}
	}
	if open == 0 {
		confirmed = true
		for _, c := range cs {
			confirmed = confirmed && c.Confirmed
		}
		return Done, tier, confirmed, ""
	}
	return best.State, best.Tier, false, best.Why
}

func awards(q Quest, v QuestView, in Input) []Award {
	if !in.Plan.touched(in.Diff, q.ID) {
		return nil
	}
	var out []Award
	d := Difficulties[in.Diff]
	for i, c := range q.Chunks {
		if k := AwardKey(q.ID, c.ID, in.Diff); v.Chunks[i].State == Done && v.Chunks[i].Confirmed && c.Meta[d] > 0 && !in.Plan.inBaseline(k) {
			out = append(out, Award{k, c.Meta[d]})
		}
	}
	if k := AwardKey(q.ID, "", in.Diff); v.State == Done && v.Confirmed && q.Meta[d] > 0 && !in.Plan.inBaseline(k) {
		out = append(out, Award{k, q.Meta[d]})
	}
	return out
}

// doneKeys are the award keys of every chunk and quest that reads Done, with or without a column: the baseline.
func doneKeys(vs []QuestView, diff int) []string {
	var out []string
	for _, v := range vs {
		for _, c := range v.Chunks {
			if c.State == Done {
				out = append(out, AwardKey(v.ID, c.ID, diff))
			}
		}
		if v.State == Done {
			out = append(out, AwardKey(v.ID, "", diff))
		}
	}
	return out
}

// CanStart says whether a Start tap may be accepted for the chunk now: only a Ready chunk (a locked,
// out-of-order, available or done one is refused with the reason).
func CanStart(views []QuestView, quest, chunk string) error {
	for _, v := range views {
		if v.ID != quest {
			continue
		}
		for _, c := range v.Chunks {
			if c.ID != chunk {
				continue
			}
			if c.State != Ready {
				return fmt.Errorf("questboard: %s/%s is %s, not ready%s", quest, chunk, c.State, because(c.Why))
			}
			return nil
		}
		return fmt.Errorf("questboard: quest %q has no chunk %q", quest, chunk)
	}
	return fmt.Errorf("questboard: unknown quest %q", quest)
}

func because(why string) string {
	if why == "" {
		return ""
	}
	return " (" + why + ")"
}

// Transition is one row of the state machine: entering To from From needs evidence of at least Need on
// Source. It documents what Derive enforces; the tests drive every row.
type Transition struct {
	From, To State
	Source   string
	Need     Tier
}

// Transitions is the table of design 1.3 and 1.4. From "" means any state.
var Transitions = []Transition{
	{"", Locked, "start or a new difficulty: a prerequisite is not Done on T1/T2", TierNone},
	{Locked, Available, "prerequisite quests Done", TierSave},
	{Locked, Ready, "prerequisite quests Done, readiness safe or risky", TierSave},
	{Available, Ready, "readiness word (model, no quest bit)", TierNone},
	{Ready, Available, "readiness word fell", TierNone},
	{Ready, Accepted, "the player's Start tap (plan record)", TierNone},
	{Available, Accepted, "the player's Start tap (plan record)", TierNone},
	{Accepted, InProgress, "the executor runs the chunk", TierNone},
	{InProgress, Awaiting, "objective met (T3 enough to stop) or pending bit on T1/T2", TierProxy},
	{Accepted, Awaiting, "restart with the objective already met", TierProxy},
	{Awaiting, Done, "quest word Done; T3 shows done as confirming", TierProxy},
	{InProgress, Blocked, "the executor gave up (no quest evidence needed)", TierNone},
	{"", Blocked, "the game failed the quest (Rescue on Mount Arreat, U)", TierProxy},
	{"", Skipped, "the player's Skip or the difficulty file", TierNone},
	{"", Done, "quest word Done in the save or memory, board never ran it (quiet, no award)", TierSave},
}
