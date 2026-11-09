# docs/research: topical index

One line per doc: what it shows, status, PR. Status: current = still the best source; historical = finished or executed, kept as the record; superseded = replaced (its banner says by what). Archived docs (stub left here) live in [../archive/](../archive/README.md). Upstream = psi29a/d2decomped. PR rule: a differential against game.exe's own code, see [../process/upstream-pr-standard.md](../process/upstream-pr-standard.md). Where we are overall: [../plan/STATUS.md](../plan/STATUS.md); docs map: [../README.md](../README.md).

## Fidelity and the risk map (where d2d may differ from game.exe)
- [d2d-fidelity-riskmap.md](d2d-fidelity-riskmap.md) ranks the places d2d can move our four gated metrics; script `tools/riskmap/`. current.
- [per-kill-time-split.md](per-kill-time-split.md) per-kill time, original vs d2d, and where it goes. current.
- [d2d-whisper-lan-oracle.md](d2d-whisper-lan-oracle.md) a packet-level oracle idea for town layout and spawns. current (card d2d-lan-oracle-demo).
- [mon-wake-trace.md](mon-wake-trace.md) monster wake-up radius and think cadence equal game.exe (oracle, 0 differ). current; result: clean negative, no PR.
- [upstream-gaps/](upstream-gaps/) item traces behind the first PR candidates (charged-bolt hit, act 1 monster HP, monster density) with the oracle scripts. historical.

## XP and kill time
- [xp-award-trace.md](xp-award-trace.md) XP per monster traced in game.exe: identical to d2d. current.
- [upstream-pr-candidates-xp.md](upstream-pr-candidates-xp.md) XP candidate list (none ready). historical.
- [spawn-density-trace.md](spawn-density-trace.md) spawn density and champion/unique rolls equal game.exe. current.
- [upstream-pr-candidates-spawn.md](upstream-pr-candidates-spawn.md) spawn candidate list. historical.
- [gap-casts-per-kill.md](gap-casts-per-kill.md) why the original needs fewer casts per kill (a probe artifact). current.
- [cast-trace-gameexe.md](cast-trace-gameexe.md) cast timing traced in game.exe; basis of PR #128 (merged). current.
- [skill-bind-trace.md](skill-bind-trace.md) skill bind (C->S 0x3c) and cast packets/gates, game.exe vs d2d vs the bot (oracle, decomp-queue item 4). current; result: server refuses non-row / missing / level-0 selects, no leftskill check; d2d World stores any id (D1, 81,756 of 127,032 -> 0 under the rule, PR not built); on d2d BindSkills binds nothing in the game (B1); P26 drill recipe inside.

## Get-hit and recovery
- [movement-and-recovery-trace.md](movement-and-recovery-trace.md) walk/run speed and hit recovery traced in game.exe. current.
- [monster-mods-trace.md](monster-mods-trace.md) champion / unique / minion modifiers, game.exe vs d2d vs the bot (oracles, decomp-queue item 13). current; result: d2d's boss inits match on the same base except ghostly speed, berserker level, CE length; d2d reads boss stats at the boosted level (D1) and gives minions nothing (D2); PR candidate not built.
- [walk-speed-trace.md](walk-speed-trace.md) walk/run speed and the movement step, game.exe vs d2d vs the bot (oracle, decomp-queue item 12). current; result: player speeds equal for the bot, d2d monster RN uses Run not Velocity (+ chill, floor, velocitypercent terms; 712,362 cases, 425,665 -> 0); walking time is bot-side (B1 run belief never reset). PR candidate: [pr-walk-evidence-pack.md](pr-walk-evidence-pack.md) ([proof](pr-walk-proof.md), [draft](pr-walk-draft.md)).
- [static-field-chill-trace.md](static-field-chill-trace.md) Static Field and the cold length / chill on the hit, game.exe vs d2d vs the bot (oracles, decomp-queue item 6). current; result: d2d Static Field S1-S6 (309,664 cases, 115,103 -> 0), cold C1/C2 (28,739 of 101,170, no patch); drill recipe. PR candidate: [pr-static-evidence-pack.md](pr-static-evidence-pack.md) ([proof](pr-static-proof.md), [draft](pr-static-draft.md)).
- [pets-merc-trace.md](pets-merc-trace.md) pets, the merc and monster targeting, game.exe vs d2d vs the bot (oracle vs d2d's C++, decomp-queue item 8). current; result: nearest wins, player first, pets newest first, 55-subtile cut; merc leash before the fight; d2d merc target clean (0 of 100,000), D1 pet tie order 1,254 of 100,000 -> 0 (no PR); bot B1-B5; drill recipe.
- [pr-get-hit-rule-proof.md](pr-get-hit-rule-proof.md) the get-hit decision matches game.exe. current. PR #134 (merged upstream 2026-10-31); evidence: [pr-get-hit-rule-evidence-pack.md](pr-get-hit-rule-evidence-pack.md). Body and review replies: archived ([draft](../archive/research/pr-get-hit-rule-draft.md), [review 1](../archive/research/pr-get-hit-rule-review-draft.md), [review 2](../archive/research/pr-get-hit-rule-review2-draft.md)).
- [pr-get-hit-length-proof.md](pr-get-hit-length-proof.md) the get-hit length (AnimData speed x (50 + FHR)) matches game.exe. current. PR #136 (merged by the maintainer 2026-10-31, upstream 4778a47c); evidence: [pr-get-hit-length-evidence-pack.md](pr-get-hit-length-evidence-pack.md). Body: [archived draft](../archive/research/pr-get-hit-length-draft.md).
- [pr-get-hit-render-proof.md](pr-get-hit-render-proof.md) the get-hit frame rule (step once, stand on the last frame) matches game.exe. candidate, rebased onto upstream main 4778a47c, ready to open on the user's go; evidence: [pr-get-hit-render-evidence-pack.md](pr-get-hit-render-evidence-pack.md), body: [pr-get-hit-render-draft.md](pr-get-hit-render-draft.md). Nothing pushed or posted.
- [upstream-pr-candidates-movement.md](upstream-pr-candidates-movement.md) stub: findings that became #134 and #136. archived.

## Missile damage
- [pr-missile-damage-fine-proof.md](pr-missile-damage-fine-proof.md) a missile's element damage keeps its 256ths. current. PR #135 (merged by the maintainer 2026-10-31, upstream 62282965); evidence: [pr-missile-damage-fine-evidence-pack.md](pr-missile-damage-fine-evidence-pack.md). Bodies: archived ([v2](../archive/research/pr-missile-damage-fine-draft-v2.md), [v1](../archive/research/pr-missile-damage-fine-draft.md)).
- [pr-missile-damage-fine-measured.md](pr-missile-damage-fine-measured.md) measured effect of the fix on casts per kill. current.
- [monster-attack-trace.md](monster-attack-trace.md) monster melee and missile attacks on the player, game.exe vs d2d (oracle). current; result: melee clean, two narrow missile gate deviations (none early), monster rules explain ~0 of the XP/h gap; bot findings B1-B4; first pass archived. Candidates: [upstream-pr-candidates-monster.md](upstream-pr-candidates-monster.md).

## Lut Gholein and level generation (DRGL)
- [lutgholein-trace.md](lutgholein-trace.md) why whisper `town` and d2d `drlg-dump` disagree. historical (the cause is in the proof).
- [lutgholein-proof.md](lutgholein-proof.md) the two missing level-seed steps, traced, fixed, proven. current.
- [pr-lut-gholein-draft.md](pr-lut-gholein-draft.md) and [pr-lut-gholein-evidence-pack.md](pr-lut-gholein-evidence-pack.md) the tool fix, NOT posted yet (waits for the user's go); rebased onto upstream main 4778a47c, 20/20 after. current.
- [skill-damage-trace.md](skill-damage-trace.md) skill damage on the unit path, game.exe vs d2d (synergy gate, MulDiv shares). current. Proof [pr-skill-damage-proof.md](pr-skill-damage-proof.md), body [pr-skill-damage-draft.md](pr-skill-damage-draft.md), evidence [pr-skill-damage-evidence-pack.md](pr-skill-damage-evidence-pack.md). PR candidate READY, pending the user's go; not posted. current.
- [sorc-skill-matrix.md](sorc-skill-matrix.md) every Sorceress skill, which ones the plan uses by act, what is verified (V/S/I/U matrix), ranked gap decomps (decomp-queue 14-23), the live skill-smoke drill and a per-act gate. current; proposals only (card skillinv).

## Upstream PRs: candidates, scan, posted record
- [pr-bundle-draft.md](pr-bundle-draft.md) and [pr-bundle-evidence-pack.md](pr-bundle-evidence-pack.md) the combined PR (renderer get-hit frames + Lut Gholein tool fix, two commits), prepared on upstream afa0cdae, NOT opened. current.
- [upstream-prs.md](upstream-prs.md) table of posted PRs (#127 and #128 merged). current.
- [upstream-pr-scan.md](upstream-pr-scan.md) how upstream works, open PRs and maintainers. historical (2026-10-30).
- [upstream-pr-candidates.md](upstream-pr-candidates.md) first candidate drafts (price, cast, missile). historical.
- [mpq-recheck-pr-claims.md](mpq-recheck-pr-claims.md) the price and cast data claims rechecked on the patched tables. current.
- [pr128-review-draft.md](pr128-review-draft.md) stub: PR #128 review replies, executed. archived.

## Alternatives and landscape
- [d2-landscape.md](d2-landscape.md) reimplementation/decompilation/hosting landscape. historical.
- [d2moo-fit.md](d2moo-fit.md) D2MOO as a backend: no switch. historical.
- [redivivus-sister-project.md](redivivus-sister-project.md) the Redivivus project. historical.
- [web-damage-calc-sources.md](web-damage-calc-sources.md) web sources for 1.14d damage maths. current.

## Decomp lane (game.exe traces; queue: [../decomp-queue.md](../decomp-queue.md))
- [belt-pickup-trace.md](belt-pickup-trace.md) belt and ground-potion pickup traced in game.exe vs d2d vs the bot (findings B1-B5, d2d deviation D1). current.
- [click-handlers-trace.md](click-handlers-trace.md) shift-click to the belt, ctrl-click sell, inventory-open pickup in game.exe vs d2d (D3 proven gap, PR candidate [pr-click-proof.md](pr-click-proof.md) / [draft](pr-click-draft.md) / [evidence](pr-click-evidence-pack.md); D2, D4 client-only). current.
- [pr-belt-proof.md](pr-belt-proof.md), [pr-belt-draft.md](pr-belt-draft.md), [pr-belt-evidence-pack.md](pr-belt-evidence-pack.md) upstream PR candidate for belt scrolls: ready, local branch only, pending the user's go. current.
- [price-trace.md](price-trace.md) vendor repair / sell / buy prices and gold caps traced in game.exe vs d2d vs the bot (d2d D1-D7, bot B1-B6); proof [pr-price-proof.md](pr-price-proof.md); follow-on D9 / D10 [pr-price2-proof.md](pr-price2-proof.md) ([draft](pr-price2-draft.md), [evidence](pr-price2-evidence-pack.md)). current.
- [town-portal-trace.md](town-portal-trace.md) Tome of Town Portal use traced in game.exe vs d2d vs the bot. current.
- [waypoint-trace.md](waypoint-trace.md) Waypoint panel and 0x49 travel traced in game.exe vs d2d vs the bot; PR candidate [pr-wp-proof.md](pr-wp-proof.md) ([draft](pr-wp-draft.md), [evidence](pr-wp-evidence-pack.md)). current.
- [pr-tome-proof.md](pr-tome-proof.md), [pr-tome-draft.md](pr-tome-draft.md), [pr-tome-evidence-pack.md](pr-tome-evidence-pack.md) upstream PR for inventory tome use: OPEN #149 (2026-10-31).

## Process, review and public prep
- [code-review-2026-10-30.md](code-review-2026-10-30.md) review of the day's merges; section 9 lists the bug-fix cards. current (open items: [../BACKLOG.md](../BACKLOG.md)).
- [proposed-glossary-terms.md](proposed-glossary-terms.md) terms proposed for the user's grill session (GLOSSARY.md is a guard file). open until the grill.
- [README-public-draft.md](README-public-draft.md) draft of the public README. draft; the user approves it for the repo root (see its banner).
- [public-readiness-audit.md](public-readiness-audit.md) pre-public audit (leak-check allow-listed at this path, so it stays). historical.

## Decompiled-source discipline
Every doc here tags claims V (run or read by the author), D (data table read), S (secondary), U (unverified). No claim without a tag may be relied on; an upstream PR needs zero U in its causal chain. Reading the real 1.14d tables: [../process/mpq-data-reading.md](../process/mpq-data-reading.md). The oracle scripts are `tools/emu/*.py`; raw captures stay on the worker host.

## Generality / architecture
- [bot-generality.md](bot-generality.md) framework vs stencil: what is fitted to class, act, clvl, difficulty and engine, a vocabulary for generality, target layers, ranked experiments and a card plan. current.
- sk-armor-trace.md: armors and Energy Shield (SKARMOR, decomp row 15); oracle tools/emu/sk_armor.py
- sk-lightning-trace.md (SKLIGHT, 2026-11-01): Chain Lightning hop rule and NextHit as unit state justhit; Lightning one unit a frame; no PR candidate
- sk-resist-trace.md: SKRESIST, decomp row 21 (monster resists, immunities, curses, pierce); oracle tools/emu/sk_resist.py, d2d test mode tools/emu/sk_resist_test.cpp.
- sk-melee-trace.md: the player's plain Attack (to-hit, damage build, Blunt vs undead, barehanded) in game.exe vs d2d; oracle tools/emu/sk_melee.py.
- sk-melee2-trace.md: the plain Attack on the real d2d build (per-part differences, D7 and the shield, off-weapon ED); oracle tools/emu/sk_melee2.py, d2d test mode tools/emu/sk_melee2_test.cpp.
- sk-nova-trace.md (SKNOVA, 2026-11-01): Frost Nova's 64-missile ring from the truncated tables at 0x6e1288 / 0x6e1388, reach 12-12.7 subtiles (V run); d2d hits a monster with every touching ring missile (D1, model ~6x); no PR candidate
- sk-passives-trace.md: SKPASS, masteries / pierce / Warmth / Telekinesis / synergies / +skills (V read, D, I replay)

## Watch UI and product design
- [ui-design.md](ui-design.md) dashboard IA, idle and ARPG game UIs, design systems, CSS architecture, accessibility, responsive and motion research behind [../ui-design-system.md](../ui-design-system.md); finds the gear strip CSP break. current (TASK-228).
