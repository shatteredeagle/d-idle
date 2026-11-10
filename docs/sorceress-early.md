# Sorceress early game (clvl 1 to 12) and friends readiness (TASK-267, sorcready, 2026-11-10)

> status: proposal plus two applied data changes. The user-approved v3 table is NOT changed. Tags: V verified here (file or test named), S the SME's secondary reading, U unverified. Source plan: `agent/sorcplan` `docs/research/sorceress-table-plan.md` (not yet on main when written).

## 1. Does the wizard's Sorceress pick use sorceress_normal30? No (V)
- The wizard writes `wizard.env` with `D2D_NAME`, `D2D_CLASS` (d2d index) and `D2BOT_GOALS` only (`cmd/d2dsetup/wizard.go` `finishLocked`; asserted in `cmd/d2dsetup/d2dsetup_test.go`). No leveler table is written, and the bot has no env or flag for `levelerBuild` (only the config key, `internal/bot/config.go`).
- So a fresh Sorceress plays `fallback-lightning-v2` (BUILDV2 default, `leveler.SelectBuild`; `TestDefaultSorceressTableUnchanged`) and a fresh Paladin `paladin-normal30-v2`. `sorceress_normal30` runs only when config `levelerBuild` names it; then `planForClass` takes combat `lightning` from the table (V `TestClassPickDoesNotSelectNormal30`, `cmd/d2bot/sorcready_test.go`). The wizard's "normal30" label for the Sorceress therefore describes a table she is not given by default.
- The classpacing numbers (clvl 4 at 60 min) exist for both `sorceress_normal30` and `fallback-lightning-v2` (`docs/soaks/classpacing.md`, tags sor1 and sor2).
- Side finding (V, test run): `planForClass(Paladin, "sorceress_normal30")` returns no error (class-bound check only in `LoadBuild` callers); a wrong explicit table is not refused at the plan step. Not changed.

## 2. SME consult (d2-sme, 2026-11-10; mostly EXP, the repo has no verified per-level Fire/Charged Bolt mana or Warmth numbers)
| question | answer | tag |
|---|---|---|
| Warmth point | One point is worth it: +30 percent mana recovery (+12 per extra level); a modest gain at a 35-40 mana pool, saves maybe 10 to 20 percent of Recovery time, not the wall. Place it at clvl 2 in place of the first Charged Bolt point; the Den point stays Frozen Armor | S (class-sorceress.md) / U |
| Attack order clvl 1-12 | Fire Bolt and Ice Bolt cost about 2.5 to 3 mana; Charged Bolt about 3 mana and three bolts (crowd skill, weak on single targets); staff Attack is free but melee on a 40-life hero. Keep the profile order (Charged Bolt, Fire Bolt, Ice Bolt); below one cast use a potion, else retreat; do not stand in Recovery | U (numbers need skills.txt) |
| Frozen Armor, Static Field | Frozen Armor worth it from clvl 5, cast once per fight or area; Static Field skip (the bot cannot use it, the table spends no point on it) | U |
| Kit clvl 1-12 | 4 to 6 mana potions a trip plus the starting minor heals, then more health potions as gold allows; two health and two mana columns; 1 TP scroll plus 1 or 2 spare; town trip at mana potions 0 or health potions 1 or a full inventory | U |
| Lightning 12 / Chain Lightning 19 | Both re-hit on d2d: replace by a cold line (Ice Blast from clvl 6+, Glacial Spike or Blizzard later) per `docs/sme/class-sorceress.md`; keep Telekinesis 11 and Teleport 18 | S |
Gap named by the SME: extract Fire Bolt, Charged Bolt, Ice Bolt mana and damage per level (skills.txt) and measure mana regen per second at clvl 1 to 12.

## 3. Applied (data, no table change)
- `internal/combat/classdata/profiles.json`, profile `sorceress/lightning`: new `avoid` list (Chain Lightning, Lightning). `Profile.AttackNames` drops listed names from the band it returns (`internal/combat/classdata.go`, a generic optional field; Paladin and the Sorceress blizzard profile have none; V `TestAvoidListIsPerProfile`, `TestAttackOrder`). Reason: both are Known broken on d2d (`docs/capabilities.md`). At clvl 19+ she attacks with Charged Bolt, Fire Bolt (and Ice Bolt in the last band). Reverting is deleting the `avoid` entry. The cast-gap test lifts the list for itself (`castgap_test.go`) because it tests Lightning timing.
- No attack-order change: the generic ladder already drops to the cheapest affordable spell under `EconomyManaPct` (`internal/resource/attack.go`), and a Sorceress has no free Attack rung to reorder as the Paladin had; the SME did not ask for one. Mana-wall relief is Warmth, potions and table points, all below.
- Consequence to decide (open Q3): the table still spends points on Lightning (12) and Chain Lightning (19-30), which the bot will now never cast, so from clvl 12 the build stops growing its damage.

## 4. Proposed table diff (NOT applied, needs the user)
`build-sorceress_normal30.json` (and v3, which it equals for clvl 2..30), shown as rows:
```
clvl 2   Charged Bolt 1  ->  Warmth 1                 (Charged Bolt first point moves to clvl 3)
clvl 12  Lightning 1     ->  Ice Blast 1 (clvl 6 skill, needs Ice Bolt? check unlocks-sorceress.json)
clvl 19-30  Chain Lightning ..  ->  Glacial Spike / Blizzard line, or Charged Bolt, per the user
```
The Ice Blast prerequisite and the unlock ladder (`unlocks-sorceress.json`, `TestNoBuildRowIsRefusedByAnUnlock`) must be checked first; the cold line also changes the combat profile key (`blizzard`), so this is a new table file, not an edit. U: none of it simulated.

## 5. Open questions for the user
1. Should a fresh Sorceress default to `sorceress_normal30` (v3) instead of `fallback-lightning-v2`? Needs a wizard or default-table change; today the wizard label and the played table differ.
2. Add one Warmth point at clvl 2 (table change to the approved v3)?
3. With Lightning and Chain Lightning avoided, keep the lightning table (damage stalls at Charged Bolt from clvl 12) or switch Sorceress to a cold-line table for friends, or ship her "early game only" (clvl 12)?
4. Ship the Sorceress only after three fresh first-hour records pass (section 6)?

## 6. Live measurement plan (NOT run; needs `agent/firsthour` (TASK-245) and `agent/devreset` (TASK-263) merged and the dev sandbox)
Floors today (V `data/smoke/rules.json`): Sorceress clvl 2 at minute 20, 3 at minute 45 (baseline minus 1 from classpacing: clvl 4 at 60 min). They stay; a first-hour record at or above them is the pass line (plan section 7).
Per day, three consecutive days, one live job at a time, worker host `nice 19`:
```
tools/dev-reset.sh --to d2play-20261108 --class sorceress --name <name> --yes   # fresh hero, zero setup writes, archives the old state
# let the bot play 60 realtime minutes at 1x
tools/firsthour.sh --class sorceress [--start <RFC3339 of the first d2bot start line>]   # appends the row to docs/probes/firsthour.md
tools/dev-sandbox.sh restore --to d2play-20261108 --from last --yes   # when the Paladin hero is wanted back
```
Offline-simulated alternative: `tools/smoke.sh --class sorceress --minutes 45 --record`. Log each created hero in `docs/probes/test-characters.md`. Exit code 1 of firsthour means not a full fresh hour; nothing is written.
