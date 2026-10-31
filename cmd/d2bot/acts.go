package main

import (
	"context"
	"errors"

	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/quest"
	"d2lod-bot/internal/task"
	"d2lod-bot/internal/town"
)

// Act 2 to 5 pieces of the task adapter (docs/act2.md, docs/act3.md). Classes are V (monstats hcIdx, quest/act2.go and
// act3.go); every dialog row below is U until a live run opened the menu (docs/act2.md C2, docs/act3.md A3-4). A wrong
// row shows up as a failed step (needs_attention), never as a guessed click.

// Names of NPCs the task nodes do not name themselves (internal/task has no constants for them yet).
const (
	npcFara     = "Fara"
	npcLysander = "Lysander"
	npcAsheara  = "Asheara"
	// Act 4 and 5 NPCs without a task constant (shops and Nihlathak's town self are not quest steps yet).
	npcJamella   = "Jamella"
	npcHalbu     = "Halbu"
	npcNihlathak = "Nihlathak"
)

// npcByAct is the class of an NPC per act (1-based); index 0 is "any act". The same name is a different unit in each
// act for Warriv, Meshif and Cain.
var npcByAct = map[string]map[int]int{
	task.NPCAkara:    {0: quest.Akara},
	task.NPCKashya:   {0: quest.Kashya},
	task.NPCCharsi:   {0: quest.Charsi},
	task.NPCWarriv:   {1: quest.Warriv, 2: quest.WarrivA2},
	task.NPCCain:     {1: quest.CainTown, 2: quest.CainA2, 3: quest.CainA3, 4: quest.CainA4, 5: quest.CainA5}, // A4 246 vs 265 is U (docs/act4.md); Act 2 Cain's class V live (docs/act2.md A2), Act 1 and 3 U
	task.NPCAtma:     {0: quest.Atma},
	task.NPCDrognan:  {0: quest.Drognan},
	task.NPCJerhyn:   {0: quest.Jerhyn},
	task.NPCMeshif:   {2: quest.Meshif, 3: quest.MeshifA3},
	task.NPCTyrael:   {0: quest.TyraelTomb, 4: quest.Tyrael, 5: quest.TyraelA5}, // A4 367 is the one that matters (Izual, Diablo, the hop)
	task.NPCGreiz:    {0: quest.Greiz},
	npcFara:          {0: quest.Fara},
	npcLysander:      {0: quest.Lysander},
	task.NPCHratli:   {0: quest.Hratli},
	task.NPCAlkor:    {0: quest.Alkor},
	task.NPCOrmus:    {0: quest.Ormus},
	npcAsheara:       {0: quest.Asheara},
	npcJamella:       {0: quest.Jamella},
	npcHalbu:         {0: quest.Halbu},
	task.NPCLarzuk:   {0: quest.Larzuk},
	task.NPCMalah:    {0: quest.Malah},
	task.NPCAnya:     {0: quest.AnyaNPC},
	task.NPCQualKehk: {0: quest.QualKehk},
	npcNihlathak:     {0: quest.NihlathakNPC},
}

// npcClassIn is the class of the NPC in the given act (0 = unknown, any act).
func npcClassIn(npc string, act int) (int, bool) {
	m, ok := npcByAct[npc]
	if !ok {
		return 0, false
	}
	if c, ok := m[act]; ok {
		return c, true
	}
	c, ok := m[0]
	return c, ok
}

// actTalkPaths are the (npc, pick) rows of Acts 2 and 3; init in task.go merges them into talkPaths. Atma's Talk plays
// the introduction at once (V live), the rest follow the Talk / Trade / Cancel shape and open a submenu whose
// Introduction row speaks the quest line (U). Meshif's "Sail East" row only exists once act2_done is set (quest.Menus).
var actTalkPaths = map[[2]string][]string{
	{task.NPCAtma, "quest"}:           {"Talk"},
	{task.NPCAtma, "reward"}:          {"Talk"},
	{task.NPCCain, "horadric scroll"}: {"Talk"},
	{task.NPCDrognan, "arcane"}:       {"Talk"},
	{task.NPCJerhyn, "arcane"}:        {"Talk"},
	{task.NPCJerhyn, "act2"}:          {"Talk"},
	{task.NPCTyrael, "reward"}:        {"Talk"},
	{task.NPCMeshif, "sail east"}:     {"Sail East"},
	{task.NPCMeshif, "golden bird"}:   {"Talk"},
	{task.NPCAlkor, "golden bird"}:    {"Talk"},
	{task.NPCAlkor, "lam esen"}:       {"Talk"},
	{task.NPCHratli, "blade"}:         {"Talk"},
	// Acts 4 and 5 (all U, nothing was opened live): the quest talks are the plain Talk line, as for Act 1 to 3 rewards.
	// Anya's personalize pick is an item choice that no row path expresses (still only the Talk); Larzuk's socket pick
	// is town.Socket, called from dialogAdapter.Talk before this table is read.
	{task.NPCTyrael, "fallen angel"}:      {"Talk"},
	{task.NPCTyrael, "diablo"}:            {"Talk"},
	{task.NPCTyrael, "harrogath"}:         {"Travel to Harrogath"}, // main-level row, shown once terrors_end is done (quest.Menus)
	{task.NPCLarzuk, "sockets"}:           {"Talk"},
	{task.NPCQualKehk, "rescue"}:          {"Talk"},
	{task.NPCQualKehk, "rite of passage"}: {"Talk"},
	{task.NPCMalah, "potion"}:             {"Talk"},
	{task.NPCMalah, "scroll"}:             {"Talk"},
	{task.NPCAnya, "personalize"}:         {"Talk"},
}

// mercRows is the hire-window row per act (1-based) for the wanted aura, filled after the live check that reads the
// window (docs/act2.md M1, docs/act3.md A3-4). Empty: the hire stops with a prereq fault and the step is routed around,
// because the aura of a row is only visible on screen and a guessed row costs gold.
var mercRows = map[int]int{}

// mercHirer is bot.TaskDeps.HireMerc over town.Hire.
func mercHirer(tw *town.Town, e *env.Env) func(ctx context.Context, act int, aura string) error {
	return func(ctx context.Context, act int, aura string) error {
		// Soft rule (plan-v2 10.1): never pay Kashya. The Rogue comes free after Blood Raven through the claim step
		// (task.MercClaimer); an Act 1 hire here would spend the gold the kit needs. No task step asks for it today.
		if act == 1 {
			return task.Errorf(task.KindPrereq, "merc: no paid Kashya hire; the Rogue is claimed free after Blood Raven")
		}
		s, err := e.Snapshot()
		if err != nil {
			return err
		}
		w := town.MercWant{Act: act, Diff: data.Difficulty(s.Player.Difficulty), Aura: aura, Row: -1}
		if r, ok := mercRows[act]; ok {
			w.Row = r
		}
		rep, err := tw.Hire(ctx, w)
		switch {
		case errors.Is(err, town.ErrMercRowUnknown), errors.Is(err, town.ErrMercUnverified), errors.Is(err, town.ErrNoCandidate), errors.Is(err, town.ErrNoMercNPC):
			return task.Wrap(task.KindPrereq, err)
		case err != nil:
			return err
		case !rep.Hired:
			return task.Errorf(task.KindResource, "mercenary not hired: %s", rep.Skipped)
		}
		return nil
	}
}
