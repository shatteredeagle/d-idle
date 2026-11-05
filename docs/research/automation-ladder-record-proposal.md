# Automation ladder: proposed record text (READY TO ACCEPT)

> status: PROPOSAL. Text for the user's own grill session or an explicit coordinator brief that names the guard files. Nothing here is written to `docs/decisions.md`, `docs/adr/` or `GLOSSARY.md`. Source of the decisions: the user, 2026-11-05, recorded in [automation-ladder.md](automation-ladder.md) section 7.

Tags: **V** = read in this repo (file cited), **D** = the user's stated decision, **U** = my judgment.

## 1. decisions.md

### 1a. Index line (style of the neighbouring entries, V docs/decisions.md lines 60 to 66)

Add under the ADR index, after the ADR 0015 line:

```
- [ADR 0016](adr/0016-automation-ladder.md) (2026-11-05, user): d.idle is an incremental Diablo II on d2d; no direct hero control; automation tiers T0 to T5 unlock by learn by doing (same answer N times in a row, N by risk, revocable) and are raised by sacrificing items valued by one worth rule at an altar; fresh clvl 1 start; learned rules survive hardcore rebirth; full ladder inside Act 1 Normal first, other acts as expansions; Gate 2/3 are the T4 acceptance, T5 gets its own gate, Gate A stays separate; amends ADR 0015 "done means" and un-parks the idle layer
```

### 1b. Current-truth bullets (U, only if the user wants the top block touched)

- Replace in "Deferred or parked": remove "idle layer" (it becomes the ladder; the 24 h Act 1 clear stays the fidelity acceptance, not the product bar).
- Add: `**Product shape** (ADR 0016): the ladder T0 to T5 (docs/research/automation-ladder.md); backlog work is ranked by the tier it serves; Gate 2/3 are the T4 acceptance.`

## 2. ADR 0016 draft

File name: `docs/adr/0016-automation-ladder.md` (format of V docs/adr/0015-goals-acceptance-on-game-exe.md: front matter, Context and Problem Statement, Decision Outcome, Consequences).

```markdown
---
status: accepted
date: 2026-11-05
decision-makers: user
---
# 0016: Automation as progression: the tier ladder T0 to T5

## Context and Problem Statement

ADR 0015 fixed the engineering goals (acceptance on game.exe, stability on d2d) and parked the idle layer. The product discussion of 2026-11-05 compared four shapes: (1) watch the hero, (2) job runner, (3) an abstract simulator calibrated by real runs, (4) automation as progression. The user ranked 4 first, 1 second, 2 third and rejected 3, so the real engine (d2d) keeps running the game. The current engineering (leveling, Act 1 clear gates) sits in the middle of that product, and goal-driven farming is untouched. [research/automation-ladder.md](../research/automation-ladder.md) has the tier-by-tier design.

## Decision Outcome

1. The player never controls the hero directly (no click-to-move, no casting). "Manual" means task orders and answering the hero's questions.
2. Automation is progression: six tiers, T0 Order-taker, T1 Tidy, T2 Runner, T3 Planner, T4 Questor, T5 Farmer. Each tier needs the one below (a linear ladder).
3. A tier unlocks by learn by doing. At a decision point the hero stops, rests safely in town and asks. The same answer in the same context N times in a row teaches a rule; a different answer resets the count. N depends on risk: 2 safe, 3 normal, 5 risky. Rules are listed and revocable. A sacrifice can skip one repetition. N is tunable data. A new act or difficulty is a new context.
4. Quality is raised by sacrificing items at an altar (a custom d2d object). Anything can be sacrificed; one worth rule values items (runes and perfect gems highest, gold and junk early, gems mid, runes late). The farm goal reserves the runes its runeword needs. No separate currency.
5. Every run starts from a fresh clvl 1 character; there is no premade hero. Leveling to farming strength is part of the game.
6. Failure shows as needs input (a question card, hero safe in town), incident (a plain-language journal report) and a visible uptime stat.
7. Hardcore death is rebirth: the character resets, learned rules and altar quality stay.
8. Tiers are capabilities, acts are content. The full ladder T0 to T5 is built inside Act 1 Normal first (Countess for the early rune goals); Acts 2 to 5 follow as expansions.
9. Gate 2 and Gate 3 are the T4 acceptance (Act 1 clear). T5 gets its own farm-goal drill and gate. Gate A (fidelity acceptance) stays as ADR 0015 defines it.
10. The end state is a fully automated farming bot driven by a goal ("farm X runes") that the player watches.

## Consequences

* Amends ADR 0015 item 1 and the "Done means" bullet of decisions.md: the fresh uncheated 24 h Act 1 clear plus the A:B verdict stays as the fidelity acceptance (Gate A, G-A, G-D), but it is the T4 acceptance of the product, not the product bar. Un-parks "idle layer" (ADR 0015 item 10) as this ladder; stream-grid, d2dmesh and the container stay deferred.
* Backlog work is ranked by the tier it serves. T0/T1 reliability work continues only where it blocks a soak. The open gaps are the decision-point store, the altar object, the question-card UI and the T5 planner ([decision-point-store-spec.md](../research/decision-point-store-spec.md) is the first of them).
* Soaks and gates stay comparable by running with a fixed, scripted answer log; "writes after start: 0" is unchanged. Hardcore rebirth adds a test state (rules present, character fresh).
* The card cap of 12 (ADR 0015 item 11) still applies: new tier work needs a slot or replaces a card.
* Not decided here: the final decision-point catalogue, the watch view, the altar's numeric costs.
```

## 3. GLOSSARY.md terms (style of V GLOSSARY.md: bold term, definition line, _Avoid_ or _Not to be confused with_ line)

Add under a new heading `### Automation ladder` in the Product section:

```
**Automation tier**:
One of six hero capabilities, T0 Order-taker to T5 Farmer, unlocked in order. Tiers are capabilities; acts are content. Lives in `docs/research/automation-ladder.md`.
_Avoid_: "level" (that is the character level).

**Learn by doing**:
Unlocking a rule by answering: the same answer in the same context N times in a row (N by risk: 2 safe, 3 normal, 5 risky) teaches it; a different answer resets the count; learned rules are revocable. Spec: `docs/research/decision-point-store-spec.md`.
_Avoid_: "training" or "ML"; it is a deterministic count.

**Sacrifice** (and **altar**):
Spending an item (gold, gem, rune, junk) at the altar, a custom d2d object, to raise the quality of a tier. Items are valued by one worth rule. The altar is the object; a sacrifice is the act.
_Not to be confused with_: selling to a vendor.

**Needs input**:
The state in which the hero waits, safe in town, for the player to answer a decision point (a question card). Not a failure. The older bot log line `leveling: blocked: needs input` (V internal/bot/task.go) is the first instance of the word.
_Not to be confused with_: incident.

**Incident**:
A failure the hero could not resolve, reported in plain language in the journal. Rare at high tiers.

**Uptime**:
The share of wall time the hero was acting rather than waiting for input or in an incident.

**Order**:
A task the player gives the hero ("go to the Blood Moor", "run Countess x20", "farm 3 Ist"). The only manual control; not a click.
_Avoid_: "command" (the d2d control socket has commands).
```
