# Verify: vendor prices follow-on (fix/vendor-prices-followon-v2, da80d0ab)

> status: research, current (2026-11-02). Independent Opus verifier, card verifyprices. Own harness written before reading the author's proof and draft: `tools/emu/vpr_oracle.py` (game.exe's own code under the emulator), `tools/emu/vpr_probe.inc` (a local `--vpr` mode compiled into a real d2d `test_game`, never committed to d2d), `tools/emu/vpr_cmp.py`, `tools/emu/vpr_dis.py`. Tags: V = read in the disassembly or run (how stated), S = secondary or by reading only, U = unverified. Worker host only, CPU kept to about 2 cores (builds `-j1`, `nice`). Nothing pushed, nothing posted.

## Verdict

**NOT READY as the branch stands; READY WITH CHANGES after a cut and a rebase.**

- Blocker (V): the branch does **not** merge into current upstream main (3852c904): `git merge-tree` reports 3 content conflicts in `components/rules/rules.hpp`. The reason: upstream main already has the maintainer's own commit a9548d38 "Store prices past 0xffff divide by 1024 before the NPC mult (0062faaf)" (2026-11-01), which does what our aed23fc8 does (and also wraps the repair cost to 32 bits). The draft's and proof's "rebase onto upstream main is clean" banners are stale.
- The repair gate (da80d0ab) is not upstream (upstream's `repair_cost` comment still says the FUN_00629930 checks "aren't made"). Cherry-picked alone onto 3852c904 it conflicts only in `tests/test_rules.cpp` (one hunk); rules.hpp, gamedata_load.cpp and store.md merge.
- Both rules hold against game.exe on my own cases (0 mismatches on head attributable to the change); details below.

Exact changes for a PR:
1. Drop aed23fc8 (divide-first): upstream has it. Drop the `bigmul` parts of the oracle commit 777df9d7 (`price_edge.py`, `price_claims.py`) or cut them to the repair gate.
2. Rebase da80d0ab onto current upstream main; resolve the `tests/test_rules.cpp` hunk (keep the bow / crossbow asserts; the `dear` assert still uses upstream's `big` item). My local resolution (drop the conflict markers, keep both sides) is in section 4; built (-Werror) and the 7 suites are green, and its probe gives the same counts as the head (V: 0 gate mismatches on 4,664, 0 mult mismatches above 0xffff, the same 8 / 10 harness residuals).
3. Fix the text: bows are nodurability 1 too (V, below), so "worn bows (Repair 0)" is not the reason a bow is refused in vanilla data; say the ItemTypes Repair half is redundant with nodurability for every vanilla base and kept for fidelity (mods). Title loses the divide-first half: "Vendor repair: no price without ItemTypes Repair or durability (FUN_0062e660)".
4. Re-run `pr_guard` and `presubmit --quick` on the rebased head (done here locally: 0 FAIL).

## 1. Rules and addresses against the disassembly (V, capstone over game.exe, `vpr_dis.py`)

| claim | read | verdict |
|---|---|---|
| Divide-first at 0062faac | `cmp [ebp-0x10],0xffff; jle` then `test mult; je`; `cdq; and edx,0x3ff; add; sar 10; imul mult` (signed, toward 0), else `mult*price` then the same signed /1024. 32-bit `imul` both ways. | HOLDS |
| Sell at 0062fae5 | edi x `[row+8]` but the zero test is `ecx` = `[row+4]` (the other column) | HOLDS; the `ponytail` is right and unreachable: no npc row has a 0 mult (V: all 17 game rows, `vpr_oracle.py rows`) |
| Repair at 0062fb1d | `[ebp-8]` x `[row+0xc]`, own zero test | HOLDS |
| Quest loop 0062fb70..0062fc53 | 3 flags at row+0x10.., FUN_0065c310(q, flag, 0) or (.., 1) == 1; each of the three prices divides first on its own running value | HOLDS |
| Gate call 0062efee | mode 3: FUN_0062e660 false -> return 0 | HOLDS |
| FUN_0062e660 | identified (flag 0x10), not ethereal (0x400000); FUN_0062a840 (stat 0xcc charged skills: any with left < max) -> repairable; else ItemTypes record byte +8 (Repair) must be set and FUN_0062e5d0 (Repair, Throwable FUN_0062ba80, stackable FUN_006289f0, not ethereal) or FUN_00629930 | HOLDS |
| FUN_00629930 | ItemsTxt +0x113 nodurability == 0, +0x112 durability > 0, FUN_00625e00 (stat 0x49 maxdurability) != 0, stat 0x98 indestructible <= 0 | HOLDS (d2d's `indestructible()` treats a negative stat 152 as indestructible, game.exe does not: unreachable) |
| Ponytail "with charges missing, a refused item still adds its base" | 0062f9e8: when FUN_00629930 is false the durability scaling is skipped and `[ebp-8]` keeps the full base, so the repair price is base x rep + charges; d2d gives charges only | HOLDS as a stated remaining deviation (S how often: no vanilla bow or throwing potion carries charges that I know of) |

## 2. Counts, base vs head (V)

Builds: my own clones on the worker host, base 8a45cfc9 (= da80d0ab~3) and head da80d0ab, g++ 16.2, RelWithDebInfo, `-DD2D_WERROR=ON` (CI flags plus -Werror): both rc 0. Cases: `vpr_oracle.py gen 2026 60000` (seed 2026, not the author's 7 / 11): 60,000 M cases (pre-mult price P: edges 0, 1, 1023, 1024, 0xfffe..0x10001, 0x103ff, 0x10400, 0x1ffff, 0x20000, 99999, 100000, 2^20, 2^20+1023, 2^24-1; 40 % in 0xf000..0x12000; 40 % log-uniform to 2^24; random npc of the 17, difficulty 0..2, random quest-flag bits) and 7,908 G cases (every one of 659 ItemsTxt bases x 12 states: durability half / whole / 0 / no max, charges none / full / missing, 10 % unidentified, 10 % ethereal, 10 % indestructible, quantity 0..2).

Game side: M runs FUN_0062efb0's own block 0062faac..0062fc59 on the real npc row (FUN_00656900 by MonStats hcIdx) with FUN_0065c310 hooked to the case's quest bits; the sale cap (row +0x40 by difficulty) and the minimum 1 are applied by my comparator as d2d does (S). G runs FUN_0062e660 on a fake item unit with the real ItemsTxt / ItemTypes tables (game.exe's own loader, FUN_00619300), only the stat getters FUN_00625480, FUN_00625e00, FUN_006261d0 hooked. d2d side: the real `rules::item_price` / `rules::repair_cost` with the real tables; M uses one synthetic base "vprm" of cost P (a normal identified item, whose pre-mult price is P on all three columns), G the real bases.

| set | column | base differ | head differ |
|---|---|---|---|
| M, P > 0xffff (25,674) | buy | 25,631 | 0 |
| | sell | 3,444 | 0 |
| | repair | 25,477 | 0 |
| M, P <= 0xffff (34,326) | buy | 2,955 (2,048 or 1,088 mult pushes it over 0xffff before a quest mult) | 8 |
| | sell | 0 | 0 |
| | repair | 10 | 10 |
| G, something to repair (4,664) | gate (cost > 0 vs game passes) | 143 (d2d prices, game refuses; e.g. `6cb`, an elite bow, half durability) | 0 |
| G, whole (3,244) | gate | 17 + 626 | 626 |

- Head residuals (V, listed): the 8 buy and 10 repair cases are all prices that reach 0 in the mult block (P 0, 1 or 7); d2d's later minimum 1 gives 1, my comparator does not model the game's later minimum for buy and repair. Same count on base and head: harness, not the change.
- The 626 whole-item cases: game.exe's gate passes (price 1, Repair All skips it), d2d 0: the known D5 sentinel, equal on base and head.
- Nothing that matched on base differs on head (V: per-case join).
- npc rows: d2d's 17 `npc_prices` rows equal game.exe's 17 rows field by field (V).
- Data fact (V, `vpr_cmp.py types` over the MPQ's txt): 66 ItemTypes codes have Repair != 1; all 186 bases of those types have nodurability 1 (bows, Amazon bows, throwing potions, potions, ...). So for vanilla data the ItemTypes Repair test never decides alone; the nodurability test does. The change still matters in play: d2d's item maker (rules.hpp, the FUN_00556360 model) gives every weapon a drawn durability, so a dropped bow can sit below its max and base d2d priced its repair (S that game.exe draws the same for nodurability weapons).

## 3. Nothing else changes (V by grep, S for the det scenarios)

- Readers: `ItemType::repair` is read only in `repair_cost`. `repair_cost` feeds `store_repair`, `store_repair_all`, `repair_all_cost` (rules.hpp), `components/game/world/commands.cpp` (repair commands) and `apps/d2d/store.cpp` (Repair All hover). `item_price` (the npc mult change) feeds buy and sell in rules.hpp, `apps/d2d/store.cpp`, `apps/d2d/ingame.cpp`, `apps/d2d/town/net.cpp`. Gamble prices and `charges_repair_cost` are not touched by the diff.
- det scenarios: NOT RUN (needs a d2d slot, not in this card). By reading `tools/emu/det_scenarios_lab.py`: combat, merc, pet and party scenarios, no vendor action, so 0 hash changes are expected (S).

## 4. CI parity

| check | head da80d0ab | rebased (3852c904 + da80d0ab, local) |
|---|---|---|
| g++ build, CI flags + -Werror, the 7 suite targets | rc 0 | rc 0 |
| suites test_rules, test_game_rules, test_monsters, test_monster_thinks, test_monster_ai, test_combat, test_game (data dir, no SKIP) | all rc 0 | all rc 0 |
| g++ and clang++ `-fsyntax-only`, CI flags + -Wshadow -Wrange-loop-construct -Wconversion, TUs test_rules.cpp, gamedata_load.cpp, world/commands.cpp, apps/d2d/store.cpp | 0 warnings in rules.hpp, gamedata_load.cpp, test_rules.cpp (clang's other hits are -Wsign-conversion in untouched headers) | 0 warnings in our files (same as head) |
| MSVC /W4 by reading | no new nested redeclaration (C4456: `info`, `kind`, `mends`, `durable` are new names in `repair_cost`, not reused below it); no narrowing (`npc_mult` returns `long long` into `long long`; `repair` is a bool); no windows.h macro names (far, near, min, max, small, interface, IN, OUT) in new identifiers; the designated initializer `{ .type, .kind }` follows ItemInfo's order | same |
| clang-tidy "Names and includes" (tools/tidy-pr-select.py with the 3 changed files: 206 TUs) | run on test_rules.cpp and gamedata_load.cpp: 0 findings in our files; 4 in `components/drlg/outdoor.hpp` (`x`, `y` bindings), untouched by the branch (local clang-tidy 23.1.4 may be newer than CI's). The other 204 TUs NOT RUN: they see the same rules.hpp, whose #includes the diff does not change, so include-cleaner cannot change there (S) | same as head: 0 in our files, the same 4 outdoor.hpp findings |
| required direct includes | none new: the added code uses no new std or project symbol | same |
| pr_guard per commit (777df9d7, aed23fc8, da80d0ab) and range | 0 FAIL, 0 WARN each | 0 FAIL, 0 WARN |
| presubmit --quick | 0 FAIL, 1 WARN (build skipped) | 0 FAIL, 1 WARN |
| merge into upstream main 3852c904 | **3 conflicts, rules.hpp** | (is the rebase) |
| merge into open #159 (36e7fcde), #160 (df1bf0e5), #163 (5e2d3f87) | the same 3 rules.hpp conflicts each (they sit on current main) | clean against all three (merge-tree rc 0) |

## 5. The draft's claims against counted runs

| draft claim | verdict |
|---|---|
| divide-first rule and addresses | HOLDS (V), but **already upstream** (a9548d38); not a claim to send |
| repair gate rule (FUN_0062e660 / FUN_00629930 / FUN_0062e5d0) | HOLDS (V) |
| "worn bows (Repair 0), crossbows (nodurability 1) and throwing potions had a repair price" | OVERCLAIM in the reason: bows and throwing potions are nodurability 1 as well (V); the outcome (refused) holds |
| "rebase onto upstream main is clean (merge-tree)" (draft, proof and verify-price2 banners) | **FALSE today** (V: 3 conflicts) |
| author counts (46,730 / 545 / 14,356 -> 0, 15,519 -> 0, no case newly wrong) | consistent in kind with mine (all -> 0, none newly wrong); my absolute numbers differ by design (own seed, own case mix) |
| limits: no stat lists, quest flags 0 in the oracle | my harness covers quest flags (0 head mismatches) and charges at the gate; still no magic stat costs or sockets |
| "32 bits: neither side wraps below 2^20" | HOLDS; upstream's version wraps the repair cost to 32 bits anyway |

## Not run

det scenario hashes (no d2d slot); clang-tidy over the 204 other TUs; an MSVC build (read only); a full clang build with -Werror (syntax-only passes instead); the end-to-end FUN_0062efb0 price call (the oracle runs its mult block and its gate, not the whole function).

## Commands (worker host, my clones under the lab dir `verifyprices/`)

```sh
tools/emu/run-on-worker.sh vpr_dis.py 0062faac 0062fc59 0062e5d0 0062e6f0 00629930 00629990
tools/emu/run-on-worker.sh vpr_oracle.py gen 2026 60000 <lab>/verifyprices/cases.txt
tools/emu/run-on-worker.sh vpr_oracle.py game <lab>/verifyprices/cases.txt <lab>/verifyprices/g.txt
build-<tree>/tests/test_game --vpr cases.txt > d-<tree>.txt      # with tools/emu/vpr_probe.inc included before the --price mode
tools/emu/run-on-worker.sh vpr_cmp.py cmp cases.txt g.txt d-base.txt d-head.txt
```
