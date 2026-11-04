# Retired-terms banner pass (2026-11-04)

Author: Claude (d2d-general agent, job retiredterms, branch agent/retiredterms)

Scope: living docs under `docs/` that state a retired or reversed concept without a `> status:` banner. Each hit gets ONE banner line under the title, pointing at the current truth in `docs/decisions.md` ("Current truth"), ADR 0013/0015, or the removal plan. Bodies are not rewritten. Not touched: `docs/adr/*`, `docs/decisions.md`, `GLOSSARY.md`, `docs/archive/`, `openspec/`, Go code, specs.

Current truth used (docs/decisions.md, verified 2026-11-04): d2d is the product and the development backend; game.exe 1.14d is the oracle; done = fresh uncheated character, 24 h unattended d2d Act 1 clear, writes after start 0, plus the G-D equivalence. G1 and G2 retired as release gates (ADR 0013), replaced by the d2d ladder Gate 0 to 3. OpenSpec retired for new work (user, 2026-11-04). Milestones M1 to M6 superseded by goals G-A to G-F (ADR 0015). The "Gate 1 XP below baseline is a FAIL" rule was superseded the same day (ADR 0015 item 8; struck through in decisions.md).

Cap: 40 files. Banners added: 20.

## Banners added (Y)

| file | term | banner added | why |
|---|---|---|---|
| docs/champions.md | "needs a G1/G2 check" (shared code) | Y | retired obligation, stated as current |
| docs/combat.md | "needs a G1/G2 check" | Y | same |
| docs/d2d-determinism.md | "needs a G1/G2 check" (loop.go, Pacer) | Y | same |
| docs/d2d-economy.md | "Needs a G1/G2 check" section | Y | same |
| docs/d2d-loop.md | "needs a G1/G2 check" (Game.exe behaviour) | Y | same |
| docs/d2d-soak.md | Game.exe G1/G2 soak notes | Y | Game.exe soaks retired |
| docs/d2d-container/design.md | "G1/G2-style verdicts" | Y | verdicts moved to d2d ladder; container deferred (ADR 0015) |
| docs/d2d-container/risks.md | "acceptable as G1/G2 evidence?" open question | Y | question retired |
| docs/d2d-probes.md | "the original game stays the acceptance target" | Y | acceptance rule amended (ADR 0015) |
| docs/d2d-spawns.md | "The original game is the acceptance target" | Y | same |
| docs/d2d-town.md | "Game.exe stays the acceptance target" | Y | same |
| docs/cards.md | OpenSpec cards as the tracker | Y | retired for new work (2026-11-04) |
| docs/process/ci-checks.md | OpenSpec cards / changecheck dates | Y | OpenSpec not required for new work |
| docs/process/hooks.md | changecheck hook for OpenSpec cards | Y | same |
| docs/process/grill-workflow.md | "create change cards with changecheck new" | Y | step retired for new work |
| docs/process/trial-notes.md | "KEEP the wrapper" (OpenSpec) | Y | historical trial, recommendation superseded |
| docs/process/openspec-reference.md | OpenSpec as the set-up | Y | background only |
| docs/process/openspec-fit-notes.md | OpenSpec fit | Y | superseded |
| docs/process/matt-pocock-skills-fit.md | tracker = openspec cards | Y | superseded |
| docs/process/spec-approach-evaluation.md | OpenSpec-shaped change cards adoption | Y | superseded |

## Not bannered (N), with reason

| file or group | term | banner | why |
|---|---|---|---|
| docs/BACKLOG.md, docs/INDEX.md, docs/CHANGELOG.md, docs/decomp-queue.md | G1/G2 retired line, milestones | N | shared files: merger only (AGENTS.md); BACKLOG line 109 already states the retirement |
| docs/decisions.md, GLOSSARY.md, docs/adr/* | all | N | guard rule |
| docs/archive/* | all | N | archived, never edited |
| docs/research/README.md | index | N | guard (shared file) |
| docs/noareacards2.md | OpenSpec | N | already says "retired for new work (user 2026-11-04)" |
| docs/cheats.md:40 "retired from gates" | G1/G2 | N | accurate as written |
| docs/merc-wire.md:50 "gate character" | gate | N | the Fresha gate-character rule still holds |
| docs/process/baseline-runs.md:108 | ADR 0013 | N | accurate as written |
| docs/process/outside-prs.md, public-split.md | changecheck mention | N | tolerance notes, still true |
| docs/process/openspec-removal-plan.md | OpenSpec removal | N | already bannered (the plan itself) |
| docs/process/spec-ledger-20261104.md, spec-apply-a/b-20261104.md, card-triage-20261104.md | G1/G2 stale-term hits | N | dated work logs of today's cleanup; they record the terms, not claim them |
| docs/milestones/milestones.json | M1..M6 | N | JSON, no banner possible; flagged for a future pass |
| docs/grills/0001-*/outcome.md, prebrief.md; docs/grills/0003-*/prebrief.md | G1/G2, Game.exe | N | dated grill records (ADR 0009 era) |
| docs/soaks/*.md, docs/probes/** (logs, `.jsonl.gz`) | G1 / G2 pass or fail | N | dated evidence of past runs |
| docs/research/** traces and pr-*-draft/proof files (about 60 files) | G1/G2, milestones, "game.exe" | N | dated evidence and PR drafts; the current truth is one click away in decisions.md. A bulk banner pass needs a decision; regenerate the list with `git grep -l -E "G1/G2|milestone|OpenSpec" -- docs/research` |
| docs/research/project-review-20261103.md, backend-parity.md, baseline-audit.md | many | N (already bannered) | banner present |
| docs/process/dispatcher-brief-proposed.md, guarded-proposals.md, dispatcher-brief.md | milestones M1..M6 | N (already bannered) | banner present |
| docs/roadmap.md, plan/plan-v2.md, plan/plan-20261103.md, review-strategy.md, soak24.md, liveruns.md, g1-log.md, d2d-baselines-early.md, d2d-xp-gap.md, killfix.md, bot-small.md, act3.md, act4.md, act5.md | G1/G2, milestones, acceptance | N (already bannered) | banner present |
| "Gate 1 XP below baseline is a FAIL" | XP rule | N | appears only in decisions.md (struck), research/specdelta-rejudge.md, spec-ledger (all dated); no living doc asserts it |
| docs/d2d-exit.md, forkcheats.md, forkdbg.md, soakstall.md, orig-spawns.md | "FAIL" in run logs | N | run outcomes, not the retired rule |
| docs/plan/STATUS.md | generated status block | N | generated; regenerate, do not hand-edit |

## Verification

- Banner count: `git grep -c "^> status:"` on each of the 20 files, one line each added directly under the title (plus a blank line).
- Hygiene (run after the commit): see the commit message and report; leakcheck, changecheck, merge_lines_check run in the worktree.

## Shared-file lines for the merger

None. This job edits no shared file.
