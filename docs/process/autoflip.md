# Autoflip

User policy 2026-11-27: a finished feature goes to the sandbox and is monitored. `tools/flip/autoflip.py` plans and runs the flips so the coordinator does not flip row by row; `flip.py` stays the only thing that touches a sandbox.

- `autoflip.py plan [--json]`: reads `data/darklaunch.json`; takes rows with status dark and owner coordinator (never `by_design`, never owner rows). Instance, bundle group, window and value come from the row's optional `flip_plan` {instance, bundle, window, value}, else from `tools/flip/autoflip.json` by knob prefix (`quests.`/`items.`/`loot.` to questy, `combat.`/`town.`/`bot.`/`nav.`/`invdesk.`/`ui.`/`input.` to doofer; names in `log_only_rows` go to both). A knob that the deployed build of the instance does not list in `ops status --why` knobs is skipped with the reason. Bundle = same instance and group, at most 6 knobs (more split into further bundles). Only one bundle per instance is ready; the rest are QUEUED (one window per instance). Prints the exact `flip.py ... --window 2h --commit-ledger` line per bundle.
- `autoflip.py run [--dry-run]`: per instance, the first bundle only; refuses when a window is open or a hard counter is nonzero; runs the flip.py dry-run first, then the real flip, and ends with a one-line summary. The watcher (`flip.py watch`) must be alive.
- `autoflip.py graduate-list`: rows with status enabled, `enabled_at` set and no `revert_reason`; prints per row the file of the default (`Key: "<knob>"` in `internal/env/tuning*.go`), the grep of the knob name in non-test Go (old-path sites) and the ledger edit. It edits nothing: hand the text to a worker as the brief of a graduation card.

Tests: `python3 -I tools/flip/test_autoflip.py` (fake sandbox of test_flip.py).
