# Cards

> status: OpenSpec is retired for new work (user, 2026-11-04); cards are moving to Backlog.md (docs/decisions.md Current truth; docs/process/openspec-removal-plan.md).

> The 17 cards below (12 active, 5 parked) were copied into `backlog/` on 2026-11-04 (stage 1, see docs/backlog.md); this file stays the interim list until stage 2.

Stage 1 of the OpenSpec removal plan (docs/process/openspec-removal-plan.md, branch agent/removalplan, not yet on main): this table is the list of active work. The user approved it on 2026-11-04 after the card triage. Each row links to its plain markdown card under `openspec/changes/<id>/` (tasks.md header holds owns, exclusive and gates).

Rules (ADR 0015 items 10 and 11):
- At most 12 rows in the table below. Every row maps to exactly one goal G-A..G-F.
- One row per active job. A job without a card is named by its branch `agent/<id>`.
- Parked cards stay open but are off the cap: they are listed in "Parked" and are not counted. Nothing is deleted.
- "Needs a decision" cards stay open with git evidence only; they are not counted and not archived until the user decides.

Goals: G-A acceptance on game.exe, G-B Act 1 clear on d2d, G-C stability and basics, G-D measurement, G-E upstream fixes, G-F rules (see ADR 0015).

## Active (12)

| # | id | goal | status | what | owner / exclusive | next step | card |
|---|---|---|---|---|---|---|---|
| 1 | m3-act1run | G-B | deferred 0/3 | fresh Sorceress clears Act 1 unattended on d2d | - | not run live; absorbs the quest-chain and readiness floors text | [card](../openspec/changes/m3-act1run/) |
| 2 | m3-andariel | G-B | deferred 0/3 | Andariel fight card with the SME decisions (docs/andarielsme.md) | - | first drill per the SME card | [card](../openspec/changes/m3-andariel/) |
| 3 | m3-merc | G-B | proposed 0/4 | free Rogue run for hours on d2d, merc alive share | - | the soak (revive knob and hire drill already archived) | [card](../openspec/changes/m3-merc/) |
| 4 | gameexe-retire-soak | G-A | deferred 0/2 | game.exe validation paths; becomes the Gate A card | - | rewrite the text for ADR 0015 (acceptance, not retirement) | [card](../openspec/changes/gameexe-retire-soak/) |
| 5 | armorhit | G-E | in-progress 6/1 | Frozen Armor striker fix for upstream | - | PR claim (one batch at a time) | [card](../openspec/changes/armorhit/) |
| 6 | nexthit | G-E | in-progress 6/2 | NextHit justhit fix for upstream | - | PR claim | [card](../openspec/changes/nexthit/) |
| 7 | skmiss | G-E | in-progress 5/2 | missile roll max fix for upstream | - | differential, then PR | [card](../openspec/changes/skmiss/) |
| 8 | skresistfix | G-E | in-progress 6/1 | curse-on-immune fix for upstream | - | PR claim | [card](../openspec/changes/skresistfix/) |
| 9 | m1-liveproof | G-C | in-progress 13/2 | live drills for the basics; M1 exit | - | the two open rows of the M1 exit list | [card](../openspec/changes/m1-liveproof/) |
| 10 | fixbelt | G-C | in-progress 3/1 | low-kit belt window fix | - | live proof in a seed-set record | [card](../openspec/changes/fixbelt/) |
| 11 | fixprogress | G-C | in-progress 2/1 | trip-loop progress fix | - | live proof in a seed-set record | [card](../openspec/changes/fixprogress/) |
| 12 | ledgerkey | G-F | in-progress 2/1 | cheat ledger key per save (uncheated rule) | - | merge check, then close | [card](../openspec/changes/ledgerkey/) |

## Parked (not counted)

Parked by ADR 0015 item 10 or by the triage: they stay open, nobody works on them, and they come back by moving into the table above (which needs a free row).

| id | why parked | card |
|---|---|---|
| d2d-container-soak | the container soak is deferred (ADR 0015 item 5) | [card](../openspec/changes/d2d-container-soak/) |
| d2dmesh | named in ADR 0015 item 10 | [card](../openspec/changes/d2dmesh/) |
| stream-grid | named in ADR 0015 item 10 | [card](../openspec/changes/stream-grid/) |
| m2-roam | area-choice work (noarea cards) is archived; revisit after Act 1 runs | [card](../openspec/changes/m2-roam/) |
| m2-tripwp | same as m2-roam | [card](../openspec/changes/m2-tripwp/) |

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
