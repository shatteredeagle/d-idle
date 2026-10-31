package task

// Act3Options tunes the Normal Act 3 template.
type Act3Options struct {
	Merc       bool // hire Asheara's Iron Wolf first
	LamEsen    bool // +5 stat points (needs a quest reader to avoid refetching)
	Blade      bool // Blade of the Old Religion (rare ring); flow U
	GoldenBird bool // opportunistic: only hands in a figurine already in the pack
}

// NormalAct3 is the Normal Act 3 campaign: the three relic legs, the Council (clvl 24, runs.json), Khalim's Will and the
// orb, Mephisto (clvl 28, runs.json) and the red portal. The floors of the relic legs and optionals are U.
func NormalAct3(o Act3Options) *Campaign {
	c := &Campaign{Name: "normal_act3"}
	add := func(q, title string, min int) {
		c.Entries = append(c.Entries, Entry{Quest: q, Title: title, MinLevel: min, Act: 3})
	}
	if o.Merc {
		add(QMercAct3, "Hire the Iron Wolf", 1)
	}
	if o.GoldenBird {
		add(QGolden, "The Golden Bird", 1)
	}
	if o.Blade {
		add(QBlade, "Blade of the Old Religion", 22)
	}
	if o.LamEsen {
		add(QLamEsen, "Lam Esen's Tome", 22)
	}
	add(QEye, "Khalim's Eye", 20)
	add(QBrain, "Khalim's Brain", 22)
	add(QHeart, "Khalim's Heart", 22)
	add(QCouncil, "The High Council", 24)
	add(QKhalim, "Khalim's Will and the Compelling Orb", 24)
	add(QGuardian, "The Guardian (Mephisto)", 28)
	add(QAct3Done, "Travel to Act 4", 28)
	return c
}

// NormalActs123 is Acts 1 to 3 in one campaign.
func NormalActs123(a1 Act1Options, a2 Act2Options, a3 Act3Options) *Campaign {
	c := NormalActs12(a1, a2)
	c.Name = "normal_act1_3"
	c.Entries = append(c.Entries, NormalAct3(a3).Entries...)
	return c
}
