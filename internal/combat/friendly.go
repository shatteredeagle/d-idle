package combat

import (
	"strings"

	"d2lod-bot/internal/d2"
)

// Why a unit counts as friendly (never a target, never a threat).
type FriendlyWhy int

const (
	NotFriendly    FriendlyWhy = iota
	WhyOwner                   // owner words name the player or a friendly unit
	WhyNPC                     // monstats npc flag (townspeople, quest givers)
	WhyHirelingAI              // monstats AI Hireable
	WhySummonAI                // monstats AI of a class summon (table below)
	WhyClass                   // class id in the fallback table (no monstats row, or a row without AI)
	WhyMinionState             // child state 105
)

func (w FriendlyWhy) String() string {
	return [...]string{"no", "owner", "npc", "hireling-ai", "summon-ai", "class", "minion-state"}[w]
}

// friendlyAI are the monstats.txt AI values that only ever belong to units the player controls or hires.
// VERIFIED (extracted monstats.tsv of the live 1.14d install, every row with that AI, 2026-10): NecroPet = golems 289-292,
// valkyrie 357, necroskeleton 363, necromage 364, wolf 430, bear 431 (the Hireable and NecroPet rows are exactly the
// units below, no hostile row shares them); Hireable = 271 338 359 560 561; DruidWolf 420 421; DruidBear 428; Raven 419;
// Totem 422-424 (druid spirits); Vines 425; CycleOfLife 426 427; Hydra 351-353; AssassinSentry 410-412 415;
// BladeCreeper 413; InvisoPet 414; ShadowWarrior 417; ShadowMaster 418. That these rows are friendly in play is
// VERIFIED LIVE for Valkyrie 357 (owner = player id), the clay golem 289, necroskeleton 363 (state 105) and Sorceress Hydra 351-353 (state 105, owner words 0); the rest is
// UNVERIFIED in the live game (no druid, assassin or sorceress summon was ever cast).
var friendlyAI = map[string]FriendlyWhy{
	"hireable": WhyHirelingAI,
	"necropet": WhySummonAI, "druidwolf": WhySummonAI, "druidbear": WhySummonAI, "raven": WhySummonAI,
	"totem": WhySummonAI, "vines": WhySummonAI, "cycleoflife": WhySummonAI, "hydra": WhySummonAI,
	"assassinsentry": WhySummonAI, "bladecreeper": WhySummonAI, "invisopet": WhySummonAI,
	"shadowwarrior": WhySummonAI, "shadowmaster": WhySummonAI,
}

// friendlyClass is the same set by class id for when monstats is missing or has no AI column, plus classes that
// the AI column cannot name: the Amazon decoy (356 dopplezon, AI Idle, V in monstats), Necromancer bone prison walls
// (340-343, AI Idle), the Act 2 guard 358 (act2guard3, a hirelable-looking town guard, U) and the hireling classes.
// Hireling classes 271 (V live: hired Rogue "Heather", owner words 0/0), 338, 359, 560, 561 (V monstats AI Hireable,
// not hired live); 358 comes from the brief and is U.
var friendlyClass = map[int]FriendlyWhy{
	271: WhyClass, 338: WhyClass, 358: WhyClass, 359: WhyClass, 560: WhyClass, 561: WhyClass,
	289: WhyClass, 290: WhyClass, 291: WhyClass, 292: WhyClass, 357: WhyClass, 363: WhyClass, 364: WhyClass,
	430: WhyClass, 431: WhyClass, 419: WhyClass, 420: WhyClass, 421: WhyClass, 428: WhyClass,
	422: WhyClass, 423: WhyClass, 424: WhyClass, 425: WhyClass, 426: WhyClass, 427: WhyClass,
	351: WhyClass, 352: WhyClass, 353: WhyClass,
	410: WhyClass, 411: WhyClass, 412: WhyClass, 413: WhyClass, 414: WhyClass, 415: WhyClass, 417: WhyClass, 418: WhyClass,
	356: WhyClass, 340: WhyClass, 341: WhyClass, 342: WhyClass, 343: WhyClass,
}

// Owner types of the owner words (UnitAny type numbers, V live: the Valkyrie reads type 0 / owner = player id).
const (
	ownerPlayer = 0
)

// friendlyKind decides one unit on its own (no owner chain): monstats npc flag, AI, class table, minion state.
func friendlyKind(u d2.Unit, cat Catalog) FriendlyWhy {
	if cat != nil {
		if m := cat.Monster(u.Class); m != nil {
			if m.NPC {
				return WhyNPC
			}
			if w, ok := friendlyAI[strings.ToLower(m.AI)]; ok {
				return w
			}
		}
	}
	if w, ok := friendlyClass[u.Class]; ok {
		return w
	}
	if u.Monster.HasState(d2.StateMinion) {
		return WhyMinionState
	}
	return NotFriendly
}

// Friendlies maps the id of every friendly monster in the snapshot to why. A unit is friendly when it is a friendly kind,
// or its owner words name the player (type 0 and the player id; ids are per unit type, so type matters) or another
// friendly unit (a minion's own summon). Owner words read 0 for Necromancer minions and the hired Rogue (V), which is
// why the kind tests exist. Revived monsters keep their own class and need state 105 (U: not seen live).
// cat may be nil (class table and states only).
func Friendlies(snap *d2.Snapshot, cat Catalog) map[uint32]FriendlyWhy {
	out := map[uint32]FriendlyWhy{}
	if snap == nil {
		return out
	}
	var pid uint32
	if snap.Player != nil {
		pid = snap.Player.ID
	}
	for _, u := range snap.Monsters {
		if w := friendlyKind(u, cat); w != NotFriendly {
			out[u.ID] = w
		} else if pid != 0 && u.OwnerID == pid && u.OwnerType == ownerPlayer {
			out[u.ID] = WhyOwner
		}
	}
	for pass := 0; pass < 4; pass++ { // owner chains: a minion's summon, bounded
		grew := false
		for _, u := range snap.Monsters {
			if _, ok := out[u.ID]; ok || u.OwnerID == 0 {
				continue
			}
			if _, ok := out[u.OwnerID]; ok && u.OwnerType != ownerPlayer { // type of a minion's owner U (should be 1)
				out[u.ID] = WhyOwner
				grew = true
			}
		}
		if !grew {
			break
		}
	}
	return out
}

// IsFriendly is the one question every target picker asks: is u a minion, mercenary, pet, summon, townsperson or
// anything else that must never be attacked or avoided as hostile. The player itself and other party members are
// players, not entries of snap.Monsters, so they never reach a picker; a unit with the player's id is friendly anyway.
func IsFriendly(snap *d2.Snapshot, u d2.Unit, cat Catalog) bool {
	_, ok := FriendlyWhyOf(snap, u, cat)
	return ok
}

// FriendlyWhyOf is IsFriendly with the reason.
func FriendlyWhyOf(snap *d2.Snapshot, u d2.Unit, cat Catalog) (FriendlyWhy, bool) {
	// Unit ids are per type (V live: Combi is player id 1 and the town monster "an evil force" is monster id 1), and
	// every unit passed here is a monster, so there is no player-id test on u itself.
	if w := friendlyKind(u, cat); w != NotFriendly {
		return w, true
	}
	w, ok := Friendlies(snap, cat)[u.ID]
	return w, ok
}
