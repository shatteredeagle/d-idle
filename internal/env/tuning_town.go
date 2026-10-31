package env

// TownTuning holds the town knobs. Owner: the town agent; add fields, defaults and Knob entries here only.
// Percent knobs are 0..100; "Ms" knobs are milliseconds on the env Clock.
type TownTuning struct {
	// Healing and trip triggers.
	HealBelowLifePct  int // visit the healer when life is below this
	HealBelowManaPct  int // ... or mana is below this (0 disables the mana trigger)
	TripMinHPPotions  int // NeedsTrip: fewer healing potions than this in belt+pack
	TripMinMPPotions  int // NeedsTrip: fewer mana potions than this (only when the plan has a mana column)
	TripBeltColumnPct int // NeedsTrip: any planned belt column filled below this percent
	TripPackUsedPct   int // NeedsTrip: backpack cells used at or above this percent
	TripRepairPct     int // NeedsTrip: an equipped item below this durability percent
	// Restocking.
	BeltFillPct     int // refill each potion type up to this percent of its planned columns
	TPScrollTarget  int // town portal scrolls to hold when there is no tome
	TPMinGold       int // below this gold the first town portal scroll is not bought and its absence is no trip reason (0 = off): potions first
	TPTomeMinGold   int // buy a Tome of Town Portal when gold is at least this and none is owned
	IDScrollTarget  int // identify scrolls to hold when there is no tome
	MinGoldReserve  int // never spend below this on optional buys (scroll top-up, extra potions)
	KeyTarget       int // keys to hold (0 = do not buy)
	RepairBelowPct  int // repair when any equipped item is below this durability percent
	StashGoldBase   int // keep this much gold in the pack, plus StashGoldPerLvl per level, and deposit the rest
	StashGoldPerLvl int // see StashGoldBase
	BeltBuyLevel    int // buy the cheapest belt that adds potion slots from this clvl on (docs/sme/potion-policy.md); 0 = never
	SellJunk        int // 1: sell items the pickit says to sell, 0: only stash/keep
	// Poverty mode (Rebuy after a death).
	RebuyGearTier   int // 0 potions only, 1 + weapon and body armor, 2 + helm, boots, gloves, belt, 3 + shield
	RebuyPotionPct  int // share of the rebuy budget potions may use
	RebuyItemMaxPct int // one item may cost at most this percent of the rebuy budget
	// Interaction timing and retries.
	NPCStopDist     int // subtiles from the NPC at which the walk stops before clicking it
	NPCOpenTries    int // attempts to open an NPC dialog
	UITimeoutMs     int // wait for a panel flag after a click
	ClickSettleMs   int // pause after a UI click before the memory is re-read
	MoveTries       int // attempts to move one item between grids (verified each time)
	PortalSettle    int // ms to wait after arriving before using a portal again
	ChoreBudgetS    int // seconds one Chores call may take before it gives up
	SellMaxS        int // seconds the sell chore alone may take (it is capped, then the other chores go on)
	PortalReuseDist int // ToTown uses an already open portal within this many subtiles instead of a scroll
	GreetingWaitS   int // after clicking an NPC, wait this long for its menu (first contact plays a speech of about 60 s)
	GreetingSkip    int // how a first-contact NPC greeting is skipped: 0 wait, 1 click the NPC again, 2 Esc then click, 3 click the NPC tile without hover (internal/town/skip.go)
	SpareKeep       int // spare potions (pickit tier C) and identify scrolls kept per item code when a full pack is sold down; the surplus is sold
	FutileMinutes   int // a trip reason a visit left unfixed stays quiet this long, whatever the gold
	PotionsFirst    int // 1: kit potions before the first town portal scroll, which needs clvl TPFirstLevel and gold for it (SME EXP rule of thumb); 0 = scroll first
	TPFirstLevel    int // PotionsFirst: no first town portal scroll below this clvl (Cold Plains kit level, plan data kit_clvl 4)
	IDScrollNeed    int // 1: buy identify scrolls only while an unidentified set, rare, unique or crafted item is in the pack; 0 = always up to IDScrollTarget
	BankFirst       int // 1: deposit the gold above the keep line (never below the minimal kit) before the shop; 0 = after it
	RepairLatch     int // 1: a repair that changed nothing is not retried until durability changed or gold grew by FutileGoldStep; 0 = retry every visit
	FutileGoldStep  int // after a visit that could not fix a trip reason (cannot afford, not sold), NeedsTrip stays quiet about it until gold grew by this
}

func DefaultTownTuning() TownTuning {
	return TownTuning{
		HealBelowLifePct: 70, HealBelowManaPct: 30,
		TripMinHPPotions: 2, TripMinMPPotions: 1, TripBeltColumnPct: 50, TripPackUsedPct: 80, TripRepairPct: 20,
		BeltFillPct: 100, TPScrollTarget: 2, TPTomeMinGold: 450, IDScrollTarget: 2, MinGoldReserve: 100, KeyTarget: 0,
		RepairBelowPct: 30, StashGoldBase: 300, StashGoldPerLvl: 50, SellJunk: 1, BeltBuyLevel: 10,
		NPCStopDist: 4, NPCOpenTries: 3, UITimeoutMs: 2500, ClickSettleMs: 150, MoveTries: 3,
		RebuyGearTier: 1, RebuyPotionPct: 40, RebuyItemMaxPct: 60, PortalSettle: 4500, ChoreBudgetS: 150, SellMaxS: 60, FutileGoldStep: 100, FutileMinutes: 10, SpareKeep: 4, GreetingWaitS: 90, GreetingSkip: 0, PortalReuseDist: 40,
		PotionsFirst: 1, TPFirstLevel: 4, IDScrollNeed: 1, BankFirst: 1, RepairLatch: 1,
	}
}

func townK(key, label, unit, desc string, def, min, max int, higher bool) Knob {
	return Knob{Key: "town." + key, Label: label, Unit: unit, Desc: desc,
		Default: float64(def), Min: float64(min), Max: float64(max), HigherIsBetter: higher}
}

func townKnobs() []Knob {
	d := DefaultTownTuning()
	return []Knob{
		townK("healBelowLifePct", "Heal below life", "%", "Visit the healer in town when life is below this.", d.HealBelowLifePct, 10, 100, true),
		townK("healBelowManaPct", "Heal below mana", "%", "Visit the healer when mana is below this; 0 disables.", d.HealBelowManaPct, 0, 100, true),
		townK("tripMinHPPotions", "Trip: min healing potions", "count", "Go to town when fewer healing potions remain (belt and pack).", d.TripMinHPPotions, 0, 8, true),
		townK("tripMinMPPotions", "Trip: min mana potions", "count", "Go to town when fewer mana potions remain (only if the belt plan has a mana column).", d.TripMinMPPotions, 0, 8, true),
		townK("tripBeltColumnPct", "Trip: belt column fill", "%", "Go to town when a planned belt column is filled below this.", d.TripBeltColumnPct, 0, 100, true),
		townK("tripPackUsedPct", "Trip: pack full", "%", "Go to town when the backpack is this full.", d.TripPackUsedPct, 40, 100, false),
		townK("tripRepairPct", "Trip: durability", "%", "Go to town when an equipped item is below this durability.", d.TripRepairPct, 0, 60, true),
		townK("beltFillPct", "Belt fill target", "%", "Refill each potion type to this share of its planned columns.", d.BeltFillPct, 25, 100, true),
		townK("tpScrollTarget", "Town portal scrolls", "count", "Scrolls to hold when no tome is owned.", d.TPScrollTarget, 1, 20, true),
		townK("tpMinGold", "TP scroll min gold", "gold", "Below this gold the first town portal scroll is not bought and a missing scroll is no trip reason, so a poor character puts its gold into potions first (XPGAP: 62% of the gold of a fresh d2d character went to scrolls). 0 = always buy at the scroll price.", d.TPMinGold, 0, 1000, false),
		townK("tpTomeMinGold", "TP tome min gold", "gold", "Buy a Tome of Town Portal when gold is at least this.", d.TPTomeMinGold, 0, 5000, false),
		townK("idScrollTarget", "Identify scrolls", "count", "Scrolls to hold when no tome is owned.", d.IDScrollTarget, 0, 20, true),
		townK("minGoldReserve", "Gold reserve", "gold", "Optional purchases never spend below this.", d.MinGoldReserve, 0, 10000, false),
		townK("keyTarget", "Keys", "count", "Keys to hold; 0 never buys keys.", d.KeyTarget, 0, 12, true),
		townK("repairBelowPct", "Repair below", "%", "Repair when any equipped item is below this durability.", d.RepairBelowPct, 0, 90, true),
		townK("stashGoldBase", "Gold kept in the pack", "gold", "Deposit inventory gold above this plus level times stashGoldPerLvl. Death takes ALL carried gold (LIVE G1 attempt 12: 1125 to 0), so keep little.", d.StashGoldBase, 0, 20000, false),
		townK("stashGoldPerLvl", "Deposit gold per level", "gold/lvl", "Deposit inventory gold above stashGoldBase plus level times this.", d.StashGoldPerLvl, 0, 20000, false),
		townK("beltBuyLevel", "Buy a belt from clvl", "clvl", "From this level on a town visit buys the cheapest belt with more potion slots than the worn one (after the potion kit, never below the gold reserve plus a minimal kit). 0 = never. docs/sme/potion-policy.md.", d.BeltBuyLevel, 0, 40, true),
		townK("sellJunk", "Sell junk", "bool", "1 sells items the pickit rates as sell; 0 keeps them.", d.SellJunk, 0, 1, true),
		townK("rebuyGearTier", "Rebuy gear tier", "tier", "After a death: 0 potions only, 1 weapon and body armor, 2 also helm/boots/gloves/belt, 3 also shield.", d.RebuyGearTier, 0, 3, true),
		townK("rebuyPotionPct", "Rebuy potion share", "%", "Share of the rebuy budget potions may use before gear.", d.RebuyPotionPct, 0, 100, true),
		townK("rebuyItemMaxPct", "Rebuy item cost cap", "%", "One rebuy item may cost at most this share of the budget.", d.RebuyItemMaxPct, 10, 100, true),
		townK("npcStopDist", "NPC stop distance", "subtiles", "Distance from an NPC at which the walk ends and the click starts.", d.NPCStopDist, 2, 10, false),
		townK("npcOpenTries", "NPC open attempts", "count", "Attempts to open an NPC dialog before the chore is skipped.", d.NPCOpenTries, 1, 6, true),
		townK("uiTimeoutMs", "Panel timeout", "ms", "Wait for a panel flag after a click.", d.UITimeoutMs, 800, 8000, true),
		townK("clickSettleMs", "UI click settle", "ms", "Pause after a UI click before memory is re-read.", d.ClickSettleMs, 40, 800, false),
		townK("moveTries", "Item move attempts", "count", "Attempts to move one item between grids (each verified in memory).", d.MoveTries, 1, 6, true),
		townK("portalSettle", "Portal settle", "ms", "Wait after arriving in an area before using a portal.", d.PortalSettle, 1000, 8000, false),
		townK("portalReuseDist", "Open portal reuse range", "subtiles", "ToTown walks into an open portal this close instead of reading a scroll.", d.PortalReuseDist, 0, 100, true),
		townK("greetingWaitS", "NPC greeting wait", "s", "Wait for an NPC menu after a click (first contact plays a long speech).", d.GreetingWaitS, 0, 180, false),
		townK("greetingSkip", "NPC greeting skip", "mode", "How the first-contact greeting is skipped when the menu is not open 1 s after the click: 0 wait it out (greetingWaitS), 1 click the NPC again, 2 Esc then click again, 3 left click on the NPC tile without hover. docs/npc-skip.md.", d.GreetingSkip, 0, 3, false),
		townK("futileGoldStep", "Futile trip back-off", "gold", "After a town visit could not fix a trip reason, do not ask again until gold grew by this.", d.FutileGoldStep, 0, 5000, false),
		townK("futileMinutes", "Futile trip quiet time", "min", "A trip reason a visit left unfixed is not raised again for this long.", d.FutileMinutes, 0, 120, false),
		townK("spareKeep", "Spare supplies kept", "count", "A full pack sells spare potions and identify scrolls beyond this many per kind (they fill the pack but are never belted).", d.SpareKeep, 0, 30, true),
		townK("sellMaxS", "Sell chore cap", "s", "Time the sell chore alone may take before it is cut off.", d.SellMaxS, 15, 300, true),
		townK("potionsFirst", "Potions before first scroll", "bool", "1 buys the kit potions before the first town portal scroll, which then needs tpFirstLevel and gold for one scroll (SME EXP rule of thumb, U on d2d). 0 = scroll first (old).", d.PotionsFirst, 0, 1, true),
		townK("tpFirstLevel", "First scroll from clvl", "clvl", "potionsFirst: no first town portal scroll below this level (Cold Plains kit level). SME EXP rule of thumb.", d.TPFirstLevel, 1, 30, false),
		townK("idScrollNeed", "Identify scroll only if valuable", "bool", "1 buys identify scrolls only while an unidentified set, rare, unique or crafted item is in the pack; 0 = always idScrollTarget (old).", d.IDScrollNeed, 0, 1, true),
		townK("bankFirst", "Bank before shop", "bool", "1 deposits gold above the keep line (never below the minimal kit) before the shop; 0 = the shop spends first (old).", d.BankFirst, 0, 1, true),
		townK("repairLatch", "Failed repair latch", "bool", "1 retries a repair that changed nothing only after durability changed or gold grew by futileGoldStep; 0 = every visit (old).", d.RepairLatch, 0, 1, true),
		townK("choreBudgetS", "Chore budget", "s", "Time one Chores call may take before it gives up.", d.ChoreBudgetS, 30, 600, true),
	}
}
