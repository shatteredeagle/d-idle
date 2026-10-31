package task

import (
	"context"
	"time"
)

// Normal Act 2 quest handlers and campaign (docs/act2.md). Same shape as act1.go: a handler returns the nodes of one
// quest. Area ids, item display names and the boss/NPC classes are V against the real game tables (see
// internal/quest/act2.go); every stage proxy, talk option and object name is U until the live checks of docs/act2.md ran.
//
// The registration happens in init() so act1.go (owned by INTEGRATE2) stays untouched. Capabilities the Actions
// interface does not have (the Horadric Cube, the mercenary hire) are optional interfaces below: the bot's Actions
// value implements them or the step fails with a Prereq fault.

// Act 2 areas (V levels.txt).
const (
	Sewers2         Area = 48
	Sewers3         Area = 49
	FarOasis        Area = 43
	LostCity        Area = 44
	CanyonOfMagi    Area = 46
	PalaceCellar3   Area = 54
	HallsOfDead2    Area = 57
	HallsOfDead3    Area = 60
	ClawViper2      Area = 61
	MaggotLair3     Area = 64
	DurielsLair     Area = 73
	ArcaneSanctuary Area = 74
	KurastDocks     Area = 75
)

// Act 2 quest ids (catalog.md "quest.<id>"). The campaign uses them as group ids; the bot's quest reader maps the ones
// with a save word (radament, horadric_staff, arcane_sanctuary, summoner, duriel, act2_complete = QAct2Word) and leaves the
// fetch legs (horadric_cube, viper_amulet, staff_of_kings) to the item proxies.
const (
	QRadament    = "radament"
	QCube        = "horadric_cube"
	QViper       = "viper_amulet"
	QShaft       = "staff_of_kings"
	QStaff       = "horadric_staff"
	QArcane      = "arcane_sanctuary"
	QSummoner    = "summoner"
	QDuriel      = "duriel"
	QAct2Done    = "act2_hop"      // campaign group: Jerhyn and Meshif; done when the world is in act 3
	QAct2Word    = "act2_complete" // save word 15 ("able to go to Act 3"); the quest reader maps it
	QMercAct2    = "merc_act2"
	NPCAtma      = "Atma"
	NPCDrognan   = "Drognan"
	NPCJerhyn    = "Jerhyn"
	NPCMeshif    = "Meshif"
	NPCTyrael    = "Tyrael"
	NPCGreiz     = "Greiz"
	BossRadament = "Radament"
	BossSummoner = "The Summoner"
	BossDuriel   = "Duriel"
	BossFangskin = "Fangskin"
	// Item display names (V items.txt Display column).
	ItemCube        = "Horadric Cube"
	ItemShaft       = "Shaft of the Horadric Staff"
	ItemStaffTop    = "Top of the Horadric Staff"
	ItemStaff       = "Horadric Staff"
	ItemBook        = "Book of Skill"
	ItemScroll      = "Horadric Scroll"
	ItemStaffPlaced = "Horadric Staff placed" // pseudo item: the adapter sets 1 once the orifice took the staff
	ItemBookRead    = "Book of Skill read"    // pseudo item: 1 once the adapter right-clicked the book this game
	ItemMercenary   = "Mercenary"             // pseudo item: 1 while a hired mercenary is alive
)

// Transmuter is the Horadric Cube capability (quest.Cube is the same contract on the quest side). The bot's Actions
// implements it over town/cube.go once that exists.
type Transmuter interface {
	Transmute(ctx context.Context, inputs []string) error
}

// CubeOut is the same capability with the result codes, as town.Town and quest.Cube have it.
type CubeOut interface {
	Transmute(ctx context.Context, inputs []string) ([]string, error)
}

// MercHirer is the hire capability (town.Hirer.Hire with MercWant{Act: act, Diff: Normal, Aura: aura}).
type MercHirer interface {
	HireMerc(ctx context.Context, act int, aura string) error
}

func init() {
	handlers[QRadament] = radamentNodes
	handlers[QCube] = cubeNodes
	handlers[QViper] = viperNodes
	handlers[QShaft] = shaftNodes
	handlers[QStaff] = staffNodes
	handlers[QArcane] = arcaneNodes
	handlers[QSummoner] = summonerNodes
	handlers[QDuriel] = durielNodes
	handlers[QAct2Done] = act2HopNodes
	handlers[QMercAct2] = mercNodes
}

// a2Stage is the stage of an Act 2 quest: the reader's value, raised by what the world itself proves. Fetch legs have
// no reward to claim, so holding the item is Done; a boss kill is Cleared (Done where there is nothing to turn in).
func a2Stage(w World, q string) Stage {
	s := w.Quest(q).Stage
	raise := func(p Stage) {
		if p > s {
			s = p
		}
	}
	switch q {
	case QRadament:
		if w.Killed[BossRadament] {
			raise(StageCleared)
		}
	case QCube:
		if w.Items[ItemCube] > 0 {
			raise(StageDone)
		}
	case QShaft:
		if w.Items[ItemShaft] > 0 || staffMade(w) {
			raise(StageDone)
		}
	case QViper:
		if w.Items[ItemStaffTop] > 0 || staffMade(w) {
			raise(StageDone)
		}
	case QStaff:
		if w.Items[ItemStaff] > 0 {
			raise(StageCleared)
		}
		if staffPlaced(w) {
			raise(StageDone)
		}
	case QArcane:
		if w.Area == ArcaneSanctuary || w.WPs[ArcaneSanctuary] {
			raise(StageDone)
		}
	case QSummoner:
		if w.Killed[BossSummoner] {
			raise(StageCleared)
		}
	case QDuriel:
		if w.Killed[BossDuriel] {
			raise(StageCleared)
		}
		if w.Act >= 3 {
			raise(StageDone)
		}
	case QAct2Done:
		if w.Act >= 3 {
			raise(StageDone)
		}
	}
	return s
}

func staffMade(w World) bool { return w.Items[ItemStaff] > 0 || staffPlaced(w) }

// staffPlaced: the pseudo item is per game (a restart clears it), so the persisted quest word counts too.
func staffPlaced(w World) bool {
	return w.Items[ItemStaffPlaced] > 0 || w.Quest(QStaff).Stage >= StageDone || w.Area == DurielsLair || w.Killed[BossDuriel] || w.Act >= 3
}

func a2At(q string, s Stage) Pred { return func(w World) bool { return a2Stage(w, q) >= s } }
func a2Is(q string, s Stage) Pred { return func(w World) bool { return a2Stage(w, q) == s } }
func a2has(item string) Pred      { return func(w World) bool { return w.Items[item] > 0 } }
func a2in(a Area) Pred            { return func(w World) bool { return w.Area == a } }
func a2or(p, q Pred) Pred         { return func(w World) bool { return p(w) || q(w) } }

// home2 walks back to town (Act 2's town is not the Rogue Encampment, so StepTravel's town case does not apply) when
// there is something to do there. It is needed only while the character still stands in the quest's own area: a Need
// that was true anywhere in the field would make the campaign re-select a finished group whenever the next quest takes
// the character out of town (Campaign.complete asks every step), and the two groups would ping-pong.
func home2(id string, in Area, when Pred) Step {
	s := StepGoHome(id)
	s.Need = func(w World) (bool, Evidence) { return !w.InTown && w.Area == in && when(w), EvMemory }
	return s
}

func a2gate(min int, why string) func(World) *Fault { return gate(min, why) }

// Radament's Lair: Atma, Sewers 3 (waypoint at Sewers 2), kill Radament, read the Book of Skill, open the Horadric Scroll
// chest, Atma again, then Cain reads the scroll (which starts the Staff chain). Radament's monster level is 16 (V).
func radamentNodes() []Node {
	q := QRadament
	killed := a2At(q, StageCleared)
	claimed := a2At(q, StageDone)
	got := a2or(a2has(ItemScroll), a2or(claimed, a2At(QStaff, StageStarted)))
	bookDone := a2or(a2has(ItemBook), a2or(a2has(ItemBookRead), claimed))
	start := StepTalk(id(q, "talk-atma"), "Talk to Atma", NPCAtma, "quest", not(a2Is(q, StageNone)), nil)
	start.Optional = true
	pickBook := StepPickUp(id(q, "book"), ItemBook, bookDone, func(w World) bool { return w.Killed[BossRadament] })
	pickBook.Optional = true // already read after a restart: nothing on the ground
	chest := openAndTake(id(q, "chest"), "Horadric Scroll chest", ItemScroll, got, a2in(Sewers3))
	chest.Optional = true
	cain := StepTalk(id(q, "cain"), "Show the scroll to Cain", NPCCain, "horadric scroll", not(a2has(ItemScroll)), nil)
	cain.Optional = true
	return []Node{
		{Step: start},
		{Step: StepWaypoint(id(q, "wp-sewers2"), Sewers2, killed)},
		{Step: withReady(StepTravelA2(id(q, "to-sewers3"), "Go to Sewers level 3", Sewers3, killed), a2gate(14, "Radament needs clvl 14"))},
		req(Node{Step: StepKill(id(q, "kill"), BossRadament, Sewers3, killed, 8*time.Minute)}, id(q, "to-sewers3")),
		{Step: pickBook},
		{Step: StepInteract(id(q, "read"), "Read the Book of Skill", "item:"+ItemBook, not(a2has(ItemBook)), nil)},
		{Step: chest},
		{Step: home2(id(q, "home"), Sewers3, a2Is(q, StageCleared))},
		{Step: StepTalk(id(q, "reward"), "Talk to Atma", NPCAtma, "reward", not(a2Is(q, StageCleared)), nil)},
		{Step: cain},
	}
}

// fetch builds a chest leg: waypoint, travel, optional guard kill, open or touch the object, pick the item up, go home.
func a2fetch(q string, wp, to Area, guard string, object, item string, done Pred, budget time.Duration, min int) []Node {
	n := []Node{
		{Step: StepWaypoint(id(q, "wp"), wp, done)},
		{Step: withReady(StepTravelA2(id(q, "to"), "Go to "+object, to, done), a2gate(min, q+" needs clvl "+a2itoa(min)))},
	}
	if guard != "" {
		k := StepKill(id(q, "guard"), guard, to, a2or(done, func(w World) bool { return w.Killed[guard] }), budget)
		k.Optional = true
		n = append(n, req(Node{Step: k}, id(q, "to")))
	}
	n = append(n,
		req(Node{Step: openAndTake(id(q, "object"), object, item, done, a2in(to))}, id(q, "to")),
		Node{Step: home2(id(q, "home"), to, done)})
	return n
}

// openAndTake clicks a chest or altar and picks up what drops, as one step: the world cannot see the drop on the ground,
// only the item in the pack, so a step that only opened would never read as finished.
func openAndTake(sid, object, item string, done, ready Pred) Step {
	s := StepInteract(sid, "Open "+object, object, done, ready)
	s.Do = func(ctx context.Context, x Actions) error {
		if err := x.Interact(ctx, object); err != nil {
			return err
		}
		return x.PickUp(ctx, item)
	}
	return s
}

func a2itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return a2itoa(n/10) + string(rune('0'+n%10))
}

// Horadric Cube: chest in Halls of the Dead 3 (waypoint at Halls of the Dead 2). The Cube is needed to assemble the
// Staff and for every cube recipe afterwards, so it is also a goal of its own (catalog quest.horadric_cube).
func cubeNodes() []Node {
	return a2fetch(QCube, HallsOfDead2, HallsOfDead3, "", "Horadric Cube chest", ItemCube, a2At(QCube, StageDone), 0, 16)
}

// The Tainted Sun: the altar in Claw Viper Temple 2 (Lost City -> Valley of Snakes -> temple 1 -> 2) drops the Amulet
// of the Viper; Fangskin guards it.
func viperNodes() []Node {
	return a2fetch(QViper, LostCity, ClawViper2, BossFangskin, "taintedsunaltar", ItemStaffTop, a2At(QViper, StageDone), 6*time.Minute, 18)
}

// Staff of Kings: chest in Maggot Lair 3 (Far Oasis -> lair 1 -> 2 -> 3).
func shaftNodes() []Node {
	return a2fetch(QShaft, FarOasis, MaggotLair3, "", "Staff of Kings chest", ItemShaft, a2At(QShaft, StageDone), 0, 18)
}

// The Horadric Staff: put shaft and amulet into the Cube in town, then in the Canyon of the Magi find the tomb that holds
// the orifice (the adapter scans the seven tombs) and place the Staff. Done is the reader or the pseudo item
// ItemStaffPlaced; entering Duriel's Lair proves it too.
func staffNodes() []Node {
	q := QStaff
	placed := a2At(q, StageDone)
	assemble := Step{ID: id(q, "assemble"), Title: "Assemble the Horadric Staff", Phase: "chore", Budget: 3 * time.Minute, Est: 30 * time.Second,
		Need: func(w World) (bool, Evidence) {
			return !placed(w) && w.Items[ItemStaff] == 0 && w.Items[ItemShaft] > 0 && w.Items[ItemStaffTop] > 0, evOf(w)
		},
		Ready: func(w World) *Fault {
			switch {
			case !w.InTown:
				return &Fault{Kind: KindPrereq, Why: "the Cube is used in town", Insert: []Step{StepGoHome(id(q, "assemble/home"))}}
			case w.Items[ItemCube] == 0:
				return &Fault{Kind: KindPrereq, Why: "no Horadric Cube in the pack"}
			}
			return nil
		},
		Do: func(ctx context.Context, x Actions) error {
			in := []string{ItemShaft, ItemStaffTop}
			switch t := x.(type) {
			case Transmuter:
				return t.Transmute(ctx, in)
			case CubeOut: // town.Town (matches display names)
				_, err := t.Transmute(ctx, in)
				return err
			}
			return Errorf(KindPrereq, "the bot has no Horadric Cube capability")
		}}
	place := StepInteract(id(q, "place"), "Place the Staff in the orifice", "tal-orifice", placed, func(w World) bool { return w.Items[ItemStaff] > 0 && w.Area == CanyonOfMagi })
	place.Budget = 10 * time.Minute // the tomb scan walks up to seven tomb levels
	return []Node{
		{Step: assemble},
		{Step: StepWaypoint(id(q, "wp-canyon"), CanyonOfMagi, placed)},
		{Step: withReady(StepTravelA2(id(q, "to-canyon"), "Go to the Canyon of the Magi", CanyonOfMagi, func(w World) bool { return placed(w) || w.Items[ItemStaff] == 0 }), a2gate(20, "the Canyon needs clvl 20"))},
		req(Node{Step: place}, id(q, "to-canyon")),
	}
}

// The Arcane Sanctuary (optional): Drognan and Jerhyn send you to the portal at the end of Palace Cellar 3.
func arcaneNodes() []Node {
	q := QArcane
	done := a2At(q, StageDone)
	talk := func(step, npc string) Node {
		s := StepTalk(id(q, step), "Talk to "+npc, npc, "arcane", a2At(q, StageStarted), nil)
		s.Optional = true // U: whether the talks gate the portal
		return Node{Step: s}
	}
	return []Node{
		talk("drognan", NPCDrognan),
		talk("jerhyn", NPCJerhyn),
		{Step: StepWaypoint(id(q, "wp-cellar1"), 52, done)},
		{Step: withReady(StepTravelA2(id(q, "to-cellar3"), "Go to Palace Cellar 3", PalaceCellar3, done), a2gate(20, "the Palace Cellar needs clvl 20"))},
		req(Node{Step: StepInteract(id(q, "portal"), "Take the Arcane Sanctuary portal", "Arcane Sanctuary portal", done, a2in(PalaceCellar3))}, id(q, "to-cellar3")),
	}
}

// The Summoner (optional): Arcane waypoint, find and kill him, touch his journal. Teleport shortens it a lot (U: the
// executor has no capability flag yet, so the level gate stands in).
func summonerNodes() []Node {
	q := QSummoner
	killed := a2At(q, StageCleared)
	done := a2At(q, StageDone)
	journal := StepInteract(id(q, "journal"), "Touch the Summoner's journal", "Summoner's journal", done, func(w World) bool { return w.Killed[BossSummoner] && w.Area == ArcaneSanctuary })
	journal.Optional = true
	return []Node{
		{Step: StepWaypoint(id(q, "wp-arcane"), ArcaneSanctuary, killed)},
		{Step: withReady(StepTravelA2(id(q, "to-arcane"), "Go to the Arcane Sanctuary", ArcaneSanctuary, killed), a2gate(24, "the Summoner needs clvl 24"))},
		req(Node{Step: StepKill(id(q, "kill"), BossSummoner, ArcaneSanctuary, killed, 12*time.Minute)}, id(q, "to-arcane")),
		{Step: journal},
		{Step: home2(id(q, "home"), ArcaneSanctuary, killed)},
	}
}

// Duriel (The Seven Tombs): through the lair door once the Staff is placed, kill him, Tyrael's talk ends the quest.
func durielNodes() []Node {
	q := QDuriel
	killed := a2At(q, StageCleared)
	ready := func(w World) *Fault {
		if w.Level < 24 {
			return &Fault{Kind: KindDanger, Why: "Duriel needs clvl 24"}
		}
		if w.Potions < minPotions {
			return &Fault{Kind: KindResource, Why: "not enough potions", Insert: []Step{StepChores(id(q, "chores"))}}
		}
		return nil
	}
	door := StepInteract(id(q, "door"), "Enter Duriel's Lair", "duriel-door", a2or(killed, a2in(DurielsLair)), staffPlaced)
	door.Ready = ready
	tyrael := StepTalk(id(q, "tyrael"), "Talk to Tyrael", NPCTyrael, "reward", not(a2Is(q, StageCleared)), a2in(DurielsLair))
	tyrael.Ready = nil // Tyrael stands in the lair, not in town
	return []Node{
		{Step: door},
		req(Node{Step: StepKill(id(q, "kill"), BossDuriel, DurielsLair, killed, 15*time.Minute)}, id(q, "door")),
		{Step: tyrael},
		{Step: home2(id(q, "home"), DurielsLair, a2At(q, StageCleared))},
	}
}

// Act 2 complete: Jerhyn (U) and Meshif's boat to the Kurast docks. Done is the world seeing act 3 (EvMemory).
func act2HopNodes() []Node {
	q := QAct2Done
	done := func(w World) bool { return w.Act >= 3 }
	ready := a2At(QDuriel, StageCleared)
	jerhyn := StepTalk(id(q, "jerhyn"), "Talk to Jerhyn", NPCJerhyn, "act2", a2At(QAct2Word, StageDone), ready)
	jerhyn.Optional = true // U: the word may already be set by Tyrael
	return []Node{
		{Step: jerhyn},
		{Step: StepTalk(id(q, "meshif"), "Sail east with Meshif", NPCMeshif, "sail east", done, ready)},
	}
}

// Mercenary: Greiz's Desert Mercenary, Normal auras Defiance / Prayer / Blessed Aim (V hireling.txt). Needs the
// pseudo item ItemMercenary to learn whether one is alive; without it the step would hire twice, so it stays
// Optional-free and Ready blocks on missing gold only.
func mercNodes() []Node {
	q := QMercAct2
	hire := Step{ID: id(q, "hire"), Title: "Hire the Desert Mercenary", Phase: "chore", Budget: 3 * time.Minute, Est: 30 * time.Second,
		Need: func(w World) (bool, Evidence) { return w.Items[ItemMercenary] == 0 && w.Act >= 2, EvMemory },
		Ready: func(w World) *Fault {
			switch {
			case !w.InTown:
				return &Fault{Kind: KindPrereq, Why: "Greiz is in town", Insert: []Step{StepGoHome(id(q, "hire/home"))}}
			case w.Gold < 1000:
				return &Fault{Kind: KindResource, Why: "not enough gold to hire (U: about 360 at merc level 9, more with levels)"}
			}
			return nil
		},
		Do: func(ctx context.Context, x Actions) error {
			h, ok := x.(MercHirer)
			if !ok {
				return Errorf(KindPrereq, "the bot has no mercenary hire capability")
			}
			return h.HireMerc(ctx, 2, "Defiance")
		}}
	return []Node{{Step: hire}}
}
