# Hire window / hirer NPC menu — game.exe 1.14d vs d2d vs bot (job hirewindow)

> Status: current. Tags: V read = read in upstream's decompile notes and
> source (not re-read in the disassembly here), V run = observed running,
> D = derived from MPQ tables (hireling.txt through `tools/emu/run-on-worker.sh
> mpq-table.sh hireling`), S = shown by an upstream proof, I inferred,
> U unverified. Nothing was run in a game; no emulator differential was
> built (see "Not done").

## Answer in one line

Kashya's menu shows **hire only above character level 7**. The d2d lab
drill (livedrills2, D8.h) used a low-level throwaway character, so the
menu was `Kashya / talk / cancel`. This is **not a d2d gap**: d2d
implements the rule, the menu entry, the list and the window.

## Q1. game.exe rule

- Q1.1 NPC menu table at `0x726c48`, 48 records of 39 bytes, found by
  `FUN_004b2e30`, built by `FUN_004b4830`. Kashya's static record has one
  entry (talk) plus cancel. [S: upstream `docs/research/re/npc-menu.md`;
  V read there]
- Q1.2 `FUN_004b66b0` (opening an NPC): for Kashya (hcIdx 0x96 = 150) with
  character level (stat 0xc) **above 7**, `FUN_004b6410` rewrites her
  record to three entries, string[1] = 0xd45 "hire", handler `0x4b5c60`.
  Reads only stat 0xc; not quest-gated, not an NPC-table flag. [S, V read
  upstream]
- Q1.3 Other act hirers (Greiz 0xc6, Asheara 0xfc, Tyrael 0x16f, Qual-Kehk
  0x203): their static records already carry "hire" (only Kashya is
  rewritten at runtime); `FUN_004b6440` inserts "Resurrect %s: %d" (0x1507)
  while a dead merc's name is known. [S; the other hirers having no level
  gate is I: upstream only names Kashya's rewrite]
- Q1.4 `talk` is a separate sub-menu (handler `0x4b6c70`): lines
  introduction (0xd47), gossip (0xd43), quest names, cancel. This matches
  the live d2d menu in livedrills2 (`talk, introduction, gossip, cancel`),
  so hire is never behind `talk`. [S: d2d `open_talk_menu`; V run of the
  menu shape in livedrills2 (d2d, not game.exe)]
- Q1.5 Free Rogue: the Sisters' Burial Grounds reward `FUN_00579180` hires
  the first offer of Kashya's list with no window and no menu entry. [S]
- Q1.6 Window (`FUN_004b5c60`): 490x350, header 0xd24 "Your Gold: %d  Hire
  which Mercenary?" (carried + stash), a list (`FUN_004bf8f0`, 8 rows
  visible, scrolls), rows `name - Lvl: n  Life: n  Def: n  Cost: n`, then
  cancel. [S, V read upstream]
- Q1.7 List (server `FUN_00576070`): the seller's first hireling.txt row
  for difficulty and version gives NameFirst..NameLast, one entry per name,
  ten picks offered; rolled anew when none is left (`FUN_00577010`).
  Offer (`FUN_006637f0`): rows of the act/difficulty/version sharing the
  first row's Level, one picked; level = max(2, clvl-5+rand(5)); with
  d = level - row Level: life = max(40, HP+HP/Lvl*d), cost =
  max(Gold, Gold*(15d+100)/100), exp = (level+1)*Exp/Lvl*level^2. [S:
  upstream `tools/emu/hire.py`: 720 offers and 400 lists identical to
  game.exe's code in the emulator, i.e. V run upstream]
- Q1.8 Codes and prices (D, hireling.txt, LoD version 100):
  Act 1 Rogue = Class 271 (hcIdx), Ids 0/1 (Fire / Ice) normal, 2/3 NM,
  4/5 Hell; first row Level 3; Gold 100 (Fire) / 150 (Ice) Normal,
  6000 / 7500 NM, 12500 / 14000 Hell. Act 2 = 338 (Ids 6..14), Act 3 =
  359 (Ids 15..23), Act 5 = 561 (Ids 24..29). Seller hcIdx 150 / 198 / 252 /
  515. Normal Act 1 at clvl 8: level 3..7 so cost 100..160 (Fire) or
  150..240 (Ice) — a price window, not one price.
  The strings `RG / GU / IW / 0A` are the bot's labels (tables.go), not
  hireling.txt columns: hireling.txt has no such code. The 0A/IW mapping
  stays U (I: IW = 359 Iron Wolf, 0A = 561 Barbarian by their MonStats
  ids).
- Q1.9 Aura rows: hireling.txt Skill1..Skill6 columns per row (same table,
  D); not decomposed here.

## Q2. d2d

d2d implements all of it (S, read in the source): `kNpcMenus` Kashya = talk
only (`npc_menu.hpp:30`); `open_npc_menu` (`apps/d2d/ui.cpp:61`) rewrites it
to talk + hire for `clvl > 7`; `NpcMenuState::kHire` sends `cmd::OpenHire`
(`town.cpp:1778`) → `World::hire_list` / `hire_offers` (`world.cpp:1520`),
`open_hire_menu` (`store.cpp:26`), `cmd::Hire` (`world.cpp:2118`),
`rules::merc_offer` / `hire` (`rules.hpp:1612`, `1680`), Kashya's reward
`world.cpp:1537`. Hirelings list: `gamedata_load.cpp:1381`.

## Q3. Deviations / gaps

- D1: none proven. d2d's menu gate (`clvl > 7`) equals game.exe's (V read
  upstream; V run upstream for the list/offer math). Differential for the
  gate itself: none exists; buildable by running `FUN_004b66b0` in the
  emulator with stat 0xc = 7 and 8 and comparing the record's entry count
  against `open_npc_menu` — U.
- The drill cannot "fail on d2d": D8.h failed because of its character's
  level, not a d2d gap. No upstream issue/PR needed. D8.h can pass on d2d
  once the throwaway reaches level 8 (cheat/setup), then `hire` appears.
- The bot's paid-hire path may claim: menu flow and price window above (D),
  once run at clvl >= 8. It cannot yet claim 0A/IW codes (U) or any quote
  (never observed live).

## Q4. Correct menu flow for the drill

Setup: character level >= 8, gold >= 300 (Normal Act 1 worst case 240).
Click Kashya → menu `Kashya, talk, hire, cancel` → click `hire` → window
(header "Your Gold: N", up to 8 visible rows of ten offers) → click a row →
hired (gold paid, carried first then stash). With a dead merc a
`Resurrect` line sits before `hire`. Cant-afford drill (D8.h2): gold below
the cheapest row (clvl 8 Normal: 100 for a Fire offer; set gold 0..99).
Check `cmd/d2drill/hire_lab.go` waits for the `hire` line instead of
failing at menu open when clvl < 8 (B2).

## Bot findings

- B1: `cmd/d2drill/hire_lab.go:67` `hireFlow` fails with "no hire line"
  without checking the character level; the level-7 gate is the cause (I).
- B2: `internal/d2dsrc/tables.go:91-93` `IW`/`0A` are labels mapped from
  MonStats ids, never observed in a hire quote (U).
- B3: the specs' "free Rogue granted without a window (V-d2d)" is
  consistent with Q1.5, but the window is shown for the paid hire.

## Drill recipe (U → V)

1. In the lab binary set the throwaway to level 8 (docs/forkcheats.md level
   verb), give 5000 gold; walk to Kashya, click: expect 4 lines with `hire`.
2. Click `hire`; read the list: expect `Lvl` 3..7, `Cost` 100..160 (Fire) or
   150..240 (Ice), 10 rows (8 visible, wheel scrolls).
3. Repeat at level 7: no `hire` line (the gate edge, 7 is not above 7).
4. Click a row; read the merc type: expect Id 0 or 1 (class 271).
5. Same in Act 2/3/5 hirers for 338 / 359 / 561 and their prices.

## Matrix-cell proposals

None (no skill matrix row is touched). Proposed note for `docs/livedrills2.md`
(owner's file, not edited): "no hire line: character level was not above 7".

## Not done

No emulator differential run; no disassembly re-read (the decompile is
upstream's notes, tag S/V read upstream). The runner was reachable
(hireling.txt table read worked).

## Shared-file lines for the merger

BACKLOG: hirewindow: Kashya's hire entry needs clvl > 7 (docs/research/hire-window-trace.md); re-run D8.h at level 8, not a d2d gap.
INDEX: docs/research/hire-window-trace.md — hirer menu gate and hire window, game.exe vs d2d vs bot.
CHANGELOG: hire-window trace: Kashya's menu gets "hire" only above character level 7; d2d matches; D8.h failed on character level.
