package task

import "time"

// Normal Act 1 quest handlers (docs/goals/execution.md section 3, planner.md). A handler is data plus a few lines:
// it returns the nodes of one quest. Tags: area ids are V per the repo docs (levels.txt not re-read here), quest
// stage proxies and talk option names are U until the dialog layer and the quest reader are verified live.

// Quest returns the state of q: the reader's value (EvMemory/EvSave) or, when the reader knows less, a stage derived
// from this game's observations (EvProxy). A proxy can say Cleared; it never says Done, except where the world
// itself proves it (Act 2 reached, a quest with no reward).
func (w World) Quest(q string) QuestState {
	s := w.Quests[q]
	p := proxyStage(w, q)
	if p.Stage > s.Stage {
		return p
	}
	return s
}

func proxyStage(w World, q string) QuestState {
	st := func(s Stage) QuestState { return QuestState{Stage: s, Ev: EvProxy} }
	switch q {
	case QDen:
		if w.Cleared[DenOfEvil] {
			return st(StageCleared)
		}
	case QBurial:
		if w.Killed[BossBR] {
			return st(StageCleared)
		}
	case QTools:
		if w.Items[ItemMalus] > 0 {
			return st(StageCleared)
		}
		if w.Killed[BossSmith] {
			return st(StageStarted)
		}
	case QTower:
		if w.Killed[BossCount] {
			return QuestState{Stage: StageDone, Ev: EvProxy}
		}
	case QAndariel:
		if w.Act >= 2 {
			return QuestState{Stage: StageDone, Ev: EvMemory}
		}
		if w.Killed[BossAndy] {
			return st(StageCleared)
		}
	case QAct1Done:
		if w.Act >= 2 {
			return QuestState{Stage: StageDone, Ev: EvMemory}
		}
	case QCain:
		if w.Items[ItemCainSc] > 0 {
			return st(StageStarted)
		}
	}
	return QuestState{}
}

// QuestDone reports whether q needs no more work.
func QuestDone(w World, q string) bool { return w.Quest(q).Stage >= StageDone }

func stageIs(q string, s Stage) Pred { return func(w World) bool { return w.Quest(q).Stage == s } }
func stageAtLeast(q string, s Stage) Pred {
	return func(w World) bool { return w.Quest(q).Stage >= s }
}
func not(p Pred) Pred         { return func(w World) bool { return !p(w) } }
func levelAtLeast(n int) Pred { return func(w World) bool { return w.Level >= n } }

// turnIn runs while the objective is met and the reward is not claimed.
func turnIn(q string) Pred { return stageIs(q, StageCleared) }

// needStarted reports whether talking to the quest giver has not been done yet.
func notStarted(q string) Pred { return stageIs(q, StageNone) }

// optional marks a step skippable: without a quest reader its Need cannot tell "not done" from "done earlier".
func optional(s Step) Step { s.Optional = true; return s }

// withReady adds r to the step's own Ready (the field gate of StepTravel, StepClear and StepKill runs first).
func withReady(s Step, r func(World) *Fault) Step {
	base := s.Ready
	if base == nil {
		s.Ready = r
		return s
	}
	s.Ready = func(w World) *Fault {
		if f := base(w); f != nil {
			return f
		}
		return r(w)
	}
	return s
}

// gatePotions is gate plus a potion floor that inserts a chores trip, so a character that escaped a boss fight to
// town (boss.ErrEscaped) restocks before the walk back. The floors match boss.Script.MinPotions.
func gatePotions(minLvl int, why string, minPots int, q string) func(World) *Fault {
	lvl := gate(minLvl, why)
	return func(w World) *Fault {
		if f := lvl(w); f != nil {
			return f
		}
		if w.Potions < minPots {
			return &Fault{Kind: KindResource, Why: "not enough potions", Insert: []Step{StepChores(id(q, "chores"))}}
		}
		return nil
	}
}

func gate(min int, why string) func(World) *Fault {
	return func(w World) *Fault {
		if w.Level < min {
			return &Fault{Kind: KindDanger, Why: why}
		}
		return nil
	}
}

func req(n Node, ids ...string) Node { n.Requires = ids; return n }
func nodes(s ...Step) []Node {
	out := make([]Node, len(s))
	for i, x := range s {
		out[i] = Node{Step: x}
	}
	return out
}

// handlers maps a quest id to its nodes.
var handlers = map[string]func() []Node{
	QDen:      denNodes,
	QBurial:   burialNodes,
	QCain:     cainNodes,
	QTools:    toolsNodes,
	QTower:    towerNodes,
	QAndariel: andarielNodes,
	QAct1Done: hopNodes,
}

// Handler returns the nodes of quest q (nil when unknown).
func Handler(q string) []Node {
	if h := handlers[q]; h != nil {
		return h()
	}
	return nil
}

func id(q, step string) string { return "quest." + q + "/" + step }

// Den of Evil: Akara, clear area 8 (every monster, V), back to Akara. Reward: 1 skill point (Leveler spends it).
func denNodes() []Node {
	q := QDen
	done := stageAtLeast(q, StageCleared)
	talk := StepTalk(id(q, "talk-akara"), "Talk to Akara", NPCAkara, "quest", not(notStarted(q)), nil)
	talk.Optional = true // U: the quest may count without the greeting
	// Claim probe (soak 2026-10-29: Corpsefire died twice, den_of_evil stayed 0x001c, no Akara visit was ever tried). It sits
	// before the clear: it only becomes needed after a sweep ended with the word still open, and it goes first so the planner
	// does not send the character back into an empty Den. Optional, so a probe that changes nothing is skipped after the
	// executor's failure budget instead of looping.
	probe := StepTalk(id(q, "claim-probe"), "Ask Akara about the Den", NPCAkara, "reward",
		func(w World) bool { return !w.Swept[DenOfEvil] || w.Quest(q).Stage != StageStarted }, nil)
	probe.Optional = true
	return []Node{
		{Step: talk},
		{Step: probe},
		{Step: StepWaypoint(id(q, "wp-cold-plains"), ColdPlains, done)},
		{Step: StepTravel(id(q, "to-den"), "Go to the Den of Evil", DenOfEvil, done)},
		req(Node{Step: StepClear(id(q, "clear"), "Clear the Den of Evil", DenOfEvil, done, 15*time.Minute)}, id(q, "to-den")),
		{Step: StepTravel(id(q, "home"), "Return to camp", RogueEncampment, not(turnIn(q)))},
		{Step: StepTalk(id(q, "reward"), "Claim the Den reward", NPCAkara, "reward", not(turnIn(q)), nil)},
	}
}

// Sisters' Burial Grounds: kill Blood Raven, Kashya reward (rogue hire is a separate goal).
func burialNodes() []Node {
	q := QBurial
	done := stageAtLeast(q, StageCleared)
	return []Node{
		{Step: StepWaypoint(id(q, "wp-cold-plains"), ColdPlains, done)},
		{Step: withReady(StepTravel(id(q, "to-burial"), "Go to the Burial Grounds", BurialGrounds, done), gateReady(q))},
		req(Node{Step: StepKill(id(q, "kill"), BossBR, BurialGrounds, done, 8*time.Minute)}, id(q, "to-burial")),
		{Step: StepTravel(id(q, "home"), "Return to camp", RogueEncampment, not(turnIn(q)))},
		{Step: StepTalk(id(q, "reward"), "Talk to Kashya", NPCKashya, "reward", not(turnIn(q)), nil)},
	}
}

// Search for Cain (optional, off by default): bark from the Dark Wood tree, Akara reads it, cairn stones in Stony
// Field open the Tristram portal, free Cain from the cage. Stone order is random per game (U): the adapter brute
// forces it inside Interact("cairn-stones").
func cainNodes() []Node {
	q := QCain
	inTristram := func(w World) bool { return w.Area == Tristram }
	opened := func(w World) bool { return w.Items[ItemPortal] > 0 || inTristram(w) } // portal seen (EvProxy)
	free := stageAtLeast(q, StageDone)
	return []Node{
		{Step: StepWaypoint(id(q, "wp-dark-wood"), DarkWood, free)},
		{Step: StepTravel(id(q, "to-wood"), "Go to the Dark Wood", DarkWood, func(w World) bool { return w.Items[ItemBark] > 0 || w.Items[ItemCainSc] > 0 || free(w) })},
		req(Node{Step: StepInteract(id(q, "tree"), "Take the Inifuss bark", "Tree of Inifuss",
			func(w World) bool { return w.Items[ItemBark] > 0 || w.Items[ItemCainSc] > 0 || free(w) }, func(w World) bool { return w.Area == DarkWood })}, id(q, "to-wood")),
		{Step: StepTalk(id(q, "read"), "Give the bark to Akara", NPCAkara, "bark",
			func(w World) bool { return w.Items[ItemBark] == 0 || free(w) }, nil)},
		{Step: StepTravel(id(q, "to-stony"), "Go to Stony Field", StonyField, func(w World) bool { return opened(w) || free(w) })},
		req(Node{Step: StepInteract(id(q, "stones"), "Touch the cairn stones", "cairn-stones",
			func(w World) bool { return opened(w) || free(w) }, func(w World) bool { return w.Area == StonyField })}, id(q, "to-stony")),
		{Step: StepTravel(id(q, "to-tristram"), "Enter Tristram", Tristram, func(w World) bool { return inTristram(w) || free(w) })},
		{Step: StepClear(id(q, "clear"), "Clear Tristram", Tristram, func(w World) bool { return w.Cleared[Tristram] || free(w) }, 12*time.Minute)},
		{Step: StepInteract(id(q, "cage"), "Free Cain", "Cain's Cage", free, inTristram)},
		{Step: StepTravel(id(q, "home"), "Return to camp", RogueEncampment, func(w World) bool { return !free(w) && w.Area != Tristram || w.InTown })},
	}
}

// Tools of the Trade: needs clvl 8 (V). Smith in the Barracks drops the Horadric Malus; Charsi imbues one item
// (which item is U: the imbue pick is "best" until the item chooser exists).
func toolsNodes() []Node {
	q := QTools
	cleared := stageAtLeast(q, StageCleared)
	lvl := gateReady(q)
	charsi := StepTalk(id(q, "talk-charsi"), "Talk to Charsi", NPCCharsi, "quest", not(notStarted(q)), nil)
	charsi.Optional = true // U: the Malus may drop without the greeting
	charsi = withReady(charsi, gatedReady)
	return []Node{
		{Step: charsi},
		{Step: StepWaypoint(id(q, "wp-cold-plains"), ColdPlains, cleared)},
		{Step: withReady(StepTravel(id(q, "to-barracks"), "Go to the Barracks", Barracks, cleared), lvl)},
		req(Node{Step: StepKill(id(q, "kill"), BossSmith, Barracks, func(w World) bool { return w.Killed[BossSmith] || cleared(w) }, 8*time.Minute)}, id(q, "to-barracks")),
		{Step: optional(StepPickUp(id(q, "malus"), ItemMalus, stageAtLeast(q, StageDone), func(w World) bool { return w.Killed[BossSmith] }))},
		{Step: StepTravel(id(q, "home"), "Return to camp", RogueEncampment, not(turnIn(q)))},
		{Step: StepTalk(id(q, "reward"), "Give the Malus to Charsi", NPCCharsi, "imbue:best", not(turnIn(q)), nil)},
	}
}

// Forgotten Tower: Countess in Tower Cellar 5. No turn-in; the kill completes the quest (V). Runes drop here.
func towerNodes() []Node {
	q := QTower
	done := stageAtLeast(q, StageDone)
	lvl := gateReady(q)
	return []Node{
		{Step: StepWaypoint(id(q, "wp-black-marsh"), BlackMarsh, done)},
		{Step: withReady(StepTravel(id(q, "to-cellar"), "Go to the Tower Cellar", TowerCellar5, done), lvl)},
		req(Node{Step: StepKill(id(q, "kill"), BossCount, TowerCellar5, func(w World) bool { return w.Killed[BossCount] || done(w) }, 10*time.Minute)}, id(q, "to-cellar")),
		{Step: StepTravel(id(q, "home"), "Return to camp", RogueEncampment, func(w World) bool { return w.InTown || !done(w) })},
	}
}

// Sisters to the Slaughter: Andariel in Catacombs 4 (clvl 22, runes.json, U). Needs potions and a TP; poison in
// Normal is handled by Fighter (antidotes are part of Chores). Ready inserts a chore trip when short.
func andarielNodes() []Node {
	q := QAndariel
	killed := stageAtLeast(q, StageCleared)
	ready := gateReady(q)
	return []Node{
		{Step: StepWaypoint(id(q, "wp-cold-plains"), ColdPlains, killed)},
		{Step: withReady(StepTravel(id(q, "to-catacombs"), "Go to Catacombs level 4", Catacombs4, killed), ready)},
		req(Node{Step: StepKill(id(q, "kill"), BossAndy, Catacombs4, killed, 12*time.Minute)}, id(q, "to-catacombs")),
		{Step: StepTravel(id(q, "home"), "Return to camp", RogueEncampment, func(w World) bool { return w.InTown || !killed(w) })},
	}
}

// Act 1 complete: Warriv "go east" (U option name). Done is the world seeing act 2 (a memory read, EvMemory).
func hopNodes() []Node {
	q := QAct1Done
	done := func(w World) bool { return w.Act >= 2 }
	return nodes(
		StepTalk(id(q, "warriv"), "Sail east with Warriv", NPCWarriv, "go east", done, func(w World) bool { return QuestDone(w, QAndariel) || w.Quest(QAndariel).Stage >= StageCleared }),
	)
}
