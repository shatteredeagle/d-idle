package quest

// Dialog rows of the Act 3 NPCs. All U: they follow the Act 1/2 shape (Talk / Trade / Cancel); the real lists may
// carry quest rows that change the count, so docs/act3.md live check A3-4 must record them before a handler picks a
// row by position. Menu tags are U everywhere.
func init() {
	talk := []Row{{"Introduction", "U", "", nil, 0}, {"Gossip", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}
	for _, npc := range []int{Hratli, Alkor, Ormus, CainA3, Asheara} {
		Menus = append(Menus, Menu{npc, LevelTalk, talk})
	}
	for _, npc := range []int{Hratli, Alkor, Ormus} {
		Menus = append(Menus, Menu{npc, LevelMain, []Row{{"Talk", "U", LevelTalk, nil, 0}, {"Trade", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}})
	}
	Menus = append(Menus,
		Menu{CainA3, LevelMain, []Row{{"Talk", "U", LevelTalk, nil, 0}, {"Identify", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},
		Menu{Asheara, LevelMain, []Row{{"Talk", "U", LevelTalk, nil, 0}, {"Hire", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},
	)
}
