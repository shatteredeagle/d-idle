# Queue speed changes (TASK-312)

> status: current. Source: docs/research/merge-queue-profile-2026-11-11.md. Code lives in the ops repo (`tools/mergeq.py`, text in its `docs/merging.md` item 10); this page is the public summary.

1. A recorded `ci: red` of a merged item expires when a later green head of main contains it (`red (fixed by <sha>)`, one worker log line). A red no green head contains still holds the queue.
2. Main is judged by the `ci` workflow only (`MERGEQ_CI_WORKFLOWS`, default `ci`). Another workflow failing (desktop-electron) is a WARN line and `other workflow red: <name>` in `mergeq status`, no hold.
3. The auto-Done follow-up head counts as an in-flight push (cap `MERGEQ_NOWAIT_MAX` 3, a red result holds everything). The generated shared-files commit is part of the item's own head already.
4. `MERGE_TIMINGS=1` is in the worker env file; read `result.timings` after a day, then decide the gate trims (profile candidates a, b, e).

V: unit and rehearsal tests in the ops repo (`python3 -I tools/tests/test_mergeq.py`, fake gh, scratch bare repo). U: live behaviour until the worker restarts.
