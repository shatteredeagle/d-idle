package quest

// Dialog rows of the Act 2 NPCs, appended to Menus at start-up (dialog.go is not edited). Only Greiz's three
// lines are V (screenshot on i2, town/merc.go). Everything else follows the Act 1 shape and is U: the real list may
// carry quest topic rows (as Akara's does) that change the line count, which is why each NPC's table is checked
// in docs/act2.md live check C2 before a handler picks a row by position.
func init() {
	std := func(npc int, tag string) []Menu {
		return []Menu{
			{npc, LevelMain, []Row{{"Talk", tag, LevelTalk, nil, 0}, {"Trade", tag, "", nil, 0}, {"Cancel", tag, "", nil, 0}}},
			{npc, LevelTalk, []Row{{"Introduction", tag, "", nil, 0}, {"Gossip", tag, "", nil, 0}, {"Cancel", tag, "", nil, 0}}},
		}
	}
	for _, npc := range []int{Drognan, Fara, Elzix, Lysander, Geglash} {
		Menus = append(Menus, std(npc, "U")...)
	}
	Menus = append(Menus,
		// V live (Navi on i1, 2026-10-29, before the Radament quest was spoken): Atma = Talk / Cancel, no Trade row, and
		// Talk plays the Radament introduction straight away (no submenu; the quest word 9 reads 0x0004 afterwards).
		Menu{Atma, LevelMain, []Row{{"Talk", "V", "", nil, 0}, {"Cancel", "V", "", nil, 0}}},
		// V live (Palli, i2): Greiz = Talk / Hire / Cancel. The Talk submenu is U.
		Menu{Greiz, LevelMain, []Row{{"Talk", "V", LevelTalk, nil, 0}, {"Hire", "V", "", nil, 0}, {"Cancel", "V", "", nil, 0}}},
		Menu{Greiz, LevelTalk, []Row{{"Introduction", "U", "", nil, 0}, {"Gossip", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},
		// Warriv in Lut Gholein: the way back west (U row text "Go West", always offered).
		Menu{WarrivA2, LevelMain, []Row{{"Talk", "U", LevelTalk, nil, 0}, {"Go West", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},
		Menu{WarrivA2, LevelTalk, []Row{{"Introduction", "U", "", nil, 0}, {"Gossip", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},
		// Cain after the rescue (class U) and Tyrael in Tal Rasha's Chamber: Talk only.
		Menu{CainA2, LevelMain, []Row{{"Talk", "U", LevelTalk, nil, 0}, {"Identify", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},
		Menu{CainA2, LevelTalk, []Row{{"Introduction", "U", "", nil, 0}, {"Gossip", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},
		Menu{TyraelTomb, LevelMain, []Row{{"Talk", "U", LevelTalk, nil, 0}, {"Cancel", "U", "", nil, 0}}},
		Menu{TyraelTomb, LevelTalk, []Row{{"Introduction", "U", "", nil, 0}, {"Gossip", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},
		// Meshif at the A3 docks: the way back west (U).
		Menu{MeshifA3, LevelMain, []Row{{"Talk", "U", LevelTalk, nil, 0}, {"Sail West", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},
		Menu{MeshifA3, LevelTalk, []Row{{"Introduction", "U", "", nil, 0}, {"Gossip", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},
	)
}
