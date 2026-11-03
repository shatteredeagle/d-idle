# Batch 2 as two batches: skill casting (A) and monster damage roll (B)

> status: DRAFT bodies, not posted (card b2rebase2, 2026-11-03). Use these if the user chooses two batches (reasons: [pr-upstream-rules-20261103.md](pr-upstream-rules-20261103.md)); open A first, B after A merges ("keep one batch open at a time"). Both branches are on upstream main `005c7cab` on the private fork remote only. Evidence text, counts and limits are those of [pr-batch2-body.md](pr-batch2-body.md) (sections 1 to 3 for A, section 4 and the ponytail list for B); tags V / S / U as there. Not re-run for these heads: builds and suites of the split branches (their patches are `range-diff` `=` to the bundle's; V: A's tree equals the bundle head's parent tree, the merge of A and B equals the bundle tree).

---

## Batch A: Fidelity fixes against game.exe 1.14d: skill mana check and payment, Teleport target rules

Branch `local/batch2-mana-teleport`, head `dc6618ae`, 3 commits, 10 files, +389 / -12, no conflict (V: cherry-pick of `486e8ff1`, `4fd9b356`, `9dda1012`; `range-diff` `=` for all three).

### What changes

d2d refused a cast under the minmana floor and paid the floor; game.exe checks the formula alone and pays `max(formula, minmana x 256)` only with enough mana. d2d let Teleport land anywhere and refused a blocked target; game.exe lands only in the caster's room or its near list and moves a blocked target to the nearest free spot (rings 1 to 49).

### Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `0056c160` | the mana check: formula only, no minmana floor |
| `0056bfe0` | the payment: `max(formula, minmana x 256)`, nothing when short or mana and lvlmana are 0 |
| `00554ea0` | the Teleport warp: room test, then the spot search |
| `00463740` | the target's room in the caster's room or its near list |
| `0064e7b0`, `0064dea0` | the spot search: the subtile itself if free, else the first free square ring, 1 to 49, least |dx| + |dy| |

### Claims

| Commit | Claim or change | Evidence | Re-check |
|---|---|---|---|
| `de5a6bdb` | No minmana floor in the check; payment only with enough mana | `0056c160`, `0056bfe0` | `pb2_mana.sh 100000 7 BASE_BIN HEAD_BIN`: 190,840 cases, 105,978 compared, base 281 differ, head 0 (V, re-run for the bundle head) |
| `abcedaba` | Teleport lands only in the caster's room or its near list | `00554ea0`, `00463740` | `pb2_tele.sh 100000 1 BASE_BIN D1_BIN HEAD_BIN`: 40,974 not-near cases, 15,854 equal before, 40,974 after (V) |
| `dc6618ae` | Teleport lands on the nearest free spot of a blocked target (needs the previous commit) | `00554ea0`, `0064e7b0`, `0064dea0` | same run: 1,461 blocked-target cases, 0 before, 1,461 after; 100,000 of 100,000 equal at the head (V) |

### How to re-check

```sh
python3 tools/pr_guard.py
cmake --build build && ctest --test-dir build --output-on-failure
```

Plus the oracle calls in the table. Size: 10 files, 401 added + deleted lines, 3 commits.

Agent: Claude Code

---

## Batch B: A monster's hit is rolled in 256ths, absorbed, then reduced

Branch `local/batch2-monsterdmg-pr`, head `bbd339ad`, 1 commit, 9 files, +346 / -50, no conflict (V: cherry-pick of `824da884` onto main; git auto-merged `fight.hpp` and `tick.cpp`, `range-diff` `=`). It touches `fight.hpp` and `tick.cpp`, which batch A also touches, in other hunks: open it after A merged (rebase is then clean, V: merge-tree of the two is clean).

### What changes

d2d rolled a monster's hit in whole points, inclusive. game.exe rolls `(min << 8) + rand((max - min) << 8)`, raises the absorb event on the unreduced physical (Bone Armor, Energy Shield), then reduces the record, flat first, then percent, in 256ths. Energy Shield is NOT equal after this change (it runs after the armors on the reduced total; game.exe takes it per record field on the unreduced value); the body says so and lists the cases (main body section 4).

### Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `0057b420` | the roll in 256ths, min at least 1, max at least min + 1 |
| `0057c1e0` | the record: absorb event on the unreduced physical, then the reduction, the life stat takes 256ths |
| `0057bf80` | the reduction, flat then percent |
| `005c6370`, `005ca6c0` | Bone Armor and Energy Shield absorbs |

### Claims

| Commit | Claim or change | Evidence | Re-check |
|---|---|---|---|
| `bbd339ad` | Roll in 256ths, absorb on the unreduced physical, then reduce; fixed records (min == max) keep their value | `0057b420`, `0057c1e0`, `0057bf80`, `005c6370`, `005ca6c0` | `v4_run.py 6000 20261103` (354,000 frames, 0 differ on no-shield sets, base up to 47,896), `mdmg2.py 20000 20261103`, `v3_run.py 10000 314159` (S: run on the bundle's earlier head, same patch; to be re-run on the Windows build worker) |

### How to re-check

```sh
python3 tools/pr_guard.py
cmake --build build && ctest --test-dir build --output-on-failure
```

Plus the oracle calls in the table. Size: 9 files, 396 added + deleted lines, 1 commit.

Agent: Claude Code
