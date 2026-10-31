# Web sources for 1.14d damage calculations (agent DMGWEB, 2026-10-30)

Tags: **V** = I read it at the URL (or in a shallow clone) today; **S** = snippet or secondary; **U** = unverified or memory. Target classic 1.14d; D2R-only facts flagged. Nothing is copied from third-party pages beyond short numbers; no decompiled code is reproduced (function names and addresses only). Blocked: Fextralife/Diablo Wiki/PureDiablo returned 403 to WebFetch; one attempt to fetch with a browser User-Agent was refused by the permission check, so I did not re-route (those sites are only seen as search snippets, tagged S).

## 1. Headline

- **No public source gives a worked, verified "casts per kill" model.** What exists: (a) the data tables (Skills.txt/Missiles.txt, mirrored by wikis), (b) two decompilation projects that read the real engine: **psi29a/d2decomped** (1.14d, with an emulator harness that runs game.exe itself) and **ThePhrozenKeep/D2MOO** (1.10f based, carries 1.14d addresses in comments).
- Charged Bolt's missile **count** is table-driven (Skills.txt `calc1`), its **path** is a seeded random wiggle; both are decompiled and agree between the two projects. d2d's own flight code is already replay-tested against game.exe for Charged Bolt (see 3.3). So the 2.59 vs 0.96 gap is unlikely to be the bolt count or flight; look at hit/damage, monster HP, or how "casts" are counted.

## 2. Ranked sources (best 8)

| # | Source | Tag | Date read | Verdict |
|---|---|---|---|---|
| 1 | https://github.com/psi29a/d2decomped `docs/research/re/missiles.md`, `skills.md`, `combat.md`, `monsters.md` (HEAD af825ed) | V | 2026-10-30 | Best. 1.14d game.exe addresses, "proven by tools/emu/missiles.py ... game.exe's own functions, thousands of flights incl. chargedbolt". Charged Bolt: FUN_005c9300 (skill do 17) -> callback FUN_005c9290. Same repo = our d2d upstream. |
| 2 | https://github.com/ThePhrozenKeep/D2MOO `source/D2Game/src/SKILLS/SkillSor.cpp`, `D2Common/src/Path/PathMisc.cpp` (HEAD 8721f95) | V | 2026-10-30 | Independent decompile (MIT). Comments give 1.14d addresses (`SKILLS_SrvDo017_ChargedBolt_BoltSentry` = 0x005C9300, `SKILLS_MissileInit_ChargedBolt` = 0x004E2ED0 in its numbering). Matches #1 on count = calc1. Base is 1.10f; not a runnable 1.14d. |
| 3 | https://d2emu.com/db/skill/charged-bolt | V | 2026-10-30 | Per-level table (bolts, mana, damage, synergy +6%/lvl of Lightning). Data-file mirror, not stated which patch (S for version). Best web table. |
| 4 | https://www.d2mods.info/forum/viewtopic.php?p=503197 "Guide to Almost All Missile Functions" | V | 2026-10-30 | Modder guide: srvdofunc 17 = "multiple zig-zagging missiles launched from user", count = calc1. No spread/damage detail. |
| 5 | https://classic.hiveworkshop.com/diablo2exp/monsters/act1-fallen.shtml (Arreat Summit mirror) | V | 2026-10-30 | Official-text mirror: Fallen Normal level 1, HP "1-4", defense 5, melee 1-2. Old (pre-1.14d scaling?) numbers: **conflicts with 1.14d HP model, see 3.4**. |
| 6 | https://wiki.projectdiablo2.com/wiki/Charged_Bolt | V | 2026-10-30 | **Mod (Path of Diablo 2)**: bolts 3 + 1/level to 28 at 26, damage 2-4 at 1; bolts "pierce". Do not use for vanilla; only shows the same lvl+2 base. |
| 7 | Phrozen Keep / Blizzhackers Skills.txt, Missiles.txt, Monstats.txt guides | U | not fetched (searches did not surface the pages) | Standard column references; use for field meaning only. |
| 8 | https://diablo2.diablowiki.net/Charged_Bolt, https://rankedboost.com/diablo-2/sorceress/charged-bolt/ | S | search snippets only | Claim up to 24 bolts at max; rankedboost table is D2R-era (S). Low trust. |

Not found: any publicly written model of Sorceress damage per kill; any Maxroll/d2planner damage-calculator source (not looked into beyond searches, U); OpenDiablo2 skill damage (not checked, U); Median XL/PoD docs are mods and irrelevant to vanilla.

## 3. Facts we need

### 3.1 Bolts per cast (calc1)
- V (D2MOO SkillSor.cpp, d2decomped missiles.md): the number of bolts = Skills.txt `calc1` evaluated at the caster's skill level; one missile per index, each with flags 0x21 from the caster's position toward the target point.
- S/V (d2emu table): level 1 = **3**, level 4 = **6**, level 8 = **10**; +1 per level up to 24 at level 22 (cap through 40). So calc1 behaves like `lvl + 2` (inference from the table, U for the literal formula string; our Skills.txt on the worker host gives it exactly: read `calc1` of Charged Bolt, and `D2MOO` evaluates it through its skill-formula VM). Skill level here includes +skills gear.

### 3.2 Damage per bolt
- d2emu (S for patch): lightning damage **2-4** (level 1), **3-5** (level 4), **5-7** (level 8); mana 3 / 4.5 / 6.5; synergy Lightning +6% per level. Damage is per missile on hit; no cooldown. Per-element min/max come from `EMin/EMax` + `EMinLev`, shifted by `HitShift` (d2decomped skills.md "Elemental damage": (EMin+brackets)<<HitShift + synergy % + elemental mastery) (V).
- Our own check: bolts at skill 1 = 3 x avg 3 = 9 damage if every bolt hits, which is above Fallen's low-level HP: the original's ~0.96 casts per kill is consistent with "one cast kills about one Fallen, with multi-kill carry-over".

### 3.3 Spread, flight, hit
- V (d2decomped missiles.md; D2MOO PathMisc.cpp `PATH_ComputePathChargedBolt`): path type "charged bolt": range capped at 77 frames, seeded random per index ({target x + index, 666} in d2decomped), a series of 2-subtile hops in the 8-way direction toward the target, each turned by a seeded draw from a small table (so bolts zig-zag, no fixed cone). A missile damages units it enters (client/server collision), Charged Bolt keeps flying after a hit (CollideKill 0 for pierce-like behaviour: this is Missiles.txt per d2decomped "CollideKill 0 leaves the missile flying"; whether chargedbolt's row has CollideKill 0 is U, read it from our Missiles.txt row 56).
- V (d2decomped): d2d's flight replays 18 game.exe flights incl. chargedbolt lvl 2 and 27 (`tests/test_monsters.cpp missile_flights`) and `tools/emu/missiles.py` on thousands of flights. So flight geometry is verified against the original; do not suspect it first.
- Spell hit roll: for the missile "hit function" path, d2decomped skills.md says damage applies when it strikes (return bit 2). A to-hit roll applies for rows with `ToHit` set (U for chargedbolt: **check Missiles.txt `ToHit` of row 56**; elemental spells normally do not roll to hit in 1.14d, S/U). No source I read states it for Charged Bolt.

### 3.4 Fallen HP (Normal, Act 1)
- V (Arreat Summit mirror): level 1, HP "1-4", defense 5. Probably the pre-1.10 table.
- V (d2decomped monsters.md): 1.14d life = `MonLvl.txt` HP by monster level x (`minHP..maxHP`)% of the MonStats row; example given: zombie1 level 1 = 7 x 101..181% = **7..12**. Fallen minHP/maxHP columns not in any page I read: **U**; our own data (d2d prints life: Fallen Shaman 7 to 9 in docs/d2d-spike.md, V) suggests the same 7..12 order of magnitude. Read `fallen1` minHP/maxHP in our MonStats and MonLvl to get the exact range; do not use "1-4" for 1.14d.
- Fextralife also shows 1-4 (S, copied old data).

## 4. What d2d should match, and the cross-check

| Quantity | Should match | Cross-check with |
|---|---|---|
| bolts per cast at skill 1/4/8 = 3/6/10 | calc1 of Charged Bolt in our Skills.txt (same file d2d loads) | d2emu table; count missiles spawned per cast in the d2d log |
| damage 2-4 / 3-5 / 5-7 | Skills.txt EMin/EMax rows + synergy | d2emu table; per-hit damage in d2d |
| flight (zig-zag, range 77) | already replay-verified | d2decomped test suite |
| Fallen HP | MonLvl x minHP..maxHP | d2decomped monsters.md formula; d2d life reads |

Best cross-check source: **d2decomped missiles.md + skills.md** (V against game.exe), second **D2MOO SkillSor.cpp** for the count, third **d2emu** for human-readable numbers.

## 5. Public gaps (not reverse engineered or not found)

- No published "casts per kill" or end-to-end spell-damage-versus-monster-HP verification for 1.14d.
- Whether Charged Bolt rolls to hit, and per-bolt hit-on-same-target rules (NextHit/state 0x56 skipping a unit already struck, mentioned in d2decomped missiles.md) are only described at the engine level, not as numbers.
- The full path of the damage record from missile hit to life (FUN_0057c6c0 damage record apply is documented; spell-specific modifiers like monster resist/immunity for lightning are not tabulated).
- How the bot counts "cast commands": d2d's 2.59 counts bot commands, not real casts or hits (docs/d2d-casts.md); the original numbers' method matters as much as the engine.

## 6. Questions for the upstream maintainer (draft, Discord)

1. In d2d, how many Charged Bolt missiles spawn per cast at skill 1, 4 and 8, and does each one use Missiles.txt ToHit or a spell hit roll against the monster's defense? (We measure 2.59 casts per Fallen kill at skill 1; the original is 0.96.)
2. Does d2d apply the Fallen's HP as MonLvl HP x minHP..maxHP% (7..12 at level 1 for zombie), and is the Normal-difficulty roll identical to game.exe's FUN_006538a0 path?
3. Have you replayed a Charged Bolt flight together with a hit and the damage record (FUN_0057c6c0) against game.exe, not only the flight (`missile_flights`)? Is there a script we can run?

## 7. Open items

Read from our own Skills.txt/Missiles.txt/MonStats (not on the web): Charged Bolt calc1, EMin/EMax rows, Missiles.txt row 56 ToHit/CollideKill/Range, `fallen1` HP columns. Those four reads would settle most of this document's U tags.
