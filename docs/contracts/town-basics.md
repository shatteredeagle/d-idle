---
id: town-basics
goal: G-B
status: in-force
enforced_by:
  - internal/town
---
# town-basics contract

## Purpose
What the bot does in town with gold, scrolls, potions, repairs, the stash and the pack so that a poor early character spends on survival first and never burns time or gold on trips that pay nothing. Source: docs/basics-audit.md (the d2-sme's 15 scenarios, 2026-10-31), docs/BACKLOG.md section 0 (user milestone).

> status: reconciled 2026-11-04 against docs/process/spec-ledger-20261104.md; pending deltas: none

## Requirements

### Requirement: ID scrolls only on demand
The bot SHALL buy an identify scroll only when a pack item needs identifying and gold after the purchase stays above the gold reserve; an unidentified magic item whose estimated sale value is under 100 gold SHALL be sold unidentified. Source: docs/basics-audit.md rows 2, 3, 7 and G1; d2-sme checklist 2026-10-31.

#### Scenario: Scroll held, belt full, poor
- **WHEN** gold is 120, one portal scroll is held, the belt is full and no pack item needs identifying
- **THEN** no shop window is opened and the gold stays 120

#### Scenario: Cheap unidentified ring
- **WHEN** the pack holds an unidentified ring worth under 100 gold and gold is 80
- **THEN** the ring is sold unidentified and no identify scroll is bought

### Requirement: Purchase order when poor
With town.potionsFirst 1 (default) the shop SHALL buy healing potions first (up to 3) and the first Town Portal scroll only when the gold left covers the scroll price plus town.tpScrollPotions (default 2) healing potions, with no level rule. The same gold test decides whether a missing scroll raises a town trip. Source: decision D3 2026-10-31; docs/basics-audit.md G5. Shared town code; d2d prices UNVERIFIED.

#### Scenario: Gold 150, empty belt, no scroll
- **WHEN** a clvl 4 character enters town with 150 gold, an empty belt and no portal scroll
- **THEN** healing potions are bought first (up to 3) and no scroll is bought, because the gold left does not cover scroll plus 2 potions (TestAuditPurchaseOrder)

#### Scenario: Rich enough for both
- **WHEN** a clvl 1 character enters town with 900 gold, an empty belt and no portal scroll
- **THEN** potions are bought first and a portal scroll after them (TestAuditPurchaseOrder)

#### Scenario: Old rule restored by knobs
- **WHEN** town.tpScrollPotions is 0 and town.tpFirstLevel is 4 at 150 gold and clvl 4
- **THEN** a missing scroll is a trip reason again (TestAuditPurchaseOrderOldRuleKnobs)

### Requirement: Bank before the shop
When carried gold exceeds the keep line (300 + 50 x clvl, death-rule spec) the stash deposit SHALL run before the shop, so the shop spends only from the kit budget and the surplus is banked. Source: docs/basics-audit.md row 11 and G6.

#### Scenario: Gold 1500 at clvl 5
- **WHEN** a clvl 5 character enters town with 1500 gold
- **THEN** 950 gold is banked and the pack keeps 550 before any purchase

### Requirement: Repair failure latch
After a repair attempt fails the bot SHALL log it once and MUST NOT retry the repair until carried gold or the lowest worn durability has changed. Source: docs/basics-audit.md row 10 and G4; docs/d2d-economy.md item 2 (V d2d: five retries in one run); card basicsfix.

#### Scenario: Second visit, nothing changed
- **WHEN** a repair failed on the previous visit and gold and durability are unchanged
- **THEN** the repair step is skipped and no second log line is written

### Requirement: Charms, runes and gems on a sell visit
A sell visit SHALL keep every rune, every unidentified small charm, identified charms with life, a resist, FCR, FHR, MF or skills (large and grand ones only with 3 free pack cells) and every gem of standard grade or better; chipped and flawed gems SHALL be sold until a Horadric Cube is held (pack or stash; UNVERIFIED as the stand-in for "recipe available"). town.sellLowGems 0 restores the old keeping. Source: decision D4 2026-10-31; docs/basics-audit.md row 6, G7. Shared loot code.

#### Scenario: Plain charm and rune in the pack
- **WHEN** the pack holds an unidentified small charm, a rune, a chipped gem and a flawless gem and the bot sells junk before owning the Cube
- **THEN** the charm, the rune and the flawless gem are kept and the chipped gem is sold (TestCharmsRunesGemsOnSellVisit, TestAuditS6RuneCharmAndGems)

#### Scenario: Identified charm without a kept stat
- **WHEN** a small charm is identified and shows only a damage bonus
- **THEN** it is sold on the next visit (TestCharmsRunesGemsOnSellVisit)

#### Scenario: Full pack and a grand charm
- **WHEN** a large charm with life is found and fewer than 3 pack cells are free
- **THEN** it is sold, not kept (TestBigCharmsNeedThreeFreeCells)

### Requirement: Stash gold cap
When the stash refuses a deposit because it is full the bot SHALL deposit only the difference to the cap (stash step and bank-before-shop alike), log it once, and MUST NOT fail the stash step on every later visit. Source: docs/basics-audit.md E4 and G10; the cap constant (2,500,000) is UNVERIFIED until card m1-liveproof measures it; a refused deposit into a stash holding 100,000 gold or more is taken as the cap.

#### Scenario: Full stash
- **WHEN** the stash gold is at the cap and the pack holds 2000 gold above the keep line
- **THEN** the deposit is skipped with one log line and the next visit does not retry it while the stash is still full

#### Scenario: Partial deposit
- **WHEN** the stash has room for 100 gold and the pack holds gold above the keep line
- **THEN** exactly 100 gold are deposited and the stash step does not fail

### Requirement: Recovery pickup is wired
While the recovery phase is on, the Looter handed to the bot loop SHALL be in recovery mode and SHALL report recovery totals, so that every gold pile and every sellable item up to the kit line is picked up and the `recovery: loot` line counts them. Wrapping the Looter in an adapter MUST NOT hide the recovery methods.

#### Scenario: Recovery pass through the adapter
- **WHEN** the recovery phase switches recovery mode on through the bot-facing Looter and a small gold pile and a sellable sword lie on the ground at a gold amount between the poor line and the kit line
- **THEN** both are picked up and the recovery totals are non-zero

#### Scenario: Not recovering
- **WHEN** recovery mode is off at the same gold amount
- **THEN** neither is picked up

### Requirement: Belt fill retries a vanished item id
A belt move that fails because the item id is no longer listed SHALL NOT mark the belt full; the fill SHALL re-read the pack and try the remaining potions again, and the sticky belt-full mark SHALL reset when a new game starts or the belt or the pack's potion set changes. Source: docs/primitives-review.md F1, F9.

#### Scenario: Put fails once after a re-entry
- **WHEN** the first belt put fails with item-not-found and the pack holds healing potions
- **THEN** the same fill retries and moves them, and a later fill is not blocked

### Requirement: Belt filled before leaving town
A town visit SHALL end with the pack's healing potions in the belt up to the plan, including potions bought in that visit, because d2d purchases land in the pack. Source: docs/primitives-review.md F3.

#### Scenario: Purchase lands in the pack
- **WHEN** the shop buys healing potions and they land in the pack
- **THEN** a belt step after the shop moves them before the chores end

### Requirement: Belt column holds one kind
A potion SHALL be put only into a belt column that holds its kind or is empty, never above another kind (SME, UNVERIFIED in this repo), and combat SHALL see every belt box 0..15 with column = box mod 4 on both backends. Source: docs/primitives-review.md F7, F8.

#### Scenario: Healing next to mana columns
- **WHEN** the belt row holds healing, mana, mana, mana and two more healing potions are put
- **THEN** both go into the healing column and combat counts them

### Requirement: Kit counts pack potions
The kit gate SHALL treat a healing or rejuvenation potion in the pack as present, because the belt fill places it. Source: docs/primitives-review.md F6.

#### Scenario: Empty belt, potion in the pack
- **WHEN** the belt holds no healing potion and the pack holds one, with a portal scroll
- **THEN** the kit is not missing

### Requirement: ToTown never reads a scroll while an open portal is in reach
`ToTown` SHALL enter a town portal object within `portalReuseDist` of the player on every try and SHALL read a scroll or tome only when no such portal is in reach. When every try used an open portal and none led to town, it SHALL take the no-scroll path (walk or waypoint) instead of casting. Source: drill D2.t (all 3 scrolls spent on casts beside a portal that could not be entered); a new cast only replaces the pair with the same kind of portal.

#### Scenario: First entry ignored
- **WHEN** a portal stands 10 subtiles from the player in the field with 3 scrolls in the pack and the first entry does nothing
- **THEN** ToTown enters the same portal on the next try, arrives in town and still holds 3 scrolls

#### Scenario: No portal in reach
- **WHEN** no portal object is within reach and the pack holds 2 scrolls
- **THEN** one scroll is read, its portal entered, and 1 scroll is left

#### Scenario: Open portal never leads
- **WHEN** the open portal in reach never takes the player and the pack holds 2 scrolls
- **THEN** no scroll is read and ToTown walks back from Blood Moor to the camp

### Requirement: Gold deposit works on d2d
The town stash chore SHALL deposit gold, a partial amount or the whole purse, through the controller's gold-move capability when it has one, because the d2d amount prompt starts empty and the d2d town controller cannot type. A controller without the capability SHALL keep the typed path. The deposit SHALL still be clamped to the room left under the stash gold cap and confirmed by the stash gold rising in memory.

#### Scenario: Partial deposit on a controller that cannot type
- **WHEN** the purse holds 2000 gold above a keep line of 550 and the controller only offers the gold-move capability
- **THEN** 1450 gold reaches the stash with one move and the chore reports no error

#### Scenario: Whole purse
- **WHEN** the whole purse of 1500 gold is deposited on such a controller
- **THEN** the purse is 0 and the stash holds 1500

#### Scenario: Stash nearly full
- **WHEN** the stash has room for 100 gold
- **THEN** the move is for 100

### Requirement: Repair is not attempted without gold
The repair chore SHALL NOT be needed when the character holds less gold than a lower-bound estimate of the repair price, so a character that cannot pay never opens the smith.

#### Scenario: Gold 0 with worn gear
- **WHEN** the character has 0 gold and badly damaged gear
- **THEN** no repair step runs and no error is reported, and with 500 gold the next visit repairs

### Requirement: Prices follow game.exe's formulas
The bot's repair, sell and buy price helpers SHALL follow the vendor price function of game.exe 1.14d (docs/research/price-trace.md): repair at cost x missing durability / 8 (armor scaled by defence), none for unidentified or ethereal items; sell at cost / 2, a quarter more off for ethereal and class items, capped at the NPC max buy; buy at cost x the vendor multiplier. Terms the bot cannot read SHALL stay unmodelled, making the result a lower bound.

#### Scenario: Live repair price
- **WHEN** a Short Staff of base cost 168 with 1 of 20 durability is estimated for repair
- **THEN** the price is 19 gold, the amount drill D3.r paid at Charsi

#### Scenario: Sell value
- **WHEN** a normal weapon of base cost 1000 is valued, and again as an ethereal copy
- **THEN** the values are 500 and 125, and never above the NPC max buy

#### Scenario: Vendor multiplier
- **WHEN** an item of base cost 100 is priced at Charsi, Gheed and an Act 5 vendor
- **THEN** the prices are 93, 106 and 200

### Requirement: Ground potion goes where the game puts it
A clicked ground potion (inventory panel closed) SHALL be treated as landing on the belt when a same-kind column has a free box or a column is empty, else in the pack, and the pickit, the pickup arrival check and the belt fill MUST all follow that rule. Source: docs/fixpickup.md.

#### Scenario: Matching column has room, one in the pack
- **WHEN** a healing potion lies on the ground, the belt has a healing column with a free box and the pack holds one healing potion
- **THEN** the potion is wanted

#### Scenario: Belt full for the kind
- **WHEN** no column can take the kind and the pack already holds the buffer
- **THEN** the potion is skipped by the buffer rule

#### Scenario: Other kind full, empty column free
- **WHEN** the healing column is full and another column is empty
- **THEN** the potion is wanted (the empty column takes it)

#### Scenario: Potion lands on the belt
- **WHEN** the ground unit disappears and the belt holds one more potion of its class
- **THEN** the pickup counts as arrived

### Requirement: A kind without room is not a full belt
A belt fill SHALL treat "no column can take this potion kind" as no room for that kind: the potion stays in the pack, the other kinds are still moved, and the belt MUST NOT be marked full. A potion MUST NOT open an empty belt column when that leaves a planned kind with no column, in the belt fill and for ground potions. Source: docs/fixbelt2.md.

#### Scenario: Mana potion, no free box in a mana column
- **WHEN** a sash holds healing in three columns and a full mana column, and the pack holds mana potions
- **THEN** no fill candidate exists, nothing is clicked and the belt is not marked full

#### Scenario: Healing would take the last mana column
- **WHEN** healing fills three columns of a Sorceress belt and a healing potion lies on the ground
- **THEN** it is not wanted, and a mana potion is

### Requirement: Trips must buy something
With `town.tripMinSpend` 60 (default 0 = off) a non-emergency town trip SHALL start only when the planned purchase is worth at least 60 gold or the belt holds no healing potion, so that no trip burns a portal scroll for under 30 gold. OFF by default (the A/B of 2026-10-31 failed its gates). Source: docs/goldfloor.md finding 2 (83 percent of NAVSOAK trips spent under 30 gold). Reconciled 2026-11-04 (source: 2026-11-04-goldfloor).

#### Scenario: Small plan
- **WHEN** the belt holds 2 healing potions, the plan is one 30-gold potion and a portal scroll is held
- **THEN** no trip is raised

#### Scenario: Empty belt
- **WHEN** the belt holds no healing potion
- **THEN** a trip is raised

### Requirement: No scroll rebuy while poor
With `town.tripMinSpend` above 0 (default 0 = off) a character with gold under the kit line and fewer than 2 healing potions SHALL spend on healing potions first and SHALL NOT buy a portal scroll in the same visit. Source: docs/goldfloor.md (two scrolls bought in 25 s at gold 209). Reconciled 2026-11-04 (source: 2026-11-04-goldfloor).

#### Scenario: Gold 209
- **WHEN** a visit starts with 209 gold and 1 healing potion
- **THEN** healing potions are bought and no town portal scroll is

### Requirement: Early mana cap
With `town.mpCapLevel` 10 and `town.mpCapEarly` 1 (defaults 0 = off, the old mana buying) a visit below clvl 10 SHALL buy at most `town.mpCapEarly` mana potions and a low mana belt SHALL NOT raise a trip. Source: docs/goldfloor.md finding 3 (mana 74 to 305 gold/h at 60 gold each). Reconciled 2026-11-04 (source: 2026-11-04-goldfloor).

#### Scenario: Clvl 6
- **WHEN** a clvl 6 character has an empty mana belt and 400 gold
- **THEN** one mana potion is bought and no `mp belt columns` trip is raised

### Requirement: Sell trips only when worth it
A sell trip for a poor character SHALL start only when the sellable items reach the poverty sell count (4 without a portal scroll, 6 with one) on every trip trigger, so no second trigger reopens a trip below that count. Source: docs/basics-audit.md row 4 and the G2 audit item (an audit label, not the retired gate name); docs/d2d-economy.md Fixes (+28 percent XP/h from the first door). Reconciled 2026-11-04 (source: 2026-11-04-m1-trips).

#### Scenario: Five items with a scroll
- **WHEN** gold is under 150, a portal scroll is held and 5 sellable items are in the pack
- **THEN** no sell trip starts

### Requirement: Gold trip above the keep line
When carried gold exceeds the keep line (300 + 50 x clvl) by more than the trip margin the bot SHALL raise a town trip with reason `gold` at the next safe moment, so a death does not drop the surplus. Source: docs/basics-audit.md E6 and G8; death-rule `Gold carry and corpse`. Reconciled 2026-11-04 (source: 2026-11-04-m1-trips).

#### Scenario: Rich in the field
- **WHEN** a clvl 6 character carries 1100 gold (keep line 600) and no hostile is near
- **THEN** a trip with reason `gold` is due

### Requirement: Pack-full threshold
The pack SHALL count as full at 4 or fewer free cells (one knob in cells, default 4), which raises exactly one sell visit; fresh loot after the visit ends in the pack. Source: docs/basics-audit.md row 14 and G9; d2-sme checklist scenario 14. Reconciled 2026-11-04 (source: 2026-11-04-m1-trips).

#### Scenario: Four free cells
- **WHEN** the pack has 4 free cells
- **THEN** one sell visit is due and the next pickup after it lands in the pack

## Pending deltas (not in force)
None. No open card carries a town-basics delta (ledger 2026-11-04).
