# Review of community PRs 19 (PALAURA), 22 (PALDUR) and 23 (PALSKILL), 2026-11-02

> status: card reviewprs, 2026-11-02, offline maintainer review (Opus reviewer). Nothing was posted on GitHub, no issue touched, nothing pushed. Verdicts are PROVISIONAL: the review had a 60-minute limit; every check is marked VERIFIED (with how) or NOT RUN.

Tags as in the repo: **V run** = run by this review (game.exe code under upstream's emulator on the worker host through `tools/emu/run-on-worker.sh`, or the patched 1.14d tables read with `tools/emu/mpq-table.sh`); **V read** = read here (game.exe disassembly by upstream `disasm.py` through `tools/emu/palaura_dis.sh`, or d2d source at the commit the PR cites); **S** = what a PR or one of our traces says; **U** = not checked.

PR heads reviewed: #19 `903b71d6`, #22 `59bab2b4`, #23 `041386f3` (each one commit, one doc). The three PRs touch only `docs/research/`.

## 0. Context the user should weigh (not for the comments)

- Timeline (issue comments, V read with `gh issue view N --json comments`): the contributor claimed #1 PALAURA at 03:16Z (the maintainer answered the claim at 03:39Z), #2 PALDUR at 15:37Z and #3 PALSKILL at 16:10Z. Our PALAURA, PALDUR and PALSKILL jobs were launched at 14:49Z without checking the claims. For PALAURA that was a launch over a standing, acknowledged claim: our process failure. For PALDUR and PALSKILL our jobs were already running when the claims came, but the board still showed both items open, so the contributor had no way to know: also ours (the board was not updated at launch).
- The contributor's turnaround was fast (PR 22 about 30 minutes after the claim, PR 23 about 25 minutes): both say they were written by an AI agent. That alone is fine under the guide; it is why the reproducibility asks below matter.
- Per the user's decision (coordinator message): contributor work is primary where sound, merged with its own commits; our traces stay as an independent second trace under their own names, cross-linked; the draft comments in section 5 are technical only and do not mention the overlap.

## 1. Cross-PR findings

| # | Rank | Finding | Evidence | Recommendation |
|---|---|---|---|---|
| X1 | high | **None of the 300,000 (PR 19), 100,000 (PR 22) or 120,000 (PR 23) "V run" cases can be reproduced**: no script is in the PR or the repo, PR 23 names no seed, and "0 mismatches" is against a formula the author wrote from the same reading (self-consistency, not a differential against d2d). Guide `docs/contributing/agent-work.md` section 6: "Every number can be reproduced by a command in the document"; section 3: "A number without the command that produced it is U". | V read (diffs: one `.md` each; `grep tools/emu` in the docs finds no script name) | Ask for the harness: either the scripts under `tools/emu/` (allowed here with maintainer review; we ask for it explicitly, so the "only if the issue asks" rule is met) or the exact existing script plus arguments. Until then re-tag the counts as "V run (harness not published)" and do not cite them as V. |
| X2 | medium | **Commit author is `Claude <noreply@anthropic.com>`**, not the contributor (guide section 5: "Commit as yourself"). The user wants their authorship kept: a merge keeps exactly this identity, so the contributor is not credited by git. | V read (`git log -1 --format='%an <%ae>' pr-N`) | Ask the contributor to re-commit as themselves (agent line in the trailer is welcome), or merge as is and credit by handle in the doc header and the credits file. |
| X3 | low | No `Author: <handle>` header (guide section 8 proposal). | V read | Ask for one line; trivial. |
| X4 | info | leakcheck `-mode fail`: 0 hits on all three heads; `tools/ci/gamecode_guard.sh`: ok on all three; `go run ./cmd/changecheck -cli`: 0 warnings on all three. No hostnames, paths, secrets, disassembly listings or table dumps seen in a read. The game.exe and Patch_D2.mpq SHA256 values in PR 19 are public file hashes: fine. | V run (worktrees of each head) | none |
| X5 | info (ours) | The "D" tag means "from the patched tables" in PR 19 and in our own traces, but "a deviation of d2d" in the guide section 2. The contributor copied our convention. | V read | Our docs issue, not theirs: fix the guide wording separately. |

## 2. PR 19: PALAURA (`docs/research/pal-aura-trace.md`, +323)

**Scope (V read, issue 1 and brief):** answers all four open parts (filter bits and area shape, enemy pulse damage, Prayer mana and state 85, D5/D6), stays out of the "do not redo" part, ends each part with a d2d probe and expected numbers as asked. Good fit.

### Spot checks

| Claim (PR 19) | Check | Result |
|---|---|---|
| Area search: `dx^2 + dy^2 <= r^2` inclusive, integer subtiles, caster skipped, near rooms only | S: our palaura-trace.md section 1 (2,032 placements, V run there) | **Agrees** with ours. |
| Meditation `aurafilter = 0x12001`: merc and pets never get it | V run: `palaura_apply.py friendly 120 1 none 100`: aurafilter 0x12001; caster and party ally get state 48; merc and pet get nothing. Skills table: Meditation aurafilter 73729 | **HOLDS.** Not in our trace: an addition. |
| Holy Freeze `0x4a283` skips prime evils | Skills table: Holy Freeze aurafilter 303747 = 0x4a283; MonStats: Andariel primeevil 1. S: our palaura-trace (Andariel untouched, V run there) | **HOLDS**, agrees with ours. |
| Pulse roll `elo + rand(ehi - elo)`, one roll for all targets | S: ours (6,000 pulses: 256..767, 768 never) | **Agrees.** But PR 19's own probe 2.6/1.5 text says "loses 256..768": inconsistent with its formula (top is 767). |
| `check_noaura`: Charged Bolt Sentry (411) has `noAura = 1` and is skipped; Shadow Warrior (417) has 0 | V run: `palaura_flags.py` (bit 27 = column noAura, 5 rows) and `mpq-table.sh monstats`: noAura = 1 only on rows **340..344 (bone prison 1..4, bone wall)**; 411 chargeboltsentry and 417 shadowwarrior both empty | **FAILS.** The "multi-unit scenario" result `[1, 2, 5]` (trap skipped by noAura) cannot have come from the real 1.14d MonStats row. Ask how the scenario was built; this lowers trust in the other "V run" rows until X1 is met. Ours has the right rows. |
| Prayer: "When `cur_mana >= cost`: heals, sets state 85, deducts `cost` once per pulse" | V run: `palaura_apply.py friendly 99 1 none 100` (nobody hurt, mana 100): cost 0x100, applied counter 0, **mana 100 -> 100, state 85 off**; `... friendly 99 1 caster 100` (caster at half life): applied counter 1, life 40 -> 42, **mana 100 -> 99, state 85 on** | **FAILS as stated.** Mana is paid and state 85 set only on a pulse where at least one heal raised a life value (the callback counter), exactly as our palaura-trace says. PR 19 has the counter in its table (0x5cf229) but ties it to the mana gate only. The insufficient-mana half (no heal, no payment, state 85 cleared) agrees with ours. |
| State 85 stops "all natural and % mana regeneration (0 per frame)" | V run: `palaura_apply.py regen`: state 85 on gives 0 with stat 26 = 0, but **512 with flat stat 26 = 512** (off: 520) | **HOLDS WITH LIMITS**: PR 19's own table puts flat stat 26 after the skipped block, so the detail is right; the summary "completely, 0 per frame" overstates. Agrees with ours. |
| Holy Shock min synergy skip, Holy Freeze `coldeffect >= 0` gate, passivestate rule vs d2d `passive_stats()` | | **NOT RUN.** Additions beyond our trace; plausible, U from our side. |

### Comparison with our palaura-trace.md (for the user)

- Agree: area shape, 0x12003 and 0xa783/0x4a283 meanings, roll range, insufficient-mana behaviour, state 85 on regeneration.
- PR 19 is wrong, ours right: Prayer's payment condition (V run above); noAura rows (V run above).
- PR 19 adds (sound where checked): full bit decode incl. Conviction (no LOS), Sanctuary, Redemption, Meditation's 0x12001 (V run holds), Holy Shock / Holy Freeze special cases and the passivestate deviation (not run).
- Ours adds: the 4 out-of-near-room placements, the resist / pierce formula, published scripts (`tools/emu/palaura_*.py`).

**Verdict: ACCEPT WITH CHANGES** (asks in the draft comment). After the fixes, merge as the primary PALAURA trace; add one line to ours pointing to it as the primary and theirs to ours as the independent second trace.

## 3. PR 22: PALDUR (`docs/research/durability-trace.md`, +300)

**Scope:** answers the item (chance and slot for hit, struck, block; d2d comparison from source). Good fit; the "small" deliverable became 300 lines but stays on topic.

### Spot checks

| Claim (PR 22) | Check | Result |
|---|---|---|
| 4 % weapon, 10 % armor, 10 % throwable, weights 3:5:4:4:2:2:2, blocked blow wears nothing | S: our paldur-trace (V run there, 0 differ vs d2d in 34,939 armor cases); d2d `missiles.cpp` lines 130..180 at 6935cf6 (V read: `chance = armor or throwable ? 10 : 4`, the seven slot weights) | **Agrees** with ours; the d2d citation **HOLDS** at the cited commit. |
| Throwable wear gated on `[game + 0x70]`, which PR 22 names the Expansion flag | V read: `cmp dword ptr [eax + 0x70], ebx` at 0x559e88 in FUN_00559e30 | The gate **HOLDS**; ours found it and left its meaning U. "Expansion" is not shown in PR 22: ask for the evidence (where the field is set). If it holds, it settles our open U. |
| Amazon Impale has its own durability wear `FUN_005daa40` (chance and amount from params), breaking via `FUN_0055f850` | V read: FUN_005daa40 calls FUN_00629930 (the durability check) and FUN_0055f850 (the break); it is called at **0x5dac99** inside the Impale do FUN_005dabe0 | **HOLDS.** Our paldur-trace says "the whole rule is three functions": **incomplete, the contributor is right.** The call-site address differs (PR: 0x5dad1a; seen here: 0x5dac99): ask to recheck. |
| Smite "(`FUN_005cea30`) ... calls `FUN_0057d4f0` (0x5cec35)" | V read: 0x5cea30 decodes mid-instruction (not a function start); the Smite do is FUN_005ce9f0 (PR 23 itself uses that), whose call to FUN_0057d4f0 is at 0x5cec35 | Address **wrong**, rule right: fix the tag. |
| D1 (d2d wears armor on monster missiles, corpse explosion, death blasts) | S: ours has the same D1 (game side is inference from the call graph in ours too) | Agrees; both are I for the game side. |
| D2 (d2d never sets the broken flag on a weapon worn to 0), 100,000 cases seed 0x50414c44 | | **NOT RUN.** Our trace has no D2 like this; plausible addition. Cases: see X1. |

### Comparison with our paldur-trace.md (for the user)

- Agree on every core number and on blocking. Ours has the real d2d differential (0 differ, published `pal_wear.py`, plus the verifypaldur independent check); PR 22's 100,000 cases are unpublished.
- PR 22 adds: Impale (right, ours missed it), a candidate meaning of game +0x70, and the weapon broken-flag D2. Ours adds: the differential, the seed-draw difference.

**Verdict: ACCEPT WITH CHANGES** (address fix, Impale call site, +0x70 evidence or tag I, harness). Merge as the primary PALDUR trace; we should add the Impale path to our paldur-trace as a correction crediting PR 22.

## 4. PR 23: PALSKILL (`docs/research/pal-skill-trace.md`, +175 / -108)

**Scope:** answers the item's question (dispatch, hit count, AR/ED, mana/life, knockback/stun, held button, d2d `skill_built`). But the brief named an existing file as the deliverable, so the PR rewrote it. **That collision is our board's fault** (the PALSKILL brief's `deliverable:` is the filename of the 2026-11-01 Holy Bolt / Hammer / aura trace); the rewrite itself still has to be undone.

### What the -108 lines removed (V read: `git diff origin/main...pr-23`)

Every line of the existing doc was rewritten. Lost or changed: the status line with the d2d pin (upstream 72132b99) and the method; the tool names with their commands (`pal_flags.py`, `pal_dmg.py`, `pal_aura.py`, `pal_conc.py`, `pal_hammer.py`, `pal_hammer_hits.py`); the whole drill recipe (H0..H4, A1..A3, the Might mana drill); the "PR candidate: none" reasoning; the Q1..Q9 answers condensed. And the summary now reads "0 differ of 128,640 cases ... game.exe equals d2d / formula verification": it adds the unscripted 120,000 formula cases to our 8,640-case d2d differential, so a self-check is presented as part of a d2d differential. Not justified: the old doc is another author's evidence and other docs link into its sections (`palaura-trace.md`, `palskill-trace.md`, `skcurseval-trace.md`, `nec-skill-trace.md`, `docs/coverage/*`).

### Spot checks

| Claim (PR 23) | Check | Result |
|---|---|---|
| d2d `skill_built` builds all three (fight.hpp 62-75) | V read: upstream fight.hpp lines 68-69 (kRepeatedHit, kSacrifice, kSmite) | **HOLDS**; agrees with our palskill-trace. |
| Rows: Zeal ToHit 10 / LevToHit 10, calc1 `min(par5 + lvl - 1, par6)` (2, 5), Param2 100; Sacrifice ToHit 20 / 7, mana 0, calc2 = par3; Smite mana 2, minmana 1, manashift 8, calc2 `min(250, ln12)`, no start function | V run: `mpq-table.sh skills` rows 96, 97, 106 | **HOLDS**; agrees with ours. |
| Zeal mana paid once at the start function | S: ours says the same (SKMANA) | Agrees. |
| Smite forces the hit: `or ax, 1` at 0x5cebb0 after FUN_0057ec10 | V read: call FUN_0057ec10 at 0x5ceba8, `or ax, 1` at 0x5cebb0 | **HOLDS** (instruction level). |
| D8: Smite unblockable in game.exe; d2d lets a block stop it (`combat.hpp:499`) | V read: d2d combat.hpp line 499 at 6935cf6 tests `target.block` after the `!swing.smite` hit roll, so d2d does let a block stop Smite | d2d side **HOLDS**. Game side: FUN_0057ec10 sets the block bit 0x10 and clears bit 0; `or ax, 1` sets bit 0 again with 0x10 still set. Whether damage then lands is **NOT RUN** on either side. **Disagreement with ours:** our palskill-trace D6 says game.exe also lets a block stop Smite, citing only d2d source (S). The `or ax, 1` makes ours doubtful: if a run confirms PR 23, ours is wrong and D8 is a real d2d deviation (candidate). Needs one run: FUN_005ce9f0 with the block step answering "blocked", then the target's life. |
| D9: Zeal's later hits roll back frames (Param2 100), d2d waits the full animation (`strike.cpp:901`) | V read: d2d strike.cpp around line 901 at 6935cf6: next Zeal hit only when `now_ms >= pmode_until` | d2d side **HOLDS**; game-side timing (about 4 frames) **NOT RUN**. Not in ours (our D9 is about Zeal's reach): an addition. |
| Sacrifice self-damage 8 % of dealt capped at target life | S: ours agrees on the cap; ours adds the timing question (D8, life read after the hit) | Agrees. |
| 120,000 cases, 0 mismatches | no script, no seed | U (X1). |

**Verdict: ACCEPT WITH CHANGES, structural:** restore the existing doc unchanged and make the Smite/Zeal/Sacrifice work additive: either as a new top section plus D8/D9 rows appended (old lines byte-identical), or as its own file `docs/research/pal-smite-zeal-sacrifice-trace.md` linked from the old one. We should fix the PALSKILL brief's deliverable name in open-work.md (ours). Merge after that; cross-link with our palskill-trace.md (independent second trace). If D8 holds after a run, our palskill-trace D6 needs a correction crediting PR 23.

## 5. Draft review comments (for the user to post or edit; technical only)

### PR 19

> Thanks, this is a thorough trace and it answers every open part of the item. The filter decode is very useful, and we re-ran the Meditation case on our side: 0x12001 really does leave the merc and pets out. A few things before we merge:
> 1. Prayer's payment: in our run of FUN_005cf010 with everyone at full life and 100 mana, the counter at 0x5cf229 stays 0, no mana is paid and state 85 is not set; with the caster hurt, one heal lands, 1.0 mana is paid and state 85 is set. So the cost and state 85 depend on at least one heal raising a life value, not only on `cur_mana >= cost`. Could you correct sections 3.1, 3.3, 3.4 and the summary?
> 2. noAura: in the patched 1.14d MonStats only rows 340..344 (bone prisons and bone wall) have noAura set; Charged Bolt Sentry (411) does not. Could you check how the multi-unit scenario in 1.4 got the trap skipped?
> 3. Reproducibility: please add the harness for the three suites (scripts under `tools/emu/` are welcome in this PR; we are asking for them), or name the existing script and arguments. "0 mismatches" against your own formula is a self-check; please say so where it is not compared with d2d.
> 4. Small: with state 85 the flat stat 26 is still added (your table shows that; the summary says regeneration stops completely); probe 1.5 says 256..768 while the roll's top is 767.
> 5. Please commit as yourself (an agent line in the trailer is welcome) and add an `Author:` line to the header.

### PR 22

> Thanks, this matches what we see for the core rule (4 % / 10 % / 10 %, the 3:5:4:4:2:2:2 weights, no wear on a block), and the Impale path is a real find: FUN_005daa40 does its own durability check and break. A few asks:
> 1. Smite's tag: 0x5cea30 is not a function start; the Smite do is FUN_005ce9f0 (its call to FUN_0057d4f0 is at 0x5cec35, as you say).
> 2. Impale's call site: we see the call to FUN_005daa40 at 0x5dac99 inside FUN_005dabe0, not 0x5dad1a; could you recheck?
> 3. `[game + 0x70]` as the Expansion flag: what shows that? If it is inferred, please tag it I.
> 4. Please add the harness for the 100,000 cases (seed 0x50414c44): scripts under `tools/emu/` are welcome, or the command of an existing one.
> 5. Please commit as yourself and add an `Author:` line.

### PR 23

> Thanks for the Smite, Zeal and Sacrifice trace. The rows, the `skill_built` reading and the d2d lines you cite (combat.hpp:499, strike.cpp:901) all check out on our side, and the `or ax, 1` after FUN_0057ec10 in Smite is exactly where you say. The item's brief pointed you at an existing file, which is our mistake, but the PR now rewrites that whole document, and other docs link into its sections. Could you:
> 1. Restore the existing content of pal-skill-trace.md unchanged (drill recipe, tool names, commit pin, PR-candidate note) and add your work as a new section with D8 and D9 appended, or put it in its own file, for example `docs/research/pal-smite-zeal-sacrifice-trace.md`, linked from the old one?
> 2. Keep the case counts apart: the 8,640 element cases are a d2d differential with published scripts; the 120,000 Smite/Zeal/Sacrifice cases are a formula check. Please add their script (under `tools/emu/` is welcome) and seed.
> 3. D8: the forced hit bit is clear, but the block bit 0x10 stays set; does the damage land on a blocking target? A run of FUN_005ce9f0 with the block step answering "blocked", then the target's life, would settle it. Same for D9's game-side frame count.
> 4. Please commit as yourself and add an `Author:` line.

## 6. Commands used (reproduce)

All from the worktree of this card (`local.env` copied from the main clone, gitignored); outputs in the scratch dir, not committed.

    tools/emu/run-on-worker.sh --ttl 300 palaura_apply.py friendly 99 1 none 100     # Prayer, nobody hurt
    tools/emu/run-on-worker.sh --ttl 300 palaura_apply.py friendly 99 1 caster 100   # Prayer, caster hurt
    tools/emu/run-on-worker.sh --ttl 300 palaura_apply.py friendly 120 1 none 100    # Meditation reach
    tools/emu/run-on-worker.sh --ttl 300 palaura_apply.py regen                      # state 85 on regeneration
    tools/emu/run-on-worker.sh --ttl 300 palaura_flags.py                            # noAura bit rows
    tools/emu/run-on-worker.sh --ttl 120 mpq-table.sh monstats | skills              # rows 340..344, 411, 417; skills 96, 97, 106, 114, 120
    tools/emu/run-on-worker.sh --ttl 120 palaura_dis.sh 5daa40 5dabe0 559e30 5ce9f0 5cea30   # Impale, +0x70 gate, Smite
    # d2d: upstream checkout on the worker host, `git show 6935cf6:components/rules/combat.hpp` (line 499), fight/strike.cpp (about 901), fight/missiles.cpp (130..180); fight.hpp lines 68-69
    # per PR head worktree: go run ./cmd/leakcheck -mode fail; bash tools/ci/gamecode_guard.sh; go run ./cmd/changecheck -cli

Note: these spot checks reuse our own palaura scripts, which is fine for checking a contributor's claim but is not an independent verification of OUR trace.

## 7. Not done

- NOT RUN: Smite against a blocking target (decides PR 23 D8 vs our palskill D6), Zeal frame timing, Holy Shock / Holy Freeze special cases and passivestate (PR 19), PR 22 D2 (broken flag), the PR 23 rewrite of the Holy Bolt / Hammer sections against the old text line by line.
- The fixes on our side (paldur-trace Impale correction, possible palskill D6 correction, the PALSKILL brief's deliverable name, the guide's D-tag wording) are follow-up cards, not done here.

## 8. Shared-file lines for the merger

In `openspec/changes/reviewprs/merge-lines.txt`.
