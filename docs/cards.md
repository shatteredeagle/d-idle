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
## Active (46)

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
| 18 | t0fight | T0 | proposed 0/3 | engineered drill for a fight-time question and a learned answer | - | - | [card](archive/openspec-index.md#t0fight) |
| 19 | earlydeath13 | bot | in-progress 2/3 | seed 13 dies at clvl 4 to a Gargantuan Beast on Cold Plains | - | - | [card](archive/openspec-index.md#earlydeath13) |
| 20 | tpreserve-live | bot | proposed 0/2 | prove the scroll reserve and no-portal leave with the tp field | - | - | [card](archive/openspec-index.md#tpreserve-live) |
| 21 | playsession | T0 | proposed 0/3 | one command for a LAN play session | - | - | [card](archive/openspec-index.md#playsession) |
| 22 | upstream-next | d2d | proposed 0/2 | next upstream PR batch candidates (one open batch at a time) | - | - | [card](archive/openspec-index.md#upstream-next) |
| 23 | statusrefresh | docs | proposed 0/2 | docs/plan/STATUS.md hand-written part is stale (2026-11-03) | - | - | [card](archive/openspec-index.md#statusrefresh) |
| 24 | onboardtour | watch | proposed 0/3 | onboarding slice B: tour, Help, state banners | - | - | [card](archive/openspec-index.md#onboardtour) |
| 25 | onboardctl | watch | proposed 0/3 | onboarding slice C: Pause, Stop, Reset, report hook | - | - | [card](archive/openspec-index.md#onboardctl) |
| 26 | abmerc | ops | in-progress 3/4 | paired A/B of the merc revive reserve; A/B standard | - | - | [card](archive/openspec-index.md#abmerc) |
| 27 | fidprobe-vendor | bot | proposed 0/3 | vendor stock and quote probe, phase 2 | - | - | [card](archive/openspec-index.md#fidprobe-vendor) |
| 28 | custom-overlay | T5 | in-progress 1/2 | row-merge table overlay in the d.idle fork and Go loader | - | - | [card](archive/openspec-index.md#custom-overlay) |
| 29 | fid-load-life-mana | d2d | proposed 3/4 | d2d keeps the saved life and mana at load (pilot) | - | - | [card](archive/openspec-index.md#fid-load-life-mana) |
| 30 | fid-load-maxmana | d2d | proposed 0/3 | d2d truncates the Pally maximum mana at load (19.5 to 19.0) | - | - | [card](archive/openspec-index.md#fid-load-maxmana) |
| 31 | fid-poison-res-diag | d2d | proposed 0/3 | diagnose Freshe poison resist 6 in d2d, absent in game.exe | - | - | [card](archive/openspec-index.md#fid-poison-res-diag) |
| 32 | fid-probe-doc | docs | proposed 2/2 | describe the d2s reader, scenario 15 and the new options in fidprobe.md | - | - | [card](archive/openspec-index.md#fid-probe-doc) |
| 33 | fid-fixtures | ops | proposed 0/3 | create the true fresh Paladin and a sorc-6 with gear test character | - | - | [card](archive/openspec-index.md#fid-fixtures) |
| 34 | sessionrec | T0 | proposed 0/3 | play-session command plus a session record (questions per 10 minutes) | - | - | [card](archive/openspec-index.md#sessionrec) |
| 35 | h1baseline | docs | proposed 0/3 | H4 and H1 baseline sessions (the user plays 1 h watching, 30 min from the clvl 6 save) | - | - | [card](archive/openspec-index.md#h1baseline) |
| 36 | qdensity | T0 | proposed 0/3 | question-density variant as data (ladder catalogue variants, fade 45 to 90 s, Forgotten Tower question gating) | - | - | [card](archive/openspec-index.md#qdensity) |
| 37 | returnmock | watch | proposed 0/2 | return-screen mock (day-away story, missed questions grouped by kind) | - | - | [card](archive/openspec-index.md#returnmock) |
| 38 | run4h | ops | proposed 0/2 | 4-hour unattended realtime run of the play-session command | instance:i4 | - | [card](archive/openspec-index.md#run4h) |
| 39 | smokebot | ops | proposed 0/3 | 45-minute smoke per bot change (Gate 0.5 or Gate 1 on one seed) | - | - | [card](archive/openspec-index.md#smokebot) |
| 40 | h3verdict | docs | proposed 0/2 | H3 orders session and the H1 verdict (the user plays 1 h with orders only) | - | - | [card](archive/openspec-index.md#h3verdict) |
| 41 | lanhowto | docs | proposed 0/2 | one friend over the LAN, a one-page how-to | - | - | [card](archive/openspec-index.md#lanhowto) |
| 42 | designbrief | docs | proposed 0/2 | design brief v1 (the loop that won, three metrics with targets, what is cut) | - | - | [card](archive/openspec-index.md#designbrief) |
| 43 | qqueue | T0 | proposed 0/3 | questions queue for the player''s return; remove the 45 s fade (D2) | - | - | [card](archive/openspec-index.md#qqueue) |
| 44 | dietcards | ops | proposed 0/3 | stop committing docs/cards.md (generate on demand or in CI; fix the mergeq preflight and the cards_md --check use) | - | - | [card](archive/openspec-index.md#dietcards) |
| 45 | dietarchive | docs | proposed 0/2 | archive finished research docs with stubs | - | - | [card](archive/openspec-index.md#dietarchive) |
| 46 | dietrule | docs | proposed 0/2 | docs-only merge limit rule in docs/backlog-howto.md | - | - | [card](archive/openspec-index.md#dietrule) |
<!-- cardsgen:active:end -->

## Parked (not counted)

Parked by ADR 0015 item 10 or by the triage: they stay open, nobody works on them, and they come back by moving into the table above (which needs a free row).

<!-- cardsgen:parked:begin -->
| id | why parked | card |
|---|---|---|
| d2d-container-soak | - | [card](archive/openspec-index.md#d2d-container-soak) |
| m2-roam | - | [card](archive/openspec-index.md#m2-roam) |
| m2-tripwp | - | [card](archive/openspec-index.md#m2-tripwp) |
| ordersux2 | - | [card](archive/openspec-index.md#ordersux2) |
| dpwire | - | [card](archive/openspec-index.md#dpwire) |
| ladder-t1 | - | [card](archive/openspec-index.md#ladder-t1) |
| ladder-t2 | - | [card](archive/openspec-index.md#ladder-t2) |
| ordersux3 | - | [card](archive/openspec-index.md#ordersux3) |
| custom-item-1 | - | [card](archive/openspec-index.md#custom-item-1) |
| metacurrency | - | [card](archive/openspec-index.md#metacurrency) |
| currency-rows | - | [card](archive/openspec-index.md#currency-rows) |
| currency-slice | - | [card](archive/openspec-index.md#currency-slice) |
| fid-probe-readers | - | [card](archive/openspec-index.md#fid-probe-readers) |
| fid-probe-equip | - | [card](archive/openspec-index.md#fid-probe-equip) |
| fid-probe-regen-speed | - | [card](archive/openspec-index.md#fid-probe-regen-speed) |
| fid-xp-award | - | [card](archive/openspec-index.md#fid-xp-award) |
| fid-price-pr | - | [card](archive/openspec-index.md#fid-price-pr) |
| fid-stock-oracle | - | [card](archive/openspec-index.md#fid-stock-oracle) |
| fid-belt-pr | - | [card](archive/openspec-index.md#fid-belt-pr) |
| fid-inv-limits | - | [card](archive/openspec-index.md#fid-inv-limits) |
| fid-layout-act1 | - | [card](archive/openspec-index.md#fid-layout-act1) |
| fid-transitions | - | [card](archive/openspec-index.md#fid-transitions) |
| fid-monster-move | - | [card](archive/openspec-index.md#fid-monster-move) |
| fid-missile-player | - | [card](archive/openspec-index.md#fid-missile-player) |
| fid-missile-monster | - | [card](archive/openspec-index.md#fid-missile-monster) |
| fid-boss-mods | - | [card](archive/openspec-index.md#fid-boss-mods) |
| fid-boss-skills | - | [card](archive/openspec-index.md#fid-boss-skills) |
| fid-status-effects | - | [card](archive/openspec-index.md#fid-status-effects) |
| fid-sorc-upstream | - | [card](archive/openspec-index.md#fid-sorc-upstream) |
| fid-skill-damage-pr | - | [card](archive/openspec-index.md#fid-skill-damage-pr) |
| fid-teleport | - | [card](archive/openspec-index.md#fid-teleport) |
| fid-hit-recovery | - | [card](archive/openspec-index.md#fid-hit-recovery) |
| fid-lab-rebase | - | [card](archive/openspec-index.md#fid-lab-rebase) |
| fid-quest-den-raven | - | [card](archive/openspec-index.md#fid-quest-den-raven) |
| fid-quest-andariel | - | [card](archive/openspec-index.md#fid-quest-andariel) |
| fid-wp-tp-pr | - | [card](archive/openspec-index.md#fid-wp-tp-pr) |
| fid-death | - | [card](archive/openspec-index.md#fid-death) |
| fid-gatea-ab | - | [card](archive/openspec-index.md#fid-gatea-ab) |
| fid-adapter-rows | - | [card](archive/openspec-index.md#fid-adapter-rows) |
| fid-lowvalue | - | [card](archive/openspec-index.md#fid-lowvalue) |
| fid-matrix-refresh | - | [card](archive/openspec-index.md#fid-matrix-refresh) |
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
