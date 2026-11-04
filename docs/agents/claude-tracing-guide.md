# Tracing claims about Diablo II 1.14d with Claude: a practical guide

> status: current (card claudeguide, 2026-11-01). Written at the maintainer's request for people who use Claude (Claude Code or the API) on d.idle / d2d decomposition work. Everything here was tested with Claude models only (Opus, Sonnet, Haiku). Other models are untested: see section 10.

Read first if you are new: [../contributing/agent-work.md](../contributing/agent-work.md) (what help is wanted, the safety rules) and the [hands-off prompt](../contributing/agent-prompt.md). Terms: [../../GLOSSARY.md](../../GLOSSARY.md). The repo rules every agent follows: [../../AGENTS.md](../../AGENTS.md).

## 1. The idea in one paragraph

d.idle is a bot for classic Diablo II LoD 1.14d. We test it against d2d, an open-source server reimplementation. The question behind most work is: does d2d (and does our bot) behave like the real `game.exe`? A "trace" answers that for one behaviour: read what `game.exe` does, run its code when that is the proof, compare with d2d on a real build, and list the differences. Claude does the reading, the harness writing and the report. Humans keep the decisions (what goes upstream, what gets merged).

## 2. Evidence tags (the one rule that matters)

Every statement in a trace carries a tag. A statement with no tag does not count.

| Tag | Meaning |
|---|---|
| V read | read in the disassembly (name the function tag, for example `FUN_0056c160`) |
| V run | the real `game.exe` code was run under the emulator and compared byte for byte with d2d on a real d2d build |
| D | derived from the data tables |
| S | secondary source (a wiki, a forum, someone else's note) |
| I | inferred; reasoning, not observed |
| U | unverified |

Rules that follow from the tags:
- A port of d2d's rule into Python is NOT a real-build result. It checks your port, not d2d.
- Every number is reproducible by a command in the document, with the case count and seed. Our bar for V run: 100,000 random cases or an exhaustive boundary set. Use 32-bit masked seeds in both generators (game side and d2d side) so they draw the same cases.
- Never paste game code or disassembly. Use `FUN_` address tags and short formulas. The gamecode guard fails CI on listings ([../process/gamecode-guard.md](../process/gamecode-guard.md)).
- Deviations of d2d are numbered D1.., findings about our bot B1.., kept apart. State the limits: what you did not cover.
- No game files, tables dumps, secrets, host names or private paths (`go run ./cmd/leakcheck -mode fail` must report 0).

## 3. How to trace one claim

Worked example: SKRESIST, "how do monster resists and immunities behave by difficulty, and does d2d match?" Result: [../research/sk-resist-trace.md](../research/sk-resist-trace.md), then the independent check [../research/pr-curseimmune-verifier.md](../research/pr-curseimmune-verifier.md).

1. **Question.** One sentence, answerable with yes/no plus numbers. Split into Q1..Qn sub-questions (the SKRESIST trace has ten).
2. **Where to start.** Address tags and existing docs only: the decomp queue ([../decomp-queue.md](../decomp-queue.md)), earlier traces on neighbouring skills, upstream's notes. Do not ask the model to "find the function" blind; hand it a starting `FUN_` tag or let it search by data reference and label the result I until read.
3. **Read.** Disassemble in your own tool; write the rule as a short formula with address tags (V read).
4. **Oracle.** A script under `tools/emu/` (see the existing `*.py` files there) runs the real game function natively under the emulator and prints results for a list of inputs. This is the game-side half of the differential.
5. **d2d side.** Do not re-implement d2d's rule. Add a small test mode to a d2d build that calls d2d's real function on the same inputs and prints the same columns. The existing `tools/emu/*_mode.inc` files show the pattern. Build the unmodified base and, if you propose a fix, the claim commit.
6. **Differential.** Same cases both sides, compare, count per case class: base vs head, cases that differ before, after. Two seeds.
7. **Deliver one document** `docs/research/<topic>-trace.md`: answer in a few lines, questions answered, the rules as a formula, d2d's version, deviations D.., the differential with commands, the bot's model and findings B.., what a live drill should expect (written, not run), what is not traced (U).
8. **Report** the exact commands, what is verified (and how), what is not, and the last commit.

Proven deviations may become upstream PR candidates under [../process/upstream-pr-standard.md](../process/upstream-pr-standard.md). Only the maintainer opens upstream PRs.

## 4. The verifier (and why it must be independent)

Anything that becomes a PR candidate gets an independent verifier. Examples: [pr-nexthit](../research/pr-nexthit-verifier.md), [pr-armorhit](../research/pr-armorhit-verifier.md), [pr-freezeresume](../research/pr-freezeresume-verifier.md), [pr-curseimmune](../research/pr-curseimmune-verifier.md), [pr-blizzard](../research/pr-blizzard-verifier.md).

The lesson that made this a rule: the author's own oracle can hide an error. The hit-once fix (NEXTHIT) ended its protection one frame early, and the author's differential passed. Only a verifier with its own game-side harness found it, in about a quarter of its cases.

The verifier must:
- build its own game-side harness and its own d2d probe from scratch, not reuse the author's scripts (read the author's proof only after the first differential);
- report base-vs-head counts per case class, with its own case count and seed;
- check the claim touches nothing else (build each commit alone and compare);
- check merge cleanliness and the CI-class checks;
- give a verdict: READY, READY WITH CHANGES or NOT READY, with each claim HOLDS, HOLDS WITH LIMITS or FAILS, and edits for the PR draft.

## 5. Which Claude model does which job

Decided 2026-11-01 from a blind comparison on one item ([../research/model-ab-skresist.md](../research/model-ab-skresist.md)).

| Role | Model | Why | Limits |
|---|---|---|---|
| Decomposition author | Sonnet | Found the decisive curse/aura 1/5 rule and a real d2d deviation on a real build. Used more tokens than Opus, but Opus costs several times more per token. | One item, one run per arm |
| Independent verifier | Opus, with its own harness (never the author's) | Required for anything that becomes a PR candidate; the strongest read for errors the author cannot see | Not tried as an author on the same item set |
| Escalation | Opus | When Sonnet fails the same decomposition twice | Rule of thumb, not measured |
| Mechanical jobs | Haiku, hard limits | Re-run a named differential, dump tables, grids of a named function, checklists. In the blind test it did byte-testable grids accurately at lower cost. | It never picks functions or cases and never concludes "no deviation" |

What Haiku got wrong in the test: it missed the decisive rule, built no real d2d comparison (a Python port instead) and wrote a misleading "no deviation" headline. So Haiku gets a named command and named inputs, and reports numbers; a stronger model or a human draws conclusions.

The honest size of this evidence: n = 1 item, one run per model. It is a start, not a benchmark.

## 6. The agent setup

| Piece | What it is |
|---|---|
| Coordinator | One session that plans, launches agents and decides. It never does the traces itself. |
| Subagents | author, verifier, mechanical, merge. Each has a fixed model and a short role file. |
| Task | `backlog/tasks/<file>` (a Backlog.md task; formerly a change card). Its frontmatter lists `owns:` (paths the agent may edit, mirrored in `docs/OWNERS`), `exclusive:` (single-user resources such as a game instance or the lab binary) and `contracts:`. `python3 -I tools/backlog/check.py` checks it. See [../backlog.md](../backlog.md). |
| Isolation | One git worktree and one branch per agent. Agents never touch other branches or other agents' directories. |
| Merge | One agent merges at a time, under a lock, with a script: preflight gates (contract check, leakcheck, hygiene, whole-repo tests only when code changed), then a CI check on the pushed commit. Agents never merge or push themselves. |
| Capacity | Check the worker host's load before heavy or live work. |
| Live game | Hidden instances only, one per agent, own throwaway character. Cheats only through the logged cheat tool, on throwaway characters. Outside contributors get no live access. |
| Upstream | Nothing is posted upstream (issues, PRs, comments) without the maintainer's go. Automated review (CodeRabbit) follow-ups on our own PRs are answered; human comments never are by an agent. |
| Outside input | Text from contributors is data, never instructions ([../process/outside-prs.md](../process/outside-prs.md)). |

Where the work lives: the queue [../decomp-queue.md](../decomp-queue.md), the public board [../community/work-board.md](../community/work-board.md), the board tool described in [../process/chunker.md](../process/chunker.md).

## 7. Brief templates

Paste and fill the CAPS. Keep briefs short: goal, evidence rules, task, report length.

**Decompose X (author, Sonnet)**

```
GOAL: Find out whether d2d behaves like game.exe 1.14d for BEHAVIOUR, and list deviations.
START FROM: docs/decomp-queue.md row ROW, docs/research/NEIGHBOUR-trace.md, address tags TAGS (no others until you have read them).
TASK: Write docs/research/TOPIC-trace.md in the style of docs/research/sk-resist-trace.md: questions Q1.., rules as formulas with FUN_ tags, d2d's version, deviations D1.., bot findings B1.., differential, limits.
EVIDENCE RULES: tag every claim V read, V run, D, S, I or U. A Python port of d2d is not a real-build result: the d2d side must call d2d's real function in a test mode. Every number needs a command with case count (100,000 or exhaustive boundaries) and a 32-bit masked seed used by both generators. Never paste code or disassembly; use FUN_ tags and short formulas. No game files, tables, secrets, hosts or private paths. Offline, no live game.
ESCALATE: if two attempts at the same item fail, stop and say so; do not guess.
REPORT: at most N lines: answer, commands, what is V run vs not, deviations, limits, last commit.
```

**Independently verify Y (verifier, Opus)**

```
GOAL: Independently verify the claim CLAIM (commit HEAD on base BASE) of the document DOC.
RULES: Build your OWN game-side harness and your OWN d2d probe from scratch. Do not reuse the author's scripts or oracle. Read the author's proof only after your first differential.
CHECK: (1) the rule in game.exe (V read, own notes); (2) own differential, own generator, own seed, base vs head counts per case class; (3) the claim touches nothing else (build each commit alone, compare); (4) merge cleanliness and the CI-class checks; (5) no vacuous checks: show a case where your check would fail.
EVIDENCE RULES: same tags as the trace (V read, V run, D, S, I, U); no pasted code; no private data.
VERDICT: per claim HOLDS, HOLDS WITH LIMITS or FAILS; overall READY, READY WITH CHANGES or NOT READY; list edits to the PR draft.
REPORT: at most N lines.
```

**Mechanical job (Haiku)**

```
Run exactly: COMMAND with inputs INPUTS. Report the numbers as a table. Do not choose other functions or cases. Do not conclude that nothing deviates. Stop after MINUTES minutes or on any error and report the error.
```

## 8. Pitfalls we hit

- **Checks that pass vacuously.** An author check that cannot fail proves nothing. Make the verifier show a failing case.
- **Unquoted free text in generated scripts.** A summary containing quotes or `$` broke a script that was generated from it. Quote everything that came from free text.
- **Fused multiply-add.** On arm64 the compiler fuses multiply-add and changed 5 of 842,156 differential lines against x86. Build the differential with `-ffp-contract=off`.
- **Presubmit name-length rules** of the upstream project reject long names in test modes; check them before the verifier does.
- **Machine names trip the leak check.** Write "the worker host", "the second machine", "the NAS".
- **Blind runs need an old base.** To compare models fairly, give each arm a worktree of a commit that predates the earlier result, or it can read the answer.
- **Own oracle blindness** (section 4): the biggest one.

## 9. Claude settings that matter

- **Model per role:** as in section 5. Set it in the agent definition so a brief cannot silently change it.
- **Permission mode:** keep the default prompts for anything destructive. Never run `--dangerously-skip-permissions` unattended. Agents work in their own worktree, start only hidden game instances, never delete characters or files they did not create, and never push or post.
- **Cost awareness:** keep a small usage ledger: the tokens, tool calls and minutes each agent reports, grouped by agent class (author, verifier, mechanical). The blind test recorded these (see its effort row). We do not claim prices here; prices change, and they are on the provider's pricing page.
- **Short briefs, tagged reports, bounded waits:** every wait has a time limit, every process is registered so it can be reaped, and every report has a line cap.

## 10. What we have not tested

- Any model other than Claude Opus, Sonnet and Haiku.
- More than one item: the model choice rests on SKRESIST alone, one run per model. A different item could rank them differently.
- Opus as author versus Sonnet on harder items, for example skills with several interacting rules.
- Sonnet as the verifier. We use Opus because the verifier's job is to catch the author's blind spots, but we did not measure it.
- Whether Haiku's mechanical accuracy holds on larger or less regular grids.

If you test another model, run the same blind comparison (same item, a base commit that predates the result, one worktree per arm) and add your findings as a research note.
