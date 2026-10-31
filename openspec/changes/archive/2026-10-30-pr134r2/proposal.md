## Why

CodeRabbit reviewed our upstream PR #134 (get-hit by hit class and share of life) with two actionable comments, and the PR's merge state is blocked on the maintainer. The user's standard for upstream PRs (docs/process/upstream-pr-standard.md, memory d2-pr-proof-standard) wants every review claim answered with evidence from game.exe's own code, a differential with zero mismatches after any logic change, an independent adversarial check, and nothing posted or pushed without the user's go. The round-1 card (hitrec, archived) produced the fix e3c1e535 and the replies; this card checks that round independently against the live PR state and prepares the next step.

## What Changes

- The review, its comments, the check runs and any maintainer activity on PR #134 are read in full (GET only) and the timeline against our two pushes is written down.
- Each actionable comment gets a verdict (valid / partly valid / wrong) with V/S/U evidence: the differential (`tools/emu/hit_rule.py` vs upstream's `test_combat --get-hit` on the clone at the PR head) is re-run for the documented seed and a fresh one, with mismatch counts before (the summed model) and after (per blow).
- One independent verifier (read-only) attacks the per-blow change; its verdict is attached.
- `docs/research/pr-get-hit-rule-review2-draft.md`: verdicts, numbers, draft replies in the tone of our posted ones, the exact push and reply commands (not run). No fix commit unless the evidence calls for one.
