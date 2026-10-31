# Prebrief for grill 0001: is d2d the PRIMARY backend?

status: ready (not yet grilled). Written 2026-10-30 for the interviewee (the user) and for the grilling model. How the session runs: [../../process/grill-workflow.md](../../process/grill-workflow.md). Tags: V = read in the linked doc (that doc has its own V/S/U underneath), S = secondary, U = unverified. Classic 1.14d only; d2d's D2R-style extras (widescreen, gamepad, co-op) are irrelevant.

## 1. The decision in one sentence

Should d2d (the GPL native reimplementation, our private lockstep fork) become the PRIMARY backend of the bot (test bench now, and the product/container backend later), or stay a test bench next to the original Game.exe backend?

Existing record: [ADR 0002](../../adr/0002-d2d-primary-backend-gated.md) says "likely primary, gated on proof, original stays acceptance target" (V). This grill decides whether the gates are right, whether the "likely" becomes a "yes", and what stops being done. It supersedes or confirms ADR 0002.

## 2. Already DECIDED (do not re-ask; challenge only if the user reopens)

- The container is the product end goal; native on the worker host is the dev path ([ADR 0001](../../adr/0001-container-end-goal-native-dev.md), V).
- d2d probably becomes primary if it runs containerized better than Wine (user, 2026-10-30; [decisions.md](../../decisions.md), V).
- The original game stays the ACCEPTANCE target: d2d results are not acceptance results ([spec backend-contract](../../../openspec/specs/backend-contract/spec.md), V).
- Private fork on the forge (`<owner>/d2d-fork`), GPL-3.0, no d2d code in this repo, nothing distributed, nothing posted upstream without the user ([ADR 0003](../../adr/0003-d2d-fork-private-gpl.md), V).
- Decision code stays backend-agnostic; Wine-specific container work is paused ([ADR 0002](../../adr/0002-d2d-primary-backend-gated.md), V).

## 3. Options

| | Option | Meaning |
|---|---|---|
| A | d2d PRIMARY | d2d is the default backend for development, soak and the container image. Original game only for periodic acceptance checks, then retired from the product path. Wine container work stays dead. |
| B | d2d TEST BENCH only | d2d for fast offline tests; the product and container stay on Game.exe/Wine (heavier container, Wine work resumes). |
| C | HYBRID | d2d for iteration (speed, determinism), original game as acceptance, plus oracle projects ([D2MOO as rules oracle](../../research/d2moo-fit.md), [libd2 / d2-dedicated-server](../../research/d2-landscape.md)) for cross-checks. Product backend decided later by the fidelity gate. |

Recommendation of the prebrief author (U, judgement): C now, written as an explicit path to A with a named trigger (replicated soak inside the gate plus a built container), because the unproven list below is long and each item is cheap to test.

## 4. Evidence so far

| Claim | Number | Source | Tag |
|---|---|---|---|
| Builds and runs headless, no Wine, no X display | GCC 16.2.1, dummy SDL driver | [d2d-spike.md](../../d2d-spike.md) | V (on the worker host) |
| Speed with our lockstep fork | 120x real time idle in town, 90x to 370x outdoors (the brief's 120-377x) | [d2d-fork.md](../../d2d-fork.md) | V |
| Determinism | same seed plus same inputs gives byte-identical `monsters` output | [d2d-fork.md](../../d2d-fork.md) | V |
| Fork size | 324 added lines, 9 files vs upstream e36922e | [d2d-fork.md](../../d2d-fork.md) | V |
| Fidelity measured | 7885 kills of drop/XP statistics; mana/life/stamina regen exact, Minor Healing exact, items per kill 0.21-0.35 vs live 0.40 incl. piles | [d2d-fidelity.md](../../d2d-fidelity.md) | V-d2d, not against the original except where "V-live" |
| Gaps found | tome price 250 vs 450 live; Sorceress cast period 14 ticks for every spell vs 13 (19 for Lightning/Chain Lightning); potion double fill missing; wake radius about 7 cells (cannot test on original) | [d2d-fidelity.md](../../d2d-fidelity.md), [d2d-gaps.md](../../d2d-gaps.md) | V-d2d |
| Gaps fixed on our fork | cast period (Fire Bolt 13.0, Lightning 19.1 ticks) and tome 450 on branch `lab2`; formula confirmed by two kinds of source | [d2d-lab.md](../../d2d-lab.md), [cast-timing-sources.md](../../sme/cast-timing-sources.md) | V (our fork), original-game numbers still U for price |
| Navigation | the bot's real navigator walked town -> Blood Moor -> Cold Plains -> town | [d2d-walk.md](../../d2d-walk.md) | V |
| Combat | the bot's combat layer: 51 kills, 1 boss, 0 deaths, 0 potions (fresh character, no belt) | [d2d-combat.md](../../d2d-combat.md) | V (one short run) |
| Town | Akara menu, trade, buy 3 potions and 3 tomes, sell a scroll, gold matches the dump; Charsi/Gheed/Warriv/Kashya timed out (walk limit) | [d2d-town.md](../../d2d-town.md) | V (Akara), U (others) |
| Bot on d2d | `d2bot --backend d2d` boots, dashboard/API up, goal manager ran a farm goal | [d2d-bot.md](../../d2d-bot.md) | V offline tests, partly live |
| Free Rogue, death rule, quest persistence | answered on d2d from its source and probes | [d2d-probes.md](../../d2d-probes.md) | V-d2d |
| Alternatives | D2MOO: not a standalone engine, 1.10f based, MIT, no as backend, yes as oracle. No project beats d2d as fast simulator; `jaenster/d2-dedicated-server` runs the original headless (oracle/possible product, real-time only) | [d2moo-fit.md](../../research/d2moo-fit.md), [d2-landscape.md](../../research/d2-landscape.md) | V (reads, nothing run) |
| Container drafts | Dockerfile, compose, NAS template drafted, nothing built; effort table 2-5 h build proof, 12-20 h bot entry | [d2d-container/](../../d2d-container/README.md), [risks.md](../../d2d-container/risks.md) | U (estimates) |
| Upstream state | 494 `ponytail:` shortcuts in source, multiplayer partial, Act 5 uber groups not exact (their README) | [d2d-spike.md](../../d2d-spike.md) section 11 | S |

## 5. NOT yet proven (the grill must treat each as a gate or an accepted risk)

1. The bot loop end to end on d2d (goals, planner, town trips, recovery) for hours, not pieces.
2. The replicated soak within the 20 % gate: same character, N seeds, compared with original-game soaks ([spec fidelity-gate](../../../openspec/specs/fidelity-gate/spec.md), [d2d-soak.md](../../d2d-soak.md); harness built, the real comparison not run) (U).
3. Merc on d2d: free Rogue probed on d2d only; hire/revive/gear and the merc reader on the original not done.
4. Act 2 and later: not run on either backend by the bot; d2d upstream says Act 5 is partial (S).
5. Pricing and other original-game numbers: tome 450 is our fix, original rule unexplained ([d2d-gaps.md](../../d2d-gaps.md)); wake radius, hit recovery, drops/RNG against the original untested.
6. GPL review: nobody named; nothing distributed ([gpl.md](../../d2d-container/gpl.md), not legal advice).
7. Building in a container: upstream is C++26; g++ 15 on ubuntu:26.04 never tried here (the worker host built with 16.2.1) ([design.md](../../d2d-container/design.md)).
8. Maintenance: our patches rebased once without conflict; the rate of upstream change is unknown (U).

## 6. What each answer unlocks

- A: restart container work on a d2d image now (step 1-2 of [risks.md](../../d2d-container/risks.md)), make the replicated soak the next milestone, `--backend d2d` becomes the default in dev, stop adding Game.exe-only features, plan the licence route before any image leaves the NAS.
- B: resume the Wine container (prefix, winedevice reaping, sandbox), keep d2d tools as tests, no licence plan needed while private, but the product stays heavy.
- C: the replicated soak is the gate that turns C into A; container build proof is a cheap parallel spike; keep both backends green behind env.Env; name which probes always get confirmed on the original.
- What we STOP doing in A or C-after-trigger: Wine-only hardening and Game.exe memory-offset work for new features; in B: container-on-d2d work and the d2d fork patches beyond fidelity fixes.

## 7. Questions the grill must force

1. What would make you abandon d2d? Name a measurable trigger (a gated metric over 20 % after N fix cycles? an unfixable engine gap? maintenance cost per upstream release?).
2. Which fidelity gap is unacceptable for the bot's purpose (XP/h, deaths, drops, prices, timing)? Which can we live with if the bot only needs relative behaviour?
3. Is the product the idle container that plays the REAL game's rules (so d2d is a substitute engine users never see) or a bot for the real Game.exe (d2d is only our lab)? That changes whether fidelity must be exact or good enough.
4. Who reviews the GPL side and by when (before any container leaves the NAS or the repo goes public)? What is the plan if the answer is "nobody": stay private forever?
5. Upstream relationship: PRs for the cast and price fixes (drafted, nothing posted, [d2dcast](../../../openspec/changes/d2dcast/proposal.md) and d2dprice cards)? Accept permanent fork divergence or rebase on every upstream release?
6. How much original-game confirmation is enough: every probe, a sample, or only the soak gate?
7. Time and effort: is the user willing to pay for both backends for months (ADR 0002 consequence), and what is the date when one is dropped?

## 8. Risks

- Fidelity illusion: green d2d soaks that do not transfer to the real game (mitigation: the 20 % gate, acceptance on the original).
- Upstream drift or abandonment: our fork carries the patches alone.
- GPL exposure at distribution time if nobody reviewed (stays private until decided).
- Toolchain: C++26 build in a container untried.
- Sunk cost: the Wine container and Game.exe memory reader investment is paused, not dead.
- Both backends in step costs time on every feature (contract tests mitigate).
- The user's own expectation ("it will run containerized better") is unmeasured: no container build has run.

## 9. What the grilling model must NOT do

- No code, no tool runs against the game, no GitHub, nothing posted anywhere.
- No scope creep: GPL, merc, Act 2 and the container design are separate grills ([../README.md](../README.md)); park them.
- Ask the whole current frontier per round in the skill's format with a recommended answer each, but only about THIS decision; do not ask facts that the links above answer.
- Challenge vague answers ("probably", "later", "it depends"): ask for the number, date or trigger.
- Do not present a U row as a fact or a V-d2d row as an acceptance result.
- Do not edit AGENTS.md, GLOSSARY.md or docs/agents/**; write only the ADR draft and the decisions.md line.
