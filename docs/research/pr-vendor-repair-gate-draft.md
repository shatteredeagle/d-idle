# Draft PR body: vendor repair gate (local/vendor-repair-gate-v4)

> status: draft (card prbuildprices2, 2026-11-02). NOT posted: the PR hold is in force (no push to the public fork, no PR, no comment). Replaces the earlier two-claim draft [pr-price2-draft.md](pr-price2-draft.md): the divide-first half is already upstream, this is the repair gate alone. Verifier: [pr-prices-verifier.md](pr-prices-verifier.md). Tags: V = verified (how stated), S = secondary, U = unverified.

## Branch

- `local/vendor-repair-gate-v4`, ONE commit `ee2d03f9` on the private fork's remote, on upstream/main `48908176` (the merge of #174; rebased 2026-11-02, no conflict; the earlier `local/vendor-repair-gate-v3` `f0360487` was on `cb716f10`). Counts re-run on the new base: see [rebase174-20261102.md](rebase174-20261102.md).
- Built from the verified candidate `da80d0ab` (fix/vendor-prices-followon-v2) by: dropping `aed23fc8` (divide-first: upstream has it as `a9548d38`) and the divide-first half of the oracle commit `777df9d7` (no tooling goes in this PR); cherry-picking `da80d0ab` alone; resolving its one conflict in `tests/test_rules.cpp` (kept the bow / crossbow asserts; the `dear` assert stays upstream's); correcting the text (below). The old branch is untouched, no force-push.
- Author and committer: the numeric-ID noreply identity (V: `git log --format='%an <%ae> | %cn <%ce>'` shows the same on both). Message body ends with `Agent: Claude Code` only.
- Overlap with the open PR #174 (head `489ef83d`, based on `3852c904`): V, throwaway merge of this commit with #174's head: clean, no conflicts (`rules.hpp` and `test_rules.cpp` auto-merge in different hunks, `gamedata_load.cpp` too). Nothing to resolve.

## Title

Vendor repair: no price without ItemTypes Repair or durability (FUN_0062e660)

## What

`FUN_0062efb0` mode 3 calls `FUN_0062e660` at `0x62efee`; a 0 there makes the repair price 0. `FUN_0062e660` passes an item that is identified and not ethereal and either has charges missing (`FUN_0062a840`) or whose ItemTypes row has Repair (+8) and which restocks (`FUN_0062e5d0`: Repair, Throwable `FUN_0062ba80`, stackable `FUN_006289f0`, not ethereal) or has durability (`FUN_00629930`: nodurability 0, durability column > 0, maxdurability stat != 0, not indestructible). `rules::repair_cost` made only the ethereal / unidentified checks and priced any item with a saved max durability above its current one. New `ItemType::repair` (ItemTypes "Repair", loaded in `gamedata_load.cpp`) and the two tests in `repair_cost`. Docs: `docs/research/re/store.md`.

What it changes in play (S): d2d's item maker gives every weapon a drawn durability, so a dropped bow could sit below its max and d2d priced its repair; game.exe prices 0 for it (all of vanilla's bows have nodurability 1). In vanilla data the ItemTypes Repair half is redundant: all 186 bases of the 66 types without Repair (bows, Amazon bows, throwing potions, potions, ...) have nodurability 1, as do crossbows (V: ItemTypes and weapons rows read from the MPQs by the verifier). It is kept for fidelity with mods.

## Evidence (V, re-run on the rebased head)

game.exe's own code under the emulator against the real `rules::repair_cost`, 7,908 gate cases (every one of 659 ItemsTxt bases x 12 states: durability half / whole / 0 / no max, charges none / full / missing, 10 % unidentified, 10 % ethereal, 10 % indestructible, quantity 0..2) and 60,000 price-multiplier cases (seed 2026). Base = upstream main `cb716f10`, head = this commit, both g++ with `-DD2D_WERROR=ON`:

| set | base (upstream main) | head |
|---|---|---|
| gate, something to repair (4,664): d2d prices, game refuses | 143 | 0 |
| gate, whole item (3,244): d2d prices, game refuses | 17 | 0 |
| gate, whole item: game passes (price 1), d2d 0 (the known sentinel: Repair All skips it) | 626 | 626 (unchanged) |
| mult cases, price above 0xffff (25,674): buy / sell / repair differ | 0 / 0 / 0 | 0 / 0 / 0 |
| mult cases, price at most 0xffff (34,326): buy / repair differ | 8 / 10 | 8 / 10 (harness: prices that reach 0 in the mult block, d2d's later minimum 1) |

Nothing that matched on base differs on head (per-case join). The mult rows are there to show the change does not touch the npc mults: upstream already divides first, and the verifier's earlier head-versus-old-base counts (25,631 / 3,444 / 25,477 wrong) belong to the dropped commit.

Suites on the rebased head (the data dir set, no SKIP): `test_rules`, `test_game_rules`, `test_monsters`, `test_monster_thinks`, `test_monster_ai`, `test_combat`, `test_game`: all exit 0.

CI parity: g++ and clang++ `-fsyntax-only` with the CI flags plus `-Wshadow -Wrange-loop-construct -Wconversion` over `test_rules.cpp`, `gamedata_load.cpp`, `world/commands.cpp`, `apps/d2d/store.cpp`: 0 warnings in the changed files (clang++ stops with errors in other, untouched headers because of the build's `-Werror`; none in ours). MSVC /W4 by reading: no nested redeclaration (`info`, `kind`, `mends`, `durable` are new names), no narrowing, no windows.h macro names. No new `#include` (no new std or project symbol). `pr_guard.py upstream/main HEAD`: 0 FAIL, 0 WARN. `presubmit.py --quick`: 0 FAIL; WARN build (skipped, built by hand above); WARN doc-claims on `store.md` lines 208-209 ("all", "never"): true for vanilla data (186 of 186 bases, counted), and the text says so.

## Limits

- With charges missing, game.exe's refused item still adds its full base x rep mult (`0x62f9e8`: the durability scaling is skipped when `FUN_00629930` is false); d2d charges the charges only (`// ponytail:` in the code). No vanilla bow or throwing potion carries charges that we know of (S).
- `indestructible()` treats a negative stat 152 as indestructible; game.exe does not. Unreachable (U that no mod writes one).
- No magic stat costs and no sockets in the comparison; det scenario hashes were not run (they contain no vendor action: S).

Agent: Claude Code
