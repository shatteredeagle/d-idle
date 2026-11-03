# BOTAB trace: bot-level d2d against game.exe (questions, deviations, bot findings, limits)

> status: current. Offline trace for card botab; data and statistics are in docs/research/botab-20261102.md (read that first). No game, no live run, no Go change, no d2d slot. Tags as there: V read (committed record or `git diff`), D (botab scripts on committed minute lines), S (another doc's number), I, U.

## Questions

| Q | question | answer | tag |
|---|---|---|---|
| Q1 | Is d2d within 15 percent of game.exe on XP per hour at bot level? | Not shown and not refuted. Every XP/h ratio interval is INCONCLUSIVE; the narrowest useful one, clvl 12-13 with 5 d2d runs against 3 game.exe lanes, is 1.19 [0.72, 1.99] | D |
| Q2 | Which comparisons in the records are apples to apples? | One: d2d gate run 1102 (bot 050a509) and the game.exe late set (bot 74abe06f66bf) run identical bot code (`git diff 050a509 74abe06f66bf -- internal cmd go.mod` is empty). Everything else differs in bot commit, character history, clock, seed or independence | V read (git diff), D |
| Q3 | Is kills per minute the same? | d2d is lower: 0.73 to 0.78 of the game.exe mean in every comparison at clvl 12-13 and 1-6 (A3: 0.76 [0.63, 0.92]); the same-code lane has 6.6 kills per minute against 7.2 to 8.5. Part may be counting, not the game (B1) | D, cause U |
| Q4 | Potions, town entries, time per area? | Potions per hour: no separable difference (1.30 [0.44, 3.89]). Town entries per hour: d2d lower, 0.70 [0.50, 0.98] (late1,2,4), inflated on game.exe by restarts re-entering town. Area shares differ (area 4 31 to 43 percent on d2d against 24 to 34; area 9 4 to 12 against 11 to 20) | D |
| Q5 | Deaths per hour? | No statement: 1 event against 1 event in the 11-14 band (rate ratio 0.09 [0.001, 6.7]); 0 against 0 at clvl 12-13 | D |
| Q6 | What do the missing answers need? | Section 5 of the data doc: the fresh clvl 1-6 cell needs 5 d2d lanes plus 2 game.exe lanes at one commit as a pilot; the clvl 12-13 cell needs about 17 clean lanes per arm at CV 13 and 15 percent (37 at the CVs seen now) | D (sample formula), I (CVs for new runs) |

## Deviations d2d against game.exe (bot level; none is a proven rule deviation)

A deviation here is a difference of an outcome the bot measures. None of these is attributed to a rule: the rule-level work is in docs/coverage/rule-coverage.md and the SK* traces. Each has a cause that is U.

| # | d2d / game.exe | tag | differential under tools/emu? |
|---|---|---|---|
| D1 | Kills per minute at clvl 12-13: d2d 5.3 to 6.6, game.exe 7.2 to 8.5 (ratio about 0.76) | D (value), U (cause: spawn density per area, kill pace of the fight loop on the lockstep clock, or the counting in B1) | Partly buildable. Spawn density: run game.exe's monster placement code (the drlg/spawn emulation of seeddist and seedspawn) for the areas 3, 4, 9 at one map seed and compare unit counts with d2d's room spawns; oracle = the unit list. Fight pace and counting have no emulator differential: they need a lab drill (section 5 of the data doc, item 3) |
| D2 | XP/h by band: d2d 1-4 about 0.63 to 0.71 of the single Fresha chain run; bands 5-7 and 8-10 straddle it; 11-14 about 0.68 to 0.80 of older runs, none significant | D (value), U (cause) | No. A rule-level XP difference would show in the monster-XP formula (level difference, party) which has its own rule row; this is not a bot differential |
| D3 | Town entries per hour: d2d 4.8, game.exe 6.1 to 8.1 | D, U (restarts re-enter town on game.exe lanes) | No emulator differential; recount with a per-trip counter once the minute line has one (B2) |
| D4 | Area mix at clvl 12-13: d2d is in area 4 more and in area 9 less | D, U (map and route guard inputs; same seed on every d2d run) | Map: yes, the layout differential of AB-SEEDPIN (drlg-dump against a live room dump). The bot's route choice: replay the planner on both maps offline (bot-side, B-type test) |
| D5 | Clock: d2d lockstep runs 6 to 12 virtual minutes per wall minute, game.exe k = 1 | V read (the records) for the ratio; U for any effect on outcomes | Yes in principle: 3 d2d lanes at k = 1 against lockstep (section 5 item 3) |

## Bot findings (our bot against the comparison; described, NOT changed)

| # | finding | where | tag |
|---|---|---|---|
| B1 | The `kills` field is not a game statistic: a kill is counted when a monster seen alive is seen dead in a later snapshot. A monster that dies and is removed between two snapshots is never counted, and the length of time a corpse stays in the monster list can differ between d2d and game.exe. So kills per minute may depend on snapshot cadence and corpse lifetime, on top of the real kill rate | internal/bot/observe.go:590 to 631 (the `aliveSeen && !counted` loop, `o.kills += len(kills)` at line 631); lifetime counter internal/bot/counters.go:28 | V read (code), U (effect size) |
| B2 | The minute line has no town-trip or portal counter, no clock factor, no bot commit, no seed or map id and no difficulty field. Town trips can only be derived from area changes, which a restart also produces; a comparison needs the commit, k and the seed in the record, not in prose | internal/bot/observe.go:959 to 962 (the minute line) | V read |
| B3 | `restarts` means different things per backend: on game.exe the menu count (also crash re-entries), on d2d only bot-started games and lost game processes. The "weak" flag of the baseline set keys on it, so a game.exe lane is flagged weak (3 of 4 late lanes) by a counter that d2d lanes cannot raise by the same events; d2d restarts of the same size would not be flagged | internal/bot/observe.go:251 to 257 (`restartCount`) | V read |
| B4 | late3 played 82 percent of its 119 minutes in area 2 (Blood Moor) at clvl 12 with 21 restarts and lost its XP (233 XP/h net, potions counter 0). Cause not found in the records (no log text read); it is a bot or harness loop that the d2d runs did not show (d2d: at most 14 restarts in a 950-minute band, and at most 16 percent of minutes in area 2) | docs/probes/baseline-runs/20261102/late3-minutes.jsonl.gz, record.md | D, U (cause) |

## PR candidates

None. No d2d deviation is proven: no differential exists for a bot-level metric, and no rule differential was run here (nothing under tools/emu was run; the evidence is committed minute lines).

## Matrix and ledger updates

- docs/coverage/ledger.tsv: no row. The ledger needs rule tests with case counts that appear in the evidence doc; nothing here proves a rule. rule-coverage.md is therefore not regenerated by this card.
- Matrix proposals: none for the skill matrices. For the fidelity-gate spec, the proposal of docs/research/tierb-design-20261102.md section 4.1 stands and gets one input from this card: the A:B record should carry the bot commit, k, seed or map id and the restart reason, which the minute line lacks today (B2).

## Dead ends

- Kit sets (kit, kitk2, kitb): the minute logs are not in this repo; only the per-lane numbers of tierb section 6 could be used (S). A recompute with the same extractor is open until the sets are committed under docs/probes/baseline-runs/.
- The archived baselineruns branches were not found as branches in this checkout; the committed late set under docs/probes/baseline-runs/20261102/ is the only game.exe minute log at the current code.
- Pair correlation between arms could not be measured: no pinned seed on game.exe.

## Limits

All of the data doc's section 6, plus: the 15 percent margin follows the brief, the spec says 20 percent; area names are the standard level ids from memory (I); one same-code d2d lane against game.exe lanes proves nothing statistically.

## Missing glossary term (for the coordinator)

"Apples to apples" comparison of two arms (same bot commit, class, clvl band, k, seed policy) is used in the fidelity docs without a glossary entry; proposed one-line entry: *comparable pair: two runs that differ only in the backend (same bot code, character history, clvl band, clock factor k and seed policy)*.

## Shared-file lines for the merger

See openspec/changes/botab/merge-lines.txt.
