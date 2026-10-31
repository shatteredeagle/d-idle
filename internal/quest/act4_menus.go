package quest

// Dialog rows of the Act 4 NPCs. All U: they follow the Act 1..3 shape (Talk / Trade / Cancel); docs/act4.md open
// question 4 (which Cain stands here) and a live row dump must settle them before a handler picks a row by position.
// Tyrael (367) is in dialog.go.
func init() {
	talk := []Row{{"Introduction", "U", "", nil, 0}, {"Gossip", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}
	for _, npc := range []int{Jamella, Halbu, CainA4, CainA4b} {
		Menus = append(Menus, Menu{npc, LevelTalk, talk})
	}
	for _, npc := range []int{Jamella, Halbu} {
		Menus = append(Menus, Menu{npc, LevelMain, []Row{{"Talk", "U", LevelTalk, nil, 0}, {"Trade", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}})
	}
	for _, npc := range []int{CainA4, CainA4b} {
		Menus = append(Menus, Menu{npc, LevelMain, []Row{{"Talk", "U", LevelTalk, nil, 0}, {"Identify", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}})
	}
}
