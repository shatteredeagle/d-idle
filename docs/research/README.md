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

## Get-hit and recovery
- [movement-and-recovery-trace.md](movement-and-recovery-trace.md) walk/run speed and hit recovery traced in game.exe. current.
- [pr-get-hit-rule-proof.md](pr-get-hit-rule-proof.md) the get-hit decision matches game.exe. current. PR #134 (open); evidence: [pr-get-hit-rule-evidence-pack.md](pr-get-hit-rule-evidence-pack.md). Body and review replies: archived ([draft](../archive/research/pr-get-hit-rule-draft.md), [review 1](../archive/research/pr-get-hit-rule-review-draft.md), [review 2](../archive/research/pr-get-hit-rule-review2-draft.md)).
- [pr-get-hit-length-proof.md](pr-get-hit-length-proof.md) the get-hit length (AnimData speed x (50 + FHR)) matches game.exe. current. PR #136 (merged by the maintainer 2026-10-31, upstream 4778a47c); evidence: [pr-get-hit-length-evidence-pack.md](pr-get-hit-length-evidence-pack.md). Body: [archived draft](../archive/research/pr-get-hit-length-draft.md).
- [pr-get-hit-render-proof.md](pr-get-hit-render-proof.md) the get-hit frame rule (step once, stand on the last frame) matches game.exe. candidate, rebased onto upstream main 4778a47c, ready to open on the user's go; evidence: [pr-get-hit-render-evidence-pack.md](pr-get-hit-render-evidence-pack.md), body: [pr-get-hit-render-draft.md](pr-get-hit-render-draft.md). Nothing pushed or posted.
- [upstream-pr-candidates-movement.md](upstream-pr-candidates-movement.md) stub: findings that became #134 and #136. archived.

## Missile damage
- [pr-missile-damage-fine-proof.md](pr-missile-damage-fine-proof.md) a missile's element damage keeps its 256ths. current. PR #135 (merged by the maintainer 2026-10-31, upstream 62282965); evidence: [pr-missile-damage-fine-evidence-pack.md](pr-missile-damage-fine-evidence-pack.md). Bodies: archived ([v2](../archive/research/pr-missile-damage-fine-draft-v2.md), [v1](../archive/research/pr-missile-damage-fine-draft.md)).
- [pr-missile-damage-fine-measured.md](pr-missile-damage-fine-measured.md) measured effect of the fix on casts per kill. current.
- [monster-attack-trace.md](monster-attack-trace.md) monster damage to the player, game.exe vs d2d. current. Candidates: [upstream-pr-candidates-monster.md](upstream-pr-candidates-monster.md).

## Lut Gholein and level generation (DRGL)
- [lutgholein-trace.md](lutgholein-trace.md) why whisper `town` and d2d `drlg-dump` disagree. historical (the cause is in the proof).
- [lutgholein-proof.md](lutgholein-proof.md) the two missing level-seed steps, traced, fixed, proven. current.
- [pr-lut-gholein-draft.md](pr-lut-gholein-draft.md) and [pr-lut-gholein-evidence-pack.md](pr-lut-gholein-evidence-pack.md) the tool fix, NOT posted yet (waits for the user's go); rebased onto upstream main 4778a47c, 20/20 after. current.

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

## Process, review and public prep
- [code-review-2026-10-30.md](code-review-2026-10-30.md) review of the day's merges; section 9 lists the bug-fix cards. current (open items: [../BACKLOG.md](../BACKLOG.md)).
- [proposed-glossary-terms.md](proposed-glossary-terms.md) terms proposed for the user's grill session (GLOSSARY.md is a guard file). open until the grill.
- [README-public-draft.md](README-public-draft.md) draft of the public README. draft; the user approves it for the repo root (see its banner).
- [public-readiness-audit.md](public-readiness-audit.md) pre-public audit (leak-check allow-listed at this path, so it stays). historical.

## Decompiled-source discipline
Every doc here tags claims V (run or read by the author), D (data table read), S (secondary), U (unverified). No claim without a tag may be relied on; an upstream PR needs zero U in its causal chain. Reading the real 1.14d tables: [../process/mpq-data-reading.md](../process/mpq-data-reading.md). The oracle scripts are `tools/emu/*.py`; raw captures stay on the worker host.

## Generality / architecture
- [bot-generality.md](bot-generality.md) framework vs stencil: what is fitted to class, act, clvl, difficulty and engine, a vocabulary for generality, target layers, ranked experiments and a card plan. current.
