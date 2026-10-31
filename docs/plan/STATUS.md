# Where we are (hand-written part last edited 2026-10-30 evening by REVIEWDOC; the generated block at the bottom is current)

One page for the user. Decisions: [../decisions.md](../decisions.md) and [../adr/](../adr/README.md); plan: [plan-v2.md](plan-v2.md); live history: [../g1-log.md](../g1-log.md); work queue: [../BACKLOG.md](../BACKLOG.md); doc map: [../INDEX.md](../INDEX.md). Counts that change (merges, attempts) are not copied here: `git log`, g1-log and the generated block own them. Tags: V = verified (how, in the linked doc), V-d2d = measured on the reimplementation, U = unverified.

## Headline
- **d2d is the primary backend** ([ADR 0009](../adr/0009-d2d-primary-backend.md), grill 0001): v1 product is the d2d container, Game.exe is frozen, stays the acceptance reference and retires at the d2d container's 24 h soak; leaving d2d is break glass only. Only the four gated metrics of [fidelity-gate](../../openspec/specs/fidelity-gate/spec.md) must match the original within 20 %.
- **Game.exe soaks are done and judged** (g1-log): G1 attempt 18 PASS (2 h, 0 deaths, 9.1k XP/h); G2 8 h on Soaky (clvl 14 -> 15, 4 deaths, 7.1k kills, 4.1k XP/h, kill injection PASS with the user's one-off permission, criteria 6/7 closed); Fresha 4 h fresh Sorceress (clvl 6 -> 11, 3 deaths, 13.4k XP/h). Both found the same bug class (healing potions in the belt, none drunk; portal cast starvation; vendor dialog ignoring clicks): fixed offline by potionbug, portalfail, npctrade, manabackoff, rejuvuse, equipconfirm (live effect U, Game.exe frozen).
- **The whole bot runs on the lockstep fork** (`d2d.lab5`, 22-33x real time, V-d2d): navigation, combat, loot, town (Akara, stash, repair, portal), merc behind a knob. Replicated soak 4 seeds x 2 virtual h: XP/h 3013 then 3869 after ECONOMY (baseline 6600: -54 % then -41 %), kills/h 132 -> 167 (baseline 500), deaths 0 (the bot chickens out instead). **The fidelity gate is NOT met**; verdict INCONCLUSIVE below the sample minimums ([../d2d-loop.md](../d2d-loop.md), [../d2d-economy.md](../d2d-economy.md)).
- **The gap is mostly bot-side, not d2d** (research, V): seconds per kill 23.6 on d2d vs 9.1 original, of which search/stall 5.5 vs 0.3 and town/recovery 5.2 vs 3.0 ([research/per-kill-time-split.md](../research/per-kill-time-split.md)); casts per kill was a probe artifact; XP per monster identical clvl 1-7 ([research/xp-award-trace.md](../research/xp-award-trace.md)); monster placement equals game.exe per room ([research/spawn-density-trace.md](../research/spawn-density-trace.md)). One real d2d gap: spell damage truncated to whole points (-17 % at skill 1), PR branch built, not pushed (prmeasure measures first).

## Merged today (by area; details in the archived cards, `openspec/changes/archive/2026-10-30-*`)
- d2d stack: adapter, control client, pacer/virtual clock, world source, combat, town, loot, exits, roam, economy, speed (Prefetch, immediate verbs), merc wire, tap viewer (`cmd/d2dview`, `/stream`, sanitized `/api/pub`), replicated-soak tool (`cmd/d2dsoak`, gate v2 with exact Poisson bounds), per-kill time split, 4-slot lab pool (`tools/d2dslot.sh`), floating the worker host capacity rule (`tools/capacity.sh`), a Windows worker as a WSL2 d2d worker.
- Game.exe bot (frozen, acceptance): route/tower guards, death autopsy, recovery XP, early-game tables, inventory/stash, waypoints, nav stuck memory, potion and portal fixes, NPC visit/hover/skip/trade, two soaks at once, minute-line proxy fields (`dmgTaken`, `hpMinPct`).
- Upstream (psi29a/d2decomped): PR #127 prices posted, PR #128 cast fixes and replies posted (both user-approved); cast trace, XP trace, spawn trace, upstream gap items and three PR drafts (nothing else posted); MPQ reading method; Redivivus, D2MOO, landscape, whisper/LAN-oracle research.
- Process: change cards + 7 specs + `changecheck`, ADRs 0001-0009, grill workflow and agent, needs-you panel, board `/changes`, forge runners on the NAS/the worker host/a Windows worker/Mac worker A (Mac worker B offline), CI on master/PR, pre-push hook, public page with live soak status and a private runs cache, plugin pin.

## Open now
- Cards: d2d-container-build, d2d-container-bot, d2d-container-soak (the retirement soak), d2d-default-backend, d2d-replicated-soak, d2d-lan-oracle-demo, stream-grid; deferred: gameexe-retire-soak, orig-decision-probes. Running agents: killfix, forkrebase, prmeasure, decompmon, decomprec, containerbuild, stridegrid, lanoracle, licensedecide.
- Unmerged, reviewed in [research/code-review-2026-10-30.md](../research/code-review-2026-10-30.md): agent/killgap (per-kill time buckets), agent/detloop (determinism: the guard runs on the stepping goroutine; the full bot loop was not deterministic before it), agent/replay (record/replay of d2d sessions; two High findings to fix before or with the merge).
- Review findings to card (same doc, section 9): d2dsoak exit codes and empty/crashed seeds counted as clean exposure (H), replay tap cannot be stopped with a client attached and panics on crafted files (H), d2dview watch speed can stall the bot (M), public page: character names and a stale cache shown as live (M).
- Not started: Act 2+ on either backend, Andariel, the free Rogue claim on Game.exe, the quit-to-town shortcut, the idle-game layer (parked by the user).

## Decisions awaiting you
1. Licence and GPL review before any d2d container ships (ADR 0003; agent licensedecide is drafting the options; the fork stays in private the forge `<owner>/d2d-fork`).
2. Which character names may appear on the public page (today the sanitizer blocks machines and people, not the agents' characters; Soaky/Fresha/Loopy are shown on purpose).
3. Grill 0003 (merc and Act 2) when the coordinator asks; the SME contradicts the plan on ROUTEGUARD clvl 20 (suggests 14) and the Andariel floor (needs-you `grill-0003`).
4. Upstream follow-ups: push the missile-damage PR after prmeasure's numbers; the cast follow-up PRs wait for the maintainer's response to #127/#128 (needs-you `pr-cast-followups`).
5. Stream and ports: the Kick stream stays off; the d2dview `/stream` URL, its deploy and the container viewer port are yours ([../streamctl-d2d.md](../streamctl-d2d.md)).

## Top risks
1. The fidelity gate (-41 % XP/h after ECONOMY) is a bot-on-d2d problem (search/stall, poverty recovery, chicken restarts) and the d2d-replicated-soak card inherits it; a d2dsoak verdict can also exit 0 on INCONCLUSIVE/FAIL today (review H1).
2. Determinism of the full bot loop depends on the unmerged detloop change; replay and the determinism finder are built on it.
3. The container is unbuilt (build, bot, soak cards all proposed); licence review blocks shipping it; Windows/WSL2 (a Windows worker) runs d2d but not the container.
4. The public page is PUBLIC: the sanitizer is fail-closed on hosts, IPs, paths and tokens, but names and the stale-cache "live" label are the gaps above.
5. Game.exe is frozen with known live bugs fixed only offline (potions not drunk, trade dialog, `restarts` counter counts 1 of 17 recycles); it is still the acceptance reference for the 20 % gate.

<!-- statusgen:begin (generated by `go run ./cmd/changecheck status -write`; do not edit by hand) -->
### Active changes
| change | status | agent | owns | exclusive |
|---|---|---|---|---|
| [d2d-container-soak](../../openspec/changes/d2d-container-soak/) | proposed | d2d-container-soak | `docs/d2d-container-soak.md` | `soak-slot:1` |
| [d2d-default-backend](../../openspec/changes/d2d-default-backend/) | proposed | d2d-default-backend | `cmd/d2bot/d2dbackend.go` | - |
| [d2d-replicated-soak](../../openspec/changes/d2d-replicated-soak/) | proposed | d2d-replicated-soak | `docs/d2d-replicated-soak.md` | `soak-slot:2` |
| [m1-loot](../../openspec/changes/m1-loot/) | proposed | m1-loot | `internal/loot/pickit_starter.json`<br>`internal/loot/charms_test.go`<br>`docs/loot-charms.md` | - |
| [m2-abdesign](../../openspec/changes/m2-abdesign/) | proposed | m2-abdesign | `tools/ab/**`<br>`docs/ab-design.md` | - |
| [m2-earlybase](../../openspec/changes/m2-earlybase/) | proposed | m2-earlybase | `tools/d2dsoak/baselines-early.json`<br>`docs/d2d-baselines-early.md` | - |
| [m2-soakexit](../../openspec/changes/m2-soakexit/) | proposed | m2-soakexit | `cmd/d2dsoak/**`<br>`docs/d2d-soak.md` | - |

### Gates
- spec [backend-contract](../../openspec/specs/backend-contract/spec.md): 7 requirements
- spec [bot-loop](../../openspec/specs/bot-loop/spec.md): 3 requirements
- spec [combat-loop](../../openspec/specs/combat-loop/spec.md): 6 requirements
- spec [d2d-farm](../../openspec/specs/d2d-farm/spec.md): 8 requirements
- spec [dashboard-pub](../../openspec/specs/dashboard-pub/spec.md): 3 requirements
- spec [death-rule](../../openspec/specs/death-rule/spec.md): 8 requirements
- spec [early-game-policy](../../openspec/specs/early-game-policy/spec.md): 10 requirements
- spec [fidelity-gate](../../openspec/specs/fidelity-gate/spec.md): 7 requirements
- spec [lab-rules](../../openspec/specs/lab-rules/spec.md): 9 requirements
- spec [licence-policy](../../openspec/specs/licence-policy/spec.md): 7 requirements
- spec [merc-policy](../../openspec/specs/merc-policy/spec.md): 7 requirements
- spec [process-hygiene](../../openspec/specs/process-hygiene/spec.md): 9 requirements
- spec [public-split](../../openspec/specs/public-split/spec.md): 10 requirements
- spec [replay](../../openspec/specs/replay/spec.md): 7 requirements
- spec [soak-gates](../../openspec/specs/soak-gates/spec.md): 10 requirements
- spec [upstream-pr](../../openspec/specs/upstream-pr/spec.md): 14 requirements
- change d2d-container-soak: 24 h soak report.txt says PASS with exit 0
- change d2d-container-soak: memory curve and periodic d2d restart interval recorded
- change d2d-default-backend: go test ./cmd/d2bot -run Backend passes
- change d2d-default-backend: bot started with no flag uses d2d
- change d2d-replicated-soak: d2dsoak verdict printed for all four gated metrics, 5 or more seeds
- change d2d-replicated-soak: gaps listed as explained or as fidelity bugs
- change m1-loot: go test -p 1 ./internal/loot passes including TestCharmsRunesGemsOnSellVisit (every rune, every identified charm with a kept stat and every gem the rule keeps survive; the plain charm is sold only after identification)
- change m1-loot: go test -p 1 ./internal/town -run TestAuditS6 passes after the audit row 6 test is updated to the D4 rule (chipped and flawed gem sold, flawless gem, charm and rune held)
- change m1-loot: go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes
- change m2-abdesign: python3 -m unittest discover tools/ab passes (paired statistic on a fixture of 8 seeds x 2 arms; the required-runs formula)
- change m2-abdesign: docs/ab-design.md re-analyses the xpgap A/B (docs/probes/xpgap) with the paired statistic and states the runs per arm for a 10 percent effect
- change m2-abdesign: go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes
- change m2-earlybase: tools/d2dsoak/baselines-early.json has at least 2 runs, each with xp_per_hour, kills_per_hour, deaths_per_hour, deaths_per_1000_kills and a source line; rules gate kills_per_hour (rel 0.2) and deaths_per_1000_kills (rel 0.2, abs 1.0)
- change m2-earlybase: go test ./cmd/d2dsoak passes (the baseline file loads) and `d2dsoak compare -base tools/d2dsoak/baselines-early.json` on the soakfull runs in docs/probes/soakfull prints a verdict for all four metrics
- change m2-earlybase: go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes
- change m2-soakexit: go test ./cmd/d2dsoak passes including TestVerdictExitCodes (PASS 0, INCONCLUSIVE 2, FAIL 3, NO-DATA 3) for run and compare
- change m2-soakexit: go test ./cmd/d2dsoak passes including TestExcludedSeedsAreNotExposure (a seed with 0 minute lines adds 0 h and appears under excluded seeds)
- change m2-soakexit: go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes

### Recent merges
- 51cad66 10-31 01:34 `Merge agent/roadmap: milestone roadmap, 15 cards, decisions D1-D6` (roadmap)
- 895b17c 10-31 01:33 `Merge origin/main into agent/roadmap` (roadmap)
- 478499f 10-31 01:24 `Merge agent/basicsfix: town-basics fixes` (basicsfix)
- 0226219 10-31 01:15 `basicsfix: merge origin/main, delta spec early-game-policy, skip_specs false` (-)
- f62a9f5 10-31 01:15 `Merge agent/xpgap: XP gap A/B knobs and probes` (xpgap)
- 30ddfab 10-31 01:05 `Merge agent/specfix` (spec-backfill-combat)
- 46f0fda 10-31 01:00 `Merge agent/basicsaudit` (basicsaudit)
- 5b6f98f 10-31 00:56 `Merge agent/champ: champions as elite, not boss` (champ)
- c5b4ecc 10-31 00:44 `Merge agent/botsmall` (botsmall)
- 1481f63 10-31 00:40 `Merge agent/doctrim: doc trim` (doctrim)
<!-- statusgen:end -->
