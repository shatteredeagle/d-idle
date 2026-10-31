# Mini-grill: small and medium decisions asked inline

Decided by the user 2026-10-30. Tags: V = read in a file, U = our judgement. Big decisions keep the full grill ([grill-workflow.md](grill-workflow.md)).

## 1. Mini or full grill

| | mini-grill (inline) | full grill (own pane) |
|---|---|---|
| size | one decision, answerable from options we can list | open-ended, we cannot list the options yet |
| reversibility | cheap to reverse (a knob, a threshold, a timer, a doc rule) | expensive (backend, architecture, a contract many cards depend on) |
| unknowns | at most 3 to 4 | more, or the unknowns are the question |
| where | main chat, `AskUserQuestion` (clickable) | `tools/grill.sh`, a head box pane next to the main session |
| record | `mini-rounds.md` | ADR draft + decisions.md line |

Rule of thumb (U): if you can write the options and a recommendation on one page, mini. If you cannot, or a wrong answer costs days, full. After 3 mini rounds with still-unclear answers, escalate to a full grill.

## 2. Round format

The coordinator prepares first (one page, template in section 6): facts with their sources and V/S/U tags (from the prebrief or `verified-claims.md`), the options, the recommendation. Then one `AskUserQuestion` call per round:
- at most 4 questions per round, at most 3 rounds per mini-grill;
- header at most 12 characters; 2 to 4 options per question; the recommended option FIRST, its label ending in "(Recommended)"; previews allowed for options that are code or config;
- one decision per question; facts the model can look up are never asked.

## 3. Recording without filling the main session

The coordinator does not transcribe. After each round it hands the raw answers (and any free-text notes the user typed) to a subagent (`d2-worker` or `general-purpose`, Haiku is enough), which appends to `docs/grills/<n>-<slug>/mini-rounds.md` (create with the prebrief header if missing; for a decision with no grill number use `docs/grills/mini/<date>-<slug>.md`):

```
## Round 1 - 2026-10-30
Q1 <header>: <question>
  options: A (Recommended) ..., B ..., C ...
  answer: A
  note: <user free text, verbatim, or none>
```

The subagent commits on its branch and replies with one line (file, round number). Nothing else comes back into the main chat.

## 4. From answers to a record

Agents never write ADRs, `GLOSSARY.md` or `docs/decisions.md` on their own (AGENTS.md guard rule). Path:
1. The user says "record it".
2. The coordinator's brief to a worker names the exact files (`docs/decisions.md` one line, and an ADR only if the decision matters beyond its card), pointing at `mini-rounds.md` as the source.
3. If an answer changes a numeric gate or a contract in `openspec/specs/`, it goes through a change card with a spec delta, never an edit of the spec.
4. A decision the user wanted but did not say to record stays in `mini-rounds.md` only; it is not binding until recorded.

## 5. Worked example (2026-10-30)

SOFTGUARD iowait limit (the soak-gates spec holds the number; the facts below are illustrative of the format, U, not copied from the SOFTGUARD report):
- Facts: one line per fact with its source, e.g. "limit value: <spec path> (V)", "runs that crossed it: <report> (S)".
- Q1 "iowait": "What iowait limit aborts a soak?" options: keep the spec value (Recommended) / exempt the first 5 minutes / raise the limit / drop the guard.
- Answer lands as a numeric-gate change, so step 4.3 applies: change card + delta spec, not an edit of the spec.

Board-refresh timer (U on the exact numbers, the example shows the format):
- Q2 "Refresh": "How often does the status board refresh the generated STATUS block?" options: on every agentstatus call (Recommended) / every 5 minutes by timer / manual only.
- A pure tooling knob: recorded in `mini-rounds.md`, and a worker brief names the timer file; no ADR.

Both fit one round of 2 questions.

## 6. Pre-brief template, mini variant (one page)

```
# Mini pre-brief: <slug>
status: draft | asked | recorded
## Decision
<one sentence, ends with a question mark>
## Facts (with source and tag)
- <fact> (V, <path or command>)
## Options
A. <option> (Recommended): <one line why>
B. ...
## Recommendation
<A, and the one fact that decides it>
## What each answer unlocks
- A: <card or change that follows>
- B: ...
## Escalate to a full grill if
<the condition>
```

Save as `docs/grills/<n>-<slug>/mini-prebrief.md` only if the decision is kept; otherwise it lives in the coordinator's brief to the recording subagent.
