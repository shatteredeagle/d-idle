# Cards

> status: OpenSpec is retired for new work (user, 2026-11-04); cards are moving to Backlog.md (docs/decisions.md Current truth; docs/process/openspec-removal-plan.md).

> The 17 cards below (12 active, 5 parked) were copied into `backlog/` on 2026-11-04 (stage 1, see docs/backlog-howto.md); this file stays the interim list until stage 2.

Stage 1 of the OpenSpec removal plan (docs/process/openspec-removal-plan.md, branch agent/removalplan, not yet on main): this table is the list of active work. The user approved it on 2026-11-04 after the card triage. Each row links to its plain markdown card under `openspec/changes/<id>/` (tasks.md header holds owns, exclusive and gates).

Rules (ADR 0015 items 10 and 11):
- No cap on rows (the cap of 12 was removed by the grill 0005 outcome, ADR 0016). Every row maps to exactly one goal G-A..G-G.
- One row per active job. A job without a card is named by its branch `agent/<id>`.
- Parked cards stay open but are listed in "Parked". Nothing is deleted.
- "Needs a decision" cards stay open with git evidence only; they are not counted and not archived until the user decides.

Goals: G-A acceptance on game.exe, G-B Act 1 clear on d2d, G-C stability and basics, G-D measurement, G-E upstream fixes, G-F rules (see ADR 0015), G-G automation ladder (ADR 0016).

<!-- cardsgen:active:begin -->
## Active (21)

| # | id | goal | status | what | owner / exclusive | next step | card |
|---|---|---|---|---|---|---|---|
| 1 | m3-act1run | G-B | deferred 0/3 | fresh Sorceress clears Act 1 unattended on d2d | soak-slot:3 | - | [card](archive/openspec-index.md#m3-act1run) |
| 2 | m3-andariel | G-B | deferred 0/3 | Andariel fight card with the SME decisions (docs/andarielsme.md) | soak-slot:3 | - | [card](archive/openspec-index.md#m3-andariel) |
| 3 | m3-merc | G-B | proposed 0/4 | free Rogue run for hours on d2d, merc alive share | soak-slot:3 | - | [card](archive/openspec-index.md#m3-merc) |
| 4 | gameexe-retire-soak | G-A | deferred 0/2 | game.exe validation paths; becomes the Gate A card | - | - | [card](archive/openspec-index.md#gameexe-retire-soak) |
| 5 | m1-liveproof | G-C | in-progress 13/15 | live drills for the basics; M1 exit | soak-slot:0 | - | [card](archive/openspec-index.md#m1-liveproof) |
| 6 | ledgerkey | G-F | in-progress 2/3 | cheat ledger key per save (uncheated rule) | - | - | [card](archive/openspec-index.md#ledgerkey) |
| 7 | decision-point-store | G-G | in-progress 3/5 | learn-by-doing store: answers, streaks, rules, fading thoughts | - | tier order: after T4/Gate A cards per ADR 0016 | [card](archive/openspec-index.md#decision-point-store) |
| 8 | watch-screen | G-G | proposed 5/5 | Thoughts feed, orders, uptime, rules list and journal on d2dview | - | tier order: after T4/Gate A cards per ADR 0016 | [card](archive/openspec-index.md#watch-screen) |
| 9 | t0-slice | G-G | in-progress 3/5 | wire T0 decision points into combat with the untrained default | - | tier order: after T4/Gate A cards per ADR 0016 | [card](archive/openspec-index.md#t0-slice) |
| 10 | decisions-mirror | G-F | deferred 0/4 | ADRs mirrored read-only into backlog/decisions for the board | - | - | [card](archive/openspec-index.md#decisions-mirror) |
| 11 | testerwatch | G-F | proposed 1/3 | separate watch token, tap video source switch | - | - | [card](archive/openspec-index.md#testerwatch) |
| 12 | testerimage | G-F | deferred 0/2 | >- | - | - | [card](archive/openspec-index.md#testerimage) |
| 13 | testerlive | G-F | proposed 0/3 | run the tester flow in the real d2d container | - | - | [card](archive/openspec-index.md#testerlive) |
| 14 | testerdocs | G-F | proposed 0/2 | d2d tester guide and first-stranger checklist | - | - | [card](archive/openspec-index.md#testerdocs) |
| 15 | testerreport | G-F | proposed 0/1 | scrubbed report bundle for the d2d tester | - | - | [card](archive/openspec-index.md#testerreport) |
| 16 | testerlicence | G-F | proposed 0/2 | GPL package for the d.idle fork shipped with its source | - | - | [card](archive/openspec-index.md#testerlicence) |
| 17 | testerpack | G-F | proposed 0/1 | build the d2d tester pack | - | - | [card](archive/openspec-index.md#testerpack) |
| 18 | testertap | G-F | proposed 2/2 | tap screenshot frame source for the watch video | - | - | [card](archive/openspec-index.md#testertap) |
| 19 | testerd2dimage | G-F | proposed 0/2 | d2d tester image, realtime entry, compose file, ports | - | - | [card](archive/openspec-index.md#testerd2dimage) |
| 20 | testerd2dwiz | G-F | proposed 0/2 | first-run wizard for the d2d tester container | - | - | [card](archive/openspec-index.md#testerd2dwiz) |
| 21 | testerrt | G-F | proposed 0/2 | realtime ladder check on d2d for the tester build | - | - | [card](archive/openspec-index.md#testerrt) |
<!-- cardsgen:active:end -->

## Parked (not counted)

Parked by ADR 0015 item 10 or by the triage: they stay open, nobody works on them, and they come back by moving into the table above (which needs a free row).

<!-- cardsgen:parked:begin -->
| id | why parked | card |
|---|---|---|
| d2d-container-soak | - | [card](archive/openspec-index.md#d2d-container-soak) |
| d2dmesh | - | [card](archive/openspec-index.md#d2dmesh) |
| stream-grid | - | [card](archive/openspec-index.md#stream-grid) |
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
