# Card triage and decisions without an ADR, 2026-11-04

> status: proposal for the lead, docs only (job cardtriage). Nothing was moved, edited or archived; no ADR or decisions.md text was written, proposed text is below only. Goals G-A..G-F, the cap of 12 and the parking wording come from [ADR 0015](../adr/0015-goals-acceptance-on-game-exe.md). Tags: V = read in this repo (file or `git log`), I = judgement, U = the repo does not settle it. Earlier pass: [research/card-triage-20261103.md](../research/card-triage-20261103.md); since then 17 cards were archived (V, `ls openspec/changes/archive`: soak-minutes-virtual, andarielgate, andarielplan, andarielsme, andarielspec, merchire, mercrevive, noarea, noareacards, palprep, palverify, rtleave, backlogapply and the gate1multi runs).
>
> Note on OpenSpec: the user retired OpenSpec for new work on 2026-11-04, but the 36 open cards below still live under `openspec/changes/` and `changecheck` still reads them (I). This doc creates no card. Closing means "archive with the merge hash", which the merger does; whether the tooling survives the retirement is not settled in the repo (U).

## Task A: open cards

36 open cards (V, `ls openspec/changes` without archive). Cap 12, so **24 over the cap**. Result: KEEP 12, CLOSE 12, PARK 5, DROP 7.

"Tasks" is ticked/open from tasks.md (V). "Merged" is from the 2026-11-03 triage or the merge commits it cites (V there, not re-checked branch by branch). Last activity is `git log` date of the card directory (V).

| card | status, tasks | what | goal | last act. | merged or superseded (evidence) | rec |
|---|---|---|---|---|---|---|
| m3-act1run | deferred 0/3 | fresh Sorceress clears Act 1 unattended on d2d | G-B | 10-31 | not run (STATUS: nothing past Act 1 ran live, V) | KEEP 1 |
| m3-andariel | deferred 0/3 | Andariel fight card; amended with the SME decisions | G-B | 11-04 | docs/andarielsme.md; plan/spec/SME cards archived (V) | KEEP 2 |
| m3-merc | proposed 0/4 | free Rogue run for hours on d2d, merc alive share | G-B | 10-31 | mercrevive, merchire archived 11-04 (V) cover the revive knob and hire drill, not this soak | KEEP 3 |
| gameexe-retire-soak | deferred 0/2 | game.exe validation paths; becomes the Gate A card | G-A | 10-30 | text is about retirement, ADR 0015 changed the aim; needs a rewrite (I) | KEEP 4 |
| armorhit | in-progress 6/1 | Frozen Armor striker fix for upstream | G-E | 11-01 | merged (2026-11-03 triage); open task is the PR claim | KEEP 5 |
| nexthit | in-progress 6/2 | NextHit justhit fix for upstream | G-E | 11-01 | merged; open: PR claim | KEEP 6 |
| skmiss | in-progress 5/2 | missile roll max fix for upstream | G-E | 11-01 | merged; open: differential/PR | KEEP 7 |
| skresistfix | in-progress 6/1 | curse-on-immune fix for upstream | G-E | 11-01 | merged; open: PR claim | KEEP 8 |
| m1-liveproof | in-progress 13/2 | live drills for basics; M1 exit | G-C | 10-31 | merged; 2 open rows are the M1 exit list | KEEP 9 |
| fixbelt | in-progress 3/1 | low-kit belt window fix | G-C | 11-01 | merges 825cc2ca, 5b1fbcea (V); a 3-seed set later archived (gate1multi-*, V); live proof U | KEEP 10 |
| fixprogress | in-progress 2/1 | trip-loop progress fix | G-C | 11-01 | merged; live proof U | KEEP 11 |
| ledgerkey | in-progress 2/1 | cheat ledger key per save | G-F (uncheated rule) | 11-01 | no merge found on 11-03 (V there) | KEEP 12 |
| fixcastgap | in-progress 4/1 | per-skill cast gap | G-C | 11-01 | merged; only live proof open | CLOSE |
| fixmanamodel | in-progress 4/1 | mana regen/booking model | G-C | 11-01 | merged; only live proof open | CLOSE |
| fixskilllvl | in-progress 3/1 | +skill level bonuses | G-C | 11-01 | merged; only live proof open | CLOSE |
| skmeleewire | in-progress 4/1 | wire melee blow model to snapshot | G-C | 11-01 | merged; only live proof open | CLOSE |
| buildv3 | in-progress 3/1 | fallback-lightning-v3 table, opt-in | G-B | 11-01 | merged; v3 default switch deferred by ADR 0015 item 10 (V) | CLOSE |
| drills6 | in-progress 4/1 | live rerun of four fixes | G-C | 10-31 | merged; open task is a note | CLOSE |
| gate1run-20261101 | in-progress 2/1 | generated Gate 1 record (NOT PASSED) | G-C | 11-01 | records live in docs/soaks; seed-11 single runs dropped (ADR 0015 item 8) | CLOSE |
| gate1run-20261102 | in-progress 2/1 | generated Gate 1 record (PASSED) | G-C | 11-03 | same; branch merge state U | CLOSE |
| goldfloor | deferred 3/0 | gold floor levers, off by default | none (parked by its own result) | 11-03 | delta spec not applied (2026-11-03 triage, V) | CLOSE |
| paldurtrace | in-progress 4/0 | durability trace for the Paladin | G-E | 11-03 | all ticked; palprep/palverify archived 11-04 (V) | CLOSE |
| palskilltrace | in-progress 5/0 | Smite/Zeal/Sacrifice trace | G-E | 11-03 | all ticked; palverify merge e079944c, palprep 9bf824f0 (V) | CLOSE |
| mapitem | in-progress 4/1 | contiguous stream map | none (stream) | 10-31 | merged; live check is a nicety | CLOSE |
| d2d-container-soak | proposed 0/3 | 24 h container soak | none | 10-30 | ADR 0015 item 5 defers it | PARK |
| d2dmesh | deferred 7/0 | peer-to-peer farm mesh | none | 10-31 | named in ADR 0015 item 10; design branch merge U | PARK |
| stream-grid | deferred 4/3 | public stream grid page | none | 10-31 | named in ADR 0015 item 10 | PARK |
| m2-roam | deferred 0/4 | cut walk time between areas | G-B | 10-31 | area-choice work: noarea, noareacards archived 11-04 (V) | PARK |
| m2-tripwp | deferred 0/3 | cheaper town trips | G-B | 10-31 | same | PARK |
| d2d-default-backend | proposed 0/3 | flip --backend default to d2d | none | 10-31 | ADR 0013 made d2d the lab backend; whether the flip happened U | DROP |
| d2d-replicated-soak | proposed 0/3 | replicated fidelity soak | G-D | 10-30 | replaced by the seed set and the statistical A:B (ADR 0015 items 6, 8) | DROP |
| m1-trips | deferred 0/5 | trip trigger waste | G-C | 10-31 | minor next to stability (2026-11-03 triage, I) | DROP |
| m2-latebase | deferred 0/3 | single-seed late baseline | G-D | 10-31 | single seeds dropped (item 8) | DROP |
| orig-decision-probes | deferred 0/2 | probes on the original game | G-D | 10-30 | superseded by A:B and Gate A | DROP |
| m3-act1chain | deferred 0/3 | quest chain after Blood Raven | G-B | 10-31 | fold the chain text into m3-act1run | DROP |
| m3-readiness-model | deferred 0/4 | readiness replaces fixed floors | G-B | 10-31 | ADR 0011 and andarielsme set floors (15/17 plus life floor); fold into m3-act1run | DROP |

### The 12 kept, ranked, and why
1-3: the G-B path (act1run, Andariel, free Rogue) is what the project is for and has the only recent decisions. 4: the only G-A card; ADR 0015 puts game.exe acceptance first, but its text is about retirement and must be rewritten. 5-8: G-E upstream items, merged offline, waiting on the maintainer; ADR 0015 says one batch at a time, so they cost nothing to hold. 9: M1 exit list. 10-11: fixbelt and fixprogress each fixed a named Gate 1 failure; keep until a seed-set record shows them live (U whether the later gate1multi records already do). 12: ledgerkey protects the "uncheated" rule; small. Ties 9-12 are a judgement (I); the other G-C fix cards would fit the same slot, they are CLOSE only because their one open task is the same live proof, and that proof should become one BACKLOG line each (the merger does that) rather than twelve cards.

UNSURE: whether the user accepts closing cards whose live proof never ran (fixcastgap, fixmanamodel, fixskilllvl, skmeleewire); whether gate1run-20261102's branch is merged (U); whether d2d-default-backend's flip already happened (U).

## Task B: decisions with only a decisions.md bullet

Source: the dated bullets since 2026-11-01 in docs/decisions.md (V). "ADR?" is my recommendation.

| date | decision | ADR? |
|---|---|---|
| 11-01 | every minute in d2d soak gates is a virtual minute (belt window 5, no kill/XP 10) | no: the soak-gates spec holds it (card soak-minutes-virtual archived 2026-10-31, V); an ADR would only copy the spec |
| 11-01 | build flexibility: new code derives from build data and profiles, unknown class or table is an error, never a Sorceress fallback | yes |
| 11-01 | approve fallback-lightning-v3 rows, default stays v2 | no: ADR 0015 item 10 defers the switch |
| 11-01 | contributor guide and docs approved | no: process text, no architecture |
| 11-02 | memory writes allowed, bot never depends on them | already ADR 0014 |
| 11-03 | Gate 1 belt rule (poverty windows reported, not failing) | no: soak-gates spec delta (research/specdelta-rejudge.md) |
| 11-03 | Gate 1 XP rule: one seed, inconclusive, inside tolerance passes | yes, to reconcile: ADR 0015 item 8 says XP/h is never a per-run verdict, this bullet says one seed can pass on XP; the repo does not say which survives (U) |
| 11-03 | seed-set XP verdict, single-seed re-runs dropped, milestone time is the headline | no: ADR 0015 items 8 and 9 |
| 11-03 | rule coverage dropped, container deferred, one page of truth, gate character treated as Sorceress | no: ADR 0015 items 5, 10, 11 |
| 11-04 | m3-andariel plan decisions (docs/andarielsme.md) | yes |

### Proposed ADRs (text for the lead only)

**ADR 00NN: Gate verdicts on XP come from a seed set; the 2026-11-03 single-seed pass rule is withdrawn (or kept)**
The soak-gates spec and ADR 0015 item 8 conflict with the 2026-11-03 bullet. Decide one: (a) per-run gates judge stability, invariants and funded belt windows only, XP/h is judged on at least 5 seeds with 2 holdouts, the single-seed "inconclusive passes" note is dropped; or (b) keep the note as a stopgap until the first seed set exists. Records say which rule judged them.

**ADR 00NN: New code is build-driven; there is no Sorceress fallback**
Class, skill order, combat bands and kit come from the plan table, combat profile and class traits (research/build-flexibility-audit.md). An unknown class or table is an error. Reason: the gate character class is itself unconfirmed (STATUS open question), so a silent default can void gate numbers.

**ADR 00NN: Andariel fight policy for the first runs**
No Static Field opener until a drill measures it; fight floor clvl 15 with a living merc, 17 without, plus a life floor from the diagnostic run; kit 8 healing, 3 antidotes, 4 mana, tome with 2 charges. A declared cheat setup (with handicapped variants and at least 3 no-merc runs) may replace the road; the hold lifts after 5 clean wins in a row, `writes after start: 0`, user decides. Merc revive: knob first (`merc.goldFarmMaxLevel` 18, reserve 100 percent of a refill).

The virtual-minutes rule needs no ADR (above); if the lead wants one for discoverability: "In d2d soak gates every minute is a virtual minute; belt window 5, no kill and no XP 10; the spec soak-gates is the owner."

## Shared-file lines
Docs-only change, no BACKLOG/INDEX/CHANGELOG line needed from this job; the archive and BACKLOG moves above are the merger's.
