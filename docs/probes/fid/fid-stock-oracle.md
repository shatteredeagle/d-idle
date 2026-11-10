# fid-stock-oracle (TASK-77, matrix rows VN04 VN05): vendor stock generation and refresh, d2d against game.exe

> status: finished 2026-11-09 by agent decomp8 (Sonnet). Result: **stock generation (which item codes are asked, how many, quality and item level, and the proxy seed left over) is 0 differ on current upstream main 83fc7ba1: 40,800 stock lines (400 seeds x 17 vendors x 6 act/difficulty/clvl jobs) and 14,400 merc offer lines, game.exe's FUN_00576980 / FUN_00576070 against the real build.** The refresh rule is partly read: game.exe generates a vendor's stock once at the first open (V read), and has two regenerate paths gated by flags whose setters are NOT traced (U); no timer or level-up trigger found. d2d keeps stock after the first open. AC #1 and #2 proven for the generation, AC #3 half done (U), AC #4 (live stock lists) NOT done. Nothing posted, no PR candidate.

Tags: V run (game.exe's code in the emulator against the real compiled d2d `drlg-dump store` / `hire`), V read (disassembly with upstream's `disasm.py`, `tools/emu/d8_dis.sh`), D, S, I, U. d2d: upstream main 83fc7ba1. Scripts: `tools/emu/d8_store.sh FIRST-LAST`, `d8_more.sh`, `d8_dis.sh`, `d8_callers.sh`, `d8_gl.sh`, `d8_src.sh`. Earlier proof: seedspawn-trace.md section 3.5 (200 seeds on the 2026-11-02 scratch build); this card re-runs it on main and adds 200 seeds.

## Q1. What does game.exe ask for, and does d2d ask the same? 0 differ, V run

`store.py` (upstream tools/emu): per job a proxy seed `{s, 666}` (s = index x 0x9e3779b1), each vendor 0..16 with its candidate lists (`FUN_00536d50`: 12-byte rows {min, max, magicMin, magicMax, code, magicLvl}, plus the permanent codes), an act byte, a difficulty and a character level, six jobs per vendor: (act 0, Normal, clvl 1), (1, Normal, 18), (2, Normal, 40), (1, Nightmare, 30), (3, Hell, 70), (4, Hell, 90). `FUN_00576980` runs natively; the item maker `FUN_00576330` is hooked to record the ask `code:quality@ilvl` and return a fake item. d2d: `drlg-dump ... store` prints `rules::store_asks` the same way. One line per job, the proxy seed left at the end of the line.

| seeds | store lines | differ | hire lines (`hire.py` vs `drlg-dump hire`) | differ |
|---|---|---|---|---|
| 1..200 | 20,400 | **0** | 7,200 | **0** |
| 201..400 | 20,400 | **0** | 7,200 | **0** |

This covers "item count", "quality", "item level" and the seed chain (a line carries the asks in order and the seed after; one differing draw anywhere changes every later field). The 17 vendor slots are the NPC proxy records of all five acts (the in-game vendor behind each index was not mapped here, I); the rule is one function for all, so Akara, Charsi, Gheed and Kashya (Act 1, vendors by the same table) are in it.

### Limits (honest)

- The item maker is hooked: the **affix rolls themselves** (which prefix and suffix, their values) are not compared by this run. They belong to the item generation proven by the treasure and item oracles (IT05: first drops 0 differ, `treasure-trace.md`, S) but the vendor's call into it (magic level from the row, `magicLvl`) is only the ask here. U for a vendor-made magic item end to end.
- Normal clvl-1 and 18 jobs are the bot's early game; Nightmare/Hell jobs are also included and equal.
- The result is for the same proxy seed. That d2d's proxy seed equals game.exe's after the same game start is seedspawn-trace.md item 2 (the seed chain after level creation, V run).

## Q2. Refresh: when does stock re-roll? (AC #3)

game.exe (V read, `d8_callers.sh 576980`: three call sites, 0x578bcb, 0x579116, 0x5794c2; `d8_dis.sh 578b30`):

| Address | What | Tag |
|---|---|---|
| FUN_00578b30 (entry 0x578b30) | looks the NPC up (`FUN_00535f10` -> record `esi`), `edi = esi + 0x20`. At 0x578bbc: **`cmp byte [edi], 0; jne` skip, else `FUN_00576980(player, record)` (0x578bcb) and `[edi] = 1`**. So the stock is made once per NPC per game, at the first interaction; a second open finds the flag and keeps the stock. The next flag `[edi+1]` does the same for the hire list: `FUN_00576070` only for NPC classes 0x96, 0x203, 0xc6, 0xfc (0x578bd9..0x578bff), then `[edi+1] = 1` | V read |
| 0x579105..0x579116 (a function ending at 0x579124, entry before 0x5790c0) | if `[record+0x23] != 0`: `[esi] = 0`, regenerate with `FUN_00576980`, `[esi] = 1` | V read; what sets `[+0x23]` (the only byte store `[reg+0x23] = 0` found is at 0x56a12b; no store of 1 found by the sweep) U |
| 0x57949e..0x5794c7 (the open-trade path, calls `FUN_00578b30` at 0x579475 first) | if `[record+0x27] != 0` (and the trade flags of the packet are set, for NPC classes other than 0x96, 0x203, 0xc6): `[record+7] = 0`, `FUN_00536d10`, `FUN_00576980`, `[edi] = 1` | V read; the setters of `[+0x27] = 1` in the binary (sweep `byte ptr [reg + 0x27], 1`): 0x5b8b17 and 0x5bbce7, in other functions whose structs were not identified (a buyback or inventory-change flag is plausible, I) U |
| a 4-minute timer | the earlier seed-study item 7.3 claim ("rerolls every 4 minutes"): **no timer read in the three paths above**; a search for other code that clears or sets the flags, or a tick-driven vendor loop, was NOT done beyond the two flag sweeps | U |

d2d (V read, `components/game/world/commands.cpp` lines 50-60 on 83fc7ba1, `d8_src.sh`): `cmd::OpenTrade` takes `stocks.find(trade->npc)`; found: the kept stock is shown again (mode 0, first tab); not found: `open_store(...)` generates it on the proxy seed and the result is kept. The Gamble tab (`trade->gamble`) is generated fresh on every open (`open_gamble`). `OpenHire` regenerates `hire_list(act())` on every open (comment "ten offers (FUN_00576070)"). So:

| Rule | game.exe | d2d | Verdict |
|---|---|---|---|
| first open generates stock | yes ([rec+0] flag) | yes (`open_store`, kept) | SAME (V read both) |
| second open keeps stock | yes unless flag +0x23 or +0x27 set | always keeps | equal if those flags are never set in the bot's play (U) |
| new game | all flags zero: fresh | fresh | SAME (I) |
| level-up | no trigger read | none | no deviation known (U) |
| time | no timer found | none | no deviation known (U) |
| hire list | made once, flag [rec+1], only for classes 0x96/0x203/0xc6/0xfc | regenerated on every open | **D1 (V read both)**: is the list the same? `hire_list` uses the proxy/hire seeds; the offer lines are equal for a fresh list (7,200 lines), but d2d recalculating on every open may re-roll after the party or level changed; whether it re-draws the seed was not read (U) |

## Q3. Live lists (AC #4): not done

No game ran here. The TASK-56 live stock read (vendor probe, fidprobe phase 2) is the only source: not available in this run (U). When it exists, compare per vendor with `store.py`'s asks for the same map seed (d2d `--seed N`, `drlg-dump ... store` for that seed's proxy chain; the vendors' record order follows the NPC's first-open order, V read: a stock is made on the first open, so the first vendor opened takes the first proxy draws).

## Deviations (d2d against game.exe)

| id | what | tag | differential |
|---|---|---|---|
| D1 | hire list regenerated on every open in d2d, made once per NPC in game.exe | V read both | could be built: call `FUN_00576070` twice in the emulator with the flag handling (`store.py`'s harness hooks the record) and `OpenHire` twice in `test_game` with a player level change in between; oracle: the second list equals the first |
| D2 | flags `[+0x23]` / `[+0x27]` regenerate stock in game.exe, never in d2d | V read game.exe, U on the setters | trace the writers' structs (functions around 0x5b8b00 and 0x5bbcd0) and run those paths under the emulator |

## Bot findings

B1: any bot assumption that stock never changes after the first visit holds in d2d and in game.exe's normal path; a re-roll in game.exe after some NPC event (D2) is not excluded (U), so a plan that buys the same item across visits should re-read the vendor list (whether the bot does was not checked, U). No code changed.

## PR candidate

No PR candidate: generation equals; D1 and D2 are V read only and the existing differential (`store.py`/`hire.py`) does not exercise a second open.

## Drill recipe (lab, not run)

- Stock persistence: lab binary, throwaway character (docs/forkcheats.md setup), open Akara, `debug` the stock list (docs/forkdbg.md), close, wait 5 virtual minutes, level up, reopen: d2d expected identical (kept); the same on game.exe to settle D2 (U).
- Hire list: open the Rogue hire panel in Act 1 twice with a different character level between (cheat setup, a throwaway): compare the two lists; game.exe expected identical.
- Stock vs oracle: pin a map seed with `--seed N`, open the Act 1 vendors in the order the oracle uses, compare the asked codes and item levels with `d8_store.sh N-N` for that seed.

## Matrix-cell update (proposed)

VN04: "UPDATE 2026-11-09 (decomp8, fid-stock-oracle.md): stock asks (codes, count, quality, item level, seed chain) 0 differ on 83fc7ba1, 400 seeds x 17 vendors x 6 jobs = 40,800 lines plus 14,400 offer lines (V run); affix rolls hooked (U for a vendor-made magic item end to end); live lists not read (U)."
VN05: "UPDATE 2026-11-09 (decomp8): game.exe makes stock once per NPC at the first open (flag byte at record+0x20, V read 0x578bbc); two regenerate paths gated by flags +0x23 / +0x27 with untraced setters (U); no timer or level-up trigger found; d2d keeps stock after the first open (V read). Hire list: game.exe once per NPC, d2d on every open (D1, V read both)."
