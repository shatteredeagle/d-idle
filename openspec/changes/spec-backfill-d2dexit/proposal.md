## Why

Card d2dexit (archived 2026-10-30) changed d2denv, d2dsrc and d2dworld so the exit to Blood Moor is found (it was a hang waiting for a link) but merged with skip_specs; docs/spec-skip-audit.md judged it needs-spec (effort S). The link wait, liveness beat and log lines are backend behaviour, so they are ADDED to backend-contract.

## What Changes

- Delta ADDED to backend-contract: link wait outlives the verb timeout, default link wait, missing layout verb warning, arrival diagnostics, liveness beat, one unknown-fields line
- Nothing in internal/** or cmd/** changes
