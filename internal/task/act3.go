package task

import (
	"context"
	"time"
)

// Normal Act 3 quest handlers (docs/act3.md). Same shape as act2.go. Tags: area ids, item codes and NPC/boss classes are
// V (docs/act3.md read them from the real 1.14d tables, internal/quest/act3_real_test.go re-checks them); every object
// name the adapter must understand, every talk option, the relic chest mapping and the orb mechanics are U until the
// live checks A3-4/A3-5 ran. Registration is in init() so act1.go stays untouched.

// Act 3 areas (V levels.txt).
const (
	SpiderForest   Area = 76
	FlayerJungle   Area = 78
	KurastBazaar   Area = 80
	UpperKurast    Area = 81
	Travincal      Area = 83
	SpiderCavern   Area = 85
	FlayerDungeon3 Area = 91
	KurastSewers2  Area = 93
	RuinedTemple   Area = 94
	Durance2       Area = 101
	Durance3       Area = 102
	Fortress       Area = 103
)

// Act 3 quest ids. They are the saves ids where one exists (golden_bird, blade_old_religion, lam_esen, guardian); the relic
// legs, the Council and the orb have no save word of their own (Khalim's Will is word 18), so they run on item proxies.
const (
	QEye         = "khalims_eye"
	QBrain       = "khalims_brain"
	QHeart       = "khalims_heart"
	QCouncil     = "blackened_temple" // the three-member kill that drops the Flail (word 21)
	QKhalim      = "khalims_will"     // cube the Will, equip it, smash the Compelling Orb (word 18)
	QGuardian    = "guardian"         // Mephisto (word 22)
	QAct3Done    = "act3_hop"         // the red portal; done when the world is in act 4
	QGolden      = "golden_bird"
	QBlade       = "blade_old_religion"
	QLamEsen     = "lam_esen"
	QMercAct3    = "merc_act3"
	NPCAlkor     = "Alkor"
	NPCOrmus     = "Ormus"
	NPCHratli    = "Hratli"
	BossCouncil  = "High Council"
	BossMephisto = "Mephisto"
	// Item display names (U: items.txt Display column was not dumped for these; the codes are V in quest/act3.go).
	ItemEye        = "Khalim's Eye"
	ItemHeart      = "Khalim's Heart"
	ItemBrain      = "Khalim's Brain"
	ItemFlail      = "Khalim's Flail"
	ItemWill       = "Khalim's Will"
	ItemJade       = "Jade Figurine"
	ItemGolden     = "The Golden Bird"
	ItemGidbinnRaw = "The Gidbinn"
	ItemLamTome    = "Lam Esen's Tome"
	ItemOrbSmashed = "Compelling Orb smashed" // pseudo item: 1 once the adapter smashed the orb this game
)

// OrbSmasher is the optional capability behind the orb step: equip the quest weapon from the pack, left-click the
// Compelling Orb until it breaks, re-equip the old weapon (docs/act3.md X5). Missing capability is a Prereq fault.
type OrbSmasher interface {
	SmashOrb(ctx context.Context, weapon string) error
}

func init() {
	handlers[QEye] = eyeNodes
	handlers[QBrain] = brainNodes
	handlers[QHeart] = heartNodes
	handlers[QCouncil] = councilNodes
	handlers[QKhalim] = khalimNodes
	handlers[QGuardian] = guardianNodes
	handlers[QAct3Done] = act3HopNodes
	handlers[QGolden] = goldenNodes
	handlers[QBlade] = bladeNodes
	handlers[QLamEsen] = lamEsenNodes
	handlers[QMercAct3] = merc3Nodes
}

func orbSmashed(w World) bool {
	return w.Items[ItemOrbSmashed] > 0 || w.Killed[BossMephisto] || w.Area == Durance2 || w.Area == Durance3 || w.Act >= 4
}

// a3Stage is the reader's stage raised by what the world proves. The relic items disappear into the Cube, so once the
// Will exists (or the orb is gone) every earlier leg counts as done.
func a3Stage(w World, q string) Stage {
	s := w.Quest(q).Stage
	raise := func(p Stage) {
		if p > s {
			s = p
		}
	}
	willMade := w.Items[ItemWill] > 0 || orbSmashed(w)
	switch q {
	case QEye:
		if w.Items[ItemEye] > 0 || willMade {
			raise(StageDone)
		}
	case QBrain:
		if w.Items[ItemBrain] > 0 || willMade {
			raise(StageDone)
		}
	case QHeart:
		if w.Items[ItemHeart] > 0 || willMade {
			raise(StageDone)
		}
	case QCouncil:
		if w.Killed[BossCouncil] {
			raise(StageCleared)
		}
		if w.Items[ItemFlail] > 0 || willMade {
			raise(StageDone)
		}
	case QKhalim:
		if w.Items[ItemWill] > 0 {
			raise(StageCleared)
		}
		if orbSmashed(w) {
			raise(StageDone)
		}
	case QGuardian:
		if w.Killed[BossMephisto] {
			raise(StageCleared)
		}
		if w.Act >= 4 {
			raise(StageDone)
		}
	case QAct3Done:
		if w.Act >= 4 {
			raise(StageDone)
		}
	case QGolden:
		if w.Items[ItemJade] > 0 || w.Items[ItemGolden] > 0 {
			raise(StageStarted)
		}
	case QLamEsen:
		if w.Items[ItemLamTome] > 0 {
			raise(StageCleared)
		}
	case QBlade:
		if w.Items[ItemGidbinnRaw] > 0 {
			raise(StageCleared)
		}
	}
	return s
}

func a3At(q string, s Stage) Pred { return func(w World) bool { return a3Stage(w, q) >= s } }
func a3Is(q string, s Stage) Pred { return func(w World) bool { return a3Stage(w, q) == s } }

// Relic legs: waypoint, travel, open the chest, take the relic, go home. Which chest class holds which relic is U (S
// Kolbot); the object names are what the adapter's Interact must resolve in that area. The monster levels of the
// areas are 21..24 (V), the gates are one level under the Normal range plus a margin for a weak build (U).
func eyeNodes() []Node {
	return a2fetch(QEye, SpiderForest, SpiderCavern, "", "Khalim's Eye chest", ItemEye, a3At(QEye, StageDone), 0, 20)
}

func brainNodes() []Node {
	return a2fetch(QBrain, FlayerJungle, FlayerDungeon3, "", "Khalim's Brain chest", ItemBrain, a3At(QBrain, StageDone), 0, 22)
}

// The Heart sits in Kurast Sewers 2, entered from the Bazaar (V Vis 80 -> 92 -> 93); the lever/stairs trick (S) is the
// nav layer's business.
func heartNodes() []Node {
	return a2fetch(QHeart, KurastBazaar, KurastSewers2, "", "Khalim's Heart chest", ItemHeart, a3At(QHeart, StageDone), 0, 22)
}

// The High Council (Travincal): kill the three members, take Khalim's Flail. Which member drops it is U (all six share
// the Council TC, V), so the pickup is by item name after the whole group is dead.
func councilNodes() []Node {
	q := QCouncil
	done := a3At(q, StageDone)
	killed := a3At(q, StageCleared)
	flail := StepPickUp(id(q, "flail"), ItemFlail, done, func(w World) bool { return w.Killed[BossCouncil] && w.Area == Travincal })
	return []Node{
		{Step: StepWaypoint(id(q, "wp"), Travincal, done)},
		{Step: withReady(StepTravelA2(id(q, "to"), "Go to Travincal", Travincal, done), gatePotions(24, "the Council needs clvl 24", minPotions, q))},
		req(Node{Step: StepKill(id(q, "kill"), BossCouncil, Travincal, killed, 15*time.Minute)}, id(q, "to")),
		req(Node{Step: flail}, id(q, "to")),
		{Step: home2(id(q, "home"), Travincal, a3Is(q, StageCleared))},
	}
}

// Khalim's Will: cube Flail + Heart + Eye + Brain in town (4 inputs, V recipe, never run live), then at the orb equip it
// and smash the Compelling Orb. Done is the pseudo item set by the adapter, or being past Travincal.
func khalimNodes() []Node {
	q := QKhalim
	done := a3At(q, StageDone)
	have := func(w World) bool {
		return w.Items[ItemFlail] > 0 && w.Items[ItemHeart] > 0 && w.Items[ItemEye] > 0 && w.Items[ItemBrain] > 0
	}
	assemble := Step{ID: id(q, "assemble"), Title: "Cube Khalim's Will", Phase: "chore", Budget: 3 * time.Minute, Est: 30 * time.Second,
		Need: func(w World) (bool, Evidence) { return !done(w) && w.Items[ItemWill] == 0 && have(w), evOf(w) },
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
			in := []string{ItemFlail, ItemHeart, ItemEye, ItemBrain}
			switch t := x.(type) {
			case Transmuter:
				return t.Transmute(ctx, in)
			case CubeOut:
				_, err := t.Transmute(ctx, in)
				return err
			}
			return Errorf(KindPrereq, "the bot has no Horadric Cube capability")
		}}
	smash := Step{ID: id(q, "orb"), Title: "Smash the Compelling Orb", Phase: "fight", Budget: 4 * time.Minute, Est: 45 * time.Second,
		Need: needIf(done, func(w World) bool { return w.Items[ItemWill] > 0 && w.Area == Travincal }),
		Do: func(ctx context.Context, x Actions) error {
			o, ok := x.(OrbSmasher)
			if !ok {
				return Errorf(KindPrereq, "the bot cannot equip a quest weapon and smash the orb")
			}
			return o.SmashOrb(ctx, ItemWill)
		}}
	return []Node{
		{Step: assemble},
		{Step: StepWaypoint(id(q, "wp"), Travincal, done)},
		{Step: withReady(StepTravelA2(id(q, "to"), "Go to Travincal", Travincal, func(w World) bool { return done(w) || w.Items[ItemWill] == 0 }), gate(24, "the orb needs clvl 24"))},
		req(Node{Step: smash}, id(q, "to")),
	}
}

// Mephisto: through the opened stairs to Durance 2 (waypoint) and 3, kill him (boss.Mephisto routes and fights). The
// reader's Done is the word set by the kill; the portal is the next group.
func guardianNodes() []Node {
	q := QGuardian
	killed := a3At(q, StageCleared)
	return []Node{
		{Step: withReady(StepTravelA2(id(q, "to-durance3"), "Go to Durance of Hate 3", Durance3, killed), gatePotions(28, "Mephisto needs clvl 28", minPotions, q))},
		req(Node{Step: StepKill(id(q, "kill"), BossMephisto, Durance3, killed, 20*time.Minute)}, id(q, "to-durance3")),
	}
}

// The red portal. It exists only in the game of the kill (S Kolbot), so this group owns a kill of its own that ignores
// the reader: a restarted bot that finds the quest Done but the world still in act 3 fights Mephisto again (U: he
// respawns in a new game).
func act3HopNodes() []Node {
	q := QAct3Done
	done := func(w World) bool { return w.Act >= 4 }
	kill := StepKill(id(q, "kill"), BossMephisto, Durance3, func(w World) bool { return done(w) || w.Killed[BossMephisto] }, 20*time.Minute)
	return []Node{
		{Step: withReady(StepTravelA2(id(q, "to"), "Go to Durance of Hate 3", Durance3, func(w World) bool { return done(w) || w.Killed[BossMephisto] }), gatePotions(28, "Mephisto needs clvl 28", minPotions, q))},
		req(Node{Step: kill}, id(q, "to")),
		{Step: StepInteract(id(q, "portal"), "Take the red portal", "red portal", done, func(w World) bool { return w.Killed[BossMephisto] && w.Area == Durance3 })},
	}
}

// Golden Bird: opportunistic. The Jade Figurine's dropper is U, so this only runs when j34 is already in the pack:
// Meshif turns it into the Golden Bird, Alkor pays +20 life.
func goldenNodes() []Node {
	q := QGolden
	meshif := StepTalk(id(q, "meshif"), "Give the figurine to Meshif", NPCMeshif, "golden bird", func(w World) bool { return w.Items[ItemJade] == 0 }, nil)
	alkor := StepTalk(id(q, "alkor"), "Give the Golden Bird to Alkor", NPCAlkor, "golden bird", func(w World) bool { return w.Items[ItemGolden] == 0 }, nil)
	return []Node{{Step: optional(meshif)}, {Step: optional(alkor)}}
}

// Blade of the Old Religion (optional): the Gidbinn altar in the Flayer Jungle, Hratli takes it. Flow U.
func bladeNodes() []Node {
	q := QBlade
	n := a2fetch(q, FlayerJungle, FlayerJungle, "", "gidbinn altar", ItemGidbinnRaw, a3At(q, StageCleared), 0, 22)
	hratli := StepTalk(id(q, "hratli"), "Give the Gidbinn to Hratli", NPCHratli, "blade", func(w World) bool { return w.Items[ItemGidbinnRaw] == 0 }, nil)
	return append(n, Node{Step: optional(hratli)})
}

// Lam Esen's Tome (optional, +5 stat points): the Ruined Temple altar off the Bazaar, then Alkor. Needs a quest reader
// to know the reward was claimed (the tome leaves the pack when handed in, which the proxy does see).
func lamEsenNodes() []Node {
	q := QLamEsen
	n := a2fetch(q, KurastBazaar, RuinedTemple, "", "Lam Esen altar", ItemLamTome, a3At(q, StageCleared), 0, 22)
	alkor := StepTalk(id(q, "alkor"), "Give the tome to Alkor", NPCAlkor, "lam esen", func(w World) bool { return w.Items[ItemLamTome] == 0 }, nil)
	return append(n, Node{Step: optional(alkor)})
}

// Asheara's Iron Wolf. The row is U (docs/act3.md section 4).
func merc3Nodes() []Node {
	q := QMercAct3
	hire := Step{ID: id(q, "hire"), Title: "Hire the Iron Wolf", Phase: "chore", Budget: 3 * time.Minute, Est: 30 * time.Second,
		Need: func(w World) (bool, Evidence) { return w.Items[ItemMercenary] == 0 && w.Act >= 3, EvMemory },
		Ready: func(w World) *Fault {
			switch {
			case !w.InTown:
				return &Fault{Kind: KindPrereq, Why: "Asheara is in town", Insert: []Step{StepGoHome(id(q, "hire/home"))}}
			case w.Gold < 1500:
				return &Fault{Kind: KindResource, Why: "not enough gold to hire (U)"}
			}
			return nil
		},
		Do: func(ctx context.Context, x Actions) error {
			h, ok := x.(MercHirer)
			if !ok {
				return Errorf(KindPrereq, "the bot has no mercenary hire capability")
			}
			return h.HireMerc(ctx, 3, "Cold")
		}}
	return []Node{{Step: hire}}
}
