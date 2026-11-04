# Independent verifier: PALSKILL D2/D3 (Blessed Hammer: Concentration share, undead / demon bonus) (PALVERIFY, 2026-11-04)

> status: verifier record for the private fork candidate `local/pal-d2d3-r2` (head 9e9622eb: 6d6873bf + 9e9622eb on upstream main e5de6c10). Author's evidence (`pr-palskill-body-r2.md` on agent/palprep, `pal-skill-trace.md`) read only after my own counts. Offline: no game, nothing pushed or posted, the fork checkout not touched, the author's `hammer_hit.py` / `test_hammer_hit.cpp` not used. Verdict: section 6.

Tags: **V run** (my harness or builds on the worker host), **V read** (my disassembly with the base tree's `tools/emu/disasm.py`, or the source), **S** read without a run, **U** not checked.

## 1. The claim, as the commits state it

- 6d6873bf: Blessed Hammer's missile magic min and max are scaled at launch by the caster's Concentration: influence = damagepercent(state 0x2a) x Param1 / 8 toward 0 (FUN_006461d0), then MulDiv(stat, 100 + influence, 100) (FUN_005d0040). d2d: `aura_influence`, `influenced`, `Missile::influence` set in `Fight::fire` (player cast only), applied in `Fight::strike`.
- 9e9622eb: the missile row with pSrvDmgFunc 5 (only `blessedhammer`, dParam1 = dParam2 = 50) adds MulDiv(roll, dParam, 100) to the rolled element per class (undead, demon) before resistance (FUN_005ad2f0). d2d: loader keeps pSrvDmgFunc / dParam1 / dParam2, `MissileDamage::undead_pct / demon_pct`, `Target::demon`, `missile_blow` adds the bonus.
- Claimed: 0 differing cases on the author's harness (plain monsters, all monster ids), nothing else changes.

## 2. My harness (copies here)

- `hammer_game.py` (game.exe under the emulator, data tables booted): FUN_00644d50 / FUN_00644e40 for min / max; FUN_006461d0 natively on a fake caster (state plumbing hooked to the case's damagepercent), the do's MulDiv via the real FUN_00483360; then the REAL hit sequence of FUN_005ae0d3..0x5ae18e: FUN_005a89a0 rolls the record on the missile's seed, the row's pSrvDmgFunc is called through the table at 0x73c960 (entry 5 = 0x5ad2f0, read from the image), then FUN_0057c6c0. FUN_0063e990 / FUN_0063e940 run natively on the real MonStats rows. `PV_DIRECT=1` skips FUN_005a89a0 (the author's shape); `PV_BOTH=1` sets the undead and demon bits on every MonStats row.
- `test_palprobe.cpp`, dropped into `tests/game/` of scratch copies only: the REAL `missile_damage`, (head) `aura_influence` with the hammer's `par[0]` and `influenced`, the loaded `blessedhammer` row's dmg_func / dmg_par1 / dmg_par2 as `Fight::strike` copies them, `make_monster(row)` and `Monster::target()` (the real undead / demon flags), `missile_blow` on the case's seed, `hurt()`. `PV_PREDRAW=1` advances the Rng by the draw FUN_005a89a0 makes first.
- Inputs differ from the author's: seeds 20261104, 777, 4242 (author: 1 and 2); every MonStats row twice, clvl 1..99 (author: clvl 30), hammer level 1..99, synergies 0..99, damagepercent edges -100, -60, -8, -1, 1, 7, 8, 9, 1000, 25000, 400000 plus the Concentration values, resistance -100..150 including 100 and 101, life 1, 2, 3000.
- Builds: base e5de6c10 and head 9e9622eb, separate copies, configured and built from clean (Release, g++), on the worker host. Scripts `build.sh`, `run.sh`, `cmp.py`, `suites.sh`, `sorc_regress.sh`; raw output `results.txt`.

## 3. Counts (V run)

Domain: 100 of 734 MonStats rows are not `targetable()` in d2d (`hurt()` refuses; no hit lands in play): excluded and counted (14,154 / 7,204 / 825 cases). Classes from game.exe's own flags: 123 undead, 185 demon, 0 both (V run).

| seed, cases in domain | build | direct (author's shape) | full hit path, d2d stream advanced once | full hit path, d2d one draw |
|---|---|---|---|---|
| 20261104, 88,826 | base | 63,360 differ (56,364 min/max) | 63,362 | 70,973 |
| | head | **0** | **0** | 33,546 (life only) |
| 777, 45,776 | base | 32,639 | 32,638 | 36,498 |
| | head | **0** | **0** | 16,919 (life only) |
| 4242, 5,155 | base | 3,699 | 3,699 | 4,070 |
| | head | **0** | **0** | 1,560 |
| 4242 PV_BOTH (every row undead and demon), 5,155 | head | **0** | | |

Per class at base (seed 20261104, direct): plain without Concentration **0 of 16,914** (must stay equal: it does), plain with Concentration 29,211 of 29,289, undead without 2,846 of 6,181, demon without 4,133 of 9,214 (the bonus missing). Head: 0 in every class. min / max: head 0 differ everywhere, so the Concentration part holds at the record level for every damagepercent edge, including negatives (toward-0 truncation) and 400000 (MulDiv's wide branch).

Nothing else moved (V run): upstream's own Sorceress differential (`missile_hit.py gen 20000 99`, 64,000 cases) 0 differ on base and on head; `test_combat`, `test_rules`, `test_game_rules`, `test_game`, `test_monsters` rc 0 on both; d2d's undead / demon flags equal game.exe's on all 734 rows (base and head); `mon_bin` is the identity (734 of 734).

## 4. Findings

- **medium, the double draw (V read + V run).** On a real hammer hit game.exe draws the magic roll TWICE from the missile's seed: FUN_005a89a0 fills the record (0x5ae150), then FUN_005ad2f0's FUN_005a8c70 re-rolls the missile's element and overwrites +0x20 (0x5a8d88..0x5a8da3) before adding the bonus. The author's oracle (commit message: "runs ... FUN_005ad2f0 and FUN_0057c6c0") starts at the dmg func, so "the roll drawn on the case's seed" is the SECOND draw of the real path. With the full path, head differs on 33,546 / 16,919 cases (life only) until d2d's stream is advanced by that first draw, then 0. d2d rolls on the shared fight Rng, not on a per-missile seed, so the distribution is the same and gameplay is not affected (I, arithmetic: the second draw is uniform on the same range); it is a modelling note the PR body should state, not a refutation. Recommendation: one sentence in the body ("the hit path rolls the record first (FUN_005a89a0) and FUN_005a8c70 rolls the magic again; the oracle starts at FUN_005ad2f0, the draw that counts").
- **medium, CI include check (V run, clang-tidy 23 with the repo's `.clang-tidy`).** `tests/game/test_hammer_hit.cpp:13` `#include <d2s_items.hpp>`: `misc-include-cleaner` "not used directly", and `WarningsAsErrors: '*'`. The tidy job in PR mode runs on the changed `.cpp` files, so "Names and includes" fails on this file (S: CI's clang-tidy is the ubuntu-26.04 package, not run here). Smallest fix: delete that one line in 6d6873bf (the file introduced there). Changed TUs `fight/missiles.cpp` and `gamedata_load.cpp`: 0 findings at head and base. The `outdoor.hpp:1052/1090` binding-name errors my local version prints are in an unchanged file, identical on base (not this PR).
- **low, latent generality (V read).** `missile_blow` adds `mul_div(roll, pct, 100)` for every element, but FUN_005a8c70's return value (the MulDiv base) is the roll x 25 for poison (0x5a8e1a) and its record field for stun is different; only `blessedhammer` (magic) has pSrvDmgFunc 5, so no effect today. Recommendation: a comment ("magic only; poison would use roll x 25") or gate on etype 4.
- **low, other dmg funcs (V run).** game.exe has 23 rows with a pSrvDmgFunc (1, 2, 3, 4, 7..14; e.g. firewall 3/19, icearrow 2/100, moltenboulder 14/1/50); the loader now carries the column for all, only 5 is used. Out of this claim; worth one line under Limits so no reader assumes the column is modelled. d2d's map holds 21 of them (two rows with duplicate or empty names, U).
- **low, `Fight` glue (S).** Not driven by any test (author says so, U). Read: influence is taken at fire time from the first `concentration` aura over the caster, applied at strike to the damage computed then (fine); Concentration's damagepercent from d2d's `eval_calc` is 60 + 15 x (lvl - 1) at levels 1..40 (V run, `--conc`), which matches the known 1.14d values (S, not against game.exe here).

## 5. The author's evidence, read last

- Agreement: the claim, the addresses and their reading (FUN_005d0040 tail, FUN_006461d0 with Param1 4, FUN_005ad2f0 per class, FUN_0063e990 / FUN_0063e940, row 92 50 / 50) match my disassembly and my image reads; 0 differ at head matches my 0 on three other seeds and a wider domain; "both can apply" holds on my synthetic both-row run (0 differ).
- Not tested by the author: the full hit path (section 4), clvl other than 30, damagepercent above 1000 or below -20, life at 1 to 3, a monster both undead and demon (no real row is: the body could say so).
- Disagreement: none on numbers. My base counts cannot be compared one to one (other generator), but the pattern is the same: base differs on every class the change touches and on none it does not.

Side checks:
- (a) Authorship and trailers (V run): both commits author = committer = `shatteredeagle <26069301+...@users.noreply.github.com>`, bodies end `Agent: Claude Code`, no `-x` line, no `Co-Authored-By`. This matches the merged upstream commits by the same author (last 300 non-merge commits: `Agent: Claude Code` on 45) and CONTRIBUTING ("Say which agent helped", "The commit is theirs"). `pr_guard.py e5de6c10 9e9622eb`: 0 FAIL, 0 WARN; pr_guard does not check identity or trailers at all.
- (b) clang-tidy: see section 4 (fails; delete line 13).
- (c) One PR or two (opinion): one PR. Same subsystem (the hammer's hit damage), the second commit edits the lines the first adds (`Fight::strike`, `missile_blow`, the test), its count is measured on top of the first, and CONTRIBUTING allows a one-subsystem batch with one claim per commit. Split only if the maintainer asks.
- (d) Body tags: overreach or untagged items:
  - "the roll on the missile's seed" (oracle section): true for the oracle, not for game.exe's hit path (two draws): add the sentence above.
  - "That calc equals game.exe's on every Paladin aura row (21,760 cases, 0 differ: an internal earlier run, S)": cites a run upstream cannot re-run; drop the number or point to a command.
  - "D4, D6, D7 of the trace", "H3 / H4 of the trace", "debug cast 113 then 112": internal references (our trace doc and lab binary), meaningless upstream; describe them in words.
  - "A merc or pet never casts Blessed Hammer in d2d" and "Melee and weapon-based hits do not use these paths": untagged; they are S (source read), say so.
  - "Counts in this table are S" while the header row says run results: fine as tagged, but the "main" / "after commit 1" columns are from a different base (8c9a92c7); my run supplies an independent base count on e5de6c10 (section 3) the body may cite.
  - "The g++ all-targets stage failed on three targets this change does not touch ... (U)": state whether base fails the same way before posting, or leave the stage out; as written it invites the question.
  - "the clang-tidy finding ... has not been fixed here (a one-line removal if the maintainer wants it)": CI fails on it, so it is not optional; fix it in 6d6873bf.

## 6. Verdict

**CONFIRMED WITH CAVEATS.** At head, d2d's Blessed Hammer hit (min / max with Concentration, undead / demon bonus, resistance, life) equals game.exe's on 0 of 139,757 differing in-domain cases over three new seeds, all 734 MonStats rows (100 not targetable excluded), every class and edge tried, and the both-class synthetic; base differs on 99,698 and on no case of the class the change must not touch; the Sorceress differential and the upstream suites are unchanged. Caveats: (1) the equality is with the second of game.exe's two draws (state it); (2) `test_hammer_hit.cpp:13` fails CI's include check (delete the line); (3) the bonus code is magic-only in effect (comment or gate); (4) the PR body's internal references and untagged claims (section 5 d). The `Fight` glue is not driven by a test (S read only).
