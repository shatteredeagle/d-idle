# Upstream PR standard: what a d2decomped PR of ours must prove first

> status: current (agent PRPROOF, 2026-10-30). User rule 2026-10-30: "I want PRs to be as close to absolutely sure with verifiable proof that the change pushes towards 1:1." Applies to every candidate for psi29a/d2decomped before the coordinator may ask the user for a go. First applied to fix/missile-damage-fine: docs/research/pr-missile-damage-fine-proof.md.

Their rules come first and are read every time: upstream `CONTRIBUTING.md`, `AGENTS.md` (sections 2, 5, 6), `.github/pull_request_template.md`, `tools/pr_guard.py`, `.github/CODEOWNERS`. In short: one claim per PR, game.exe 1.14d is the ground truth, evidence a reviewer re-runs on their own copy, no game data or decompiled code in the repo, `FUN_` only for an entry (`LAB_` inside a function), commits authored by the user, `Agent: <name>` in the body, nothing posted without the user's go.

## 1. Primary proof: a differential test against game.exe's own code

- The proof is an oracle in upstream's `tools/emu` style: game.exe's function(s) run in the emulator on the same inputs as d2d's code, outputs compared. Both sides in the loop: the real game.exe instructions (hooks only on what is not the claim: stat getters, events, logging; every hook listed in the script's docstring) and the real d2d code (its test binary in a `--<mode>` that prints the same lines, as `skill_tips.py` / `test_game --skill-tips` do), joined by `diff`.
- Coverage: exhaustive over the boundary cases (thresholds, zero, caps, the formula's branch edges) and at least 100,000 random cases, or exhaustive when the input space allows. Sequences, not single calls, when state carries over (life, timers).
- Mismatches are counted BEFORE the patch and must be ZERO after. "Before" comes from the old d2d code run on the same cases (a throwaway build of upstream/main with the same mode, or a line-for-line copy of the old arithmetic inside the oracle, cited by file and line).
- The exact commands a reviewer pastes (`uv run python <oracle>.py N SEED > g.txt; build/tests/<test> --<mode> SEED N | grep ... | diff g.txt -`) are in the PR body. A tool under `tools/` needs the maintainer's CODEOWNERS review: say so in the body; the oracle may also live outside the PR as "evidence commands" with the script attached to the PR conversation, never game data.

## 2. Address-anchored claims, no unverified link

- Every function, table, offset and traced behaviour in the causal chain carries a tag: V (run or read by us, how), S (upstream doc or secondary source), U (unverified). The chain from the game.exe code to the d2d change has NO U link; a U is either verified (emulator run, disassembly read with `disasm.py`) or taken out of the claim and listed under limits.
- Write what each address shows in the template's table, strongest first (assert strings, then the code that reads the field, then the data). A `FUN_` is an entry in `game-functions.tsv`; `pr_guard.py` must say 0 FAIL / 0 WARN.

## 3. Statistics are corroboration only

Numbers from probes (casts per kill, XP/h, kills per run) never carry the claim: they are noisy, load-dependent and show direction at best. They may appear in the body as a "Measured effect" paragraph that says what they do and do not show, after the proof.

## 4. Scope: one claim, and nothing else changes

- One behaviour per PR. A second gap found on the way (game.exe disagrees with d2d elsewhere) becomes a note under limits and a candidate issue, not a second change.
- Prove nothing else moved: upstream's own tests green (`ninja test_combat test_game`, `ctest`), `tools/presubmit.py --quick` with its warnings answered, `pr_guard.py` clean, and the state hashes of our scripted scenarios (`tools/d2ddet`, `det.py`) unchanged apart from the claimed field. Every caller of a changed function is listed with what it now does differently (nothing, or the claimed thing).

## 5. Independent adversarial verification

A second agent, not the author, gets the branch, the oracle and the proof doc with one brief: break the claim. Counterexamples, wrong assumptions about game.exe (hooks that hide behaviour, a caller that passes other flags), side effects on other callers, cases the generator never produces. Its report (verdict, what it tried, what it found) is attached to the proof doc verbatim. A finding is fixed and the differential re-run, or it becomes a stated limit.

## 6. Limits disclosed

The PR body says plainly what the proof does not cover (the domain of the generator, hooked functions, callers not exercised, data cases excluded), in a "Limits" paragraph, before the checklist.

## 7. Go / no-go checklist and the evidence pack

| # | Gate | Pass means |
|---|---|---|
| 1 | Differential | boundary cases exhaustive, >= 100k random, mismatches before > 0, after = 0, commands in the body |
| 2 | Addresses | every link V or S, none U, pr_guard 0 FAIL 0 WARN |
| 3 | Statistics | labelled corroboration, or absent |
| 4 | Scope | one claim; their tests green; presubmit answered; scenario hashes unchanged apart from the claim; callers listed |
| 5 | Verifier | independent report attached, findings fixed or listed as limits |
| 6 | Limits | written in the body |
| 7 | Their rules | template filled, `Agent: Claude Code`, author the user, no Claude trailer, commit body <= 3 lines, no game data |
| 8 | Spacing | at least one hour since our last PR, follow-ups bundled in one push, user's go for each push and the PR |

The evidence pack for the user is one page, `docs/research/pr-<topic>-evidence-pack.md`: the claim in one sentence; the proof summary (what ran against what); the numbers (cases, mismatches before/after per scenario); the commands; residual risks; the verifier's verdict; `PROOF: complete` or `PROOF: incomplete` with the reason; a recommendation (go / no-go / narrow the claim); the push and PR commands, not run.

## 8. Spacing and disclosure

- At least one hour between our PRs (CodeRabbit limits reviews per author, docs.coderabbit.ai rate limits); bundle follow-up commits into one push; wait for `gh pr checks --watch` and answer threads before the next push, each with the user's go.
- `Agent: Claude Code` in the PR body. Commits are the user's: author and committer `shatteredeagle <26069301+shatteredeagle@users.noreply.github.com>`, no `Co-Authored-By` or other trailer, subject short, body at most three lines.
- Nothing is pushed, posted, commented or labelled by an agent. The coordinator presents the evidence pack; the user runs or approves each command.
