// Package talk is the bot's town NPC dialog layer: which menu row is clicked for which (NPC, pick), the class of an NPC
// per act, the row visibility of a menu (Akara's respec line), the sequence of one talk (open, click the path, wait out
// the speech, close, check the effect) and the Akara reward retry. It moved out of cmd/d2bot (package main) unchanged so
// that internal/drill can run the bot's own code path (docs/fixtalk.md); the game I/O stays behind Services and env.Env.
package talk

import (
	"d2lod-bot/internal/quest"
	"d2lod-bot/internal/task"
)

// Names of NPCs the task nodes do not name themselves (internal/task has no constants for them yet).
const (
	NPCFara     = "Fara"
	NPCLysander = "Lysander"
	NPCAsheara  = "Asheara"
	// Act 4 and 5 NPCs without a task constant (shops and Nihlathak's town self are not quest steps yet).
	NPCJamella   = "Jamella"
	NPCHalbu     = "Halbu"
	NPCNihlathak = "Nihlathak"
)

// ByAct is the class of an NPC per act (1-based); index 0 is "any act". The same name is a different unit in each
// act for Warriv, Meshif and Cain.
var ByAct = map[string]map[int]int{
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
	NPCFara:          {0: quest.Fara},
	NPCLysander:      {0: quest.Lysander},
	task.NPCHratli:   {0: quest.Hratli},
	task.NPCAlkor:    {0: quest.Alkor},
	task.NPCOrmus:    {0: quest.Ormus},
	NPCAsheara:       {0: quest.Asheara},
	NPCJamella:       {0: quest.Jamella},
	NPCHalbu:         {0: quest.Halbu},
	task.NPCLarzuk:   {0: quest.Larzuk},
	task.NPCMalah:    {0: quest.Malah},
	task.NPCAnya:     {0: quest.AnyaNPC},
	task.NPCQualKehk: {0: quest.QualKehk},
	NPCNihlathak:     {0: quest.NihlathakNPC},
}

// ClassIn is the class of the NPC in the given act (0 = unknown, any act).
func ClassIn(npc string, act int) (int, bool) {
	m, ok := ByAct[npc]
	if !ok {
		return 0, false
	}
	if c, ok := m[act]; ok {
		return c, true
	}
	c, ok := m[0]
	return c, ok
}

// actTalkPaths are the (npc, pick) rows of Acts 2 and 3; init below merges them into Paths. Atma's Talk plays
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
	// is town.Socket, called from Adapter.Talk before this table is read.
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

// Paths maps (npc, pick) to the row labels to click.
var Paths = map[[2]string][]string{
	{task.NPCAkara, "quest"}: {"Talk", "Den of Evil"},
	// The reward is paid by the first Talk while it is pending and there is no topic list then (V live, i4 2026-10-28,
	// docs/quest.md "Akara dialog by quest state"): clicking a second row only hit the neighbouring lines.
	{task.NPCAkara, "reward"}:  {"Talk"},
	{task.NPCKashya, "reward"}: {"Talk"},
	// The quest giver visit (docs/quest-givers.md): the first Talk with Kashya is the plain line (U live, same shape as her reward Talk).
	{task.NPCKashya, "quest"}: {"Talk"},
	{task.NPCCharsi, "quest"}: {"Talk"},
	// Akara reads the Scroll of Inifuss on the plain Talk with it in the pack (S, docs/quest.md check 4); Charsi takes the
	// Malus the same way (S, check 3). Both are U live. The imbue item pick follows in Talk (town.Imbue).
	{task.NPCAkara, "bark"}:        {"Talk"},
	{task.NPCCharsi, "imbue:best"}: {"Talk"},
	{task.NPCCharsi, "imbue"}:      {"Talk"},
	{task.NPCWarriv, "go east"}:    {"Go East"},
}

func init() {
	for k, v := range actTalkPaths {
		Paths[k] = v
	}
}
