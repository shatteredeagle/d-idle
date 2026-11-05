---
id: m1-liveproof
goal: G-B
status: proposed
enforced_by: prose
---
# Proposed contract text: m1-liveproof (primitives)

> status: PROPOSED, not in force. Converted at OpenSpec removal stage 3 from the delta spec of card `m1-liveproof` (git tag `openspec-final`, path `openspec/changes/m1-liveproof/specs/primitives/spec.md`). When the work lands, the branch that implements it folds this text into a NEW contract `docs/contracts/primitives.md` (no main file yet) and deletes this file. Backlog task: title starts with `m1-liveproof:`.

# Primitives Specification

## Purpose
The basic game actions the bot depends on (belt, pickup, portals, vendors, movement, combat casts, build and lifecycle), each with a definition of correct that a scripted live drill on a hidden d2d can pass or fail. Source: docs/primitives.md, card m1-liveproof.

## Requirements (new)

### Requirement: Primitive drills
Each primitive in docs/primitives.md SHALL have a drill that sets up through the logged cheat tool with a reason per step on a throwaway character on a hidden instance, reads the end state through the StateReader, and reports PASS, FAIL or BLOCKED against the pass condition in that row. A drill never reads the bot's own log lines as proof.

#### Scenario: Belt fill drill
- **WHEN** the belt is empty and the pack holds 6 healing and 4 mana potions
- **THEN** the drill passes only if the belt grid shows the planned potions and the pack holds none of them

#### Scenario: Missing verb
- **WHEN** a drill needs a verb the d2d binary lacks
- **THEN** the verdict is BLOCKED and names the verb

### Requirement: Clean UI around every drill
The harness SHALL check before and after each drill that no panel, dialog or popup is open and the cursor holds no item, and SHALL stop the drill group when the check fails.

#### Scenario: Cursor holds an item
- **WHEN** a drill ends with an item on the cursor
- **THEN** the drill is FAIL and the remaining drills of the group do not run

### Requirement: High criticality needs a live pass
A primitive of criticality S or A SHALL have a drill PASS recorded in docs/primitives.md, or a FAIL with a named fix card, before policy changes that depend on it merge.

#### Scenario: Unproven belt primitive
- **WHEN** P1 has no live drill PASS
- **THEN** the coordinator does not merge a policy change that depends on the belt
