# Review: batch C (mana on the action frame, the scheduler tick, the armor group), independent Opus check

> status: review, 2026-11-03, agent reviewbatch2c (the cport worktree was merged and removed while this ran, so this file is on its own branch). Subject: private fork branch `local/batch2-C-r3`, head `4ed8b651` (commits `7620ce97`, `eb1f87fa`, `4ed8b651` on batch A `41afdc6a`), and [pr-batch2-C-rows.md](pr-batch2-C-rows.md). Nothing built or run on the worker host or the Windows build worker; nothing pushed or posted. Tags: **V run**, **V read** (read in game.exe's disassembly or the branch, address given), **D** (data: the 1.14d Skills.txt / States.txt copies earlier cards extracted, one md5 across five copies), **S** (an earlier doc), **I** (inference), **U** (unverified).

How it was read: an objdump of the 1.14d game.exe (Intel syntax) from an earlier card's scratch, the PE tables read with a few lines of Python, the branch cloned read-only from the fork's private remote into a scratch dir, the cport card's run log (`cport-run.sh`, 683 s, 2026-11-03 12:46 EST) and its parity log.

## Verdicts

| Commit | Verdict | Why, in one line |
|---|---|---|
| `7620ce97` pay on the action frame | **ship with edits** | the cut-cast rule is now V read (below); the rows' reason for leaving `cast_summon` / `start_swing` at the command is wrong, and the evidence cited for "a cut cast costs nothing" is not the code that does it |
| `eb1f87fa` action frame on the scheduler tick | **ship with edits** | correct for the Sorceress SC record it was counted on; claim and body read as every class and need narrowing (or the oracle widened); it cannot stand without commit 1 |
| `4ed8b651` armor group | **ship with edits** (small) | the oracle runs the real `FUN_0056c740`; add the caller that proves a cast reaches it, and name the groups |

No blocker. Two high findings (H1, H2) are text that would be false or unsupported in front of the maintainer; both fix with edits to the rows and the commit bodies, no code change needed to ship.

## Question 1: does a get-hit clear the queued cast event? Yes (V read)

| Address | What it shows | Tag |
|---|---|---|
| `0x6e1740` (player mode table, 8-byte rows) | row 4 (GH) = `0x57fe30`; rows 7, 8, 10..16, 18 = `0x57fe90` (SC is row 10); row 9 (BL) = `0x57fdc0`; row 0 (DT) = `0x580ec0` in the second column | V read (bytes from the PE) |
| `FUN_0057fe30` (GH start) | `FUN_00553570` (mode), `FUN_005533d0`, `FUN_00620210`, then **`FUN_00553990`** at `0x57fe78`, then `FUN_005539b0` | V read |
| `FUN_00553990` | `FUN_00540e60(game, unit, 0, 0)` then `(game, unit, 1, 0)` | V read |
| `FUN_00540e60` | walks the unit's event list (`FUN_00553980`: unit `+0xdc`) and removes (`FUN_00540cd0`) each event whose type byte (`+0`) equals the argument; filter 0 = every value | V read |
| `FUN_005416b0` | files type at `+0`, frame at `+4`, value at `+0x14`, index at `+0x18`; a frame at or before the current one becomes current + 1 | V read |
| `FUN_005533d0` | the server mode start: resets `+0x4e`, `+0x30`, `+0x44`, runs `FUN_00623f50`, sets `+0x48`; touches no event | V read (agrees with S pr-get-hit-length-proof.md) |
| `FUN_0057fe90` (SC start), `0x57fdc0` (BL), `0x580ec0` (DT) | each calls `FUN_00553990` before scheduling (`0x57feca`, `0x57fe08`, `0x580f98`) | V read |
| `FUN_00623b10` for mode 10 | switch at `0x623b6b`: modes 9..14 return 0, so the oracle's hook "SC start frame 0" equals the real function | V read |

So the SC mode's type-0 action event (value 1) is removed when GH (or BL, DT) starts, before `FUN_00580460` can run the do: a cast cut before its event pays nothing. The oracle's `break` on `n > cut` and d2d's `set_pmode` clear model exactly this; the U in the rows and in SORCDIFF2 becomes **V read**. Still not run: the oracle does not execute `FUN_0057fe30` (a Python rule stands in), so the 21,084 cut cases test d2d against a read, not against running code. To make it V run: in `cast_mana.py`, after filing the SC events, call `FUN_00553990` natively at the cut tick against the real event list (needs `FUN_005417d0` un-hooked and a game struct with the `+0xb8` buckets; `verify_freeze_resume.Oracle` already runs that queue, S).

Same-tick order (the case generator avoids it; not claimed): `FUN_005414d0` dispatches each frame's buckets in the order given by the unit-type table `0x6e0b9c` = `[0, 1, 3, 2, 4]`: missiles (`+0x204`), players (`+0x4`), monsters (`+0x104`), objects, items (V read). Removing an event while a bucket is walked is safe (`FUN_00540cd0` moves the walker's next pointer at `+0xa18`, V read). So on the event's own tick a monster's melee blow comes after the player's event (the cast pays), a missile's hit before it (cut) (I: that the blow and the get-hit run inside the monster's or missile's own event). Worth one line in Honest limits.

## Question 2: does `owe` / `settle_cast` match where game.exe pays? For the bot's casts yes; the rows' rule is wrong elsewhere

Read in game.exe (V read):

- Start `FUN_0056f640`: `srvstfunc` (`+0x2c`) indexes `0x732140`. A null slot (srvstfunc 0, 30, 66..90) skips the start function and the payment (`0x56f773` -> `0x56f7af`). Otherwise the start function runs and, if it returns non-zero, **pays only when the row's flag `+8 & 0x20` is clear** (`0x56f78c..0x56f7aa`). That flag is `UseManaOnDo` (D; batch A's `sk_mana.py` reads it the same way, S).
- Do `FUN_0056f7f0` (from `FUN_00580460` -> `FUN_0056fc50`, call at `0x56fc92`): for a player or monster it pays (`0x56fa7a`) when the start slot is null **or** `UseManaOnDo` is set (`0x56fa57..0x56fa6e`).

Rule: **pay at the command iff the skill has a start function and not UseManaOnDo; else pay in the do.** No player skill has a non-zero srvstfunc with a null slot (D x V read), so for players "srvstfunc none" is the first half exactly. Against the branch (V read):

| d2d path | skills (D, player rows) | game.exe pays | branch pays | |
|---|---|---|---|---|
| `Fight::cast` -> SC branch settles (`strike.cpp:1008`) | Frozen Armor and the other SC self states, Teleport, curses | do | frame | agree |
| `Fight::cast_missile`, SC mode -> SC branch | Fire Bolt, Charged Bolt, Frost Nova, the SC missiles | do | frame | agree |
| `Fight::cast_missile`, A1 mode -> `strike()` missile branch (`strike.cpp:583`) | Magic Arrow | do | frame | agree (not in the cases) |
| `cast_missile` with a start function | Multiple Shot, Fire Arrow, Inferno, Telekinesis, Thunder Storm, Hydra, ... | command | command (`owe` pays at once) | agree |
| **Plague Javelin (25), Blade Fury (266)**: start function **and** UseManaOnDo | | **do** | command | **differ** (as before the batch; `owe` does not read the flag) |
| **`Fight::cast_summon`** (`summons.cpp:61`, `pay`) | Raven, Clay / Blood / Fire Golem, Valkyrie, Decoy, Oak Sage, Spirit Wolf, Grizzly, Vines, Poppy, Shadow Warrior / Master, ...: all **srvstfunc none** | **do** | command | **differ** (unchanged by the batch) |
| **`Fight::start_swing`** (`strike.cpp:306`, `pay`), srvstfunc-none rows | Smite, Frenzy, Double Swing, Hunger | **do** | command | **differ** (unchanged by the batch) |

No leak: every `owe` that records reaches a `settle_cast` (SC branch or the A1 missile branch) or is dropped by `set_pmode` to another mode; no srvstfunc-none spot skill has A1 / TH animation (D), the only route that would record and never settle. The bot's Sorceress casts (Charged Bolt, Frozen Armor, Frost Nova, Fire Bolt, Teleport) are all on agreeing rows.

## Question 3: the armor-group oracle runs the real function (V read)

`armor_group.py` calls `FUN_0056c740` natively with game.exe's States table (`0x744304 +0xbc`, 60-byte rows, group word `+0x1e`); hooked: the has-state test `FUN_00639df0`, the clear `FUN_00639db0` (recorded) and the stat-list lookup `FUN_006256b0` (returns 0, which skips `FUN_006277e0` / `FUN_00626cd0`, the stat removal; no effect on "kept"). The body (`0x56c740..0x56c83a`) is the group loop the doc describes. Notes: the oracle passes 0 as the argument while the self-state do passes 1 (`0x5c94f0`); the argument only decides whether the new state's own index is cleared, so the X != Y answers do not change (V read). Pairs with the same state (X != Y, sx == sy) are printed 1 without a call: consistent with the d2d side's `aurastate !=` exclusion, but not run. The link "a cast reaches it" is `FUN_005c9480` (srvdofunc 18: Frozen, Shiver, Chilling, Bone, Cyclone Armor, Burst of Speed, Fade, Venom), which calls it at `0x5c94f4`; it is not in the commit's evidence. Groups (D, States.txt): 1 = frozenarmor, shiverarmor, chillingarmor, bonearmor (and justhit); 2 = quickness (Burst of Speed, 258), fade (267); 3 = wolf, bear, feralrage, maul, delerium (223, 228, 232, 233, 350). Groups 1 and 2 go through do 18 (V read); group 3 is do 116 / 120, and whether those dos call `FUN_0056c740` was not read (other callers: `0x55b800`, `0x5beda0`, `0x5c6c50`, `0x5c6cb0`, `0x5c6ec0`, `0x5d7a00`, `0x5d8f50`; U which is which).

## Question 4: counts from the log (not re-run)

Not re-run: the Windows build worker's queue daemon was not running, and starting it for an 11-minute build load is not "cheap" on a machine whose gaming use comes first; the worker host is busy with Gate 1 seeds. From the cport run log (V read of the log): mana 53,628 cases, C head 0 differ, C~2 10,560 differ (first diff `48 1 7 640 7` vs `48 1 6 640 6`: action tick 7 vs 6); game side began 42,168, refused 11,460, began and paid nothing 21,084; d2d C began and paid nothing 21,084; armor 5,476 lines, 0 differ; A's sk_mana 105,978 compared, 0 different on both builds; Teleport seeds 1 and 9, 100,000 same on both builds; `test_cast_mana: ok`, `test_armor_group: ok`; parity lane on `4ed8b651` every stage rc 0 including tidy. These match the rows. Asserts stay on in Release (`tests/CMakeLists.txt` strips NDEBUG, V read), so the no-argument "ok" lines mean the asserts ran.

The run script's own rc fields are not real (V run of the same shell shape): `build ... rc=$?` prints the `$(git rev-parse)` substitution's status, not ninja's, and `test ... | tail -2; echo rc=$?` prints `tail`'s. The rows' "pass on the C head (rc 0)" holds only through the "ok" lines.

## Findings

| # | Rank | Finding | Where | Recommendation |
|---|---|---|---|---|
| H1 | high | "`cast_summon` and `start_swing` keep `pay` (a start function or ammo pays at the command, `0056f640`)" is false: every player summon and Smite / Frenzy / Double Swing / Hunger have srvstfunc none and pay in the do in game.exe (D x V read `0x56f773`, `0x56fa57`). "Inferno, ..., Hydra and the ammo skills still pay at the command" is true except Plague Javelin (an ammo skill) and Blade Fury, which have UseManaOnDo and pay in the do | pr-batch2-C-rows.md "How it sits" row 3 and Honest limits bullet 2; `tick.cpp` `owe` comment "A start function pays at the command" | rows: "`cast_summon` and `start_swing` still pay at the command; in game.exe summons and srvstfunc-none swings pay in the do too (not moved, not claimed); so do Plague Javelin and Blade Fury (UseManaOnDo)". Code comment: "a start function pays at the command unless UseManaOnDo (FUN_0056f640, 0x56f78c)". Optional follow-up commit: `owe` in `cast_summon` (the SC branch already settles before `summon`) and the flag in `owe` |
| H2 | high | "A cast cut off before it costs nothing" cites `00580460`, `0056bfe0`, which say nothing about the cut; the proof is the GH start clearing the event | commit `7620ce97` body line 2; Claims row 1 Evidence; Evidence table | add `0057fe30` (GH start calls `00553990` at `0x57fe78`), `00553990` (clears event types 0 and 1) and `00540e60` to the Evidence table (sorted into batch A's) and the Claims cell; body: "a get-hit start clears the queued event (FUN_00553990)"; Honest limits: the cut is V read, modelled not run in the oracle; the same-tick order |
| M1 | medium | `eb1f87fa` changes every class's SC action frame, but the oracle reads only the first Sorceress SC record (`records[0]` of `SOSC`) and the body's "FCR 9-18, 42-61, 105-199" is Sorceress-only | commit body, Claims row 2, `cast_mana.py` | widen (`animdata_records(blob, cls + "SC")` for the seven classes, the dense FCR loop per record; cheap) or word it "a Sorceress cast's action frame" in claim and body. Also say A1-mode casts (Magic Arrow) keep the prate scaling |
| M2 | medium | Commits 1 and 2 do not stand alone: 2 edits 1's new test file and its `strike.cpp` hunk has 1's `settle_cast()` in context; 1's counted check is 10,560 differ until 2 is in | PR body | a body line like batch A's Teleport note: "commit 2 needs commit 1; 1's differential reaches 0 only with 2 (the residual is the frame tick)". Better, optional: a `cast_mana` check of `paid_tick == fired_tick` plus the amount (0 differ at commit 1 alone), so commit 1 carries its own passing oracle |
| M3 | medium | Claims row 1 Re-check "before this commit every one of the 42,168 casts that began differs" was measured on upstream main plus the harness (`cb07926f`, SORCDIFF2), not on this commit's parent `41afdc6a` | Claims row 1 | "on upstream main plus the harness (SORCDIFF2) every one of the 42,168 differs; on A the harness cannot run before the move" |
| L1 | low | Armor commit Evidence is only `0056c740`; add `005c9480` (do 18 calls it at `0x5c94f4`). Honest limits can now name the groups (2: Burst of Speed, Fade; 3: Werewolf, Werebear, Feral Rage, Maul, Delerium Change) and say the dos of group 3 (116, 120) were not read | Claims row 3, Honest limits bullet 4 | edit as said |
| L2 | low | The Evidence rows in the C rows file are not in address order (`005539b0`, `00580460`, `0056f7f0`, `0056c740`); batch A's table is sorted | Evidence table | insert each row at its sorted place in A's table; run `pr_guard.py` on the body file |
| L3 | low | The run script's rc fields are not the build's or the test's (Question 4) | ops repo `cport-run.sh` | `set -o pipefail`; take `$?` right after ninja |
| L4 | low | `owe` keeps the `lvl` of the command; game.exe's do pays with the level the event carries; the same unless the level changes mid-cast (I) | `tick.cpp` | none now |

## Rules check per commit (V read)

- Subjects 65 to 67 characters, bodies of at most 3 lines, `Agent: Claude Code`, numeric noreply author, no co-author line: all three commits.
- One claim per commit with its test and oracle: 1 yes (harness, fix, test); 2 yes (rule and assert; its oracle is commit 1's); 3 yes.
- Includes of both new tests sorted, `<combat.hpp>` in place; `// test-links: game` present; new tests picked up by the `GLOB_RECURSE` in `tests/CMakeLists.txt`.
- Beyond H1, H2, M1 and M3, the Honest limits are accurate (Frozen Armor trigger not covered, player only, no live run).

## Shared-file lines for the merger

- BACKLOG: batch C rows: apply the H1, H2, M1-M3 edits of review-batch2-C-opus.md (rows file and commit bodies) before the PR update
- INDEX: docs/research/review-batch2-C-opus.md - independent check of batch C: cut-cast rule V read (FUN_0057fe30 -> FUN_00553990), pay sites vs game.exe, armor oracle runs the real function
- CHANGELOG: review of batch C: the cut-cast assumption verified by read; wording fixes before the PR update
