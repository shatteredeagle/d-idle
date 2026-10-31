package task

import (
	"context"
	"time"
)

// Normal Act 4 quest handlers (docs/act4.md). Same shape as act3.go. Tags: area ids, item codes and boss classes are V
// (docs/act4.md read them from the real 1.14d tables, internal/quest/act45_real_test.go re-checks them); every object
// name the adapter must understand, every talk option, the Hellforge procedure and the Tyrael hop are U until the live
// checks A4-5..A4-8 ran. Registration is in init() so act1.go stays untouched.

// Act 4 areas (V levels.txt). Fortress (103) is in act3.go.
const (
	OuterSteppes    Area = 104
	PlainsOfDespair Area = 105
	CityOfTheDamned Area = 106
	RiverOfFlame    Area = 107
	ChaosSanctuary  Area = 108
)

// Act 4 quest ids. They are the save-word ids (izual 25, hellforge 26, terrors_end 27) except the hop group.
const (
	QIzual       = "izual"
	QHellforge   = "hellforge"
	QDiablo      = "terrors_end"
	QAct4Done    = "act4_hop" // Tyrael's portal; done when the world is in act 5
	BossIzual    = "Izual"
	BossHephasto = "Hephasto the Armorer"
	BossDiablo   = "Diablo"
	// Item display names (U: the Display column was not dumped; codes are V in internal/quest/act4.go).
	ItemSoulstone = "Mephisto's Soulstone"
	ItemHammer    = "Hell Forge Hammer"
	ItemForgeUsed = "Hellforge used" // pseudo item: 1 once the adapter placed the Soulstone and smashed it
)

// ForgeUser is the optional capability behind the Hellforge: put the Soulstone on the forge and hit it with the
// Hammer. The procedure (equip + attack the object, or cursor drop) is U (docs/act4.md A4-6). Missing capability is a
// Prereq fault.
type ForgeUser interface {
	UseForge(ctx context.Context, soulstone, hammer string) error
}

func init() {
	handlers[QIzual] = izualNodes
	handlers[QHellforge] = hellforgeNodes
	handlers[QDiablo] = diabloNodes
	handlers[QAct4Done] = act4HopNodes
}

// a4Stage is the reader's stage raised by what the world proves.
func a4Stage(w World, q string) Stage {
	s := w.Quest(q).Stage
	raise := func(p Stage) {
		if p > s {
			s = p
		}
	}
	switch q {
	case QIzual:
		if w.Killed[BossIzual] {
			raise(StageCleared)
		}
	case QHellforge:
		if w.Items[ItemHammer] > 0 {
			raise(StageStarted)
		}
		if w.Killed[BossHephasto] {
			raise(StageCleared)
		}
		if w.Items[ItemForgeUsed] > 0 {
			raise(StageDone)
		}
	case QDiablo:
		if w.Killed[BossDiablo] {
			raise(StageCleared)
		}
		if w.Act >= 5 {
			raise(StageDone)
		}
	case QAct4Done:
		if w.Act >= 5 {
			raise(StageDone)
		}
	}
	return s
}

func a4At(q string, s Stage) Pred { return func(w World) bool { return a4Stage(w, q) >= s } }
func a4Is(q string, s Stage) Pred { return func(w World) bool { return a4Stage(w, q) == s } }

// The Fallen Angel: City of the Damned waypoint, walk on to the Plains of Despair, kill Izual, Tyrael pays 2 skill
// points. Izual's monster level is 29 (V); cold builds are poor against his 75 cold resist (V). Floor 30 (R).
func izualNodes() []Node {
	q := QIzual
	killed := a4At(q, StageCleared)
	return []Node{
		{Step: StepWaypoint(id(q, "wp"), CityOfTheDamned, killed)},
		{Step: withReady(StepTravelA2(id(q, "to"), "Go to the Plains of Despair", PlainsOfDespair, killed), gatePotions(30, "Izual needs clvl 30", minPotions, q))},
		req(Node{Step: StepKill(id(q, "kill"), BossIzual, PlainsOfDespair, killed, 15*time.Minute)}, id(q, "to")),
		{Step: home2(id(q, "home"), PlainsOfDespair, a4Is(q, StageCleared))},
		{Step: StepTalk(id(q, "reward"), "Talk to Tyrael", NPCTyrael, "fallen angel", not(a4Is(q, StageCleared)), nil)},
	}
}

// The Hellforge (optional): needs the Soulstone from the Mephisto kill in the pack (a quest item, it survives the act
// hop). Hephasto holds the Hammer at the forge in the River of Flame; the placement is U. Without the Soulstone the whole
// group has nothing to do.
func hellforgeNodes() []Node {
	q := QHellforge
	done := a4At(q, StageDone)
	hasStone := func(w World) bool { return w.Items[ItemSoulstone] > 0 }
	killed := a4At(q, StageCleared)
	place := Step{ID: id(q, "place"), Title: "Smash the Soulstone on the Hellforge", Phase: "fight", Budget: 4 * time.Minute, Est: time.Minute,
		Need: needIf(done, func(w World) bool { return w.Area == RiverOfFlame && hasStone(w) && w.Items[ItemHammer] > 0 }),
		Do: func(ctx context.Context, x Actions) error {
			f, ok := x.(ForgeUser)
			if !ok {
				return Errorf(KindPrereq, "the bot cannot place the Soulstone on the Hellforge")
			}
			return f.UseForge(ctx, ItemSoulstone, ItemHammer)
		}}
	return []Node{
		{Step: StepWaypoint(id(q, "wp"), RiverOfFlame, func(w World) bool { return done(w) || !hasStone(w) })},
		{Step: withReady(StepTravelA2(id(q, "to"), "Go to the River of Flame", RiverOfFlame, func(w World) bool { return done(w) || !hasStone(w) || w.Items[ItemHammer] > 0 }),
			gatePotions(30, "the Hellforge needs clvl 30", minPotions, q))},
		req(Node{Step: StepKill(id(q, "kill"), BossHephasto, RiverOfFlame, func(w World) bool { return done(w) || !hasStone(w) || killed(w) }, 10*time.Minute)}, id(q, "to")),
		req(Node{Step: StepPickUp(id(q, "hammer"), ItemHammer, done, func(w World) bool { return w.Killed[BossHephasto] && w.Area == RiverOfFlame })}, id(q, "to")),
		req(Node{Step: place}, id(q, "to")),
		{Step: home2(id(q, "home"), RiverOfFlame, func(w World) bool { return done(w) || !hasStone(w) })},
	}
}

// Terror's End: River of Flame waypoint, the Chaos Sanctuary, Diablo (boss.RunDiablo opens the five seals and kills the
// three seal bosses first, so this is one KillBoss call), Tyrael. Floor 30 (R, progression-normal).
func diabloNodes() []Node {
	q := QDiablo
	killed := a4At(q, StageCleared)
	return []Node{
		{Step: StepWaypoint(id(q, "wp"), RiverOfFlame, killed)},
		{Step: withReady(StepTravelA2(id(q, "to"), "Go to the Chaos Sanctuary", ChaosSanctuary, killed), gatePotions(30, "Diablo needs clvl 30", 5, q))},
		req(Node{Step: StepKill(id(q, "kill"), BossDiablo, ChaosSanctuary, killed, 30*time.Minute)}, id(q, "to")),
		{Step: home2(id(q, "home"), ChaosSanctuary, a4Is(q, StageCleared))},
		{Step: StepTalk(id(q, "reward"), "Talk to Tyrael", NPCTyrael, "diablo", not(a4Is(q, StageCleared)), nil)},
	}
}

// Tyrael's portal to Harrogath. Needs the Expansion flag (V-02 in docs/progression/verification.md) and Diablo dead;
// the row text and whether it is a dialog row or a town portal object (566, "Harrogath", V name) are U.
func act4HopNodes() []Node {
	q := QAct4Done
	done := func(w World) bool { return w.Act >= 5 }
	ready := func(w World) bool { return w.Killed[BossDiablo] || a4Stage(w, QDiablo) >= StageCleared }
	return []Node{
		{Step: StepTalk(id(q, "tyrael"), "Travel to Harrogath", NPCTyrael, "harrogath", done, ready)},
	}
}
