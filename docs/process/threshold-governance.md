# Threshold governance: a floor moves on a number, never on one death

> status: current (card thrgov, TASK-977, 2026-11-26). Why: on 2026-11-25 the road to the Dark Wood got six rules in one day, each chasing one hero's deaths (docs/research/fable-logic-review-2026-11-25.md 2.3), and the knob table grew by 58 rows in six days. Roadmap row P10.

## Rule
1. A floor, line or threshold row (min level, min life, kit count, gold per minute, potion floor, entry life percent) changes only with
   - (a) the number of the margin model (the survival model that already computes the margin per area; the `evidence_ref` starts with `margin-model:` and names the zone and model version), or
   - (b) a seed set: at least 2 runs from different seeds or heroes (`evidence: n`, n >= 2) whose summary the `evidence_ref` points to (a probe doc, a soak record, a scenario row).
   One death, one hero, one day is `evidence: 1`: it may seed a row, it may not move one.
2. Every threshold row in a governed file carries `evidence` (n, a number) and `evidence_ref` (text or doc path). An existing row that came from a single incident is seeded with `evidence: 1` and `evidence_ref: "single incident: needs seed set"`; the first change to it needs rule 1.
3. Knob budget: a fix card may not add a knob without removing one, or the owner's word, and a card that adds a knob says in its result how it graduates (its `flip_plan`, docs/process/dark-launch.md) and must not push the dark ledger over its hard cap (`data/darklaunch-cap.json`; a new dark/trial coordinator row needs a closed or retired one in the same branch). The card says so in its front matter: `knob_net: 0` (default, nothing to say) or `knob_net: +1 owner-ok` (the owner approved the net rise; name where).
4. Later (not built): safety and supply thresholds as rows in the loot-rule grammar, readable and editable by the operator (review row 7, P10).

## Enforcement: tools/ci/knobbudget.py
- Knob budget: compares the knob keys of `docs/knobs.md` (generated) at the merge base and at the branch tip. Net count up without `knob_net: owner-ok` in the branch's card is a finding.
- Evidence rows: for each file in `governed_files` of `tools/ci/knobbudget-baseline.json`, a row (a JSON object with a number of its own) whose numbers changed, or a new row, needs `evidence` and `evidence_ref` on it or on an ancestor object, with n >= 2 or a `margin-model:` ref. Editing only `evidence`, `evidence_ref` or `_comment` is not a change.
- Phase: `tools/ci/check.sh` runs it with `--report` (findings print, exit 0) for one session; then drop `--report` and it fails the branch. Skipped for non-agent branches.
- Governed today: `internal/plandata/admission.json` and `internal/plandata/povertyfarm.json`, all rows seeded n=1. Add a file to the list in the same change that seeds its rows. Not yet governed (follow-up, needs a loader that tolerates the fields): readiness floors (`internal/readiness/checks.json` rejects unknown fields), the per-build rows, the admission and route-guard numbers that still live in Go.
- Tests: `python3 -I tools/ci/test_knobbudget.py`.
