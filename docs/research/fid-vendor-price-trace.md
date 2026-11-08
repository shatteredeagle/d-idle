# fid-vendor-price: vendor buy, sell, repair and quote prices, game.exe vs d2d (TASK-76)

> status: current (agent fidprice, Sonnet, 2026-11-07; card TASK-76 "fid-price-pr", matrix rows VN01 VN02 VN03 IT08 VN06). Result: **NEGATIVE, no PR candidate.** Every deviation the earlier traces proved (price-trace.md Q5: D1 to D10) is already merged upstream (ours and later maintainer commits); on the newest upstream main the price differential has no residual that is reachable with 1.14d data. Earlier docs: [price-trace.md](price-trace.md) (the function), [pr-price-proof.md](pr-price-proof.md), [pr-price2-proof.md](pr-price2-proof.md), [verify-price2.md](verify-price2.md), [pr-vendor-repair-gate-draft.md](pr-vendor-repair-gate-draft.md). Tooling: `tools/emu/fp_price_runs.sh`, `tools/emu/fp_clamp.py` (new), `tools/emu/price_edge.py` (existing).

Tags: V read (disassembly or decompile), V run (game.exe's own code in the emulator against d2d's real `item_price` / `repair_cost` through `test_game --price`), I inferred, U unverified. Addresses are game.exe 1.14d.

## Method

1. A scratch bare clone of the private fork remote, plus upstream `psi29a/d2decomped` main fetched into it (read-only fetch), gave **upstream main 9c78f391** (2026-11-08 00:07 +0100). Never the dirty dev checkout.
2. On the worker host (nice 19, load below 8 of 16 cores, another job running): a clean clone of that commit, `test_game` and `mpq-cat` built (Release, ccache), the user's game files for `D2_MPQ_DIR`.
3. `tools/emu/price_oracle.py` (upstream's own, now with the `mods` column for magic stats and staff mods) runs game.exe's `FUN_0062efb0` (buy 0, sell 1, repair 3) natively on the loaded tables; `test_game --price` prints `rules::item_price` / `repair_cost` of the same case. Compared byte for byte (`id buy sell repair`).

## Q1. Which price deviations are PROVEN, and where are they now?

| Dev | Rule (price-trace.md Q5) | Proof so far | State on upstream main 9c78f391 |
|---|---|---|---|
| D1 armor defence (cost x def / maxac) | game.exe `0x62f0f2` | pr-price-proof.md, verifier: all hold | merged (#152; `armor_scaled`, rules.hpp:451) |
| D2 class item sells a quarter | `FUN_0062c060`, `0x62f99d` | same | merged (#152) |
| D3 quality extras only when identified | `0x62f13c` | same | merged (#152) |
| D4 no repair of an unidentified item | `FUN_0062e660` | same | merged (#152) |
| D5 whole item repair = 1, Repair All skips it | `0x62fd5c` | price-trace Q5 (ponytail in #152) | merged upstream (0ad41c4f, d53f9728, aed7635a); rules.hpp `needs_repair` |
| D6 stack quantities, throw stacks | `0x62fc75`, `0x62fcb6` | price-trace Q5 | merged upstream (1f6822ed, 3b1dfca9, af240122) |
| D7 broken ethereal sells 1 | `0x62fa60` | pr-price-proof.md | merged (#152) |
| D8 stats, staff mods, sockets | `FUN_00628e70`, `FUN_0062edd0`, `FUN_006292f0` | I in price-trace | merged upstream (b26f78a7, b88364a6, 43b768d5, 6e54f3e7); the oracle now takes them as a `mods` column |
| D9 npc mult divides first above 0xffff | `0x62faac`..`0x62fc53` | verify-price2.md: 104,422 cases, 21,259 -> 0 | merged upstream (a9548d38, `price_term`, rules.hpp:650/830) |
| D10 repair gate (Repair type, durability) | `FUN_0062e660` | verify-price2.md: 2,235 -> 0 | merged upstream (bacec633, `repairable`) |

So the "fix/vendor-prices-followon-v2" branch (queue row of decomp-queue items 10 and 11) is **superseded**: opening it would re-add what main already has, with conflicts. It must not be opened. (V read of the upstream code and `git log`; the equality is V run, below.)

## Q2. How far apart are game.exe and d2d today? (V run, counted)

Cases are the oracle's own generator (`price_oracle.py gen`, all item kinds, qualities 1 to 8, identified or not, ethereal, durability states, stacks, 17 vendors, 3 difficulties, 0 to 5 random magic stats and staff mods per case), upstream main 9c78f391 against game.exe:

| Run | Cases | buy / sell / repair differ | Cases with a difference |
|---|---|---|---|
| gen seed 7 (the number price-trace.md Q5 gave: 39,626 / 38,939 / 42,705 before #152) | 100,000 | 1 / 0 / 13 | 14 |
| gen seed 21 | 60,000 | 0 / 0 / 11 | 11 |
| `price_edge.py` seed 11 (bases around 0xffff / 0x10000 and far above, gate items) | 60,000 | 7 / 3 / 112 | 122 |
| seed 7, plain stats clamped to -20..40 (`fp_clamp.py 40`) | 100,000 | 0 / 1 / 1 | 2 |
| seed 7, clamped to -20..12 | 100,000 | 1 / 0 / 0 | 1 |
| seed 7, stat 91 negative (-1..-50) plus 0..4 all-skills | 20,000 | 0 / 0 / 0 | 0 |
| edge cases, clamped to 40 | 60,000 | 2 / 0 / 24 | 26 |

**No mismatch in any case without magic-stat mods**: every one of the 14 + 11 + 122 differing cases carries a `mods` column (none of the 16,173 mod-free edge cases differs, and none of the mod-free cases of the two seed runs). VN01, VN02 and the buy/sell/repair part of IT08 are therefore **equal to game.exe on upstream main for all tabled items** (V run). That closes the 39,626 / 38,939 / 42,705 differences of the matrix rows.

### The residual (what is NOT equal), classified

| # | Family | Evidence | Tag |
|---|---|---|---|
| R1 | Repair when buy price x missing durability exceeds 2^31 (32-bit overflow): game.exe returns a positive price, d2d floors a negative one to 1 | 12 of 13 repair differences of seed 7, 11 of 11 of seed 21, 112 of 112 edge: product above 2^31. Bisect of one case (uui, jamella, durability 14/91, `127:0:N` all skills): N=5, 10, 20, 30 agree (repair 409,600 .. 2,353,024); N=40 (buy 29.6M, product 2.28e9) game 1,424,956 against d2d 1. The game value is also not what a plain 64-bit division would give (3.1M), so game.exe's wrap order is not d2d's | V run (bisect), I (cause) |
| R2 | One case family with a negative total before the staff mods (seed 7 case 55849: crafted `uml`, mods `3:0:206,297:0:27,156:0:-16,140:0:9,107:33:15,107:242:2`). Each half alone agrees, the whole differs (game buy 880,906, d2d 1) | single-mod and half-mod runs, `fp_price_runs` log | V run, cause U |
| R3 | Edge cases with stat 91 (requirement percent) set to +149..+253 and sells at the cap | 8740, 11652, 15213, 19053 of the edge run; with stat 91 negative as real items carry it, 20,000 cases agree | V run; the positive values are off-data (I) |
| R4 | One seed-7 repair difference whose product is 1.3e9, below 2^31 (case 42973, rare `uh9`, mods with a staff mod) | the `chk` listing | V run, cause U |

Why R1 to R4 are not PR material: the buy prices involved are 26 million to 1.4 billion gold, produced by random magnitudes (a +40 all-skills stat, 300 on a stat whose real range is a few dozen). When the random magnitudes are clamped to a realistic band the repair family disappears (clamp 12: repair 0 of 100,000); what remains in the clamped runs is still at prices of 30M and more (the stat cost tables carry multipliers of many thousands). No 1.14d item is priced anywhere near that (the price model comment in price-trace.md: bases stay below 2^20 in all item tables, I). Reachability in a real game is U: the missing piece is a generator that draws only stat ranges that Properties.txt / MagicPrefix / MagicSuffix / Unique / Set rows can produce (min..max per property), and then reports the largest game.exe price it ever sees. Without it, "no real item crosses 2^31 / 255" is an inference, not a measurement.

## Q3. VN03 consumable prices (V run)

Oracle cases for tbk, ibk, tsc, isc, hp1..hp5, mp1..mp5, rvs, rvl, key at Akara, Charsi, Gheed, Fara, Drognan, Lysander, Ormus, Malah, Normal/Nightmare/Hell, stack sizes 0, 1, 4, 5, 10, 20: 2,448 cases, **2,304 agree; the 144 differences are all `ibk` and are an oracle artifact** (the oracle's fake item has no book id, so game.exe reads the first books.txt row, the TP tome). With the item's first suffix word set to the book row (0 = tbk, 1 = ibk), `fp_price_runs.sh tomes` (144 cases, 4 vendors x 3 difficulties x 6 quantities x 2 tomes): **0 differ**. This also shows (V run) that game.exe picks the tome's per-scroll price from that word: ibk goes from 50 to 40 per scroll when the word is 1.

Single-unit values are the matrix row's (tsc 100, isc 80, hp1 30, mp1 60, key 45; the run printed only the tome lines, the others are covered by the 2,304 agreeing cases). Tomes: tbk 250 + 50 per scroll, ibk 200 + 40 per scroll (qty 4: 450 and 360, V run). The live 450 at clvl 3 (matrix VN03 "unexplained") is the TP tome at **4 scrolls: 250 + 4 x 50 = 450** on both sides (V run for the price at qty 4; that the stock tome held 4 scrolls is U, the stock quantity was not read: VN04's stock probe settles it).

## Q4. Quote (hover) and gold caps

The hover quote is the same function (`FUN_0062efb0` modes 0/1/3), so VN01/VN02/IT08 cover it. The purse cap (clvl x 10,000), the stash cap (2,500,000, `FUN_00623460`, V read) and a sale that would pass the purse cap were not re-run here (price-trace.md Q4 stays: purse cap I, sale over the cap U). Upstream's `store_repair_all` now pays through the stash path as `FUN_00576d90` (rules.hpp comment), V read only.

## Deviation list for d2d vs game.exe (D1.. renumbered for this card)

| ID | Deviation | Tag | Differential |
|---|---|---|---|
| D1 | Repair price when buy x missing durability passes 2^31 (R1): d2d floors, game.exe returns a positive price | V run, I (unreachable with 1.14d data) | yes, exists: `fp_price_runs.sh gen|edge` against `price_oracle.py`; a PR needs the realistic-range generator above and a proof that some real item reaches it |
| D2 | Negative pre-staff-mod total (R2) | V run, U (cause) | yes: bisect the mods of case 55849 and print the intermediate (`FUN_00628e70` result, `FUN_0062edd0` input) by hooking the two calls in the emulator |
| D3 | Product 1.3e9 repair difference (R4) | V run, U | yes: same bisect |
| D4 | Sale over the purse cap clamps (d2d) against game.exe's sell handler | U | would need the server sell handler (not traced): run game.exe's sell path in the emulator with a purse at the cap |

None is proven to be reachable, so none is a PR claim.

## Bot findings (described, not changed)

- B1 `internal/loot/loot.go:397` SellValue is cost / 2 with the ethereal and class quarters and quality factors only when identified, capped at max buy: **matches the rule** (V read of its comment against price-trace.md Q2; its tests pin it). The old Cost/4 estimate and the repair estimate (price-trace.md Q6 B1 to B4) are fixed. Left: the stat costs of identified magic/rare/set/unique items are not modelled in `SellValue` (docstring: lower bound), while upstream now prices them (b26f78a7): a bot sale estimate for an identified rare is low by the stat costs. That is harmless for gold plans (an understatement), I.
- B2 `internal/town/chores.go:229` `repairEstimate` sums `RepairPrice` from the durability and defence stats: V read, same rule.
- B3 the 450 tome threshold `town.tpTomeMinGold` (450) in decision-point-store-spec.md is consistent with a TP tome holding 4 scrolls; with a full stock tome (20) the price is 1,250 at Akara. U which the live vendor offers; the bot buys scrolls by gold, not by the stock tome price.

## PR candidate

**No PR candidate.** Reason: the proven deviations are all merged upstream; the only differences left need prices of tens of millions that no 1.14d item reaches (I), so there is no proven d2d deviation to fix with a mismatches-before > 0 / after 0 differential. No branch `local/fid-vendor-price-r1` was created; nothing was pushed to the private remote. The old follow-on branches (`fix/vendor-prices-followon`, `-v2`, on the worker host only) are superseded and may be deleted.

## What the verifier / next worker should do (if the coordinator wants R1 closed)

1. Re-derive: `FUN_0062efb0` mode 3 at `0x62f9c2` (R = (max - dur) x R / max; the `imul` at `0x62fa45`), then the rep mult step at `0x62fb1d`; check which operand game.exe widens or shifts when the 32-bit product is negative. Compare with rules.hpp `repair_each` (`cost = std::int32_t(std::uint32_t(missing * x)) / max`, then `scale(prices.rep)`).
2. Re-run `fp_price_runs.sh gen 100000 7` on a fresh checkout of upstream main with a different seed (11, 13) and confirm the plain (mods "-") share has zero differences.
3. Build the realistic-range generator (draw each property's value from the min..max of the MagicPrefix / MagicSuffix / UniqueItems / SetItems / Runes / Gems rows) and report both the largest game.exe price and whether any case differs. If it differs, the case is a PR claim (one commit with the test and the oracle in the tree); if it never differs, close R1 to R4 as unreachable and add them to docs/d2d-deviations.md as known 32-bit wrap differences.
4. Rule conformance for any resulting PR: one subsystem (store/prices), one change per commit with test and oracle, Claims table, sorted includes, `presubmit.py --quick`, `pr_guard.py`, `-Wshadow -Werror`, clang-tidy on touched lines, numeric noreply author and committer, last line `Agent: Claude Code`, no Co-Authored-By, `tools/emu` oracle in the PR tree.

## Drill recipe (fork verbs, nothing run here)

For each U above, in the lab (docs/forkdbg.md, docs/d2d-fork.md, docs/forkcheats.md) on a throwaway character, offline no-save game:

- Tome stock (VN03 "4 scrolls"): `dump vendor` at Akara in a new Normal game, read the tbk entry's quantity and price; expected quantity x 50 + 250 = the price.
- Realistic maximum price: `debug give <code> pack q4` for the dearest Charsi/Malah stock rare and a socketed unique, hover quote (`dump ui` rects) and compare with `price_oracle.py` on the same case line; a quote over 10 million would reopen R1.
- Sale over the purse cap: put gold at clvl x 10,000 - 1, sell a high-priced item, read the purse (the game.exe side needs the same on the original client: U, not available offline).

## Matrix-cell updates (proposed text for docs/research/fidelity-matrix.md)

| Row | Now | Proposed |
|---|---|---|
| VN01 | DEV; d2d buy differs 39,626/100,000 | EQ: game.exe equals d2d main 9c78f391 on every item the tables can produce (V run: the runs of Q2, no difference without magic-stat mods; residual only at prices of 26M+), fixed upstream by #152 and later commits; class C, effort 0 |
| VN02 | DEV; sell differs 38,939 | EQ: same runs, sell differs 0 + 1 + 0 in the plain runs; residual R3 off-data |
| VN03 | DM; live 450 unexplained | EQ: d2d equals game.exe on 2,448 consumable cases (tomes with the book id set: 144 of 144); the live 450 is a TP tome with 4 scrolls (I) |
| IT08 | DEV; repair cost differ 42,705 | EQ for the tabled range (13 + 11 + 112 repair differences all at prices over 26M, mostly above the 2^31 overflow, R1); gate and whole-item rules merged (D5, D10) |
| VN06 | DM; sell 50 % | unchanged: bot `SellValue` is cost / 2 (V read); the gold plans were not re-run (bot side of AC 4 open) |

## Reproduce

```sh
# on the worker host, clean checkout of upstream main with `build` linked to its build dir
tools/emu/fp_price_runs.sh TREE BUILD OUT gen 100000 7
tools/emu/fp_price_runs.sh TREE BUILD OUT edge 60000 11
tools/emu/fp_price_runs.sh TREE BUILD OUT clamp 40 OUT_OF_GEN/cases.tsv
tools/emu/fp_price_runs.sh TREE BUILD OUT tomes
```

## Dead ends and honest limits

- The first tome run (144 `ibk` differences) looked like a real deviation (ID tome 40 per scroll against 50); it was the oracle's missing book id (see Q3). Do not cite it.
- The clamped runs are still at prices over 26M, so they do not prove reachability either way.
- No game was run, no d2d slot used. Not run: the other test binaries, clang-tidy, presubmit (no code changed).

## Shared-file lines for the merger

See `backlog/merge-lines/agent-fidprice.txt`.
