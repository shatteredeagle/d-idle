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
## Active (52)

| # | id | goal | status | what | owner / exclusive | next step | card |
|---|---|---|---|---|---|---|---|
| 1 | m3-act1run | T4 | deferred 0/3 | fresh Sorceress clears Act 1 unattended on d2d | soak-slot:3 | - | [card](archive/openspec-index.md#m3-act1run) |
| 2 | m3-andariel | T4 | deferred 0/3 | Andariel fight card with the SME decisions (docs/andarielsme.md) | soak-slot:3 | - | [card](archive/openspec-index.md#m3-andariel) |
| 3 | m3-merc | T4 | deferred 0/4 | free Rogue run for hours on d2d, merc alive share | soak-slot:3 | - | [card](archive/openspec-index.md#m3-merc) |
| 4 | gameexe-retire-soak | gate | deferred 0/2 | game.exe validation paths; becomes the Gate A card | - | - | [card](archive/openspec-index.md#gameexe-retire-soak) |
| 5 | m1-liveproof | drills | deferred 13/15 | live drills for the basics; M1 exit | soak-slot:0 | - | [card](archive/openspec-index.md#m1-liveproof) |
| 6 | decision-point-store | T0 | in-progress 3/5 | learn-by-doing store: answers, streaks, rules, fading thoughts | - | tier order: after T4/Gate A cards per ADR 0016 | [card](archive/openspec-index.md#decision-point-store) |
| 7 | t0-slice | T0 | in-progress 3/5 | wire T0 decision points into combat with the untrained default | - | tier order: after T4/Gate A cards per ADR 0016 | [card](archive/openspec-index.md#t0-slice) |
| 8 | decisions-mirror | tooling | deferred 0/4 | ADRs mirrored read-only into backlog/decisions for the board | - | - | [card](archive/openspec-index.md#decisions-mirror) |
| 9 | testerwatch | tester | in-progress 2/3 | separate watch token, tap video source switch | - | - | [card](archive/openspec-index.md#testerwatch) |
| 10 | testerlive | tester | proposed 0/3 | run the tester flow in the real d2d container | - | - | [card](archive/openspec-index.md#testerlive) |
| 11 | testerdocs | tester | proposed 0/2 | d2d tester guide and first-stranger checklist | - | - | [card](archive/openspec-index.md#testerdocs) |
| 12 | testerreport | tester | proposed 0/1 | scrubbed report bundle for the d2d tester | - | - | [card](archive/openspec-index.md#testerreport) |
| 13 | testerlicence | tester | proposed 0/2 | GPL package for the d.idle fork shipped with its source | - | - | [card](archive/openspec-index.md#testerlicence) |
| 14 | testerpack | tester | proposed 0/1 | build the d2d tester pack | - | - | [card](archive/openspec-index.md#testerpack) |
| 15 | testerd2dimage | tester | in-progress 1/2 | d2d tester image, realtime entry, compose file, ports | - | - | [card](archive/openspec-index.md#testerd2dimage) |
| 16 | testerd2dwiz | tester | in-progress 1/2 | first-run wizard for the d2d tester container | - | - | [card](archive/openspec-index.md#testerd2dwiz) |
| 17 | testerrt | tester | proposed 0/2 | realtime ladder check on d2d for the tester build | - | - | [card](archive/openspec-index.md#testerrt) |
| 18 | ordersux2 | T0 | proposed 0/5 | orders slice 2: until-level, clear then home, queue, conflict | - | - | [card](archive/openspec-index.md#ordersux2) |
| 19 | dpwire | T0 | proposed 0/3 | wire the remaining decision points into the bot | - | - | [card](archive/openspec-index.md#dpwire) |
| 20 | t0fight | T0 | proposed 0/3 | engineered drill for a fight-time question and a learned answer | - | - | [card](archive/openspec-index.md#t0fight) |
| 21 | ladder-t1 | T1 | proposed 0/3 | Tidy tier: cards and wiring for what T1 still lacks | - | - | [card](archive/openspec-index.md#ladder-t1) |
| 22 | ladder-t2 | T2 | proposed 0/3 | Runner tier: urgency stage 2 and the unwired T2 points | - | - | [card](archive/openspec-index.md#ladder-t2) |
| 23 | earlydeath13 | bot-defect | in-progress 2/3 | seed 13 dies at clvl 4 to a Gargantuan Beast on Cold Plains | - | - | [card](archive/openspec-index.md#earlydeath13) |
| 24 | tpreserve-live | bot-defect | proposed 0/2 | prove the scroll reserve and no-portal leave with the tp field | - | - | [card](archive/openspec-index.md#tpreserve-live) |
| 25 | g2f-record | gate | in-progress 3/3 | Gate 2 g2f result, diagnosis and the open decisions | - | - | [card](archive/openspec-index.md#g2f-record) |
| 26 | fork-labverbs | tester | proposed 3/3 | gate the mutating debug verbs behind a fork flag | fork-branch:d2d.lab3-labverbs | - | [card](archive/openspec-index.md#fork-labverbs) |
| 27 | playsession | T0 | proposed 0/3 | one command for a LAN play session | - | - | [card](archive/openspec-index.md#playsession) |
| 28 | onboardA | onboarding | in-progress 5/5 | onboarding slice A: setup stepper, go-ahead, access | - | - | [card](archive/openspec-index.md#onboardA) |
| 29 | upstream-next | upstream | proposed 0/2 | next upstream PR batch candidates (one open batch at a time) | - | - | [card](archive/openspec-index.md#upstream-next) |
| 30 | statusrefresh | tooling | proposed 0/2 | docs/plan/STATUS.md hand-written part is stale (2026-11-03) | - | - | [card](archive/openspec-index.md#statusrefresh) |
| 31 | ordersux3 | T0 | proposed 0/4 | orders slice 3: reorder, dismiss, cleared, Acts 2-5, screen reader | - | - | [card](archive/openspec-index.md#ordersux3) |
| 32 | onboardtour | onboarding | proposed 0/3 | onboarding slice B: tour, Help, state banners | - | - | [card](archive/openspec-index.md#onboardtour) |
| 33 | onboardctl | onboarding | proposed 0/3 | onboarding slice C: Pause, Stop, Reset, report hook | - | - | [card](archive/openspec-index.md#onboardctl) |
| 34 | friendspreview | tester | proposed 2/2 | shareable clickable preview of the flow for friends feedback | - | - | [card](archive/openspec-index.md#friendspreview) |
| 35 | mercrevive | bot-defect | in-progress 3/3 | revive reserve by clvl band per the SME | - | - | [card](archive/openspec-index.md#mercrevive) |
| 36 | shortdrills | drills | in-progress 3/3 | short live drills for the g2e/g2f fixes | - | - | [card](archive/openspec-index.md#shortdrills) |
| 37 | abmerc | tooling | in-progress 3/4 | paired A/B of the merc revive reserve; A/B standard | - | - | [card](archive/openspec-index.md#abmerc) |
| 38 | fidelity-matrix | gate | in-progress 3/3 | bot use versus game.exe, how each row is verified | - | - | [card](archive/openspec-index.md#fidelity-matrix) |
| 39 | fidprobe-core | gate | proposed 0/3 | same-save probe tool and fresh/clvl 6 scenarios | - | - | [card](archive/openspec-index.md#fidprobe-core) |
| 40 | fidprobe-vendor | gate | proposed 0/3 | vendor stock and quote probe, phase 2 | - | - | [card](archive/openspec-index.md#fidprobe-vendor) |
| 41 | tpcount | bot-defect | in-progress 2/2 | minute-line tp reads 0 in the g2g runs | - | - | [card](archive/openspec-index.md#tpcount) |
| 42 | customitems | T5 | in-progress 3/3 | feasibility of own items and drop-table overlays on d2d | - | - | [card](archive/openspec-index.md#customitems) |
| 43 | custom-overlay | T5 | in-progress 1/2 | row-merge table overlay in the d.idle fork and Go loader | - | - | [card](archive/openspec-index.md#custom-overlay) |
| 44 | custom-item-1 | T5 | in-progress 0/2 | stackable currency token from one early source | - | - | [card](archive/openspec-index.md#custom-item-1) |
| 45 | metacurrency | T5 | in-progress 0/3 | per-character and account-wide currencies from custom items | - | - | [card](archive/openspec-index.md#metacurrency) |
| 46 | currency-ledger | T5 | in-progress 2/2 | registry and account ledger (Go, offline) | - | - | [card](archive/openspec-index.md#currency-ledger) |
| 47 | currency-rows | T5 | in-progress 0/2 | token row, drop line and bank recipe overlay | - | - | [card](archive/openspec-index.md#currency-rows) |
| 48 | currency-slice | T5 | proposed 0/2 | bank step, balance line and kill drill | - | - | [card](archive/openspec-index.md#currency-slice) |
| 49 | g2grecord | gate | in-progress 3/3 | Gate 2 g2g wave 1 result | - | - | [card](archive/openspec-index.md#g2grecord) |
| 50 | g2gfixmerc | bot-defect | in-progress 3/3 | merc caps and gold-farm need on the virtual clock and the read price | - | - | [card](archive/openspec-index.md#g2gfixmerc) |
| 51 | g2gfixmouth | bot-defect | proposed 0/2 | nav mouth gate no safe click loop at repeated spots | - | - | [card](archive/openspec-index.md#g2gfixmouth) |
| 52 | g2gfixscroll | bot-defect | proposed 0/2 | no town portal scroll with funded gold leaves the hero stuck outside | - | - | [card](archive/openspec-index.md#g2gfixscroll) |
<!-- cardsgen:active:end -->

## Parked (not counted)

Parked by ADR 0015 item 10 or by the triage: they stay open, nobody works on them, and they come back by moving into the table above (which needs a free row).

<!-- cardsgen:parked:begin -->
| id | why parked | card |
|---|---|---|
| d2d-container-soak | - | [card](archive/openspec-index.md#d2d-container-soak) |
| m2-roam | - | [card](archive/openspec-index.md#m2-roam) |
| m2-tripwp | - | [card](archive/openspec-index.md#m2-tripwp) |
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
