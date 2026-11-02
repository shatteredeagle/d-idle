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
| [actgate](../../openspec/changes/actgate/) | in-progress | actgate | `docs/research/sorc-act-gate.md`<br>`openspec/changes/actgate/**` | - |
| [armorhit](../../openspec/changes/armorhit/) | in-progress | armorhit | `openspec/changes/armorhit/**`<br>`docs/research/pr-armorhit-proof.md`<br>`docs/research/pr-armorhit-draft.md`<br>`docs/research/pr-bundle-queue.md`<br>`tools/emu/armor_hit.py`<br>`tools/emu/armor_hit_mode.inc` | `fork-branch:fix/armor-hit` |
| [buildv3](../../openspec/changes/buildv3/) | in-progress | buildv3 | `internal/plandata/build-fallback-lightning-v3.json`<br>`internal/plandata/build-fallback-lightning-v2.json`<br>`docs/plan/data/build-fallback-lightning-v3.json`<br>`docs/plan/data/build-fallback-lightning-v2.json`<br>`internal/plandata/plandata_test.go`<br>`internal/leveler/build.go`<br>`internal/leveler/buildv3_test.go`<br>`internal/combat/classdata/profiles.json`<br>`internal/combat/class_test.go`<br>`internal/combat/pure_test.go`<br>`cmd/d2bot/class.go`<br>`cmd/d2bot/class_test.go`<br>`docs/buildv3.md`<br>`openspec/changes/buildv3/**` | - |
| [d2d-container-soak](../../openspec/changes/d2d-container-soak/) | proposed | d2d-container-soak | `docs/d2d-container-soak.md` | `soak-slot:1` |
| [d2d-default-backend](../../openspec/changes/d2d-default-backend/) | proposed | d2d-default-backend | `cmd/d2bot/d2dbackend_default_test.go` | - |
| [d2d-replicated-soak](../../openspec/changes/d2d-replicated-soak/) | proposed | d2d-replicated-soak | `docs/d2d-replicated-soak.md` | `soak-slot:2` |
| [drills6](../../openspec/changes/drills6/) | in-progress | drills6 | `openspec/changes/drills6/**` | - |
| [fixbelt](../../openspec/changes/fixbelt/) | in-progress | fixbelt | `docs/fixbelt.md`<br>`internal/resource/recovery.go`<br>`internal/resource/recovery_test.go`<br>`internal/bot/broke.go`<br>`internal/bot/area_ban_test.go`<br>`openspec/changes/fixbelt/**` | - |
| [fixcastgap](../../openspec/changes/fixcastgap/) | in-progress | fixcastgap | `docs/fixcastgap.md`<br>`internal/combat/castgap.go`<br>`internal/combat/castgap_test.go` | - |
| [fixmanamodel](../../openspec/changes/fixmanamodel/) | in-progress | fixmanamodel | `docs/fixmanamodel.md`<br>`internal/resource/manamodel.go`<br>`internal/resource/manamodel_test.go` | - |
| [fixprogress](../../openspec/changes/fixprogress/) | in-progress | fixprogress | `docs/fixprogress.md`<br>`internal/town/chores.go`<br>`internal/town/town.go`<br>`internal/town/fixprogress_test.go`<br>`openspec/changes/fixprogress/**` | - |
| [fixskilllvl](../../openspec/changes/fixskilllvl/) | in-progress | fixskilllvl | `docs/fixskilllvl.md`<br>`internal/d2/skilllevel.go`<br>`internal/d2/skilllevel_test.go`<br>`internal/combat/skilllevel_test.go` | - |
| [forknext](../../openspec/changes/forknext/) | in-progress | forknext | `docs/forknext.md`<br>`openspec/changes/forknext/**` | `fork-branch:lab-nexthit`<br>`d2d-lab-binary` |
| [gate1run-20261101](../../openspec/changes/gate1run-20261101/) | in-progress | gaterun | `docs/soaks/gate1-20261101.md`<br>`docs/probes/soak-logs/gate1-20261101-*`<br>`openspec/changes/gate1run-20261101/**` | - |
| [gate1run-20261102](../../openspec/changes/gate1run-20261102/) | in-progress | gaterun | `docs/soaks/gate1-20261102.md`<br>`docs/probes/soak-logs/gate1-20261102-*`<br>`openspec/changes/gate1run-20261102/**` | - |
| [ledgerkey](../../openspec/changes/ledgerkey/) | in-progress | snaplib | `internal/cheat/ledger.go`<br>`internal/cheat/ledgerkey.go`<br>`internal/cheat/ledgerkey_test.go`<br>`cmd/d2cheat/main.go`<br>`openspec/changes/ledgerkey/**` | - |
| [m1-liveproof](../../openspec/changes/m1-liveproof/) | in-progress | drills | `docs/basics-audit.md`<br>`docs/basics-live.md`<br>`docs/primitives.md`<br>`docs/probes/basics/**`<br>`cmd/d2drill/**`<br>`internal/drill/**`<br>`docs/probes/drills/**` | `soak-slot:0` |
| [m3-merc](../../openspec/changes/m3-merc/) | proposed | m3-merc | `internal/merc/**`<br>`internal/bot/merc_test.go`<br>`docs/merc-d2d.md`<br>`docs/probes/merc/**` | `soak-slot:3` |
| [mapitem](../../openspec/changes/mapitem/) | in-progress | mapitem | `docs/stream/map-contiguity.md`<br>`cmd/d2dview/stitch.go`<br>`cmd/d2dview/stitch_test.go`<br>`cmd/d2dview/feed.go`<br>`cmd/d2dview/pub.go`<br>`cmd/d2dview/main.go`<br>`cmd/d2dview/view_test.go`<br>`cmd/d2dview/web/stream.html`<br>`cmd/d2dview/web/index.html` | - |
| [nexthit](../../openspec/changes/nexthit/) | in-progress | nexthit | `docs/research/pr-nexthit-proof.md`<br>`docs/research/pr-nexthit-draft.md`<br>`tools/emu/next_hit.py`<br>`tools/emu/next_hit_mode.inc`<br>`tools/emu/next_hit_summary.py`<br>`openspec/changes/nexthit/**` | `fork-branch:fix/nexthit` |
| [skmeleewire](../../openspec/changes/skmeleewire/) | in-progress | skmeleewire | `internal/combat/melee.go`<br>`internal/combat/melee_test.go`<br>`internal/combat/resource.go`<br>`internal/data/items.go`<br>`internal/data/schema.go`<br>`internal/data/data_test.go`<br>`internal/resource/sim.go`<br>`internal/resource/wire_test.go`<br>`docs/skmeleewire.md`<br>`openspec/changes/skmeleewire/**` | - |
| [skmiss](../../openspec/changes/skmiss/) | in-progress | skmiss | `docs/research/sk-missile-trace.md`<br>`tools/emu/sk_missile_hit.py`<br>`tools/emu/sk_missile_hit_test.cpp`<br>`docs/decomp-queue.md`<br>`docs/BACKLOG.md`<br>`docs/INDEX.md`<br>`openspec/changes/skmiss/**` | `fork-branch:fix/missile-roll-top` |
| [skresistfix](../../openspec/changes/skresistfix/) | in-progress | skresistfix | `openspec/changes/skresistfix/**`<br>`docs/research/pr-curseimmune-draft.md`<br>`docs/research/pr-curseimmune-proof.md` | `fork-branch:fix/curse-immune` |
| [soak-minutes-virtual](../../openspec/changes/soak-minutes-virtual/) | in-progress | specmin | `openspec/changes/soak-minutes-virtual/**`<br>`internal/d2dsoakstat/invariants.go`<br>`internal/d2dsoakstat/invariants_test.go`<br>`internal/d2dsoakstat/invreport.go`<br>`internal/d2dsoakstat/minute.go`<br>`internal/d2dsoakstat/testdata/invreport.golden`<br>`docs/probes/postfix-soak2.md`<br>`docs/review-strategy.md`<br>`docs/d2d-soak.md`<br>`docs/geninv.md` | - |
| [wpcooldown](../../openspec/changes/wpcooldown/) | in-progress | wpcooldown | `internal/nav/waypoint.go`<br>`internal/nav/wpclick_test.go`<br>`internal/env/tuning_nav.go`<br>`docs/wpcooldown.md`<br>`docs/waypoints.md`<br>`docs/fixwp.md`<br>`openspec/changes/wpcooldown/**` | - |

### Gates
- spec [backend-contract](../../openspec/specs/backend-contract/spec.md): 34 requirements
- spec [bot-loop](../../openspec/specs/bot-loop/spec.md): 6 requirements
- spec [combat-loop](../../openspec/specs/combat-loop/spec.md): 9 requirements
- spec [d2d-farm](../../openspec/specs/d2d-farm/spec.md): 8 requirements
- spec [dashboard-pub](../../openspec/specs/dashboard-pub/spec.md): 3 requirements
- spec [death-rule](../../openspec/specs/death-rule/spec.md): 8 requirements
- spec [early-game-policy](../../openspec/specs/early-game-policy/spec.md): 17 requirements
- spec [fidelity-gate](../../openspec/specs/fidelity-gate/spec.md): 12 requirements
- spec [lab-rules](../../openspec/specs/lab-rules/spec.md): 9 requirements
- spec [licence-policy](../../openspec/specs/licence-policy/spec.md): 7 requirements
- spec [merc-policy](../../openspec/specs/merc-policy/spec.md): 7 requirements
- spec [process-hygiene](../../openspec/specs/process-hygiene/spec.md): 9 requirements
- spec [public-split](../../openspec/specs/public-split/spec.md): 10 requirements
- spec [replay](../../openspec/specs/replay/spec.md): 7 requirements
- spec [soak-gates](../../openspec/specs/soak-gates/spec.md): 19 requirements
- spec [town-basics](../../openspec/specs/town-basics/spec.md): 17 requirements
- spec [upstream-pr](../../openspec/specs/upstream-pr/spec.md): 14 requirements
- change actgate: changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict
- change armorhit: changecheck 0 warnings for this card; leakcheck -mode fail 0
- change armorhit: PR claim only with a differential on real builds: before > 0, after 0
- change buildv3: gofmt, go vet, go test -p 1 per touched package, whole repo on the worker host; changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict
- change d2d-container-soak: 24 h soak report.txt says PASS with exit 0
- change d2d-container-soak: memory curve and periodic d2d restart interval recorded
- change d2d-default-backend: go test ./cmd/d2bot -run Backend passes
- change d2d-default-backend: bot started with no flag uses d2d
- change d2d-replicated-soak: d2dsoak verdict printed for all four gated metrics, 5 or more seeds
- change d2d-replicated-soak: gaps listed as explained or as fidelity bugs
- change drills6: changecheck -cli 0 warnings; leakcheck -mode fail 0; openspec validate --all --strict; whole-repo vet and test -short on the worker host
- change fixbelt: go test -p 1 internal/resource internal/bot; changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict
- change fixcastgap: go test -p 1 ./internal/combat ./internal/env ./tools/knobdump passes including TestFightLoopLightningNeverReissuedEarly
- change fixcastgap: go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes
- change fixmanamodel: go test -p 1 ./internal/resource ./internal/combat passes; whole-repo go vet and go test -short on the worker host pass
- change fixmanamodel: go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes
- change fixprogress: go test -p 1 ./internal/town ./internal/bot passes; changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict
- change fixskilllvl: go test -p 1 ./internal/d2 ./internal/combat passes; whole-repo go vet and go test -short on the worker host pass
- change fixskilllvl: go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes
- change forknext: changecheck status -write then -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict
- change gate1run-20261101: Gate 1 criteria of the soak-gates spec judged by d2dsoak (numbers in docs/soaks/gate1-20261101.md)
- change gate1run-20261102: Gate 1 criteria of the soak-gates spec judged by d2dsoak (numbers in docs/soaks/gate1-20261102.md)
- change ledgerkey: go test -p 1 ./internal/cheat/...; go vet ./cmd/d2cheat; changecheck -cli 0 warnings; leakcheck -mode fail 0
- change m1-liveproof: docs/primitives.md: every primitive with criticality S or A has a drill verdict PASS (or FAIL with a named fix card) in its evidence column, checked by the coordinator by eye
- change m1-liveproof: docs/basics-audit.md coverage table has 0 G rows and every S1 or S2 row is C with a test name in the offline-test column and a live proof line (`grep -c '| C ' docs/basics-audit.md` is at least 15)
- change m1-liveproof: starting gold and unidentified sell prices recorded V in docs/basics-live.md with cheat ledger ids
- change m1-liveproof: tools/agentreap.sh m1-liveproof leaves nothing; changecheck -cli 0 warnings; leakcheck 0
- change m3-merc: 2 seeds x 4 virtual hours from clvl 9 after Blood Raven on hidden d2d: the Rogue claimed exactly once per seed, 0 paid Act 1 hires, merc alive at least 70 percent of field time, every revive logged with price and gold delta within the revive rule (raw in docs/probes/merc)
- change m3-merc: go test -p 1 ./internal/merc ./internal/bot -run 'Merc|Claim' passes; go test ./internal/env -run EveryKnob passes with the d2d default
- change m3-merc: go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes; tools/agentreap.sh m3-merc leaves nothing
- change mapitem: go test -p 1 ./cmd/d2dview passes including TestStitchTransitionBloodMoorColdPlains, TestStitchFieldAndTown, TestStitchFieldToCaveIsNotStitched
- change mapitem: go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes
- change nexthit: changecheck 0 warnings for this card; leakcheck -mode fail 0
- change nexthit: PR claim only with a differential on real builds: before > 0, after 0
- change skmeleewire: go test -p 1 ./internal/data ./internal/resource ./internal/combat ./cmd/d2bot; whole-repo vet and test on the worker host
- change skmeleewire: changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict
- change skmiss: differential on real d2d builds: before > 0, after 0 for any claim; pr_guard 0/0; presubmit --quick 0 FAIL; upstream tests rc 0
- change skmiss: changecheck -cli 0 warnings; leakcheck -mode fail 0; openspec validate --all --strict
- change skresistfix: changecheck 0 warnings for this card; leakcheck -mode fail 0
- change skresistfix: PR claim only with a differential on real builds: before > 0, after 0
- change soak-minutes-virtual: go test -p 1 ./internal/d2dsoakstat ./cmd/d2dsoak passes, incl. TestInvariantsGate1PostfixSoak2 (27 and 26 fail) and the clean synthetic log (passes)
- change soak-minutes-virtual: tools/openspec.sh validate --all --strict passes; changecheck -cli 0 warnings; leakcheck -mode fail 0 hits
- change wpcooldown: go test -p 1 ./internal/nav ./internal/env; changecheck -cli 0 warnings; leakcheck -mode fail 0

### Recent merges
- 18c9c1ae 11-02 15:01 `Merge agent/sorcrest2: sorcrest2: finished docs and tools work that was never merged (branch triage 2026-11-02)` (sorcrest2)
- 8c55c53f 11-02 15:01 `merge origin/main into sorcrest2` (sorcrest2)
- 5ffb9027 11-02 14:58 `Merge agent/reviewprs: reviewprs: finished docs and tools work that was never merged (branch triage 2026-11-02)` (reviewprs)
- b3a2a929 11-02 14:57 `merge origin/main into reviewprs` (reviewprs)
- e50f1bd2 11-02 14:54 `Merge agent/prbuildskmelee3b: prbuildskmelee3b: finished docs and tools work that was never merged (branch triage 2026-11-02)` (prbuildskmelee3b)
- 894788d7 11-02 14:51 `Merge agent/verifysmite: verifysmite: records (docs and generated records only)` (verifysmite)
- d6cb2f71 11-02 14:51 `merge origin/main into prbuildskmelee3b` (prbuildskmelee3b)
- 9dc7281a 11-02 14:46 `Merge agent/playerroll: playerroll: finished docs and tools work that was never merged (branch triage 2026-11-02)` (playerroll)
- 053f5ebf 11-02 14:45 `merge origin/main into verifysmite` (verifysmite)
- 9e894c1e 11-02 14:45 `merge origin/main into playerroll` (playerroll)
<!-- statusgen:end -->
