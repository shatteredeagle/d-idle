package task

// Act5Options tunes the Normal Act 5 template. The zero value is the critical path with Larzuk's sockets: Shenk,
// the Ancients, Baal.
type Act5Options struct {
	Merc      bool // hire Qual-Kehk's barbarian first (expensive, U)
	Rescue    bool // Rescue on Mount Arreat (optional, can fail)
	Prison    bool // Prison of Ice: Scroll of Resistance (+10 all resists), opens Anya's portal
	Nihlathak bool // Betrayal of Harrogath; implies Prison (the portal is the only way to the Halls)
	NoSiege   bool // skip Shenk and Larzuk's sockets
}

// NormalAct5 is the Normal Act 5 campaign: Siege (clvl 25), optional Rescue and Prison of Ice, optional Nihlathak (30),
// the Ancients (28), Baal (32). Floors are the work list of docs/act5.md (R); see the floor constants in act5.go.
func NormalAct5(o Act5Options) *Campaign {
	c := &Campaign{Name: "normal_act5"}
	add := func(q, title string, min int) {
		c.Entries = append(c.Entries, Entry{Quest: q, Title: title, MinLevel: min, Act: 5})
	}
	if o.Merc {
		add(QMercAct5, "Hire a Barbarian", 1)
	}
	if !o.NoSiege {
		add(QSiege, "Siege on Harrogath (Shenk, sockets)", floorShenk)
	}
	if o.Rescue {
		add(QRescue, "Rescue on Mount Arreat", floorRescue)
	}
	if o.Prison || o.Nihlathak {
		add(QPrison, "Prison of Ice", floorPrison)
	}
	if o.Nihlathak {
		add(QBetrayal, "Betrayal of Harrogath (Nihlathak)", floorNihlathak)
	}
	add(QAncients, "Rite of Passage (the Ancients)", floorAncients)
	add(QBaal, "Eve of Destruction (Baal)", floorBaal)
	return c
}

// NormalActs12345 is the whole Normal difficulty in one campaign.
func NormalActs12345(a1 Act1Options, a2 Act2Options, a3 Act3Options, a4 Act4Options, a5 Act5Options) *Campaign {
	c := NormalActs1234(a1, a2, a3, a4)
	c.Name = "normal_act1_5"
	c.Entries = append(c.Entries, NormalAct5(a5).Entries...)
	return c
}
