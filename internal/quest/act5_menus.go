package quest

// Dialog rows of the Act 5 NPCs. All U (shape follows Acts 1..4); Larzuk's socket pick and Anya's personalize rows
// are item selections the dialog table cannot express (docs/act5.md A5-6).
func init() {
	talk := []Row{{"Introduction", "U", "", nil, 0}, {"Gossip", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}
	for _, npc := range []int{Larzuk, AnyaNPC, Malah, NihlathakNPC, QualKehk, CainA5, TyraelA5} {
		Menus = append(Menus, Menu{npc, LevelTalk, talk})
	}
	for _, npc := range []int{Larzuk, AnyaNPC, Malah} {
		Menus = append(Menus, Menu{npc, LevelMain, []Row{{"Talk", "U", LevelTalk, nil, 0}, {"Trade", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}})
	}
	Menus = append(Menus,
		Menu{NihlathakNPC, LevelMain, []Row{{"Talk", "U", LevelTalk, nil, 0}, {"Cancel", "U", "", nil, 0}}},
		Menu{QualKehk, LevelMain, []Row{{"Talk", "U", LevelTalk, nil, 0}, {"Hire", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},
		Menu{CainA5, LevelMain, []Row{{"Talk", "U", LevelTalk, nil, 0}, {"Identify", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},
		Menu{TyraelA5, LevelMain, []Row{{"Talk", "U", LevelTalk, nil, 0}, {"Cancel", "U", "", nil, 0}}},
	)
}
