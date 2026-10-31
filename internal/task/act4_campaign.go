package task

// Act4Options tunes the Normal Act 4 template.
type Act4Options struct {
	Hellforge bool // optional: needs the Soulstone from Mephisto in the pack (gems and a rune)
	NoIzual   bool // skip the Fallen Angel (2 skill points) and go straight to Diablo
}

// NormalAct4 is the Normal Act 4 campaign: Izual (clvl 30), the optional Hellforge, Diablo (clvl 30) and the Tyrael hop.
// Floors are runs.json / progression-normal (R) against the Normal monster levels 26..28 (V); U for the Hellforge.
func NormalAct4(o Act4Options) *Campaign {
	c := &Campaign{Name: "normal_act4"}
	add := func(q, title string, min int) {
		c.Entries = append(c.Entries, Entry{Quest: q, Title: title, MinLevel: min, Act: 4})
	}
	if !o.NoIzual {
		add(QIzual, "The Fallen Angel (Izual)", 30)
	}
	if o.Hellforge {
		add(QHellforge, "The Hellforge", 30)
	}
	add(QDiablo, "Terror's End (Diablo)", 30)
	add(QAct4Done, "Travel to Act 5", 30)
	return c
}

// NormalActs1234 is Acts 1 to 4 in one campaign.
func NormalActs1234(a1 Act1Options, a2 Act2Options, a3 Act3Options, a4 Act4Options) *Campaign {
	c := NormalActs123(a1, a2, a3)
	c.Name = "normal_act1_4"
	c.Entries = append(c.Entries, NormalAct4(a4).Entries...)
	return c
}
