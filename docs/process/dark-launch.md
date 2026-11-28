# Dark-launch rule: sandbox first, graduate fast, the ledger never grows

> status: current (card darkthrottle, 2026-11-27, user directive). Supersedes the 14-day dark window of card darklaunch. Why: the arbiter, `town.plan` and `quests.boardSM` were built, shipped at 0 and never enabled (docs/research/fable-logic-review-2026-11-25.md section 2.5); the ledger then kept growing.

## Policy: SANDBOX-FIRST
A feature that is complete goes to the sandbox and is monitored; it does not wait dark.
1. Merge to main, then the dev-update deploy to the sandboxes.
2. Flip its bundle with a 2 h window and auto-revert (`tools/flip/flip.py`, below). A hard breach reverts the whole bundle.
3. A clean close (window over, no breach) makes the row `enabled`.
4. GRADUATION: the next card sets the code default to the new behaviour, deletes the knob and closes the row (`default-on`, or `retired` when the code path goes). A knob that stays at 0 after a clean close is a defect.
No step is a long wait (user decision 2026-11-26/27); the accelerated lab soak, scenarios and replays are the evidence.

A default-0 dark launch (a ledger row that stays `dark`) is only for
- features the owner gates: status `by_design` or owner rows (a product decision, "owner go only"), or
- behaviour with no sandbox signal (nothing the invariant monitor or a scenario can observe).
Everything else goes through the flip plan above.

## Rule for a card that adds a knob
1. The card's `result:` says how the knob graduates (its `flip_plan`) and the card must not push the cap below (see Cap). Knob budget: docs/process/threshold-governance.md. AGENTS.md should link this file.
2. A feature that ships behind a knob with default 0 MUST, in the same PR:
   - put `[dark-launch]` in the knob's description (it shows in docs/knobs.md);
   - add a row to `data/darklaunch.json`: `name`, `built_by` (card and commit), `dark_since`, `why_off`, `enable_criteria` (measurable), `owner` (coordinator or owner), `review_by`, `status`, `criteria_met` (yes, no, U);
   - for every row with `dark_since` from 2026-11-28 (older rows are grandfathered with their dates): `flip_plan` `{"instance": "doofer"|"questy"|"both", "bundle": "<group name>", "window": "2h"}` and `review_by` at most 3 days after `dark_since`.
3. `status` is one of `dark`, `trial` (on sandboxes), `enabled` (on in the sandbox, default still 0), `default-on`, `retired`, `by_design`. `by_design` is a permanent default the owner gates (today `resource.pointsDecisions`, `resource.gearDecisions`, `items.ladderRunewords`): never counted against the cap, never overdue, listed separately by `report.py`.
4. A `default-on` or `retired` row stays as history.

## Cap (hard)
`data/darklaunch-cap.json` `{cap, ratchet_by}` caps the rows with status `dark` or `trial` and owner `coordinator` (`by_design`, `enabled`, owner rows do not count). `check.py` exits 1 when the count exceeds `cap` and the branch grew it against its merge base with origin/main: a branch may add a row only when it closes or removes one in the same branch (net <= 0). Initial cap 27 = the count on 2026-11-27. Ratchet: once a week the coordinator lowers `cap` by one (never below the real count) and moves `ratchet_by` 7 days on; `check.py` prints a WARN when `ratchet_by` is past.

## Enforcement
`tools/ci/check.sh` runs `python3 -I tools/darklaunch/check.py`. It fails when a marked default-0 knob has no row, a row lacks a field, a new row lacks a valid `flip_plan` or has `review_by` over 3 days out, the cap is exceeded by a growing branch, or a `dark`/`trial` row is past `review_by` (decide: flip, extend with a new date and reason, or retire). Tests: `python3 -I tools/darklaunch/test_check.py`.

## Review
The coordinator runs `python3 -I tools/darklaunch/report.py` weekly (name, age dark, criteria met, owner, due; `by_design` rows listed separately) and ratchets the cap. Every item past due or with `criteria_met: yes` gets one decision: flip, extend (new `review_by` plus a changed `why_off`), or retire.

## Standard enable rule (user decision 2026-11-26/27)
Flip on the sandboxes once the scenario, drill or replay passes, watch a 2 h window with the invariant monitor, revert automatically on a hard breach (a death, a stall, a potion not drunk, a refusal-as-failure storm); otherwise the knob stays on and the row closes, then graduate it. The owner of each flip is the coordinator.

## How to flip (tool `tools/flip/flip.py`, card autorevert1)
1. Precondition: the knob has a `dark` row in `data/darklaunch.json` and its scenario, drill or replay passed. Never in a measured run (`D2BOT_RUN` set refuses); sandbox containers only; one open window per sandbox.
2. Hosts come from `local.env` through `tools/env.sh`: `D2_FLIP_RUN` (command prefix on the sandbox host, default `labrun $D2_LABRUN_HOST --`), optional `D2_FLIP_TUNING` (default `/data/instances/main/state/tuning.json` in the container), `D2_FLIP_BIN`, `D2_FLIP_STATE`. The container is named like the instance.
3. Flip (one knob or a BUNDLE: `<knob>=<value> <knob>=<value> ...`, one window, one backup, one atomic write, one restart, every knob verified, every knob needs a dark row; a breach reverts the whole bundle, each row records `revert_bundle` and `revert_breach`; `flip.py bisect <instance> --without KNOB` prints the re-flip line of the last bundle minus that knob; use `--window 45m` for a short window): `python3 tools/flip/flip.py <instance> <knob>=<value> --why "scenario X green" --window 2h` (`--dry-run` prints every step and writes nothing; run it first). The tuning file is the store envelope `{version, data:{profile, overrides}}`: the knob goes into `data.overrides` (V: real sandbox file shape), the rest stays; the old file is the backup tag `bak-flip-<ts>`; the write is an atomic rename; the restart is the whole container (`D2_FLIP_RESTART=container`, default) or only the bot launcher (`bot`, U: not drilled). Then `--verify`: it waits (`D2_FLIP_VERIFY_S`, 240) until `ops status --why` shows the value under `knobs`; if it never does, the backup is restored and nothing is recorded. On success the row is `trial` with `trial_started`.
4. Watch (one watcher for all windows, under the process registry): `tools/agentrun.sh <agent> --ttl 90000 -- python3 tools/flip/flip.py watch`. It reads `GET /api/explain` data through `d2bot ops status --why` (`invariants`, Hard rows: potion-not-drunk, life-zero-alive, refusal-storm, xp-frozen, mode-no-exit, corpse-unresolved) every 60 s.
5. A hard counter above its baseline (taken right after the restart; a child restart inside the window rebaselines at zero) reverts: backup restored, child restarted, row `dark` with `revert_reason` and `reverted_at`, one line `REVERT ...`. Ten unreadable polls in a row count as a breach. A failed restore or restart prints `REVERT FAILED` and is retried every poll; the file is never left half set.
6. No breach to the end of the window: row `enabled` (on in the sandbox, `enabled_at`), backup removed, line `CLOSE ...`. The coordinator then flips the code default and sets `default-on`.
7. `python3 tools/flip/flip.py status` lists open windows. Tests (fake sandbox and clock): `python3 -I tools/flip/test_flip.py`. U: not yet run against a real container.

### Hardening (card fliptool)
- Before any write `flip` runs a 20 s `docker inspect <instance>` through the transport; a wrong host or a missing docker stops it with a message naming `D2_FLIP_RUN`. The sandbox containers live in the docker of the worker host, so `D2_LABRUN_HOST` must name that host (not the NAS).
- `--reflip` accepts a row whose status is `enabled` (on in the sandbox, code default still 0); without it that row is refused.
- The watcher writes `<D2_FLIP_STATE>/watch.heartbeat` every poll; `flip` prints a WARNING when it is missing or older than 300 s. `flip.py watch --daemon` never exits; `flip.py watch --install-unit` prints a systemd user unit (`Restart=always`), `--write` writes it to `~/.config/systemd/user/d2flip-watch.service` (never enabled by the tool; generate it from the main checkout, the unit pins the script path).
- The ledger rows change in the checkout where `flip` or the watcher ran. `--commit-ledger` puts the flip's rows on a branch `agent/ledger-<knob>-<date>-<HHMMSS UTC>` in a sibling worktree, commits there and prints the `mergeq.sh add` line (no push, no queue). The watcher does the same by itself after every CLOSE (trial to enabled) and REVERT (to dark), with the same branch scheme (the time keeps a close and a revert on the same day apart), and writes the `mergeq.sh add` line to its log; a failure there (for example git refuses the worktree) is logged as `LEDGER COMMIT FAILED` and the rows stay uncommitted in the checkout. Queue the branch yourself.
- `flip.py` loads `local.env` from its own checkout (it points `tools/env.sh` at it), so the systemd unit needs no `Environment=` lines; the unit also sets `WorkingDirectory` to that checkout. Generate the unit from the checkout that holds `local.env` (the main one), not from a worktree.

### D2_FLIP_RUN forms (card autorevert1, after the empty-write incident)
- `labrun <machine> --`: the usual form. labrun does NOT forward stdin, so the tool never sends the tuning file on stdin: it rides as a base64 argument (limit 100000 characters), lands in `<tuning>.new`, and size and sha256 are checked before the rename; the tool then reads the file back through the transport and compares it with the intended text BEFORE the restart. Any mismatch aborts with the container untouched and no restart.
- `ssh <host>`: works the same way (the command line is the only channel used).
- A local wrapper script that runs the rest of the line in the container is fine; it must exit non-zero when the command fails.
- Tests: `python3 -I tools/flip/test_flip.py` includes a transport that drops stdin and one that empties the payload.
