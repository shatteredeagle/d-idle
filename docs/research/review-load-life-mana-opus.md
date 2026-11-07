# Review: load life/mana candidate r2 (TASK-101), independent Opus verification

> status: verification of fork branch `local/fid-load-life-mana-r2` (head 8d2f5e18, one commit on upstream main da2d96cf), 2026-11-07, fidverify1 (Opus). Re-derived with own disassembly reads and an own harness; the previous agent's conclusions were not used as input. Evidence it checks: [fid-loadhp-trace.md](fid-loadhp-trace.md), [pr-load-life-mana-draft.md](pr-load-life-mana-draft.md), queue entry in [pr-bundle-queue.md](pr-bundle-queue.md).
>
> Tags: **V read** read in game.exe 1.14d disassembly (upstream `tools/emu/disasm.py`, own `--all` sweep); **V run** game.exe code run in the upstream emulator by this review's own harness, or d2d's `test_game` built from r2 by this review; **I** inferred; **U** unverified; **S** opinion.

## Verdicts

| # | Claim | Verdict |
|---|---|---|
| C1 | game.exe's loader keeps the saved current life and mana exactly (24-bit fixed point, fraction included), no clamp to the maximum, no heal | **HOLDS** (V read + V run, incl. with the real stat-change callback registered) |
| C2 | a character *enters a game* with that life and mana (what the PR title says) | **HOLDS WITH LIMITS**: true at the end of the load; game.exe later rescales on any maximum change, and d2d r2 clamps to the maximum at its next stat refresh (see F2, F3) |
| C3 | life 0 stays 0 | **HOLDS at the loader** (V run); what follows on either side is U (F4) |
| C4 | stamina is set to the maximum | **HOLDS WITH LIMITS**: game.exe's maximum there is the full one, worn items included; d2d's is the save's own maximum (F5, pre-existing, not changed by r2) |
| C5 | the live "fraction dropped" reads were the client copy; the server keeps the fraction | **HOLDS** (V read) |
| C6 | `World::join` and `World::enter` both reach `take_character`; r2 removes no heal that game.exe does elsewhere | **HOLDS** (V read); one separate d2d divergence found on resurrect (F6, out of r2's scope) |
| C7 | the commit and PR body meet the maintainer's rules | **HOLDS**, with two low fixes to the PR body (F7) |

Overall (S): r2's code change is right and its oracle is sound. It is **READY to open after the PR body's "Honest limits" gains the F2/F3/F4 sentences** (body text only; no change to the commit is needed). No r3 was made.

## 1. The load path, read again (V read)

- FUN_00534330: magic 0xaa55aa55 at 0x534350, `cmp ebx, 0x5b` at 0x534364, version above 0x5b -> `call 0x56b180` at 0x534387; else FUN_00534020.
- FUN_0056b180: unit FUN_0056a090 (0x56b1bb), header parts 0x56a370 / 0x56a3e0 / 0x56a470, stat block FUN_0056a620 (0x56b270), then `push 6 ... call 0x625480` (0x56b283) into [ebp-0x14] and `push 8 ... call 0x625480` (0x56b28f) into [ebp-0x10]; skills 0x56a710, items 0x56a7e0, corpse 0x56a830, merc 0x56aa50 / 0x56ac10, 0x56ae50; on success `push [ebp-0x10]; push [ebp-0x14]; call 0x56af80` (0x56b3b7..0x56b3c3). Any failure returns before the restore.
- FUN_0056af80: gold 0xe and bank 0xf clamped to caps (FUN_00622e70 / FUN_00623460); `push esi; call 0x625db0; push 0; push eax; push 0xa; push esi; call 0x627260` (0x56b014..0x56b020): stamina = FUN_00625db0 (full-list getter of stat 0xb, `mov edi, 0xb0000; call 0x624f60`); equipment re-link (FUN_005390d0 loop, FUN_0056af20, FUN_005701b0); then `push [ebp+8]; push 6; ... call 0x627260` (0x56b0f9..0x56b102) and `push [ebp+0xc]; push 8; ... call 0x627260` (0x56b107..0x56b110). No compare, no `min`, no branch on the value. Then stats 0x44/0x43/0x45 = 100 and 0x1e.
- **The stat-change callback is live during the load.** FUN_0056a090 calls FUN_00555230 (0x56a22c), which calls FUN_005348c0 (0x555374), which registers FUN_0055b800 with FUN_00626d40 (`push edi; push 0x55b800; push 0; push esi; call 0x626d40`, 0x534913..0x53491c). FUN_00626d40 builds a new full stat list with the callback at +0x5c and the game at +0x60. So while items load (0x56a7e0) the maximum life/mana changes and FUN_0055b800 rescales current life; the loader's read-then-restore of stats 6/8 exists to undo exactly that. The upstream oracle hooks FUN_0056a090 and so runs without the callback; this review ran it with the callback (section 2).
- FUN_0055b800, decoded jump table (V read of bytes): index table at 0x55be04 = `00 0a 01 0a 02 ...` for stats 7, 8, 9, 10, 11; jump table 0x55bdd8 = 0x55b8ec (stat 7), 0x55b9a6 (stat 9), 0x55ba38 (stat 0xb). Stat 7 arm (0x55b8ec..0x55b95e): skip if old max <= 0 or life <= 0; old max below 0x100 counts as 0x100; `life = round(life * new / old)`, at least 1, **at most the new maximum** (`cmp ecx, edi; jge -> eax = edi`). Mana and stamina arms are the same shape. Stats 6, 8 and 0xa have no arm (index below 0 / 0x0a), so the restore at 0x56b102 fires nothing that changes life.

## 2. Own harness and runs (V run)

A separate script (`fv_load.py`, in this review's scratch clone of the fork, not in the PR) imports the upstream emulator and the r2 oracle's encoder, with its own case generator: seed 20261107, 300 cases in six kinds (dead: life 0; frac: fractions .00/.004/.5/.99/.996; above: life above the maximum; full; random x2), mana drawn from {0, max, random, above max, x.996}, stamina {0, max, random}. d2d side: `test_game --load-stats` from r2 built by this review (GCC Release, nice 19, own build dir on the worker host).

| Run | What runs for real | Result |
|---|---|---|
| `game` | FUN_0056b180 whole (as `fl_load.py`), new cases | life == file, mana == file, stamina == file's maximum: **300 / 300** |
| d2d r2 `World::enter` vs `game` | `diff` of the two vectors | **0 of 300 differ** |
| `cb` | callback FUN_0055b800 registered with the real FUN_00626d40; FUN_0056b180's order by hand with real functions: decode FUN_0056a4f0, reads FUN_00625480, "items" = real FUN_00627260 raising stats 7 and 9 by 20 points (fires the callback), restore FUN_0056af80 | the callback changed life during "items" in **300 / 300**; after the restore life and mana == file in **300 / 300** (e.g. `vabove1`: 136.0 of 133.0 -> scaled to 153.0 by the callback -> restored to 136.0) |
| `after` | callback registered, full FUN_0056b180, then the maxima lowered by 20 points (an item that stops counting after the load) | life changes in every case with life > 0 (254 / 300); life 0 stays 0 (46 / 46); no life stays above the new maximum (above-max saves: 47 / 47 clamped) |

Commands (worker host, nice 19; `P` = the emulator's venv python, `D2_MPQ_DIR` = the extracted MPQ folder):
```sh
$P fv_load.py gen 20261107 > cases.tsv
$P fv_load.py game cases.tsv > game.tsv
build/tests/test_game --load-stats cases.tsv | grep -E '^[A-Za-z_][A-Za-z_0-9]* -?[0-9]+( -?[0-9]+){5}$' > d2d.tsv
diff game.tsv d2d.tsv | grep -c '^<'          # 0
$P fv_load.py cb cases.tsv; $P fv_load.py after cases.tsv
```
`test_game` (full, r2): section 6.

## 3. The client copy (C5, V read)

Server FUN_00548760: `call 0x625480` stat 6 then `sar ebx, 8` (0x5487b7); stat 8 `sar eax, 8` (0x548855); stamina `sar ebx, 8` (0x548868). The join ends with this update: FUN_00539760 calls FUN_00548760 at 0x5399fe. Client FUN_0045d9b0: `movzx edi, word [ebp-0xf]; shl edi, 8; push 6; call 0x627260` (0x45d9d2..0x45d9df), and the same for 8 (0x45d9ea) and 0xa (0x45d9fc). So the client unit holds `floor(server) << 8`; a live read through the client sees 89 for 89.54. The PR should say "the server keeps the exact fixed-point value; the client shows whole points" (it nearly does).

What the server fraction can change in play (I): regeneration and potion amounts add to the server value, so the moment the next whole point appears on the client can differ by up to one regen step; a death check works on the server value (a save with life in (0, 1) point is alive with 0 shown). Display and the bot's chicken threshold read the client floor and are unaffected beyond that.

## 4. After the load (C2, C3)

| What | game.exe | d2d r2 | Tag |
|---|---|---|---|
| a maximum changes after the load (an item disabled by the join's requirement pass FUN_0055dbc0, Battle Orders ending) | life/mana/stamina scaled by new/old and clamped to [1, new max] (FUN_0055b800, run above); Battle Orders' end handler FUN_005cec50 -> FUN_0056b6c0 clamps 6/8/0xa to their maxima (0x56b6d1..0x56b722; only caller 0x5cec7b; registered at 0x5cee2e) | `Fight` refresh: `current = std::min(current, maximum)` on every refresh, no proportional scaling (components/game/fight/strike.cpp:190, the item_max block 172..193) | V read both; d2d's scaling gap is the trace's D3, pre-existing |
| life above the maximum at load (possible by play: saved under Battle Orders, whose share is not saved, Fight::own_stats) | kept until a maximum changes (no periodic clamp found; regen does not lower it, I) | kept by `take_character`, **clamped at the next refresh** (strike.cpp:190) | V read; I for "no other clamp" (scan of the 75 stat-6 writes was not exhaustive) |
| FUN_0055dbc0 at the join | saves 6/8/0xa into a buffer at entry (FUN_0055d9d0, 0x55d9d9..0x55d9f2) but its exit FUN_0055da70 restores only the skill slots, not life: a disabled item's max drop rescales life through the callback | `settle_requirements` runs in `take_character` before life is left alone; no scaling | V read; I for how often it bites (needs a save whose gear no longer fits) |
| life 0 at load | kept 0 by the loader; no read of stat 6 in the join path (scan: only 0x534137, old-format loader) | `take_character` keeps 0; `Fight::dead()` is `pmode == DT/DD` (fight/setup.cpp:466), so a life-0 save enters as a **living player with 0 life**; before r2 it entered at full | V run (loader), V read (d2d); U what game.exe does next and whether game.exe ever writes life 0 into a save (d2d's `Fight::die` sets life 0, fight/setup.cpp:469) |

## 5. Other heals (C6)

- `World::join` (components/game/world/setup.cpp:237 in r2, `take_character(entering)`) and `World::enter` (setup.cpp:287) both reach `take_character` (V read).
- game.exe heals that r2 must not remove, all outside the loader (V read, the scan of every `FUN_00627260` call whose stat argument is 6, 8 or 0xa): resurrect (0x54c15c..0x54c198 sets 6, 8, 0xa to FUN_00625d10 / 25d60 / 25db0, guarded by mode 0x11 at 0x54c107, I: dead mode), level-up (FUN_00570880 region, 0x570785..0x5709b3), the Battle Orders end clamp above. r2 changes none of d2d's counterparts (`Fight::revive`, `rules` level-up refill combat.hpp:1068). No heal on act change or town entry was found in game.exe's writers (I).
- Separate finding (V read, out of r2's scope): d2d `Fight::revive` (fight/setup.cpp:120..122) sets only life to the maximum; game.exe's resurrect at 0x54c17c..0x54c198 sets life, mana **and stamina**. A candidate for its own trace (not this PR).

## 6. Rules check of the commit (C7)

- Author and committer `shatteredeagle <26069301+shatteredeagle@users.noreply.github.com>`; subject plus 3 body lines; last line `Agent: Claude Code`; no Co-Authored-By (V read, `git show -s`).
- 3 files, one subsystem, change + test + oracle `tools/emu/fl_load.py` in one commit. `python3 tools/pr_guard.py da2d96cf HEAD`: `0 FAIL, 0 WARN` (V run here). Includes of `test_game.cpp` sorted, `<array>` / `<fstream>` present.
- `test_game` (full) built from r2 by this review, `D2_MPQ_DIR` set: rc 0, 29 `OK` lines, among them "OK: a save's life and mana are kept on entering a game" (V run). With the parent's `setup.cpp` (da2d96cf) rebuilt in the same tree: rc 134, `Assertion hero.character.stats.values[kLife] == 89 * 256 + 138 failed` (test_game.cpp:1301). Fails before, passes after.
- Not run by this review: `presubmit.py --quick`, clang-tidy, the other suites.

## 7. Findings, ranked

| # | Sev | Finding | Recommendation |
|---|---|---|---|
| F1 | - | C1 holds with the real callback live (section 2): the read-then-restore is the original's design | none |
| F2 | medium | The PR says "kept even above the maximum"; in d2d r2 that lasts until the next `Fight` refresh (strike.cpp:190 `current = std::min(current, maximum)`), while game.exe keeps it until a maximum changes | Add to "Honest limits": the claim is the state after the load; d2d's refresh clamp is a separate, pre-existing difference (or narrow the wording to "no clamp at load") |
| F3 | medium | game.exe rescales current life by new/old max on every later maximum change (FUN_0055b800, run here), d2d only clamps (trace D3) | Limits sentence; keep D3 as its own trace/PR, not in r2 |
| F4 | medium | Life-0 saves: r2 turns d2d's "enter at full" into "alive with 0 life" (pmode not dead); game.exe's follow-up is U | Limits sentence now; a lab drill (die, Save and Exit without respawn, rejoin on game.exe; read the client's life and mode) before anyone relies on it. Not a blocker: the loader evidence is V run |
| F5 | low | Stamina: game.exe uses the full maximum after items loaded (FUN_00625db0 at 0x56b015, after FUN_0056a7e0); d2d uses the own maximum, and its refresh raises the maximum but not current, so gear that adds max stamina leaves d2d short (I for the size) | Do not claim stamina equality with gear; the oracle has no items. Separate trace |
| F6 | low | d2d `Fight::revive` heals life only; game.exe resurrect also refills mana and stamina (0x54c17c..0x54c198) | New backlog card candidate (coordinator) |
| F7 | low | PR body: say "the server keeps the exact fixed-point value"; the docs/research/re checkbox is for traces (fix PR may leave it, but a row in net-join.md near line 334 would help the maintainer) | Edit the draft body; optional doc row as its own commit |

## Shared-file lines for the merger
See `backlog/merge-lines/agent-fidverify1.txt`.
