package task

// Act2Options tunes the Normal Act 2 template.
type Act2Options struct {
	Merc     bool // hire Greiz's Desert Mercenary first
	Arcane   bool // the Arcane Sanctuary and Summoner pair (optional, needs Teleport to be quick)
	SkipCube bool // only for a character that already has the Cube (the item proxy then completes the entry anyway)
}

// NormalAct2 is the Normal Act 2 campaign: mercenary, Radament (clvl 16, his monster level), the three fetch legs and
// the Staff (clvl 18..20), the optional Summoner pair (24) and Duriel (24), then the Meshif hop. Floors are catalog.md /
// runs.json plus the Normal monster levels in the tables (V: Radament 16, Summoner 18, Duriel 22), otherwise U.
func NormalAct2(o Act2Options) *Campaign {
	c := &Campaign{Name: "normal_act2"}
	add := func(q, title string, min int) {
		c.Entries = append(c.Entries, Entry{Quest: q, Title: title, MinLevel: min, Act: 2})
	}
	if o.Merc {
		add(QMercAct2, "Hire the Desert Mercenary", 1)
	}
	add(QRadament, "Radament's Lair", 16)
	if !o.SkipCube {
		add(QCube, "The Horadric Cube", 16)
	}
	add(QViper, "The Tainted Sun (Amulet of the Viper)", 18)
	add(QShaft, "Staff of Kings", 18)
	add(QStaff, "The Horadric Staff", 20)
	if o.Arcane {
		add(QArcane, "The Arcane Sanctuary", 24)
		add(QSummoner, "The Summoner", 24)
	}
	add(QDuriel, "The Seven Tombs (Duriel)", 24)
	add(QAct2Done, "Travel to Act 3", 24)
	return c
}

// NormalActs12 is Act 1 followed by Act 2 in one campaign (the planner picks the first unsatisfied entry, so the order
// is the dependency order).
func NormalActs12(a1 Act1Options, a2 Act2Options) *Campaign {
	c := NormalAct1(a1)
	c.Name = "normal_act1_2"
	c.Entries = append(c.Entries, NormalAct2(a2).Entries...)
	return c
}

// defaultOptions are the bot's defaults per act. Act 4 and 5 skip the optionals: the Hellforge needs the Soulstone and
// a verified forge procedure (U), Rescue can fail, Prison and Betrayal need Anya's flow (U), and the Barbarian costs
// 20000+ gold with an unknown hire row.
func defaultOptions(cain bool) (Act1Options, Act2Options, Act3Options, Act4Options, Act5Options) {
	return Act1Options{Cain: cain}, Act2Options{Merc: true}, Act3Options{Merc: true}, Act4Options{}, Act5Options{}
}

// Template builds the campaign of a goals template name (nil for an unknown one or an act without nodes yet). The
// options are defaultOptions: the Desert Mercenary and the Iron Wolf are hired, the optionals are skipped.
func Template(name string, cain bool) *Campaign {
	a1, a2, a3, a4, a5 := defaultOptions(cain)
	switch name {
	case "", "normal_act1":
		return NormalAct1(a1)
	case "normal_act2":
		return NormalAct2(a2)
	case "normal_act3":
		return NormalAct3(a3)
	case "normal_act4":
		return NormalAct4(a4)
	case "normal_act5":
		return NormalAct5(a5)
	case "nightmare_all":
		return NightmareAll(cain)
	case "hell_all":
		return HellAll(cain)
	case "normal_all":
		c := NormalActs12345(a1, a2, a3, a4, a5)
		c.Name = "normal_all"
		return c
	}
	return nil
}
