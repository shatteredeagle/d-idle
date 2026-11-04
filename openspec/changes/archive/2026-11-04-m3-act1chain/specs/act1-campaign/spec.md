# Act 1 campaign specification

## Purpose
What an unattended Normal Act 1 run on the d2d backend must achieve for a fresh character: the quest order, the evidence a quest is done, the Andariel fight rules and the time budget to reach Act 2. Source: docs/roadmap.md milestone M3, docs/sme/play-act-route.md, d2-sme consultation 2026-10-31 (EXP unless tagged).

## ADDED Requirements

### Requirement: Act 1 quest order
A fresh character SHALL do the Den of Evil first (early-game-policy), Blood Raven at the kit gate and claim the free Rogue (merc-policy), MAY skip the Search for Cain and the Forgotten Tower, and MUST kill Andariel and talk to Warriv to enter Act 2; no other quest is required for the hop. Source: d2-sme 2026-10-31 (SRC docs/sme/play-act-route.md: Cain skippable, Andariel then Warriv); card m3-act1chain.

#### Scenario: Forgotten Tower blocked by the guard
- **WHEN** the route guard blocks the tower and Cain is not done
- **THEN** the campaign still proceeds to Andariel when the floor and kit hold

### Requirement: Quest evidence on d2d
On the d2d backend a quest SHALL be marked done only on save-file (T1) or memory (T2) evidence; proxy evidence (an item in the pack, a kill event) MAY advance the step but MUST NOT skip work, and a T3-only completion is re-checked at the next save point. Source: docs/progression/quests.md section 1 (rules), card m3-act1chain (V on d2d once run).

#### Scenario: Blood Raven dead, no save yet
- **WHEN** the kill event fired but the save file has not been written
- **THEN** the campaign moves to the claim step but the quest is not recorded done until the save word shows it
