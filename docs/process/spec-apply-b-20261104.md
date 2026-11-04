# Spec apply B, 2026-11-04

Applies docs/process/spec-ledger-20261104.md and the user's decisions (2026-11-04) to the base specs of: merc-policy, backend-contract, town-basics, lab-rules, dashboard-pub, early-game-policy, bot-loop, combat-loop, licence-policy, process-hygiene, public-split, and the new act1-campaign. Only `openspec/specs/<capability>/spec.md` and this log were edited. Every changed requirement carries `Reconciled 2026-11-04 (source: ...)` in its Source line; every spec has a status line under Purpose and, where needed, a closing "Pending deltas (not in force)" section.

Tags: V = read in the repo or run (`tools/openspec.sh validate --all --strict`: 40 passed, 0 failed), U = unverified. No live run was made; nothing here is verified against the game.

Strict validate warns above about 500 characters per requirement, so long folded requirements were trimmed in wording (no number or rule changed) or split (act1-campaign record, merc-policy direction).

## lab-rules
| Folded | Left | Cleaned | Open |
|---|---|---|---|
| memrules: Memory access, Crutch rule, Cheated runs and the writes line | none | status line, Pending section (none) | none |

## dashboard-pub
| Folded | Left | Cleaned | Open |
|---|---|---|---|
| mapitem: stream map contiguous across levels; a crossing does not blank or rebuild the map | stream-grid targets `stream-scene`, not this spec | status line | none |

## town-basics
| Folded | Left | Cleaned | Open |
|---|---|---|---|
| goldfloor: Trips must buy something, No scroll rebuy while poor, Early mana cap (knobs OFF by default, stated in the text); m1-trips: Sell trips only when worth it, Gold trip above the keep line, Pack-full threshold | none | status line; "G2" in the sell-trips Source is the basics-audit label, kept and annotated | none |

## backend-contract
| Folded | Left | Cleaned | Open |
|---|---|---|---|
| fix-obj (2), fix-portal2 (3), waits-audit (2) | d2d-default-backend rows (Original game acceptance MODIFIED, Game.exe is frozen, Leaving d2d is break glass) not folded: card dropped by the user; container-deferred delta targets soak-gates, not folded | Purpose: d2d is the product, Game.exe the oracle (decision 4). "Original game remains the acceptance target" rewritten as "Game.exe is the fidelity oracle, d2d is the product" (acceptance: Act 1 cleared 24 h on d2d AND A:B EQUIVALENT). "Leaving d2d is the user's decision": if the G-D verdict is DIFFERENT and the gap needs an engine rewrite the user decides, never automatic | OPEN QUESTION: confirm break-glass wording (old 16 agent-hour and 20 percent trigger replaced by the G-D verdict). docs/decisions.md Current truth still names game.exe the acceptance target; an ADR 0015 amendment (branch agent/adramend) must land to match |

## merc-policy
| Folded | Left | Cleaned | Open |
|---|---|---|---|
| mercpolicy first: Spending order, Revive price is read, Act 2 swap, Gold farm, Merc policy on by default (ADDED); Blood Raven kit gate, Revive and stop rule, Route guard (Monastery road 14 with or without merc, Forgotten Tower 16 / 14), Merc gear and potions (potion line 50) (MODIFIED); then m3-readiness-model: Readiness model | m3-merc (4 rows) stays pending | Purpose: `merc.enabled` default 1. Route guard and Andariel floor labelled "Current implementation (interim)", numbers unchanged (road 14; Forgotten Tower 16 no merc, 14 merc; Andariel 17 no merc, 15 merc). New "Route guard floors as plan data (direction)" and "starting numbers (direction, U)": per-zone floors (`minClvl`, `minMaxLife`, `minHealPots`), raise-only death loop with exposure minimum, merc effect via gold, stash gold before a guarded zone; SME numbers 13 / 15 / Forgotten Tower 16 / Andariel 18 with life 250 and antidote, all U, not implemented | The exposure minimum has no number yet (not invented). m3-merc re-check: its "Revive and stop rule" (reviveGoldMult 2 on gold minus reserve) conflicts with the folded text (1 plus the kit reserve), and its "Revive too expensive" scenario assumes the old rule; the card needs a rebase. Because m3-merc's MODIFIED block must contain every current scenario for strict validate, the folded "Cap" and "Stop state" scenarios sit under "Spending order" (named "(Revive and stop rule)"). Readiness model scenario "Floor safety net" changed from the Monastery road (no-merc floor 16 in the old text, now 14) to the Forgotten Tower (16 no merc) so it stays true; U whether that matches the card's intent. The decision 5 numbers (13, 15, 16, 18, 250) come from the user decision, not from a delta |

## act1-campaign (new base spec)
| Folded | Left | Cleaned | Open |
|---|---|---|---|
| m3-act1chain: Act 1 quest order, Quest evidence on d2d, both in the SME wording with the SME scenarios; new Act 1 cleared record (the 14 rows with tiers as tables, split into four requirements and a lead rule so each stays under 500 characters, plus two "does not count" requirements); all tagged "decided, not verified live" | m3-act1run (Unattended Act 1 clear) and m3-andariel (Andariel fight rules) in Pending deltas | The old T3-re-check wording of Quest evidence is replaced by the SME wording | OPEN QUESTION word 6 vs word 7 (hops.go vs dialog.go); the R-B / I-A shorthand reading (required for G-B, informational for G-A) is U. "Known gaps" section lists the SME's four proposed additions (no card) |

## early-game-policy, bot-loop, combat-loop, licence-policy, process-hygiene, public-split
| Capability | Folded | Left | Cleaned | Open |
|---|---|---|---|---|
| early-game-policy | none (all in-force deltas applied) | m2-roam, m2-tripwp pending | status line | none |
| bot-loop | none | none | status line | none |
| combat-loop | none | none | status line | none |
| licence-policy | none | none | status line | none |
| process-hygiene | none | none | status line | none |
| public-split | none | none | status line | none |

The "G1 attempts 7 to 14" citation in early-game-policy is a document name (docs/g1-log.md), not a gate, and was kept.

## Counts
Folded: lab-rules 3, dashboard-pub 2, town-basics 6, backend-contract 7 (plus 2 rewritten), merc-policy 12 (5 ADDED, 4 MODIFIED, readiness model, plus 2 direction requirements written from decision 5), act1-campaign 2 (plus the cleared record). Left pending: backend-contract 3 rows (dropped card), merc-policy 4 (m3-merc), act1-campaign 2, early-game-policy 2. Open questions: 3 (break-glass wording, word 6 vs 7, m3-merc rebase) plus the unconfirmed shorthand and exposure minimum.

## Process note
The first commit contains lab-rules and dashboard-pub together (the one exception to one commit per capability); two later commits only shorten text so strict validate passes.
