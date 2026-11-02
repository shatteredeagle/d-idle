# Upstream PR candidates, inventory and ranking (card prcandidates, 2026-11-02)

> status: current, read-only inventory. Nothing pushed, nothing posted, no branch changed. Standing rules (user, 2026-11-02): up to 5 more PRs, each only after an independent Opus verification with its own harness, one Opus agent at a time, related commits bundled. Tags: V = verified by a run or a read in this card (how stated), S = secondary (a doc), U = unverified.

## 1. How this was made

- V: the private fork checkout and the PR scratch checkout on the worker host, after a read-only `git fetch upstream` (main 3852c904, "Coverity: init on the project's slug"). For every branch (local, private remote, public-fork remote) `git rev-list --count upstream/main..<branch>`, `git cherry`, merge-base distance and a subject match against `git log upstream/main` (a rebased or reworded commit shows as unmatched in `git cherry`, so the subject match decides).
- V: PR states from `gh pr list --repo psi29a/d2decomped --author <fork owner> --state all` (read only).
- S: the PR drafts, proofs and verifier records under docs/research, the ledger, BACKLOG, decomp-queue and open-work.

## 2. Corrections to the brief

- **#170 and #171 are MERGED**, not open (merge commits e09922c0 and 4bf911b3 on upstream main). Open PRs of ours are only **#159, #160, #163**. Merged: #152 #153 #155 #156 #157 #158 #164 #166 #167 #170 #171 (and older #127 #128 #134 #135 #136 #143 #146 #147 #149). The 5-slot budget therefore may be counted against the user's own number, not against five open PRs.
- The private fork's `~/d2d-lab/fork/src` holds only the lab patch set (never a PR: lab* and dethash* branches, GPL and docs/decisions.md) plus merged PR branches. The older candidate branches live in the PR scratch checkouts under the same lab directory.

## 3. Branches that are NOT candidates (already upstream, in an open PR, or private)

| Branch (head) | What it is | Status |
|---|---|---|
| fix/monster-base-velocity, fix/monster-think-paces, local/crfollow170, fix/player-missile-damage-roll, pr/missile-combined, pr/player-roll*, local/missilediff-record*, local/missilepoison-skillrows*, local/nmlevel* , local/monsterpace* | 0 commits ahead of upstream main (or only superseded copies) | merged as #164 #166 #167 #170 #171 |
| local/skabrecheck-phys (5b13a2c4), local/playerroll-lab (51e4eb1e) | the missile physical roll keeps its 256ths and the -100 floor | merged inside #166 (its commits 1ea79c57 and d67720e0); the 12 and 15 commits ahead are lab patches under one real commit. DROP. |
| trace/click (ff0d8b2f), trace/price, fix/tome-use-27113a31, fix/walk-velocity-v2 and v3, fix/nexthit, fix/static-field-chill, fix/waypoint, fix/bundle-get-hit-render-lut | subjects found on upstream main under other hashes (ecb267db, 97a590c7, 1f8f739a, 1c3fa4e5, e4a13181, 8e5ff4cd) | merged. DROP. |
| fix/armor-hit*, fix/freeze-resume*, fix/frozen-armor-and-resume* (c4e9db97) | Frozen Armor and freeze resume | open as #159 |
| fix/curse-immune-final (802dee68), fix/blizzard-shard-offsets-v2 (3ac63fa0) | curse on immune, Blizzard shard offsets | open as #160, #163 |
| lab, lab-*, lab2*, lab3..lab6, d2dtown-vendor, main (fork), dethash-* | lab patch set and harness | never a PR (decisions.md) |
| ghfork/pr-assets, ponytail, gap-map and similar | upstream-side work or docs of other authors, 0 matched candidates | not ours to send |

## 4. Candidates

Fields: branch and head (private fork), commits and files (against the merge base), deviation, evidence, risk, CI parity, draft, recommendation.

### C1. Freeze rules (fix/freeze-rules-v2, 409903a6), the one built and Opus-verified candidate

- Branch: `fix/freeze-rules-v2` in the PR scratch checkout (also `fix/freeze-rules`, bbf506c6, the pre-polish copy); 4 commits (tooling 409903a6, Blood Raven and Countess 31a63579, special monsters chilled 42460cf6, Glacial Spike freeze 4dc07e7d), 11 files, +476 -9. Not on the fork's private remote under that name (V: not among private or public-fork refs): push it there first.
- Deviation (V ledger rows states 1,11,118 and formulas F12, TESTED-DEVIATION): game.exe FUN_0057b230 resists a freeze length by cold resistance less pierce and divides it by MonsterFreezeDivisor; d2d stood any monster for the raw length and chilled it too (C5); champions, uniques and bosses are chilled instead of frozen (C3); Blood Raven and the Countess are not halved, state 118 (C4).
- Evidence: author differential 225,748 real-d2d cases, 185,746 -> 0 (docs/research/pr-freeze-proof.md). Independent Opus verifier VERIFYFREEZE (pr-freeze-proof.md section "Independent verifier"; BACKLOG): own generator and seed, 146,591 cases, 124,662 mismatches on #155's head, 6 on the candidate (freeze length 0x100001 only, unreachable); C5 and C3 HOLD WITH LIMITS, C4 HOLDS; chill set read from game.exe's own spawn path, 12,830 monsters. Not covered: Freezing Arrow, Ice Blast, item freeze.
- Risk: every freeze and chill on a monster with cold resistance and every champion, unique, superunique and act boss in Normal, NM and Hell (a Frost Nova or Glacial Spike changes from freeze to chill on them): a wide gameplay change, deliberate and measured. Players unchanged. det state hashes not run (needs a d2d slot).
- CI parity: behind upstream main by 190 commits; `git merge-tree` against 3852c904 is clean. It CONFLICTS with open #159 (add/add on tools/emu/freeze_rules.py, V merge-tree), so rebase after #159 merges or on #159's head; the commit bodies were already cut to the standard.
- Draft: docs/research/pr-freeze-draft.md (status banner says open after #155 merges; #155 is merged, banner stale).
- Recommendation: READY after rebase, a one-check re-verification on the rebased head by the Opus verifier (the earlier verifier checked bbf506c6; v2 is the polish), and the det hashes if a slot is free. Rank 1.

### C2. Vendor prices follow-on (fix/vendor-prices-followon-v2, da80d0ab)

- 3 commits (repair gate da80d0ab, npc multipliers divide first above 0xffff aed23fc8, oracle 777df9d7), 7 files, +219 -14. Not on the fork's remote (V).
- Deviation (ledger formulas F11): game.exe FUN_0062efb0 refuses repair without an ItemTypes Repair flag or without durability, and divides npc multipliers by 1024 first above 0xffff; d2d does neither.
- Evidence: author proof pr-price2-proof.md; verifier record verify-price2.md (card verifyprs2): D9 and D10 HOLD WITH LIMITS, own cases, no counter-example; on #152's 100,000-case set 9,232 / 154 / 1,608 divide-first cases and 5,163 repair-gate cases, all -> 0, 0 clean-case regressions. The verifier was not Opus (model not stated), so under the new rule it needs one Opus pass.
- Risk: low and narrow (repair of non-repairable items; large multiplier edge); the verifier's one presubmit FAIL (B905 `zip` without `strict=`) is fixed in the v2 head.
- CI parity: #152 is merged (the verifier's condition); base is 189 commits behind main; the draft says the rebase is clean (merge-tree) and the "unidentified item cannot be repaired" claim is already upstream (a3cc0f7e), so no overlap.
- Draft: docs/research/pr-price2-draft.md (title, body, commands ready).
- Recommendation: needs one Opus verify (narrow), then ready to open. Rank 2 (smallest, cheapest slot).

### C3. SKMELEE3: the player's plain melee blow (no branch yet)

- Deviations proven on the REAL d2d build (card skmelee2, ledger skills.monster 0 / formulas F03, TESTED-DEVIATION, 100,000 cases, seeds 7 and 1): D1 the physical part is cut to whole points before resistance, game.exe keeps 256ths (non-whole in 99.5 % of blows); D2 +50 percent damage against undead with a Blunt weapon, plus stat 122 (60 of 60 grid cases); D4 barehanded Strength as a damage percent; D5 attack rating floor; D6 a double floor at -90 (D4 to D6 narrow). D7, D8 (off-weapon enhanced damage) stay U or I.
- Evidence: docs/research/sk-melee2-trace.md and tools/emu/sk_melee2.py with sk_melee2_test.cpp (harness exists); independent verifier docs/research/verify-skmelee2.md (own harness, written first): D1 HOLDS, D2 HOLDS WITH LIMITS, D4 HOLDS, D5 and D6 with limits. The verifier model was Sonnet, so an Opus pass on the built branch is still owed.
- Not built: BACKLOG "SKMELEE3 (ready card)": D1 touches every `player_blow` caller and the physical resistance on 256ths (the #166 missile physical change is the model: 1ea79c57); D2 needs monster undead flags and the weapon's Blunt type plumbed into Target and Fighter.
- Risk: D1 changes every player melee blow (a Barbarian or Paladin changes by under one point on average, with kills at the 1.0 boundary); D2 changes undead fights for Blunt weapons only (wide for a mace or club user, nil otherwise). The largest player-facing change of the list.
- CI parity: no branch; can start from current main with the #166 pattern.
- Draft: none (BACKLOG card only).
- Recommendation: build D1 + D2 as one bundle (same function, same harness), then Opus verify. Rank 3. Needs a worker card (Go-free; C++ in the fork).

### C4. Ground and burst hit pipeline (no branch yet)

- Proven on a real d2d build (ledger missiles 158-162, skills.sor 47,55, sk-ground2-trace.md, sor-rest-trace.md): Blizzard standing-missile hit area D4 (size 1: 5 subtiles in game.exe, 21 in d2d; shard 5 / 13 / 21 subtile sizes), Fire Wall maker one tick short and an extra firewall on the target, and the Fire Ball and Glacial Spike burst roll D1 (game.exe rolls once per burst, d2d once per monster: 0 of 400 trials equal on d2d, 400 of 400 on game.exe). The same family as #163, which fixed only the shard offsets D1.
- Evidence: V run both sides for D4 and the burst roll (docs/research/sk-ground2-trace.md; docs/research/sor-rest-trace.md, harness sk_rest_shared.py); Fire Wall 80 of 80 cases; no independent verifier on D4 and the burst roll (pr-blizzard-verifier.md covered only D1).
- Risk: every Blizzard, Fire Ball, Glacial Spike and Fire Wall hit (a Sorceress's main plan skills): a gameplay change in both directions; the burst roll fix restructures `Fight::strike` (not a minimal commit).
- CI parity: must wait for #163 (same files in missiles and the ground tests; conflicts expected) and then be rebased.
- Draft: none.
- Recommendation: needs more evidence (an independent harness) and a build. Rank 5 (the plan skills make it worth a slot, but it is large).

### C5. Monster Act 1 AI follow-ups (no branch yet)

- Proven on the monsterai2 harness (ledger monster.ai rows; docs/research/monsterai2-trace.md, monster-ai-trace.md): Fallen flee pace 50 omitted (D1, 416 of 416 dying cases), Blood Raven pace (820 of 1,500), target-home distance (162) and boundary (3), Fallen Shaman corpse reach 24 vs 80 (1,365 of 12,000 candidates; d2d side by reading), and the earlier port-level D1 to D4 for Zombie, Brute, BloodHawk, CorruptRogue, CorruptLancer (649, 1,820, 1,686, 917, 1,656 of 3,000 each; the pace part went in #167).
- Evidence: V run on the card's own harness; no independent verifier; the card says "not the existing upstream differential" and no branch could be made then.
- Risk: Act 1 monsters only (Fallen packs, Blood Raven, Fallen Shaman): small count of monster kinds, visible behaviour change (flee speed, corpse resurrection range).
- CI parity: follows #167 (merged); no branch.
- Draft: none.
- Recommendation: needs a build and an Opus verify; a good fit for the bot's Act 1 focus. Rank 4.

### C6. Teleport D1 to D3 (no branch yet)

- Ledger skills.sor 54: d2d lands anywhere on the level; game.exe refuses outside the caster's room and its near list (D1: 25,120 of 100,000 random cases, none at the bot's 18 subtiles), blocked target uses the nearest free spot (D2, 1,323), a monster at the target is avoided (D3, 4,805). D4 (mana paid on a refused hop) is read only; D5 and D6 V read.
- Evidence: V run on the oracle against a LINE COPY of d2d's rule (docs/research/sk-teleport-trace.md section 6); the real-build side was not built ("oracle exists, real-build side not built"). An independent re-check is queued as VERIFY-TELEPORT-ROOM (open-work, status ready).
- Risk: every Teleport hop of the Sorceress; D1 removes fake long hops (a change to bot routing in d2d runs, none at 18 subtiles).
- Recommendation: needs more evidence (real-build harness, the open-work verify item). Rank 7.

### C7. Monster missile gates (no branch yet)

- Ledger formulas F14 (docs/research/monster-attack-trace.md verdict 2): (a) d2d lets a missile with no physical part be blocked (10,503 of 100,000 block cases; 653 more in town walk); (b) d2d rolls to-hit for every MissA shot, game.exe only when the row's ToHit is set (94 of 142 MissA rows). None in the early Act 1 areas.
- Evidence: V run on an oracle against the d2d rule; the after-side not on a real build. Fits the #164 family (monster missile record).
- Recommendation: needs more evidence (real-build run), low value for early areas. Rank 8.

### C8. Small sorceress and merc items (no branches)

- minmana floor D2 (docs/research/sk-mana-trace.md, verify-skmana.md: game side HOLDS on 20,000 own cases; d2d side 791 of 130,840, bites only below 1 mana point: Teleport 25+, high arrows, Might); pets and merc tie order D1 (1,254 of 100,000, rule level only, harness reordered, no effect for the Sorceress); monster life regeneration D2 (game.exe stat 0x4a, d2d none: V read, no harness against d2d); defender-side flat order and magic resist on a magic row (skdefenders-trace.md, monsterhit-trace.md, read only).
- Recommendation: each too small or too thin alone; bundle only if a slot is free after C1 to C5. Rank 6 as a bundle (minmana + pets tie), otherwise drop.

Not candidates (checked): spawn density and the champion roll (docs/research/upstream-pr-candidates-spawn.md: d2d equals game.exe), the kill-experience fix (upstream 01337ba3), treasure, merc stats, curse values (skcurseval), gear armor stats (geararmore), PALDUR/PALAURA/PALSKILL (read only, drills pending, no harness against d2d), PLAYERDEF D1 to D8 and UNIQMODS (reading only), NM / Hell monster regeneration (needs plumbing), the Act-1 only SKMISS roll top (merged as #166). Pierce: melee pierce is UNCOVERED (player_blow has no pierce input), Fire and Lightning Mastery are damage percent (REFUTED as a pierce claim, SKABRECHECK).

## 5. Ranking for the five slots

| Rank | Candidate | Independent verifier | State | Next action |
|---|---|---|---|---|
| 1 | C1 freeze rules (C5 resisted freeze, C3 special monsters chilled, C4 Blood Raven and Countess) | YES, Opus (VERIFYFREEZE, own harness) | built, 4 commits, clean on main, conflicts with #159 | push to the private remote, rebase after #159 merges, Opus re-check of the rebased head, then open |
| 2 | C2 vendor prices follow-on (repair gate, divide-first) | YES, not Opus (verify-price2, own harness) | built, 3 commits, rebase clean | one Opus pass on v2 (B905 fixed), push to the private remote, then open |
| 3 | C3 SKMELEE3: melee 256ths + undead Blunt bonus (bundle D1 + D2) | harness verified by a Sonnet verifier (verify-skmelee2), fix not built | harness ready, no branch | worker card to build both commits like #166, then Opus verify |
| 4 | C5 Act 1 monster AI (Fallen flee, Blood Raven, Fallen Shaman reach) | no | no branch | build on the monsterai2 harness, Opus verify |
| 5 | C4 ground and burst hit pipeline (Blizzard hit area D4, Fire Wall, burst roll) | no (only D1 was, in #163) | no branch, after #163 | independent harness first, then build |
| 6 | C8 minmana + pets tie order | partial (game side of minmana) | no branch | only if a slot is free |
| 7 | C6 Teleport D1 to D3 | no (VERIFY-TELEPORT-ROOM is open work) | oracle only | real-build harness |
| 8 | C7 monster missile gates | no | oracle only | real-build run; low value for early areas |

Slots 1 and 2 can be opened soonest, after one Opus verifier pass each (one at a time, per the standing rule). Slot 3 is the largest player-facing win but costs a build. Bundling: C3 as one PR (D1 and D2 together), C5 as one PR after #167's family, C1 alone. Never bundle C1 with #159 (the same file, different claims).

## 6. Evidence limits of this inventory

- V: refs, counts, merge-tree and subject matches (read-only git, 2026-11-02); S: all evidence statuses and numbers are quoted from the docs named in each entry and were not re-run here.
- U: whether `fix/freeze-rules-v2` and `fix/vendor-prices-followon-v2` still build on 3852c904 (merge-tree says the text merges; no build was run).
