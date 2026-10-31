# Boss arena procedures

Scope: the Fighter-facing procedure for each boss that a quest or farm run needs. Source of positions and sequences: `docs/archive/prior-art/02-runs.md` (Kolbot,
world coordinates; S as a source, U for 1.14d). Mechanics and rewards: `docs/goals/execution.md`. Hazard rules by difficulty: `docs/plan/data/danger-table.json`
(R). Reader's first rule: **fixed-layout levels may use the coordinates as a cross-check; random-layout levels must be found through presets**, because levels
differ per game (V nav.md "seed independence"; V map.md presets).

## 1. Common boss procedure (one `BossScript` per boss)

```go
type BossScript struct {
    Name, Act   int; Boss string
    WP          int                  // waypoint level to start from
    Route       []int                // area ids after the WP (ExitTo chain)
    Anchor      Anchor               // preset class + offset OR fixed point (Fixed true only for fixed layouts)
    Stand       []StandPoint         // ordered standing points (kite spots), optional
    Pre         []PreAction          // buffs, Static Field count, antidotes, resist check
    Hazards     []Hazard             // see section 3
    Kill        func(ctx, Deps) error // default: Fighter.Kill(bossUnit) with Safety on
    Done        func(w WorldView) bool // monster dead (mode/life) AND loot picked
    Exit        ExitPlan             // town portal after, portal to next area, or stay
    NoTP        bool                 // Ancients: no town portal, no leaving the area
}
```
Default sequence: `Chores` (potions, TP, repair, merc revive) -> `Travel(WP)` -> `ExitTo` chain -> walk to anchor -> `Pre` -> locate boss (unit scan by class id or
unique/superunique flag; poll 16 x 250 ms, V pattern in prior-art 02) -> `Fighter.Kill` -> wait 2 s for minions -> `Looter.PickUp` (quest items always) -> `Exit`.
Per-boss `Done` also emits a `kill` event with the boss flag (V events) so goal counts work with no new code.

Detection of "boss exists in this game": monster list scan by class id (Pindleskin 786, Nihlathak 794 from Kolbot, V as source, U for 1.14d), plus the
super-unique name index in monstats. Not found after N polls: `ErrBossNotFound` (a fault Kind, not an exception): rescan presets, then abort the run and
let the Cycler start a new game.

Death near a boss: death policy in bot.md already bans the area after the spiral limit; the boss script adds nothing except `Hazard` data for the danger weights.

## 2. Per-boss procedures

Columns: **arena** (area id U from runs.json/areas.json), **anchor** (V = we have the mechanism, preset type exists in `d2map`; the specific preset class number is U until dumped, V-09), **stand**, **hazards**, **kill/exit**.
Resist/immunity: read stat ids 39/41/43/45 (V ids, tracking.md) on the boss unit when it is first seen; expected values below are U and must be replaced by the monstats.txt dump (offline check O-1 in verification.md).

### Critical path (quest required)

| boss | arena | anchor | stand and approach | hazards (difficulty) | kill and exit |
|---|---|---|---|---|---|
| Andariel | Catacombs 4 (37), WP Catacombs 2 (35) | random layout: walk Catacombs 2->3->4 via `ExitTo`; boss near the far end, locate by monster scan after entering 4 | Kolbot: (22549,9520) with alternating side steps when the boss is within 10 (U, fixed relative?); keep 10+ away | poison cloud and melee poison: carry antidotes when poison res < 75% (SP rule, prior-art 02); Normal clvl floor 22 (runs.json) | kill by scan; exit by `ToTown` then Warriv hop (acts.md). Requires `Talk(Warriv)` after |
| Duriel | Duriel's Lair (73) via Canyon of the Magi WP (46) | the *correct tomb* has the Orifice object; find with `Presets(HoradricStaffHolder)` (U class); inside the tomb walk to preset offset (-11,+3) until the portal object is within 20 | Kolbot sorc: when Duriel is within 10 step to (22638,15722 or 15693) (U); no dance if Frozen Orb/Blizzard slows | very hard melee; cold slows him (S); needs the Horadric Staff placed (staff quest) | kill; after kill Tyrael appears: talk (acts.md). Normal floor clvl ~24 (runs.json) |
| Mephisto | Durance of Hate 3 (102) via WP Durance 2 (101) | fixed layout; stand (17566,8069); KO (17568,8069) (U) | optional moat trick (wait, lure to >27, cast from the moat) is a *second-phase* option; first version: ranged kite at radius 18 sidestepping when his mode is `Attacking2` (the ranged cast) | Attacking2 sidestep; Mana Burn minions in Hell; Normal floor 28 | kill, then the red portal at (17590,8068) opens; take it (acts.md A3->A4). Pick up the *Mephisto's Soulstone* (quest item) if Hell's Forge is wanted |
| Diablo | Chaos Sanctuary (108) via WP River of Flame (107) | fixed geometry with 2 seal layouts each (A/B, detected by seal preset coordinates, prior-art 02) | node path per seal (prior-art 02 lists the nodes), clear radius 30 at each; wait position near (7791,5293) (star); casters at 35 | boss wave per seal: Vizier, Seis, Infector (each seal opens its elite pack); fire/lightning walls (Kolbot adds +100 cost per "fire" object within 4); Diablo spawn watcher (monster list scan every tick, abort seal logic) | after Diablo: Tyrael talk, portal (acts.md). Hell floor clvl ~80 (prior-art 01) |
| Baal | Throne of Destruction (131) via WP Worldstone Keep 2 (129) | throne rectangle fixed (x 15072..15118, y < 5080); wave identification by monster class inside it | corners/positions in prior-art 02; wave 5 at (15094,5029) | waves 1-5 (Warped Fallen, Mummies/Cold Mage, Council, Venom Lord, Lister); hydras of wave 3 are not the player's: retreat to a corner; Souls (Burning Soul) and Dolls (Soul Killer) abort for non-immune builds | after waves, wait <= 3 min for throne Baal to leave, portal to Worldstone Chamber (132), stand (15134,5923), kill Baal; clear the difficulty. TP allowed |

### Quest bosses off the critical path or semi-optional

| boss | arena | notes |
|---|---|---|
| Blood Raven | Burial Grounds (17) | ranged fire arrows; kite; resurrects fallen (S). Reward: Kashya's merc hire. Normal floor ~clvl 6 |
| The Smith | Barracks (28) | Tools of the Trade; drops the Malus (quest item, always pick up) |
| Countess | Tower Cellar 5 (25), chain Forgotten Tower (20) -> 21..25 from WP Black Marsh (6) | find `SuperChest` preset in level 5; two layouts: chest x 12565 stand (12578,11043), x 12526 stand (12548,11083) (prior-art 02, U); her chests excluded from chest opening; runes are the reason to farm; floor 14 Normal |
| Radament | Sewers 3 (49), preset `HoradricScrollChest` | quest item chest next to the boss; Book of Skill must be *used* (right click) -> stat 5 +1 (V stat) |
| Summoner | Arcane Sanctuary (74), WP at Arcane | preset `Journal` offset (-3,-3); red portal appears after the kill; Teleport nearly required (maze) |
| Council (3) | Travincal (83) | Khalim's Will needs the Flail from one of them; split targets; casters; Mana Burn in Hell |
| Izual | Plains of Despair (105) via City of the Damned 106? (U): no WP in 105, walk from the Fortress | freeze and Iron Maiden-like; 2 skill points (V) |
| Hephasto | River of Flame (107), `HellForge` preset | only for the Hellforge quest; hammer drop |
| Shenk | Bloody Foothills (110) | super-unique, stand (3876,5130) Kolbot (U); socket reward |
| Nihlathak | Halls of Vaught (124), WP Halls of Pain (123) | preset `NihlathaksPlatform`; Corpse Explosion; abort if revived Tomb Vipers exist; Mana Burn in Hell |
| Ancients (3) | Arreat Summit (120) | **NoTP**: leaving the area or portal resets them (V battle.net); `Fighter.Safety` TownPortal action must be suppressed (execution.md). Level gates 20/40/60 |
| Pindleskin | Nihlathak's Temple (121) via Anya portal | farm; stand (10058,13234..13236) Kolbot; revived vipers abort |
| Eldritch | Frigid Highlands (111) | farm; stand (3745,5084) Kolbot |
| Cow King | Cow level | needs Wirt's Leg and the Cube; avoid his room pack (rooms marked "bad", prior-art 02) |

## 3. Hazards (data, not code)

`Hazard` = condition + response, evaluated by the Fighter's Safety each tick (the Fighter owns the dodge/step code; this table is the data it reads).

| hazard | detect | response | boss |
|---|---|---|---|
| ranged cast (`Attacking2`) | monster mode value (U, V-10) | sidestep to a point at radius 18, +-30/60/90 deg | Mephisto |
| poison cloud on the ground | missile/object list near player | step 10 away; antidote use when poison res low | Andariel |
| fire wall objects | object list, class "fire" | add cost to candidate standing points | Diablo |
| hydra (not ours) | monster class hydra, not owned by player | go to a corner, wait for expiry | Baal wave 3 |
| Souls/Dolls on entry | monster class check at arrival | leave game (Cycler) if not immune | Baal throne |
| revived Tomb Vipers | monster class check | abort run | Nihlathak, Pindleskin |
| boss minions with Mana Burn | affix list (danger-table.json) | ranged kill first, drink mana | Hell Council, Mephisto, Nihlathak |
| walk-through teleport fight | monster position jumps | do not chase | various |

## 4. Teleport and standing points

Kolbot's fixed coordinates assume Teleport (the Sorceress build has it at clvl 18, docs/plan/leveling.md). A walking bot (Paladin, pre-Teleport) needs the same
arena but traverses by `Route`+`ExitTo`; stand points that sit on the far side of an obstacle are reachable only with Teleport, so a `StandPoint` carries
`NeedsTeleport bool` and the script degrades to the nearest walkable point (nav `GoTo` tolerance) when the character has no Teleport. The class docs
(docs/classes) own the per-class Fighter profile; this file only fixes the shape.

## 5. What to write first (value order)

Andariel, Duriel, Mephisto, Diablo, Baal (the critical path through Normal), then Countess (rune income), Pindleskin/Eldritch/Shenk (Hell MF), Nihlathak.
Each script is ~100 lines of data plus a shared executor; the executor and `Hazard` evaluation live in the combat/bot boundary (packages.md `internal/bot/bosses`).
Offline test: a replay of a recorded arena (WP-REPLAY) with a fake Fighter checks that the script reaches the anchor and emits the right Fighter calls.
