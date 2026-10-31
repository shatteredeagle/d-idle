# Gear gate: graded risk instead of a hard gate (SOFTGATE)

Tags: V = read from code/logs, S = secondary (docs/sme), U = judgment, EXP = needs a live experiment.
Code: `internal/resource/gear.go` (model), `internal/bot/geargate.go` (inputs, logging, area check), knobs `resource.gearGateMode|gearRiskMedium|gearRiskHigh|gearDeathsHigh` and `bot.spiralDeaths|dangerBanMin|starterBanMin`.

## Early-gear policy (EARLYGEAR, supersedes the armor rows below for clvl < `resource.earlyGearMax`, default 15)
SRC: docs/sme/early-gear-policy.md. Below the limit "no body armor" adds no points and "no weapon" counts only for Amazon, Paladin, Barbarian, Assassin (`resource.NeedsWeapon`); a caster in Normal never counts armor at any level; the rebuy buys only the needed weapon (Amazon: a bow, crossbow or javelin). `GearInput.NeedWeapon`, `GearTune.EarlyMax`. Policy off: knob 0. Tests: `TestEarlyGearPolicy`, `TestEarlyGearPolicyGate`, `TestEarlyRebuy*`, `TestEquipFoundOnlyWhenBetter`, `TestFoundGearTripOnlyForNeededWeapon`.

## Why the hard gate went
- V (docs/g1-log.md attempts 6-7): a clvl 9 Sorceress without body armor was refused every quest step ("not ready: no body armor") for 2+ hours; once the vendor stock needed more Str than she had, gold could not fix it either.
- S (docs/sme/audit-bot-vs-human.md A5, E4): a caster fights with spells, armor is a bonus; the gate also ignored life, resists and difficulty. S (class-sorceress.md section "Normal Act 1-2"): weak spots there are mana and a few archers, not missing armor. S (play-safety.md section 1): what kills is low life with no escape (no potions, no portal), not a missing Quilted Armor (+8..15 defense, U).
- The Soaky case (V docs/deathspiral.md: clvl 8, gold 0, no potions, naked, 4 deaths in 11 min) is still caught: no potions and no gold is worth the whole HIGH threshold on its own.

## The model (U: weights are judgment, all thresholds are knobs)
`resource.GearGate(in, rebuyGold, tune)` adds points; MEDIUM at `gearRiskMedium` (3), HIGH at `gearRiskHigh` (6). Only HIGH sets `Gated` (the old meaning: first field of the act only, quest steps refused as "not ready").

| factor | points |
|---|---|
| no body armor (not below earlyGearMax, not a caster in Normal) | 1 + 2 x difficulty (N 1, NM 3, Hell 5) |
| no weapon, caster | difficulty (N 0) |
| no weapon, a class that needs one (below earlyGearMax: only these classes) | HIGH alone (cannot kill efficiently) |
| difficulty | 3 x difficulty |
| act 2-3 / act 4-5 | 1 / 2 |
| max life below 40 / 250 / 500 (N/NM/Hell) | 2 |
| lowest fire/cold/lightning resist below 0 in NM/Hell | difficulty |
| area monster level vs clvl (only for a specific area): gap >= 3 / >= 6 | 1 / 3 |
| clvl >= 5, no potions and no gold | HIGH alone |
| clvl >= 5, no potions, some gold | 2 (a trip fixes it) |
| deaths in the last hour >= `gearDeathsHigh` (4) | deaths - 4 + 2 |
| deaths in this area in the spiral window >= `gearDeathsHigh` | 2 |

Resulting behaviour (tests: `internal/resource/gear_test.go`): Sorceress clvl 9 naked in Normal act 1 = low, quests proceed (V test). Barbarian naked = high. Sorceress naked in Nightmare = high; geared in Nightmare = medium. Potions and gold both out = high. Deaths: up to 3 in the hour add nothing.
Classes: caster = Sorceress/Necromancer/Druid; Paladin counts as melee (S class-paladin.md: weapon damage is the build) -- U for Necromancer summoners who also rely on minions (still treated as caster).

## What the risk drives
a) Areas: `areaAllowed`/`routeAllowed` refuse a non-safe area only when risk HIGH, computed per area (area mlvl and local deaths included), and the quest gate (`task/gate.go`) refuses steps only on HIGH. Banned areas (AREABAN) still always refuse.
b) Rebuy: `CanRebuy` (gold >= `gearRebuyGold`, 150 U) triggers the cheap rebuy trip for a missing body armor, or a missing weapon on a non-caster. A caster without a weapon is no trip (nothing worth buying early). Found-gear pickup and EQUIPNAKED equip are unchanged.
c) Log: `gear risk: low|medium|high (reasons)` whenever the level changes (not on every point change); `gear gate: on/off` still logs the Gated flips.
d) Old behaviour: `resource.gearGateMode=0` (hard) reproduces the previous binary gate exactly (V test `TestGearGateHard`).

## Hard safety net kept
No potions and no gold (HIGH), DEATHSPIRAL area bans (docs/deathspiral.md), the deaths-per-hour pause, `FleeLifePct`, retreat belt. Death tolerance was loosened, see docs/deathspiral.md.

## To verify live (EXP)
1. A naked clvl 5-10 Sorceress in Normal act 1 with potions and gold: log `gear risk: low`, quest steps run, no `gear gate: on`.
2. Same character with potions and gold at 0: `gear risk: high`, farms Blood Moor, gets out of it by selling/gold.
3. Whether low-risk naked casters really survive Cold Plains / Burial Grounds archers in Normal (S, not measured).
4. `gearGateMode=0` still gates as before.

## Town equip "not confirmed in memory" (EQUIPCONFIRM, 2026-10-30)
- Evidence (V, soak24 i4 and soak24-i5 logs of 2026-10-30): 130 `equip: failed` lines from the town chore on i4 (Leather Gloves 45, Boots 22, Skull Cap 19, Buckler 13, Small Shield 11) plus 2 on i5 (Hand Axe, still `hax x1` in the pack afterwards). The leveler's own equip step failed the same items at the same time with `leveler: slot click refused ... stillOnCursor:true`, i.e. the item was still on the cursor 2 s after the click. So the confirmation read is right (V: cursor pointer and equipment list agree); the GAME did not accept the slot click. Other slots (belt, rArm Wand, torso) were worn fine in the same run, and Heavy Gloves worked in an earlier soak. Not a requirement problem (V: armor.tsv str 15/12/22 vs the character's str 22; gloves, boots, Hand Axe need none). Root cause of the refusal stays U.
- Fix (offline, sim-tested): `equipUnit` hovers 150 ms and clicks up to 3 points of the slot box (`hand.SlotPoint`), then puts the item back and right-clicks it in the pack (U live), and logs `equip: confirmed after N ms` or `equip: not confirmed` with `because` (item still on the cursor / worn at another body slot / id gone / back in the pack). The confirm itself already polls `UITimeoutMs`.

## EQUIPLIVE: why the slot click is refused (2026-10-30, Fresha i5)
- Evidence (V, soak24-i5 log 12:07 onward + one i5 window screenshot): all body slots were already filled (Cap, quilted armor, round shield, gloves, boots, belt, sword). The pack held the replacement pieces (Cap 12.0 vs worn 3.0, Leather Gloves 22.5 vs 2.0, Buckler) drawn on a RED tile; plain pack items were not red. Equip attempts on exactly those items fail with the item left on the cursor, three clicks each, every town trip (same item id 24 repeated).
- Hypothesis (S, confidence about 65 percent): red tile = the character cannot wear it. These are magic pieces whose affixes need a level or strength above clvl 6-7 / str ~10-15, which `viable` cannot see because it checks the base row only (armor.tsv). Fits the i4 history (non-magic starting gear and torso/wand worn fine, found Gloves/Boots/Caps/Shields refused). Not proven: the affix ids are not read from memory (ItemData prefix/suffix offsets are U).
- Not the geometry: the same slot box accepted clicks earlier in the soak (Heavy Gloves, V), the clicks land (the item stays held, not dropped).
- Fix: `equip: slot click at (x,y)` is logged per click (slot, cursor item, try, rect); an item that stays on the cursor is remembered (`refused`, by unit id and clvl) and skipped by `foundGear`/`equipUnit` until the character levels up. Follow-up: read prefix/suffix ids and apply magicprefix/suffix levelreq in `viable`.
- Belt (`belt: move failed ... did not land`): the screenshot shows the belt (4 columns) full with 3 hp and 1 mana potion; `belt.Read(...).Free()` said room was left. Code lives in beltfill.go (not owned here): check Free() against the real belt width and do not drag onto a full or other-type column.

## EQUIPREQ: effective requirements before any click (2026-10-30)
Tags: V verified (code/tests), S secondary (d2-sme), EXP not yet checked live, U unverified.
- Rule (S, d2-sme): level requirement = max(base levelreq, highest magic prefix/suffix levelreq); Str and Dex from the BASE item only. `viable`/`reject` still read the base row; `unitReject` (internal/town/require.go) adds the unit-level checks, in both the rebuy stock scan and `foundGear`/`equipUnit`.
- Seam: `town.RequirementSource.Requirement(unit, base) Req{Level,Str,Dex,Unidentified,Estimated}`, `Town.SetRequirementSource`. Default `tableReq`: quality below magic = base row unchanged (V, tests); unidentified magic or better = skipped; magic with a readable name = affix tables via `data.MagicNameLevelReq` (display text of prefix/suffix looked up in magicprefix/magicsuffix `levelreq`, highest wins, conservative on duplicate display texts); anything else magic or better = estimate base levelreq + `town.MagicLevelMargin` (6, U judgement; knob `gear.magicLevelMargin` requested, the env tuning is not in this change's paths).
- d2d source: the dump's item `name` is the display name incl. affixes (S: docs/d2d-fidelity.md saw "Hand Axe of Health"; not seen for every quality). It is NOT yet exposed to town: `ItemExtra.Name` exists in d2dsrc but `d2denv.Reader` has no `ItemName(id uint32) string` (request in the report); the town picks it up by interface assertion, so adding that method switches the exact path on. Until then d2d uses the margin estimate too. The dump has no affix ids and no requirement flag.
- Game.exe source: ItemData prefix/suffix id offsets are U; nothing is read. Margin estimate only.
- Rare, set, unique: always the margin estimate (a rare has up to 6 affixes, set/unique levelreq are in other tables: U).
- Skips log `equip: skip <item> reason=<level|str|dex|unidentified|small gain|refused> need=<n> have=<n>` once per item/clvl/reason.
- Armor swap needs a WearScore gain of 3 (`minEquipGain`, EXP).
- Refused click: item goes back to a free pack cell (hand.ClearCursor), blacklisted by (unit id, clvl, str, dex) until one changes. The right-click retry is off (`equipRightClickFallback`, S: 1.14d equips by click).
- Belt: `beltRoom` = min(slots - potions, slots - occupied cells), 0 after a failed move until the belt content changes; the drag fallback aims at the first empty cell (row 0 at the bottom, U geometry) and counts a move only when the belt total rose. Root cause of the Fresha full-belt `Free()` overcount is unknown (U: belt width from the worn belt's table row vs the real belt).
- Behaviour change to know: a magic weapon or armor at a low clvl is now skipped until clvl >= base levelreq + 6 on the margin path (before, it was attempted and often refused).
