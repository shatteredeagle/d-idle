# Friendly units (FRIENDLY)

One question for every target picker: `combat.IsFriendly(snap, unit, cat)` (`combat/friendly.go`; reason via `FriendlyWhyOf`,
whole set via `Friendlies`). `Hostiles` and `hostileMissiles` use the set, so nothing in `combat` picks or dodges a friend.
Tests: `friendly_test.go` (table, one row per unit kind) and `d2/states_test.go` (child-state chain through mem.Fake).

## Rules, in order, with evidence
| Rule | Covers | Status |
|---|---|---|
| player id | the character | not in `snap.Monsters`; the old `u.ID == player id` test was WRONG: unit ids are per type (V live: Combi is player id 1 and town monster class 152 is monster id 1, which read friendly=owner). Test removed, owner-by-player now needs owner type 0 |
| monstats `npc` flag | town NPCs, quest givers | V (town NPCs read npc=1; already skipped before) |
| monstats AI in {Hireable, NecroPet, DruidWolf, DruidBear, Raven, Totem, Vines, CycleOfLife, Hydra, AssassinSentry, BladeCreeper, InvisoPet, ShadowWarrior, ShadowMaster} | hirelings, golems, Valkyrie, skeletons, mages, wolf/bear (revived-style NecroPet rows), druid wolves/bear/ravens/spirits/vines, Sorc hydras, Assassin sentries/shadows | table V (extracted monstats.tsv, every row with that AI; no hostile row shares them); behaviour live V only for Valkyrie 357, clay golem 289, necroskeleton 363; the rest U |
| class table (same units by id, plus 356 decoy, 340-343 bone prison, 358 act2 guard, hirelings 271 338 359 560 561) | no monstats row or no AI column | 271 V live (hired Rogue read owner 0/0); 338, 359, 560, 561 row V, not hired live; 358 and 356 and 340-343 U |
| child state 105 on the unit's stat list | any Necromancer minion incl. Revive'd monsters that keep their own class | V on golem + 2 skeletons, absent on hostile Zombie/Quill Rat and on 4 hostile Fallen (live 2026-10). Also V present on EVERY friendly seen live: all town NPCs, the hired Rogue, Hydra heads. So it reads as a generic non-hostile marker, not a minion marker; meaning of 105 still U; Revive U |
| owner words: owner id = player id (any owner type), or owner is a friendly unit (chain, 4 passes) | Valkyrie, summons of minions | V Valkyrie (type 0, owner = player id); reads 0 for necro minions and the Rogue, hence the kinds above |

Not units in `snap.Monsters`: Fire Wall, Blizzard, traps that are objects, Amazon Decoy missiles etc. are objects or
missiles; missiles are filtered by owner (friendly set or player id). Party members are players, never in `Monsters`.
`d2.Reader.ReadStates(statList)` reads the child chain (StatList+0x3c newest, +0x2c prev, +0x14 state); `MonsterInfo.States` and
`HasState` expose it. It does not check the child's owner word (the player-aura code does).

## Switched (FRIENDLY2)
`bot/observe.go` (`hostile`, `hostileNear`, kill bookkeeping, killer pick), `bot/task.go` and `cmd/d2bot/display.go` (monster Friendly flag)
now call `combat.IsFriendly` / `Friendlies` through `bot.FriendCatalog(cat)` (adapter: bot `MonsterMeta` gained `AI`). `internal/boss`:
`View.Hostile` was already filled from `combat.Hostiles` in `cmd/d2bot/boss.go`; only its comment was stale. `town/merc.go:isMercClass`
stays: it asks "is this a hireling", not "is it friendly" (golems are friendly, not mercs).

## Needs a live check (d2-live)
1. Necro: golem, skeletons, mages, a Revive'd monster: `Hostiles` empty, state 105 on the revived one.
2. Druid: wolves, bear, vines, ravens, spirit totems: class id, owner words, AI table match, states.
3. Amazon: decoy (356), dopplezon after cast; hired Rogue 271 stays unattacked.
4. Assassin: shadow warrior/master, Wake of Fire / Lightning sentries, blade creeper: class ids 410-418.
5. Sorc: Hydra 351-353 as monsters (not objects).
6. Act 2 / 3 / 5 hirelings (338, 359, 560, 561) and Act 2 guard 358.
7. Hostile control: confirm a hostile Totem/turret (348-350, 372) is NOT matched, and states other than 105 on hostile summoned monsters.

## Live results (FLIVE, i2 Combi: Sorceress, cheated to clvl 30, 2026-10-29; ledger actor flive)
Tool: `tools/rigs/friendlive [radius]` (read-only dump: class, monstats AI, owner words, child states, IsFriendly verdict).
Cheats used (reason in ledger): gold 2000 (hire), heal, level 30 (Hydra needs clvl 30 + Warmth, Fire Ball, Enchant; points spent by UI clicks).
| Check | Result |
|---|---|
| 1 Necro | not testable (Combi is a Sorceress; no necro on a free instance) U |
| 2 Druid | not testable, U |
| 3 Amazon decoy 356 / dopplezon, hired Rogue 271 | Rogue 271 hired from Kashya (gold 2000 -> 1705): class 271, AI Hireable, owner type 0 id 0, state [105], IsFriendly true via hireling-ai: V again. Decoy U |
| 4 Assassin | not testable, U |
| 5 Sorc Hydra | V: Hydra cast gives three monsters (units of type monster) classes 351, 352, 353 (one per head), AI Hydra, state [105], owner words 0/0 (NOT the player), all friendly=true via summon-ai while 4 Fallen read friendly=false. The AI rule is what makes them friendly; owner words and the old player-id test would not have |
| 6 Act 2/3/5 hirelings, guard 358 | not reachable (Act 1 only), U |
| 7 Hostile control | V: Fallen (states [], owner 0/0) never matched. Hostile Totem/turret 348-350, 372 not met, U |
Findings: (a) town NPCs and a hostile-looking town unit (class 179) carry state 105, so the state rule would also flag any monster with 105; harmless for hostiles seen so far (none had it). (b) monster id 1 collides with player id 1: fixed (combat/friendly.go, test "a hostile monster whose id equals the player id"); owner chain and missile-owner now respect owner type (player = type 0). (c) Hydra owner words read 0, same as golems and the Rogue; only the Valkyrie reads the owner.
Dead end: binding a non-damage skill (Hydra) with `combatlive bindkey` fails (KnownSkills only lists attack skills); the right-skill popup was used by hand (popup order is not fixed, a click on the first slots picks Scroll/Book skills).
