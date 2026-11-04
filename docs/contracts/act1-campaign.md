---
id: act1-campaign
goal: G-B
status: in-force
enforced_by:
  - internal/quest
  - internal/d2dsoakstat/act1check.go
---
# act1-campaign contract

## Purpose
What an unattended Normal Act 1 run on the d2d backend must achieve for a fresh character: the quest order, the evidence a quest is done, and the record that says Act 1 is cleared. Source: docs/roadmap.md milestone M3 (goal G-B), docs/sme/play-act-route.md, d2-sme consultations 2026-10-31 and 2026-11-04 (S, EXP unless tagged). Evidence tiers used below: T1 save file, T2 game memory (dump), T3 proxy (log line, event).

> status: reconciled 2026-11-04 against docs/process/spec-ledger-20261104.md; pending deltas: m3-act1run, m3-andariel

All requirements in this spec are DECIDED by the user (2026-11-04, option 1 plus the SME criteria) and NOT VERIFIED LIVE: no uncheated run has exercised them on d2d.

## Requirements

### Requirement: Act 1 quest order
A fresh Normal character SHALL clear the Den of Evil (skill point spent), kill Blood Raven and claim the free Rogue, kill Andariel and talk to Warriv. The Search for Cain, the Forgotten Tower and Tools of the Trade MAY be skipped and are then recorded as skipped. No other quest is required for the hop. Decided, not verified live. Source: user decision 2026-11-04 (d2-sme wording); docs/sme/play-act-route.md; card m3-act1chain. Reconciled 2026-11-04 (source: m3-act1chain, decision 8).

#### Scenario: Den word open after a sweep
- **WHEN** the Den word is still open after the sweep (leftover monster, word 0x001c, or the dialog left open)
- **THEN** no claim is made and the bot hunts the leftover

#### Scenario: Rogue claim not seen
- **WHEN** the claim step ran but `merc: claim verified` was not logged
- **THEN** the claim step retries and the run records the failure

#### Scenario: Forgotten Tower blocked by the guard
- **WHEN** the route guard blocks the Forgotten Tower and Cain is not done
- **THEN** the campaign still proceeds to Andariel when the floor and kit hold, and the skipped quests are recorded skipped, never done

### Requirement: Quest evidence on d2d
A quest is done only when its raw word reads Done from dump quests (T2) or the .d2s (T1). Kills, items, area changes and menu rows MAY advance a step but SHALL NOT mark a quest or Act 2 done. An unknown source means unknown, not not-done. Decided, not verified live. Source: user decision 2026-11-04 (d2-sme wording); docs/progression/quests.md section 1; card m3-act1chain. Reconciled 2026-11-04 (source: m3-act1chain, decision 8).

#### Scenario: Blood Raven dead, no save yet
- **WHEN** the kill event fired but the quest word does not yet read Done
- **THEN** the campaign moves to the claim step but the quest is not recorded done

#### Scenario: Stock d2d without quest data
- **WHEN** the source returns no progress data (ErrNoProgress)
- **THEN** the quest state is unknown, not not-done

> OPEN QUESTION: word 6 or word 7. quest/hops.go gates the Warriv hop on quest word 6 while quest/dialog.go gates "Go East" on word 7 (act1_done); which word the real game sets at the Andariel kill is UNVERIFIED. The first uncheated run must log both words at the first real kill. Until then record 6 = 0x0001 alone counts as enough on d2d. (S, SME 2026-11-04.)

### Requirement: Act 1 cleared record
An Act 1 run is cleared only when its record holds all 14 rows of the four tables below (the 14-row checklist, split by topic); every row SHALL name its tier. Class R is required, I informational. R-B/I-A reads: required for goal G-B, informational for G-A (U: this reading of the SME shorthand is not confirmed). Decided, not verified live. Source: user decision 2026-11-04 (d2-sme checklist). Reconciled 2026-11-04 (source: decision 8).

#### Scenario: Cleared on a clean run
- **WHEN** every R row of the four tables holds for the goal in question
- **THEN** the record says Act 1 cleared

#### Scenario: Skipped quest
- **WHEN** Cain was skipped
- **THEN** row 11 says skipped; listing it as cleared is a defect

### Requirement: Act 1 cleared record, identity rows
The record SHALL hold these rows. Decided, not verified live.

| # | Class | Row | Tier |
|---|---|---|---|
| 1 | R | Fresh character created in the run; ledger has no entry for the name; `d2cheat ledger --since-run` prints writes after start: 0 | T3 |
| 14 | R | Every row names its tier (T1 save file, T2 memory, T3 proxy) | n/a |

#### Scenario: Ledger entry exists
- **WHEN** the ledger has an entry for the character name
- **THEN** row 1 fails

### Requirement: Act 1 cleared record, kill and arrival rows
The record SHALL hold these rows. Decided, not verified live.

| # | Class | Row | Tier |
|---|---|---|---|
| 4 | R | Andariel dead: `boss run boss=Andariel killed=true` AND word 6 Done in dump; a final T1 read agrees | T3, T2, T1 |
| 5 | R | Act 2 entered: dump area 40; after Save and Exit .d2s byte 0xA8 = 0x81 | T2, T1 |
| 6 | R | Alive at area 40, deaths inside the death rule | T2, T3 |
| 9 | R | No unspent points, no quest item in pack (Malus, bark, Cairn key) | T2 |

#### Scenario: Kill log without the word
- **WHEN** the log says `killed=true` but word 6 is not Done in dump quests
- **THEN** row 4 fails

### Requirement: Act 1 cleared record, quest and merc rows
The record SHALL hold these rows. Decided, not verified live.

| # | Class | Row | Tier |
|---|---|---|---|
| 2 | R-B / I-A | Den word 1 Done; no unspent skill points; no pending bit | T2 |
| 3 | R-B / I-A | Word 2 Done; dump merc is the Rogue; `merc: claim verified`; no Kashya gold | T2, T3 |
| 7 | R-A / I-B | After Save and Exit and rejoin words 1, 2, 6 Done (mask 0x1000), Rogue, same map id (--no-save has no T1) | T1 |

#### Scenario: Rogue missing from the dump
- **WHEN** word 2 is Done but the dump merc is not the Rogue
- **THEN** row 3 fails

### Requirement: Act 1 cleared record, time and gate rows
The record SHALL hold these rows. Decided, not verified live.

| # | Class | Row | Tier |
|---|---|---|---|
| 8 | R | Milestone times in virtual hours (Andariel dead, Act 2 entered; ADR 0015 item 9) | T3 |
| 10 | I | `boss: andariel gate pass clvl=N kit=ok` at the floor or above | T3 |
| 11 | I | Cain, Forgotten Tower, Tools each Done or skipped | T2 |
| 12 | I | Merc alive 70 percent of field time after the claim | T3 |
| 13 | I | Lut Gholein waypoint active; Act 2 movie handled (U) | T2 |

#### Scenario: No milestone times
- **WHEN** the record has no virtual-hour times for Andariel dead and Act 2 entered
- **THEN** row 8 fails

### Requirement: Evidence that never counts for the Act 1 record
The Act 1 cleared record SHALL NOT count any ledger entry for the character, a debug quest set, a debug goto, a word of 0x8001 read as a legitimate Done, a stale .d2s, or stock d2d; and SHALL NOT cite the debug quest list numbers, which differ from save words from Act 2 on. Decided, not verified live. Source: user decision 2026-11-04 (d2-sme MUST NOT COUNT). Reconciled 2026-11-04 (source: decision 8).

#### Scenario: Cheat-set word
- **WHEN** word 6 reads 0x8001 after a debug quest set
- **THEN** the run is not counted

### Requirement: Proxy and skip evidence does not count for the Act 1 record
The Act 1 cleared record SHALL NOT count proxy evidence alone (the proxyStage Andariel Done from Act 2 or later labelled EvMemory, a kill event, an item in the pack, a Warriv menu row), an area change by waypoint or portal without word 6, Cain, the Forgotten Tower or Tools skipped but listed cleared, or bit 12 differences read as changes. Decided, not verified live. Source: user decision 2026-11-04 (d2-sme MUST NOT COUNT). Reconciled 2026-11-04 (source: decision 8).

#### Scenario: Area change without the word
- **WHEN** the character reaches area 40 by portal and word 6 is not Done
- **THEN** Act 2 entered is not recorded

## Known gaps (not requirements)
S, from the d2-sme review 2026-11-04. internal/d2dsoakstat has no milestone log line and no Act 1 invariant (only "Quest done" at invariants.go:292); the claim and hop live proofs D8.c and D8.w used cheat-set records, so no uncheated run exercised them. Minimal additions proposed (U, no card yet): a `milestone name= word= clvl= virtual=` line on each Act 1 word change; an act1check section in d2dsoakstat reading the final dump quests, merc and area; logging words 6 and 7 at the first real kill; a Save and Exit plus .d2s read at the end of G-A runs.

## Pending deltas (not in force)
- m3-act1run (open card, deferred): ADDED "Unattended Act 1 clear".
- m3-andariel (open card, deferred; M3 maps to goal G-B): ADDED "Andariel fight rules".
