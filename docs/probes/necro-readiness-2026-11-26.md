# Necromancer readiness audit (card necro1, 2026-11-26)

> status: current. Updates docs/probes/necromancer-readiness-2026-11-22.md (S1 to S11 table) with the state of origin/main of 2026-11-26. Tags: V verified (how said), S read in code or data, U unverified. Drill rows run on lab setups that are CHEATED (stand-in level, gear, merc): mechanics only, never a statement about a live hero.

Verdict: NOT ready for a fresh-hero Normal Act 1 run unattended. Everything exists as data and engine code behind knobs, the Act 1 stage rows are GREEN as Necromancer in the lab (V matrix note 70), but no earned (uncheated, walked) Act 1 run has completed: the last one reached clvl 14 and never started Blood Raven (V docs/probes/necromancer-act1-drill-2026-11-22.md).

## Per concern

| concern | state | evidence |
|---|---|---|
| skill plan 1-30 | DATA, one real defect fixed here | default table `necromancer_summoner_normal30` (S: gear.DefaultBuild "nec"). It had NO `combat` key, so `planForClass` (cmd/d2bot/class.go) resolved the class default profile, whose layout has no Corpse Explosion and no Revive key, while the table names Corpse Explosion as main attack from clvl 15 (S). Fixed in this branch (entry 1). SME deltas below |
| stat plan | DATA | Vitality after clvl 2 (3 Energy + 2 Vit), Str 25 for gear (S, table tags); Strength spread rule: a stat goal cannot start at a level (S, 22 Nov audit 6.2) |
| combat profile | PARTIAL | summon/curse/golem rules are profile data (S `necro` block); caps `max` 12/4/8 are guesses, not read from skills.txt (U, S2 open); Weaken, Decrepify, Lower Resist, Fire Golem have no key because the layout stops at F8 (S) |
| Bone Armor upkeep | DONE | buff list of the summoner profile (S); cast driven by the generic buff upkeep, no Necromancer-only check (U live) |
| curses | PARTIAL | list is data (S); read-back of curse state through `debug skillstate` not wired (22 Nov audit S3) |
| mana management | PARTIAL | `armyMin` 3 skips the rest-before-pack (S necro.go); `necro.mana-budget-10min` scenario GREEN, threshold U |
| merc choice | PARTIAL | Act 2 Desert merc aura preference is class-trait data, casters want Defiance or Prayer, Holy Freeze in Nightmare (S internal/merc/policy.go AuraPrefs); no rule that a Necromancer prefers the Act 1 Rogue early (SME, below) |
| kit and belt | DONE for the caster rows | supply profiles carry `mana` lines with `manaNeedAtLeast` 2 from monastery_road on, Andariel 3 mana, 8 healing (S internal/supply/profiles.json); the Necromancer is a caster in traits (S classtraits.json). Whether a summoner needs this many mana potions is U (SME: one mana column) |
| danger and readiness floors | DATA | admission rows carry caster values: underground passage clvl 8 and 100 max life, Act 2 dungeon rows clvl +2 over melee, Arcane and Tombs 150 max life (S admission.json, all U per the file). Den: caster floor clvl 5. Andariel floor is the same for every class, 11 / 10 (user calibration 2026-11-24, S boss_levels.go): not touched |
| summon verification | PARTIAL | `CountMinions` census from the snapshot, and `combat.minionRead` (default 0) from `debug minions` (S). There is NO soak invariant like `summon-missing` (S: grep of internal/d2dsoakstat/invariants.go): an army that is dead for a whole fight raises nothing. Card |
| early economy | SAME TRAPS | potions 0 and gold 0 after the first deaths: the poverty and kit floors of admission.json are class-blind (S); a caster has the lower life floor and the same no-potion gap rule. No Necromancer poverty run exists |
| test character | MISSING | no Necromancer line in docs/probes/test-characters.md for the lab runs of 2026-11-22 (S) |

## SME comparison (d2-sme, advice only, nothing in the SME rows changed; all U unless said)

1. Corpse Explosion is overweighted: the table spends 5 points on it (clvl 15, 21, 24, 27, 29) and makes it the main attack from 15 to 30. For an unattended summoner keep 1 point (Teeth prerequisite, corpse-free fallback) and put the rest into Raise Skeletal Mage and Clay Golem.
2. The clvl 1 to 14 main attack named Raise Skeleton, which is not an attack: fixed here (Teeth).
3. Strength points at clvl 12 and 13 can wait until 17 to 20 (life first).
4. Merc: Act 1 Rogue early (cold arrow, slow), the Act 2 Desert merc (Defiance or Prayer) after Andariel.
5. Danger floors: Andariel life 180 to 200 plus 6 healing, 2 to 3 antidotes and scrolls; Act 2 sewers life 130 or more; Duriel at clvl 24 to 26 life 220 or more (no Duriel row exists); tombs only when the army has re-summon mana.
6. Belt: 3 to 4 healing columns, 1 mana column, a few Rejuvenation for bosses.
The point 1 and 5 numbers are proposals for the SME owner and the plan owner (a floor changes only with a seed set, admission.json comment); they are cards below.

## Drill rows (lab, CHEATED setups, worker host lane)

Not rerun in this branch: the one attempt on the lab of the worker host (devp29 binary; the devp30 binary is on the builder only, which did not answer) waited about 20 minutes behind another agent's drill and was cancelled for the unit tests, so nothing new. The last verdicts stand (V matrix note 70, run 20261125-141635 series, Necromancer 11): `necro.act1-andariel` GREEN, the Smith, caster belt and loop rows GREEN as Necromancer, `necro.act1-tools` RED at the reward word 0x204e (backend limit, the fork's Imbue line), `necro.tools-paladin-control` BLOCKED by its seeding. The matrix rows already carry `classes: [Necromancer]` (S scenarios JSON), so the "RED as Paladin" rows only occur when `-class` is overridden. Ready command: `labrun WORKER -- env PG_D2D_BIN=<devp30 d2d> tools/agentrun.sh necro1 --ttl 3000 -- tools/provingground/run.sh -id necro.act1-andariel,step.act1.plan_overlap.necro -class Necromancer -seed 11 -parallel 1`.

## Entries of this branch

1. Default summoner table: `"combat": "summoner"` (one copy now: internal/plandata; the internal/gear/builds copy is deleted), hotkeys of the table follow the summoner layout (a doc-only list today: S, nothing reads `Hotkeys` outside the parser), main attack for clvl 1 to 14 is Teeth. Test: `TestDefaultTableMainAttackHasAKeyInItsProfile` (cmd/d2bot) runs for Necromancer and Druid; must-fail checked (V: with the `combat` key removed it fails for the Necromancer: main attack "Corpse Explosion" has no key in profile necromancer; the Druid table passes either way). Side changes: the table copy in docs/plan/data and the leveler test that pinned the old empty combat line. Not seen in a live log (U): the default Necromancer now binds F1 to F7 as the summoner layout, not the default layout.

Test character line for docs/probes/test-characters.md (not mine to edit): none created in this branch.

## Cards (not done here)

| card | what | size |
|---|---|---|
| necro-summon-invariant | soak invariant `summon-missing`: a Necromancer fight window with zero minions after the opening cast | S |
| necro-corpse-plan | CE points 5 to 1 in the default table, SME delta 1; needs a scenario pair before it changes | S |
| necro-floors | Andariel kit/life, sewers, Duriel life floors as admission rows; seed set needed (P10) | M |
| necro-merc-rogue | merc policy: early Act 1 Rogue for casters | S |
| necro-caps | read minion caps and curse durations from skills.txt instead of the profile guesses | M |
| necro-keys | keys for Weaken, Decrepify, Lower Resist, Fire Golem (F8 is the last F key; needs a second bar or a swap rule) | M |
| necro-hotkeys-drift | build hotkeys disagree with the profile layout for most classes (S script over internal/plandata); either delete the key or make a test | S |
