# Cast timing (FCR) sources and verdicts, classic 1.14d

Research date 2026-10-30. Docs only: no game, no code. Goal: how sure can we be that a d2d patch
using the community cast-frame formula and tables is correct for 1.14d.
Tags: V = I fetched and read it (URL given), S = snippet or secondary, U = unverified or from memory.
D2R-only facts are flagged "D2R". Mods are flagged "MOD".

## 0. Bottom line

- The formula, the minus-one convention and the Lightning/Chain Lightning exception are **CONFIRMED** by two
  independent kinds of source: the reverse-engineered engine code (D2MOO, with 1.14d addresses) and the
  community tables, which my own script reproduces exactly from the formula (section 3).
- The Assassin row in the brief is wrong by one: correct is **16..9**, not 17..10 (base 17, minus one).
- d2d's own upstream doc (psi29a/d2decomped `docs/research/re/skills.md`) gives the same breakpoints but
  **without the minus one** (Sorceress 14..8 instead of 13..7) and wrongly calls Charged Bolt, Nova and
  Thunder Storm S1 casts. Anything ported from that doc is off by one tick unless corrected.
- Nobody has published an in-game frame count (video or memory sampling) that I could find. Everything
  is formula plus data derived. A one-off live measurement on a throwaway character settles the last
  unknown (section 8).

## 1. Sources actually read

| ID | Source | URL | Date read | Kind | Tag |
|---|---|---|---|---|---|
| D2MOO | `Units.cpp` of D2MOO (ThePhrozenKeep), functions annotated with 1.10f/1.13c/1.14d addresses | https://raw.githubusercontent.com/ThePhrozenKeep/D2MOO/master/source/D2Common/src/Units/Units.cpp | 2026-10-30 | decompilation-based reimplementation, derived from game code | V |
| D2MOO-H | `Units.h` (D2UnitStrc layout) | https://raw.githubusercontent.com/ThePhrozenKeep/D2MOO/master/source/D2Common/include/Units/Units.h | 2026-10-30 | same | V |
| D2D | psi29a/d2decomped `docs/research/re/skills.md` (lines ~374-395, 1084-1160), `components/rules/combat.hpp`, `components/rules/sequences.hpp`, `tests/test_combat.cpp` | https://github.com/psi29a/d2decomped (read via gh api) | 2026-10-30 | disassembly of 1.14d Game.exe plus animdata.d2 read from the 1.14d MPQ | V |
| SK113 | Skills.txt, game version 1.13 | https://github.com/fabd/diablo2 `code/d2_113_data/Skills.txt` | 2026-10-30 | primary game data (1.13, not 1.14d) | V |
| ALMAR | Almar's Guides, Faster Cast Rate | https://almarsguides.com/Computer/Games/Diablo2/Misc/GameMechanics/Breakpoints/FasterCastRate/ | 2026-10-30 | community table, no formula, no date, no version | V |
| PURE | PureDiablo breakpoints, "Updated on May 24, 2022" | https://www.purediablo.com/?p=8211 | 2026-10-30 | community table (WebFetch extraction lost the frame cells, only breakpoint lists read) | V for FCR values, S for frames |
| FEXT | Fextralife D2 wiki, Faster Cast Rate | https://diablo2.wiki.fextralife.com/Faster+Cast+Rate | 2026-10-30 | wiki, formula plus base table, no version, page chrome says Patch 2.4 so D2R-era | V (D2R-era page, values are classic engine) |
| MAXR | Maxroll D2 breakpoints and animations | https://maxroll.gg/d2/resources/breakpoints-animations (fetched with the worker host headless browser) | 2026-10-30 | guide, explicitly D2R | V (D2R) |
| PD2 | Project Diablo 2 wiki | https://wiki.projectdiablo2.com/wiki/Magic_Find (page holds the FCR section) | 2026-10-30 | MOD (PD2 changed Lightning/Frozen Orb) | V (MOD) |
| D2AND | N1v0k/D2Android-Releases `D2Breakpoints.java` | https://github.com/N1v0k/D2Android-Releases (path `source/winlator/app/src/main/java/com/winlator/d2/D2Breakpoints.java`) | 2026-10-30 | hobby project that reads animdata.d2 and cross-checks published tables; comments stripped from the file | V (code), validation claim U |
| LL | Lurker Lounge threads 6028, 11026, 4657 (2005-2006 posts by theorists) | https://lurkerlounge.com/forums/printthread.php?tid=6028 | not read: HTTP 406/409 to curl and WebFetch | forum | S (only via a web-search summary: "lightning-type spells use a sequence value, the trailing minus one is omitted, sequence speed 256") |
| ARREAT, AMAZON BASIN | classic.battle.net/diablo2exp, amazonbasin.com | Wayback availability API returned no snapshot for any FCR page I guessed; web.archive.org is blocked for WebFetch | - | - | NOT READ. Arreat Summit never had frame tables as far as I recall (U) |
| DIABLOWIKI, FANDOM | diablo2.diablowiki.net/Faster_Cast_Rate | https://diablo2.diablowiki.net/Faster_Cast_Rate | 403 | - | NOT READ |
| WOWHEAD | wowhead D2 breakpoints guide | https://www.wowhead.com/diablo-2/guide/breakpoints-by-class-fcr-ias | 2026-10-30 | page held no tables | V (empty) |

Did not use: Reddit, Diablo2.io, Blizzard forums, Nonlnear's posts (not found by search, not fetched).

## 2. The engine code (the strongest source)

D2MOO (a project that reconstructs Diablo II's D2Common.dll/D2Game.dll from the original binaries; each
function carries `//1.13c: 0x...` and `//1.14d: 0x...` addresses, so the code was matched to 1.14d). V.

**Cast rate** (`UNITS_UpdateCastAnimRateAndVelocity`, no 1.14d address printed, called from the unit
animation-rate function at 1.14d 0x00621780 area):

```
EFCR  = (120 * FCR) / (FCR + 120)          // integer division
rate  = min(100 + EFCR, 175)               // so EFCR is capped at 75
speed = rate * pAnimData->dwAnimSpeed / 100  // integer division = floor(...)
wAnimSpeed = clamp(speed, 0, 0x7FFF)
dwSeqSpeed = wAnimSpeed
```

The same table `gaAnimModeStatBasePercentage` (1.14d 0x006E8E24) has 120 for IAS/FHR/FCR/FBR and 150 for
FRW. Stat id for FCR is STAT_ITEM_FASTERCASTRATE (105).

**Which animations get this speed** (`UNITS_IsSeqAnimSpeedModulatedByFCR`, 1.14d 0x006216E0):
a player in PLRMODE_CAST always; a player in PLRMODE_SEQUENCE **only if the used skill's `seqtrans`
equals CAST (SC)**. Monsters in MONMODE_CAST. Everything else is false.

**Where the minus one comes from** (`UNITS_IsAtEndOfFrameCycle`, 1.14d 0x006217C0), frames are 8-bit
fixed point (`frames << 8`):

- plain animation (no `pAnimSeq`): ends when `current + wAnimSpeed >= frames*256`. Starting at 0 this first
  holds at tick k = ceil(frames*256/speed) - 1. **One tick less than the naive count.**
- sequence animation (`pAnimSeq` set): `dwFrameCountPrecise = seqFrames << 8`, `UNITS_StopSequence` subtracts
  `dwSeqSpeed` per tick, ends when `<= 0`: k = ceil(seqFrames*256/speed). **No minus one.**
  `UNITS_InitializeSequence` starts with `dwSeqSpeed = 256` (so FCR-less sequence skills play at the raw count).

Caveats: this is a reimplementation by humans, function names are theirs, and the exact tick at which the
server sees "action complete" relative to the tick the mode started is not stated in the code I read (open, section 6).

## 3. Reproduction of the community tables (my own computation)

I implemented `frames = ceil(256*base/floor(speed*(100+EFCR)/100)) - 1` (no minus one for the sequence) with
EFCR = min(floor(120*FCR/(120+FCR)), 75) and compared with ALMAR (V), PURE (V) and the D2AND table (V):

| Row | base@speed | Breakpoints reproduced (FCR:frames) | Matches ALMAR, PURE, D2AND |
|---|---|---|---|
| Sorceress | 14@256 | 0:13 9:12 20:11 37:10 63:9 105:8 200:7 | yes |
| Sorc Lightning / Chain Lightning (sequence, 19) | 19@256 | 0:19 7:18 15:17 23:16 35:15 52:14 78:13 117:12 194:11 | yes |
| Amazon | 20@256 | 0:19 7:18 14:17 22:16 32:15 48:14 68:13 99:12 152:11 | yes |
| Necromancer | 16@256 | 0:15 9:14 18:13 30:12 48:11 75:10 125:9 | yes |
| Paladin | 16@256 | 0:15 9:14 18:13 30:12 48:11 75:10 125:9 | yes |
| Assassin | 17@256 | 0:16 8:15 16:14 27:13 42:12 65:11 102:10 174:9 | yes |
| Barbarian | 14@256 | 0:13 9:12 20:11 37:10 63:9 105:8 200:7 | yes |
| Druid human | 15@208 | 0:18 4:17 10:16 19:15 30:14 46:13 68:12 99:11 163:10 | yes |
| Druid wolf | 15@229 | 0:16 6:15 14:14 26:13 40:12 60:11 95:10 157:9 | yes (speed 229 from FEXT, base 15 by fit) |
| Druid bear | 15@228 | 0:16 7:15 15:14 26:13 40:12 63:11 99:10 163:9 | yes (speed 228 from FEXT, base 15 by fit) |
| Necro vampire | 15@160 | 0:23 6:22 11:21 18:20 24:19 35:18 48:17 65:16 86:15 120:14 180:13 | yes (base 15@160 from MAXR) |

Every breakpoint of every row matched, including the 8-9 breakpoint tail values (e.g. 152, 163, 174, 194). This
is a strong check: a wrong base, speed, rounding or minus-one would shift at least one breakpoint.
Table sources do not say how they were derived (observation or formula). A formula-derived table would
make the match circular, which is why the code reading (section 2) and the in-game open question matter.

## 4. Claim by claim

| # | Claim | Sources agreeing | Sources disagreeing | Verdict |
|---|---|---|---|---|
| a1 | EFCR = floor(120*FCR/(120+FCR)), cap 75 | D2MOO V (code, `(120*f)/(f+120)`, rate cap 175); FEXT V; MAXR V (D2R); D2AND V; D2D V | none | **CONFIRMED** |
| a2 | speed = floor(animspeed*(100+EFCR)/100), animspeed 256 default | D2MOO V (integer division), FEXT V, MAXR V, D2AND V | D2D formula `256 + floor(256*EFCR/100)` is identical for speed 256, not for 208 (Druid: D2D omits the speed) | **CONFIRMED** for 256; for Druid human use 208 (D2MOO reads `pAnimData->dwAnimSpeed`) |
| a3 | frames = ceil(256*base/speed), plain cast one tick less | D2MOO V (end test `cur+speed >= N`); MAXR V (`ceil(... - 1)`); D2AND V (`... / step - 1`); ALMAR and PURE tables reproduce only with the minus one (section 3) | D2D (docs and `speed_frames`, tests assert 14..8 for Sorceress) has no minus one | **CONFIRMED** for the engine rule. The d2d upstream convention differs by exactly one |
| b1 | Bases: Sorc 14, Amazon 20, Necro 16, Paladin 16, Barbarian 14, Assassin 17, Druid 15@208 | D2D V (animdata.d2 of 1.14d: AMSCHTH 20, AISCHTH 17, BASCHTH 14, DZSCHTH 15, NESCHTH 16, PASCHTH 16, SOSCHTH 14); FEXT V (same numbers, Druid speed 208); table reproduction | MAXR V lists Necromancer 15 and Druid 16@208, which would give 0-FCR frames 14 and 19, contradicting every table including MAXR's own Amazon table; treat as a MAXR typo or D2R change (U) | **CONFIRMED** (2 sources plus reproduction) |
| b2 | "AISCHTH 17?" | D2D V 17; FEXT V 17; MAXR V 17 | the brief's "18" | **CONFIRMED 17** (Assassin 0-FCR cast = 16 frames) |
| b3 | Druid wolf 229, bear 228, vampire 160 speeds | FEXT V (wolf 229, bear 228), MAXR V (229, 228, vampire 160); base fit 15 | none | **LIKELY** (bases by fit, not read from animdata) |
| c1 | Sorc 0/9/20/37/63/105/200 -> 13..7 | ALMAR V, PURE V (FCR list), D2AND V, FEXT/MAXR formulas, section 3 | D2D gives 14..8 (same breakpoints) | **CONFIRMED** (frames 13..7) |
| c2 | Sorc Lightning, Chain Lightning 0/7/15/23/35/52/78/117/194 -> 19..11 | ALMAR V, PURE V, FEXT V ("separate breakpoints", base 19), MAXR V (exception formula, base 19), D2AND V, D2D V (19-step sequence 12 in sequences.hpp), D2MOO V (sequence rule) | PD2 MOD V uses 15 frames at 0 FCR and lists Chain Lightning and Frozen Orb as slow (MOD change) | **CONFIRMED** (2+ independent, code-level) |
| c3 | Amazon 0/7/14/22/32/48/68/99/152 -> 19..11 | ALMAR V, PURE V, D2AND V, section 3 | MAXR (D2R) splits Amazon by weapon, adding intermediate "action flag" rows (frames stay 19..11 at the same main breakpoints; extra rows are the action flag, D2R-era table) | **CONFIRMED** for the cast end. Whether the weapon split exists in 1.14d is not known (U) |
| c4 | Necro 0/9/18/30/48/75/125 -> 15..9; Paladin identical | ALMAR V, PURE V, D2AND V | none | **CONFIRMED** (claim was right; Maxroll Necro base is the odd one) |
| c5 | Assassin 0/8/16/27/42/65/102/174 -> 17..10 | breakpoints: ALMAR V, PURE V, D2AND V, D2D V | frames: ALMAR V, D2AND V say **16..9**; D2D says 17..10 (no minus one) | breakpoints **CONFIRMED**; frames in the brief are **WRONG by one**, correct is 16..9 |
| c6 | Druid human 0/4/10/19/30/46/68/99/163 -> 18..10 | ALMAR V, PURE V, D2AND V, FEXT V (208) | D2D lists Druid 0/9/19/32/54/86/152 (assumed speed 256), wrong for human form | **CONFIRMED** (D2D Druid row is wrong) |
| c7 | Barbarian 0/9/20/37/63/105/200 -> 13..7 (claim said base 14) | ALMAR V, PURE V, D2AND V | none | **CONFIRMED** |
| d1 | Lightning and Chain Lightning use the 19-step sequence, no minus one; speed from `anim=SQ seqnum=12 seqtrans=SC` | SK113 V (Lightning, Chain Lightning: `anim=SQ`, `seqtrans=SC`, `seqnum=12`); D2D V (sequence 12 is 19 steps for all weapon classes); D2MOO V; MAXR V | none | **CONFIRMED** |
| d2 | All other Sorc skills (Fire Bolt, Charged Bolt, Nova, Static Field, Teleport, Fire Ball, Fire Wall, Blizzard, Frozen Orb, Glacial Spike, Meteor, Thunder Storm, Hydra, Ice Bolt, Frost Nova, Ice Blast, Blaze, buffs) use the plain cast | SK113 V: all are `anim=SC seqnum=` (empty) | D2D V doc says "Charged Bolt, Nova, Lightning, Chain Lightning, Thunder Storm are S1 not SC" (contradicted by the 1.13 table, and by the 13..7 tables: S1 is not the same row); PD2 MOD makes Frozen Orb slow | **CONFIRMED** for 1.13 data; 1.14d Skills.txt not read but this block of the file did not change between 1.13 and 1.14 as far as I know (U) |
| d3 | Inferno uses a sequence (seqnum 6, 15 steps) | SK113 V (`anim=SQ seqtrans=SQ seqnum=6 seqinput=10`); D2D V (sequence 6 has 15 frames) | none | **CONFIRMED** it is a sequence. See e3: it is **not** FCR-modulated |
| d4 | Druid Arctic Blast also a sequence (18, 15 steps) | SK113 V, D2D V | none | **CONFIRMED**, not in the brief; seqtrans SQ so no FCR |
| e1 | One-tick convention (plain cast counts N-1) | D2MOO V, MAXR V, D2AND V, tables | D2D (no minus one) | **CONFIRMED** at the formula level |
| e2 | Sequences have no minus one | D2MOO V (`<= 0` test), MAXR V (exception formula), D2D V (19-step) | none | **CONFIRMED** |
| e3 | Inferno and Arctic Blast do not respond to FCR | D2MOO V (modulated only if `seqtrans == CAST`; both have `seqtrans=SQ` in SK113 V) | none seen | **LIKELY**: code reading only, no community source states it. In 1.14d Inferno is a channel, so its per-tick behavior was not examined |
| f1 | Cast delay (cooldown) is independent of FCR | MAXR V ("FCR doesn't affect cooldowns"), FEXT V, SK113 V (`delay` column in ticks: Fire Wall 35, Meteor 30, Blizzard 45, Hydra 40, Frozen Orb 25, FoH 25, Firestorm 15, Molten Boulder 50, Volcano 100, Armageddon 150, Hurricane 150), PURE V ("delay will still run for 25 frames after each cast", generic text) | none | **CONFIRMED** (values in 1.13 data match MAXR seconds x 25) |
| f2 | Teleport, Static Field, summons differ | SK113 V: Teleport and Static Field `anim=SC`, no delay; Necro summons SC; Druid summons SC | none | **CONFIRMED** no special timing (Druid SC rows carry `UseAttackRate=1`, meaning not understood, U) |
| f3 | Assassin traps and sentries | SK113 V: `anim=S2`, not SC | MAXR/PURE tables include Assassin cast but never name traps | **UNSOURCED**: D2MOO modulates only PLRMODE_CAST, so S2 traps probably ignore FCR, but no source says it |

## 5. The d2d upstream doc (what to correct)

`docs/research/re/skills.md` "Cast rate (FCR)" (V, read from GitHub main):
1. gives per-class base frames identical to the list above (good, from the 1.14d animdata),
2. drops the minus one: `frames_shown = ceil(base*256/(256+floor(256*EFCR/100)))`, so Sorceress sweeps 14..8
   and `tests/test_combat.cpp` asserts it; it says this "matches community numbers exactly" but only the
   FCR breakpoints match, the frame counts are one higher than the community ones,
3. ignores the 208 speed (Druid row 0/9/19/32/54/86/152 is wrong; real human row has 4/10/19/30/46/68/99/163),
4. says its own Necromancer/Paladin 232 breakpoint exists; EFCR caps at 75 which is FCR 200, so 232
   cannot lower anything in the real game (D2MOO cap 175, V),
5. calls the lightning family S1 and says "Lightning casts as SC" (skills.md line 648), which is
   contradicted by SK113 (SQ seq 12 for Lightning/Chain Lightning, SC for the rest),
6. states the exact call site reading stat 105 is not traced; D2MOO now names it (section 2).

## 6. Disagreements and version notes

| Topic | Disagreement | Resolution |
|---|---|---|
| Absolute frame count | D2D 14 vs community 13 for Sorceress at 0 FCR | Community and D2MOO say 13 (end test fires one tick early). Whether the bot should treat the action as lasting 13 or 14 ticks depends on how the surrounding loop counts ticks (open, section 8) |
| Maxroll base cast | Necro 15, Druid 16 (D2R page) vs 16 and 15 elsewhere | Section 3 only reproduces the tables with 16 and 15; Maxroll's own formula page is inconsistent (U whether D2R changed data) |
| Amazon weapon split | MAXR (D2R) splits by weapon | Possibly D2R-only; Almar/Pure/D2AND have one Amazon row (U for 1.14d) |
| PD2 Lightning/Chain Lightning/Frozen Orb | PD2 MOD: slow = Chain Lightning and Frozen Orb, Lightning normal, slow base 15 | MOD change; do not use |
| Version | ALMAR and PURE give no version. D2MOO addresses cover 1.10f, 1.13c, 1.14d. SK113 is 1.13. D2D data is from the 1.14d MPQ. FEXT/MAXR are D2R pages | The formula code and 25 fps tick are the same across 1.10-1.14d as far as D2MOO shows; the tables are version-independent unless Skills.txt/AnimData changed. 1.10 vs 1.14 data diffs were not examined (U) |
| Lightning mechanics | Old forum posts (LL, S only) say lightning-type use a sequence at 256 without minus one | Agrees with D2MOO and D2D |

## 7. Recommended table for the patch (1.14d, frames per cast, plain = ends one tick early, sequence = no minus one)

| Class / skills | Base@speed | FCR -> frames | Confidence |
|---|---|---|---|
| Sorceress SC (all except Lightning, Chain Lightning) | 14@256 | 0:13 9:12 20:11 37:10 63:9 105:8 200:7 | high |
| Sorceress Lightning, Chain Lightning (sequence 12, 19 steps) | 19@256 | 0:19 7:18 15:17 23:16 35:15 52:14 78:13 117:12 194:11 | high |
| Amazon | 20@256 | 0:19 7:18 14:17 22:16 32:15 48:14 68:13 99:12 152:11 | high |
| Necromancer | 16@256 | 0:15 9:14 18:13 30:12 48:11 75:10 125:9 | high |
| Paladin | 16@256 | 0:15 9:14 18:13 30:12 48:11 75:10 125:9 | high |
| Assassin | 17@256 | 0:16 8:15 16:14 27:13 42:12 65:11 102:10 174:9 | high (brief said 17..10, wrong) |
| Barbarian | 14@256 | 0:13 9:12 20:11 37:10 63:9 105:8 200:7 | high |
| Druid human | 15@208 | 0:18 4:17 10:16 19:15 30:14 46:13 68:12 99:11 163:10 | high |
| Druid wolf / bear | 15@229 / 15@228 | 0:16 6:15 14:14 26:13 40:12 60:11 95:10 157:9 / 0:16 7:15 15:14 26:13 40:12 63:11 99:10 163:9 | medium (bases by fit) |
| Necromancer vampire | 15@160 | 0:23 6:22 ... 180:13 (section 3) | medium |
| Inferno, Arctic Blast | seq 6 / seq 18, 15 steps at speed 256 | 15 ticks regardless of FCR | low (code reading only) |
| Assassin traps/sentries (S2) | - | unknown whether FCR applies | none |

Implementation rule: `E = min(120*F/(F+120), 75)`; `speed = floor(animspeed*(100+E)/100)`;
`ticks = ceil(256*base/speed)`; subtract one unless the animation is a sequence (`anim=SQ`) with `seqtrans=SC`.
For sequences that are not FCR-modulated (seqtrans other than SC): `ticks = seqFrames`.

## 8. What no source states, and what is needed for 100 percent

Claims no source states (UNSOURCED):
- That d2d's per-tick loop observes the action as complete after exactly N-1 advances (the code says the end
  test fires then; how the server then sets the next mode, and on which tick the bot sees the mode change,
  is not documented).
- FCR on Assassin traps, sentries and other S2 skills.
- Whether the weapon-dependent Amazon rows exist in 1.14d.
- A video or memory frame count by any community member. I found none.
- 1.14d Skills.txt itself (I read the 1.13 file); AnimData for wolf/bear/vampire (read only as fitted bases).

For 100 percent, run one in-game measurement later (a live agent, a throwaway character such as Testy, offline,
via the logged cheat tool if FCR gear is needed; I did not run it):

1. Read from the player unit (UnitAny, base from the player pointer in docs/d2-1.14d-memory.md; D2MOO-H layout,
   which agrees with the doc's 0x10 mode and 0x5C pStats): `+0x10 dwAnimMode` (mode CAST is 10 for players
   per `components/rules/sequences.hpp` comment, SEQUENCE is the value of PLRMODE_SEQUENCE in D2MOO),
   `+0x44 nSeqCurrentFramePrecise`, `+0x48 dwFrameCountPrecise`, `+0x4C wAnimSpeed` (int16), `+0x30 pAnimSeq`,
   `+0x3C dwSeqSpeed`. These offsets are V in D2MOO's D2UnitStrc, U for the 1.14d Game.exe process under Wine
   until read once (canary: wAnimSpeed must be 256 at rest for a Sorceress and 208-scaled for a Druid cast).
2. With FCR 0, cast Fire Bolt on a dummy target: sample at the 40 ms reader cadence (our snapshot rate is
   40-100 ms, tick is 40 ms, so use the fastest loop and record the game's tick counter if readable, else
   timestamps) and record: wAnimSpeed during the cast (expect 256), dwFrameCountPrecise (expect 14*256 = 3584),
   the number of ticks the mode stays CAST (expect 13).
3. Repeat for Lightning (expect pAnimSeq non-null, dwFrameCountPrecise 19*256 = 4864, 19 ticks) and Inferno
   (check if wAnimSpeed stays 256 when FCR is added).
4. Repeat at one FCR value per breakpoint pair around 9/20 (Sorceress) with items or the cheat tool, and one
   Druid human cast (wAnimSpeed expected floor(208*(100+E)/100)).
5. Alternative without any memory: mana-drop ticks, i.e. cast repeatedly with a mana pool and a cooldown-free skill
   (Fire Bolt) and count mana drops over a long window; ticks per cast = window / casts. Coarser, but needs
   nothing besides what the bot already reads.

Compare each measured tick count with section 7. If section 7 holds at all sampled points, the claims move from
CONFIRMED-by-code to CONFIRMED-by-observation, and the d2d off-by-one question is settled with a measured number.
