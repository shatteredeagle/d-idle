# Grill workflow: deciding with the user in a dedicated session

Decided by the user 2026-10-30. A "grill" is a relentless interview that turns an open decision into an ADR. Tags: V = read in a file (path given), U = our judgement or untested.
Background and the plugin analysis: [matt-pocock-skills-fit.md](matt-pocock-skills-fit.md) section 3. Skills read read-only from
`~/.claude/plugins/cache/claude-plugins-official/mattpocock-skills/1.3.1/skills/` (V): `productivity/grill-me/SKILL.md` is a one-liner that calls the `grilling` skill; `engineering/grill-with-docs/SKILL.md` calls `grilling` plus `domain-modeling` (writes `GLOSSARY.md` and ADRs in the working directory). Both are user-invoked only (`disable-model-invocation: true`, V), so a subagent can never run them and the coordinator should not.

## 0. Which kind of grill

Decided 2026-10-30: small and medium decisions are inline "mini-grills" in the main chat (clickable multiple choice, a subagent records the rounds): see [mini-grill.md](mini-grill.md). Everything below is the FULL grill for big, open-ended, hard-to-reverse decisions. A mini-grill that stays unclear after 3 rounds escalates to a full grill.

## 1. Rules (not optional)

1. One decision per grill. A second decision found on the way becomes a line under "Parked" in the ADR draft, not a question.
2. The user is the interviewee: 20 to 30 minutes, answers by number, says "wrap up" to end. The model asks facts of subagents/docs, never of the user.
3. The coordinator session never hosts a full grill (a 30-round interview floods it and compaction could eat the decisions) and no grill runs unattended.
4. The grill session writes no code, runs no game, posts nothing to GitHub or any chat, pushes only its own grill branch to the forge.
5. It writes exactly two things, in its own worktree: the ADR draft `docs/adr/NNNN-<slug>.md` (status `proposed`) and one line in `docs/decisions.md`. Not `AGENTS.md`, not `GLOSSARY.md`, not `docs/agents/**`, no change cards.
6. Facts come from the prebrief; every fact it cites back to the user carries its V/S/U tag from the prebrief. Anything marked U is a candidate question or a gate, not a premise.
7. The user, not the model, flips the ADR to `accepted` (README rule: user decisions only).

## 2. Who does what

| step | who | output |
|---|---|---|
| 1 pick topic | user or coordinator proposes; listed in [../grills/README.md](../grills/README.md) | topic id `NNNN-slug` |
| 2 prebrief | a `d2-researcher` or `d2-sme` subagent (coordinator launches) | `docs/grills/NNNN-slug/prebrief.md` merged to master |
| 3 grill | the user plus one dedicated Claude Code session | ADR draft + decisions.md line on branch `grill/<n>-<slug>`, pushed |
| 4 convert | coordinator launches a `spec-sme` then `d2-worker` subagent (section 5) | change cards, spec deltas |
| 5 archive | dispatcher | section 6 |

## 3. Start a grill

Primary way, as a pane next to the main session (default when run inside the head box, so the user does not switch windows): `tools/grill.sh <n> <slug>` is the same as `tools/grill.sh --pane <n> <slug>`. It splits the current pane to the right (`the head box pane split --current --direction right --cwd <worktree> --focus`, V live 2026-10-30 with a throwaway pane) and runs `claude --agent grill 'grill topic <n>'` in it via `the head box pane run`. `--here` runs claude in the current terminal instead (default outside the head box), `--print-only` prints what would run. Same checks either way (master clean and not behind, no worktree or branch, prebrief present). (U: the claude session itself was not started by the script test.)

Mechanism (one command, mechanism in [grill-agent.md](grill-agent.md)): `tools/grill.sh <n> <slug>`, e.g. `tools/grill.sh 0001 d2d-primary-backend`. It makes the worktree and starts the preconfigured `grill` agent (Opus, write access limited by a guard hook to the ADR, the decisions line and `outcome.md`) with the first message `grill topic <n>`. At the end it commits, pushes the grill branch to forge and tells you the branch name for the coordinator.

### 3b. Fallback: paste-in text (the user pastes this)

Prerequisite: the prebrief is on `main`/`master`. `<n>` = the topic number (0001), `<slug>` = its slug.

Terminal, once:
```
cd ~/code/d2lod-bot && git worktree add ~/code/d2lod-wt/grill-<n> -b grill/<n>-<slug> master && cd ~/code/d2lod-wt/grill-<n> && claude
```
(A new Claude Code session in that worktree; it is a git worktree of `~/code/d2lod-bot`, so the repo's `AGENTS.md` loads.) Model: Opus is worth it for a decision interview (U); say so in the prompt if you want it.

Then paste into the session (fill `<n>`, `<slug>`, `<title>`):

```
This is a dedicated GRILL session, not a coordinator and not a worker. I am the interviewee.
Topic: <n> <title>. Read docs/grills/<n>-<slug>/prebrief.md and docs/process/grill-workflow.md first, plus docs/adr/README.md and the newest ADR in docs/adr/ (copy its format).
Then invoke the skill /mattpocock-skills:grill-me on that decision. Follow the skill's round format: each round asks the whole current frontier of decisions as Q<n> - title, each with a recommended answer on its own line worded so that "yes" accepts it. I answer by number. Facts you need: read the docs the prebrief links, or launch a read-only subagent for a fact; never ask me a fact you can look up. Challenge vague answers ("probably", "later", "it depends") by asking what evidence or trigger would settle it, and record that trigger.
Hard rules: ONE decision only (park anything else under 'Parked'); no code, no game, no GitHub, nothing posted; do not edit AGENTS.md, GLOSSARY.md or docs/agents/**; do not run any other skill (not grill-with-docs, not domain-modeling).
Time box: 25 minutes. When I say 'wrap up' or the frontier is empty, state the decision in one sentence and the conditions you heard, and ask me to confirm it.
After I confirm: write docs/adr/<next number>-<slug>.md (status: proposed, copy the newest ADR's MADR format: Context, Considered Options, Decision Outcome, Consequences, Sources; list the gates/triggers and the Parked items), add one line to docs/decisions.md in the existing style with the ADR link, show me the diff summary, commit on this branch (git -c user.name=Claude -c user.email=noreply@anthropic.com commit, trailer 'Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>'), then git push -u forge grill/<n>-<slug> (or origin if that is the remote named in git remote -v; never GitHub). Report the branch, the commit and the ADR number. Do nothing else.
```

Why `grill-me` and not `grill-with-docs` (U, judgement): the topic is a pure decision. `grill-with-docs` makes `domain-modeling` write `GLOSSARY.md` and ADRs on its own in the plugin's short ADR format (1 to 3 sentences, status optional), which collides with our MADR format and with SETUPDOCS owning `GLOSSARY.md`. With `grill-me` the session writes the ADR itself, in our format, after the user confirmed. Use `grill-with-docs` only for a topic that is mainly about project vocabulary, and then only after the ADR format rule is in the prompt and with `GLOSSARY.md` edits allowed explicitly.

## 4. During the session

Round example (V format from `grilling/SKILL.md`, see the fit doc): `Q1 - Abandon trigger: what result on the replicated soak makes you drop d2d? Recommended: a gated metric over the 20 % gap after two fix cycles.` Answer `1 yes, 2 no: <reason>, 3 skip`. "Skip" parks it. The model recomputes the frontier and asks the next round.

## 5. After the grill: from ADR to cards

The user tells the coordinator "grill NNNN accepted". Coordinator steps:
1. Merge `grill/<n>-<slug>` via the dispatcher (docs only; check `decisions.md` has one line and the ADR status is `accepted`; if still `proposed`, the user flips it first).
2. Launch one `spec-sme` (advice) or a `d2-worker` with the brief: "implement ADR NNNN: create change cards with `go run ./cmd/changecheck new <name>` (owns/exclusive disjoint from active cards), spec deltas under `openspec/changes/<name>/specs/`, gates taken verbatim from the ADR's triggers; do not start the work itself." Run `go run ./cmd/changecheck -cli` (0 warnings) before commit.
3. The coordinator context only ever sees: ADR number, branch, card names. Superseded ADRs get `status: superseded by NNNN`, never deleted.

## 6. Archive a grill

After step 5 is merged: set the prebrief's header line `status: used by ADR NNNN`, keep `docs/grills/NNNN-slug/` in place (the Q and A history is not stored; the ADR is the record), tick the topic in `docs/grills/README.md`, delete the worktree (`git worktree remove ~/code/d2lod-wt/grill-<n>`) and keep the pushed branch until merged. A grill that ended without a decision still writes the ADR as `status: deferred` with the missing evidence listed, so the prebrief can be reused.
