# Cards

> status: OpenSpec is retired for new work (user, 2026-11-04); cards are moving to Backlog.md (docs/decisions.md Current truth; docs/process/openspec-removal-plan.md).

> The 17 cards below (12 active, 5 parked) were copied into `backlog/` on 2026-11-04 (stage 1, see docs/backlog-howto.md); this file stays the interim list until stage 2.

Stage 1 of the OpenSpec removal plan (docs/process/openspec-removal-plan.md, branch agent/removalplan, not yet on main): this table is the list of active work. The user approved it on 2026-11-04 after the card triage. Each row links to its plain markdown card under `openspec/changes/<id>/` (tasks.md header holds owns, exclusive and gates).

Rules (ADR 0015 items 10 and 11):
- No cap on rows (the cap of 12 was removed by the grill 0005 outcome, ADR 0016). Every row carries a tier or area label (docs/backlog-howto.md, Labels); the goal column shows it.
- One row per active job. A job without a card is named by its branch `agent/<id>`.
- Parked cards stay open but are listed in "Parked". Nothing is deleted.
- "Needs a decision" cards stay open with git evidence only; they are not counted and not archived until the user decides.

Goals: G-A acceptance on game.exe, G-B Act 1 clear on d2d, G-C stability and basics, G-D measurement, G-E upstream fixes, G-F rules (see ADR 0015), G-G automation ladder (ADR 0016).

<!-- cardsgen:active:begin -->
## Active (65)

| # | id | goal | status | what | owner / exclusive | next step | card |
|---|---|---|---|---|---|---|---|
| 1 | m3-act1run | T4 | deferred 0/3 | fresh Sorceress clears Act 1 unattended on d2d | soak-slot:3 | - | [card](archive/openspec-index.md#m3-act1run) |
| 2 | m3-andariel | T4 | deferred 0/3 | Andariel fight card with the SME decisions (docs/andarielsme.md) | soak-slot:3 | - | [card](archive/openspec-index.md#m3-andariel) |
| 3 | m3-merc | T4 | deferred 0/4 | free Rogue run for hours on d2d, merc alive share | soak-slot:3 | - | [card](archive/openspec-index.md#m3-merc) |
| 4 | gameexe-retire-soak | ops | deferred 0/2 | game.exe validation paths; becomes the Gate A card | - | - | [card](archive/openspec-index.md#gameexe-retire-soak) |
| 5 | m1-liveproof | bot | deferred 13/15 | live drills for the basics; M1 exit | soak-slot:0 | - | [card](archive/openspec-index.md#m1-liveproof) |
| 6 | decision-point-store | T0 | in-progress 3/5 | learn-by-doing store: answers, streaks, rules, fading thoughts | - | tier order: after T4/Gate A cards per ADR 0016 | [card](archive/openspec-index.md#decision-point-store) |
| 7 | t0-slice | T0 | in-progress 3/5 | wire T0 decision points into combat with the untrained default | - | tier order: after T4/Gate A cards per ADR 0016 | [card](archive/openspec-index.md#t0-slice) |
| 8 | decisions-mirror | docs | deferred 0/4 | ADRs mirrored read-only into backlog/decisions for the board | - | - | [card](archive/openspec-index.md#decisions-mirror) |
| 9 | testerwatch | watch | in-progress 2/3 | separate watch token, tap video source switch | - | - | [card](archive/openspec-index.md#testerwatch) |
| 10 | testerlive | container | proposed 0/3 | run the tester flow in the real d2d container | - | - | [card](archive/openspec-index.md#testerlive) |
| 11 | testerdocs | docs | proposed 0/2 | d2d tester guide and first-stranger checklist | - | - | [card](archive/openspec-index.md#testerdocs) |
| 12 | testerreport | container | proposed 0/1 | scrubbed report bundle for the d2d tester | - | - | [card](archive/openspec-index.md#testerreport) |
| 13 | testerlicence | container | in-progress 1/2 | GPL package for the d.idle fork shipped with its source | - | - | [card](archive/openspec-index.md#testerlicence) |
| 14 | testerpack | container | proposed 0/1 | build the d2d tester pack | - | - | [card](archive/openspec-index.md#testerpack) |
| 15 | testerd2dimage | container | in-progress 1/2 | d2d tester image, realtime entry, compose file, ports | - | - | [card](archive/openspec-index.md#testerd2dimage) |
| 16 | testerd2dwiz | container | in-progress 1/2 | first-run wizard for the d2d tester container | - | - | [card](archive/openspec-index.md#testerd2dwiz) |
| 17 | testerrt | container | proposed 0/2 | realtime ladder check on d2d for the tester build | - | - | [card](archive/openspec-index.md#testerrt) |
| 18 | ordersux2 | T0 | proposed 0/5 | orders slice 2: until-level, clear then home, queue, conflict | - | - | [card](archive/openspec-index.md#ordersux2) |
| 19 | dpwire | T0 | proposed 0/3 | wire the remaining decision points into the bot | - | - | [card](archive/openspec-index.md#dpwire) |
| 20 | t0fight | T0 | proposed 0/3 | engineered drill for a fight-time question and a learned answer | - | - | [card](archive/openspec-index.md#t0fight) |
| 21 | ladder-t1 | T1 | proposed 0/3 | Tidy tier: cards and wiring for what T1 still lacks | - | - | [card](archive/openspec-index.md#ladder-t1) |
| 22 | ladder-t2 | T2 | proposed 0/3 | Runner tier: urgency stage 2 and the unwired T2 points | - | - | [card](archive/openspec-index.md#ladder-t2) |
| 23 | earlydeath13 | bot | in-progress 2/3 | seed 13 dies at clvl 4 to a Gargantuan Beast on Cold Plains | - | - | [card](archive/openspec-index.md#earlydeath13) |
| 24 | tpreserve-live | bot | proposed 0/2 | prove the scroll reserve and no-portal leave with the tp field | - | - | [card](archive/openspec-index.md#tpreserve-live) |
| 25 | playsession | T0 | proposed 0/3 | one command for a LAN play session | - | - | [card](archive/openspec-index.md#playsession) |
| 26 | upstream-next | d2d | proposed 0/2 | next upstream PR batch candidates (one open batch at a time) | - | - | [card](archive/openspec-index.md#upstream-next) |
| 27 | statusrefresh | docs | proposed 0/2 | docs/plan/STATUS.md hand-written part is stale (2026-11-03) | - | - | [card](archive/openspec-index.md#statusrefresh) |
| 28 | ordersux3 | T0 | proposed 0/4 | orders slice 3: reorder, dismiss, cleared, Acts 2-5, screen reader | - | - | [card](archive/openspec-index.md#ordersux3) |
| 29 | onboardtour | watch | proposed 0/3 | onboarding slice B: tour, Help, state banners | - | - | [card](archive/openspec-index.md#onboardtour) |
| 30 | onboardctl | watch | proposed 0/3 | onboarding slice C: Pause, Stop, Reset, report hook | - | - | [card](archive/openspec-index.md#onboardctl) |
| 31 | abmerc | ops | in-progress 3/4 | paired A/B of the merc revive reserve; A/B standard | - | - | [card](archive/openspec-index.md#abmerc) |
| 32 | fidprobe-core | bot | in-progress 0/3 | same-save probe tool and fresh/clvl 6 scenarios | - | - | [card](archive/openspec-index.md#fidprobe-core) |
| 33 | fidprobe-vendor | bot | proposed 0/3 | vendor stock and quote probe, phase 2 | - | - | [card](archive/openspec-index.md#fidprobe-vendor) |
| 34 | custom-overlay | T5 | in-progress 1/2 | row-merge table overlay in the d.idle fork and Go loader | - | - | [card](archive/openspec-index.md#custom-overlay) |
| 35 | custom-item-1 | T5 | in-progress 0/2 | stackable currency token from one early source | - | - | [card](archive/openspec-index.md#custom-item-1) |
| 36 | metacurrency | T5 | in-progress 0/3 | per-character and account-wide currencies from custom items | - | - | [card](archive/openspec-index.md#metacurrency) |
| 37 | currency-rows | T5 | in-progress 0/2 | token row, drop line and bank recipe overlay | - | - | [card](archive/openspec-index.md#currency-rows) |
| 38 | currency-slice | T5 | proposed 0/2 | bank step, balance line and kill drill | - | - | [card](archive/openspec-index.md#currency-slice) |
| 39 | fid-probe-readers | bot | proposed 0/4 | probe reader gaps: d2s third reference, merc fields, affix values | - | - | [card](archive/openspec-index.md#fid-probe-readers) |
| 40 | fid-probe-equip | d2d | proposed 0/4 | probe: equip legality and identify flow | - | - | [card](archive/openspec-index.md#fid-probe-equip) |
| 41 | fid-probe-regen-speed | d2d | proposed 0/4 | probe and measure: regen, potions, player walk and stamina | - | - | [card](archive/openspec-index.md#fid-probe-regen-speed) |
| 42 | fid-xp-award | d2d | proposed 0/2 | oracle plus live: XP per kill and level-difference penalty | - | - | [card](archive/openspec-index.md#fid-xp-award) |
| 43 | fid-price-pr | d2d | proposed 0/4 | oracle: buy, sell and repair price deviations, gold plans | - | - | [card](archive/openspec-index.md#fid-price-pr) |
| 44 | fid-stock-oracle | d2d | proposed 0/4 | oracle: vendor stock generation and refresh | - | - | [card](archive/openspec-index.md#fid-stock-oracle) |
| 45 | fid-belt-pr | d2d | proposed 0/3 | oracle: belt pickup lands in belt first | - | - | [card](archive/openspec-index.md#fid-belt-pr) |
| 46 | fid-inv-limits | d2d | proposed 0/2 | oracle: stash cap, purse cap, grid sizes | - | - | [card](archive/openspec-index.md#fid-inv-limits) |
| 47 | fid-layout-act1 | d2d | proposed 0/4 | oracle: layouts of the other Act 1 areas | - | - | [card](archive/openspec-index.md#fid-layout-act1) |
| 48 | fid-transitions | d2d | proposed 0/2 | probe: area transitions and arrival position | - | - | [card](archive/openspec-index.md#fid-transitions) |
| 49 | fid-monster-move | d2d | proposed 0/3 | oracle: monster velocity, pathing, stuck cases | - | - | [card](archive/openspec-index.md#fid-monster-move) |
| 50 | fid-missile-player | d2d | proposed 0/3 | oracle: player missile flight, pierce, units per frame | - | - | [card](archive/openspec-index.md#fid-missile-player) |
| 51 | fid-missile-monster | d2d | proposed 0/3 | oracle: monster missiles and ranged attackers | - | - | [card](archive/openspec-index.md#fid-missile-monster) |
| 52 | fid-boss-mods | d2d | proposed 0/2 | oracle: unique, champion and boss modifiers (merge pr-mods) | - | - | [card](archive/openspec-index.md#fid-boss-mods) |
| 53 | fid-boss-skills | d2d | proposed 0/3 | oracle: boss skills and death mods | - | - | [card](archive/openspec-index.md#fid-boss-skills) |
| 54 | fid-status-effects | d2d | proposed 0/3 | oracle: monster states, freeze rule | - | - | [card](archive/openspec-index.md#fid-status-effects) |
| 55 | fid-sorc-upstream | d2d | proposed 0/4 | upstream: private-branch Sorceress fixes (mana timing, cast length, armor group) | - | - | [card](archive/openspec-index.md#fid-sorc-upstream) |
| 56 | fid-skill-damage-pr | d2d | proposed 0/2 | upstream: skill damage min-synergy gate and MulDiv | - | - | [card](archive/openspec-index.md#fid-skill-damage-pr) |
| 57 | fid-teleport | d2d | proposed 0/3 | oracle: Teleport hops, refusal and mana | - | - | [card](archive/openspec-index.md#fid-teleport) |
| 58 | fid-hit-recovery | d2d | proposed 0/2 | oracle: player hit recovery during casts | - | - | [card](archive/openspec-index.md#fid-hit-recovery) |
| 59 | fid-lab-rebase | d2d | proposed 0/3 | lab binary rebase: private fixes into the lab binary | - | - | [card](archive/openspec-index.md#fid-lab-rebase) |
| 60 | fid-quest-den-raven | d2d | proposed 0/3 | probe: Den of Evil and Blood Raven triggers | - | - | [card](archive/openspec-index.md#fid-quest-den-raven) |
| 61 | fid-quest-andariel | bot | proposed 0/3 | probe plus A:B: Andariel kill, Act 2 entry | - | - | [card](archive/openspec-index.md#fid-quest-andariel) |
| 62 | fid-wp-tp-pr | d2d | proposed 0/3 | upstream: waypoint and town portal deviations | - | - | [card](archive/openspec-index.md#fid-wp-tp-pr) |
| 63 | fid-death | d2d | proposed 0/2 | probe: death penalty, corpse and floor rules | - | - | [card](archive/openspec-index.md#fid-death) |
| 64 | fid-gatea-ab | d2d | proposed 0/3 | statistical A:B: end-to-end rates, Gate A | - | - | [card](archive/openspec-index.md#fid-gatea-ab) |
| 65 | fid-matrix-refresh | docs | proposed 0/2 | update the matrix as cards close | - | - | [card](archive/openspec-index.md#fid-matrix-refresh) |
<!-- cardsgen:active:end -->

## Parked (not counted)

Parked by ADR 0015 item 10 or by the triage: they stay open, nobody works on them, and they come back by moving into the table above (which needs a free row).

<!-- cardsgen:parked:begin -->
| id | why parked | card |
|---|---|---|
| d2d-container-soak | - | [card](archive/openspec-index.md#d2d-container-soak) |
| m2-roam | - | [card](archive/openspec-index.md#m2-roam) |
| m2-tripwp | - | [card](archive/openspec-index.md#m2-tripwp) |
| fid-adapter-rows | - | [card](archive/openspec-index.md#fid-adapter-rows) |
| fid-lowvalue | - | [card](archive/openspec-index.md#fid-lowvalue) |
<!-- cardsgen:parked:end -->

## Needs a decision (not counted, not archived)

The triage said close or drop, but the git evidence on main does not settle it. Each stays open until the user decides.

No cards are waiting for a decision.

Resolved from git evidence and archived on 2026-11-04: gate1run-20261102 (record and commits are on main).

Resolved by the user's decision on 2026-11-04 and archived: fixcastgap, fixmanamodel, fixskilllvl and skmeleewire closed as "offline only, live proof pending" (their live proofs are listed below); d2d-default-backend dropped (the backend flip never happened, `cmd/d2bot/d2dbackend.go` still defaults `--backend` to `game`; ADR 0015 makes d2d the lab and game.exe the acceptance target).

## Open live proofs

- Under m1-liveproof (G-C basics): fixcastgap, cast gaps per skill with the lab binary carrying fix/lightning-sequence (docs/fixcastgap.md).
- Under m1-liveproof (G-C basics): fixmanamodel, modelled mana against the dump over a fight (docs/fixmanamodel.md).
- Under m1-liveproof (G-C basics): fixskilllvl, stat vector with +skills gear and drills P1-P6 of the SKPASS trace (docs/fixskilllvl.md).
- Under m1-liveproof (G-C basics): skmeleewire, early-game soak with a staff caster and a sword fighter, drill R1 (docs/skmeleewire.md).
