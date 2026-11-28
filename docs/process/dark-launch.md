# Dark-launch rule: sandbox first, graduate fast, the ledger never grows

> status: current (card darkthrottle, 2026-11-27, user directive). Supersedes the 14-day dark window of card darklaunch. Why: the arbiter, `town.plan` and `quests.boardSM` were built, shipped at 0 and never enabled (docs/research/fable-logic-review-2026-11-25.md section 2.5); the ledger then kept growing.

## Where the ledger lives (card led1, 2026-11-28)
The ledger has two halves; nothing else is a source.
- DESIGN, in git: one file per knob, `data/darklaunch/<knob name>.json` (the knob name with its dot is the file name). Fields: `name`, `built_by`, `dark_since`, `why_off`, `enable_criteria`, `owner`, `review_by`, `status`, optional `flip_plan` and `note`. A new row is a new file, so two branches never touch the same lines. The `status` word in git is the design intent: `dark`, `enabled` (closed a clean window, default still 0), `default-on` (graduated), `retired`, `by_design`.
- RUNTIME, not in git: `<D2_FLIP_STATE>/ledger/ledger-state.json` on the head box (default `~/.cache/d2flip/ledger/`; a subdirectory, because the tool lists `*.json` of the state directory as open windows). Per knob: `state` (`trial` window open, `enabled` closed clean, `dark` reverted), `trial_*`, `enabled_at`, `revert_*`, `reverted_at`, `criteria_met`; plus a `log` of the last 300 transitions. Only `tools/flip` writes it, under a lock; no checkout is ever written by a flip, a close or a revert, so there is no ledger branch and no ledger commit of the watcher any more.
- MERGED VIEW (`tools/darklaunch/ledger.py` `rows()`; what `report.py`, `flip.py` and `autoflip.py` read): the design row plus its runtime fields. `status` is derived: a row that git calls `dark` or `enabled` shows the runtime `state` when one exists; `default-on`, `retired` and `by_design` are never moved by runtime state. `criteria_met` defaults to `U`.
- `check.py` reads git only, so its verdict is the same on every machine. It fails when a knob file carries a runtime field or the status `trial`, when a file is not named after its row, and when `data/darklaunch.json` (the old one-file ledger) comes back.
- When git must follow: after a clean CLOSE the tool prints one line, `python3 -I tools/darklaunch/ledger.py status enabled <knob>... && git add ... && git commit ...`; run it on any branch (the cap counts the row as open until then). After a REVERT nothing is committed unless git already said `enabled` (a re-flip), then the printed line sets `dark`. `report.py` repeats the pending line for rows closed clean but still `dark` in git.
- Migration: `tools/darklaunch/migrate.py split` wrote the files from the old ledger; `migrate.py state --legacy <old file>` imports the runtime fields of an old one-file ledger into the state file (used once, from the checkout the watcher had written to); `migrate.py verify --legacy <old file>` is the read-only comparison. The old content is frozen in `tools/darklaunch/testdata/legacy-2026-11-28.json`; the golden test splits it and requires the merged view to equal it, runtime fields included. An old one-file ledger still loads through every tool (`--ledger F`).
- Merge queue: the `json-list data/darklaunch.json items name` union rule of the ops repo is no longer needed (rows are separate files); it is left in place and matches nothing. The docs-class pattern of the gate scoper should name `data/darklaunch/` (ops repo, not changed here).

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
   - add the file `data/darklaunch/<knob>.json` (one JSON object; copy a neighbour): `name`, `built_by` (card and commit), `dark_since`, `why_off`, `enable_criteria` (measurable), `owner` (coordinator or owner), `review_by`, `status` (`dark`). No runtime field (`criteria_met`, `trial_*`, ...): those belong to the flip tool's state file;
   - for every row with `dark_since` from 2026-11-28 (older rows are grandfathered with their dates): `flip_plan` `{"instance": "doofer"|"questy"|"both", "bundle": "<group name>", "window": "2h"}` and `review_by` at most 3 days after `dark_since`.
3. `status` in git is one of `dark`, `enabled` (on in the sandbox after a clean window, default still 0), `default-on`, `retired`, `by_design`; `trial` (window open on a sandbox) exists only as runtime state and in the merged view. `by_design` is a permanent default the owner gates (today `resource.pointsDecisions`, `resource.gearDecisions`, `items.ladderRunewords`): never counted against the cap, never overdue, listed separately by `report.py`.
4. A `default-on` or `retired` row stays as history.

## Cap (hard)
`data/darklaunch-cap.json` `{cap, ratchet_by}` caps the rows with status `dark` in git and owner `coordinator` (a row on `trial` is `dark` in git and counts; `by_design`, `enabled`, owner rows do not count; the count never depends on the state file). `check.py` exits 1 when the count exceeds `cap` and the branch grew it against its merge base with origin/main: a branch may add a row only when it closes or removes one in the same branch (net <= 0). Initial cap 27 = the count on 2026-11-27. Ratchet: once a week the coordinator lowers `cap` by one (never below the real count) and moves `ratchet_by` 7 days on; `check.py` prints a WARN when `ratchet_by` is past.

## Enforcement
`tools/ci/check.sh` runs `python3 -I tools/darklaunch/check.py`. It fails when a marked default-0 knob has no row, a row lacks a field, a knob file carries runtime fields or a wrong name, the old one-file ledger is back, a new row lacks a valid `flip_plan` or has `review_by` over 3 days out, the cap is exceeded by a growing branch, or a `dark`/`trial` row is past `review_by` (decide: flip, extend with a new date and reason, or retire). Tests: `python3 -I tools/darklaunch/test_check.py`.

## Review
The coordinator runs `python3 -I tools/darklaunch/report.py` weekly (merged view: name, age dark, status with the runtime state, criteria met, owner, due; `by_design` rows listed separately; `--state ""` shows git only) and ratchets the cap. Every item past due or with `criteria_met: yes` gets one decision: flip, extend (new `review_by` plus a changed `why_off`), or retire.

## Standard enable rule (user decision 2026-11-26/27)
Flip on the sandboxes once the scenario, drill or replay passes, watch a 2 h window with the invariant monitor, revert automatically on a hard breach (a death, a stall, a potion not drunk, a refusal-as-failure storm); otherwise the knob stays on and the row closes, then graduate it. The owner of each flip is the coordinator.

## How to flip (tool `tools/flip/flip.py`, card autorevert1)
1. Precondition: the knob has a `dark` row (`data/darklaunch/<knob>.json`) and its scenario, drill or replay passed. Never in a measured run (`D2BOT_RUN` set refuses); sandbox containers only; one open window per sandbox.
2. Hosts come from `local.env` through `tools/env.sh`: `D2_FLIP_RUN` (command prefix on the sandbox host, default `labrun $D2_LABRUN_HOST --`), optional `D2_FLIP_TUNING` (default `/data/instances/main/state/tuning.json` in the container), `D2_FLIP_BIN`, `D2_FLIP_STATE`. The container is named like the instance.
3. Flip (one knob or a BUNDLE: `<knob>=<value> <knob>=<value> ...`, one window, one backup, one atomic write, one restart, every knob verified, every knob needs a dark row; a breach reverts the whole bundle, each row records `revert_bundle` and `revert_breach`; `flip.py bisect <instance> --without KNOB` prints the re-flip line of the last bundle minus that knob; use `--window 45m` for a short window): `python3 tools/flip/flip.py <instance> <knob>=<value> --why "scenario X green" --window 2h` (`--dry-run` prints every step and writes nothing; run it first). The tuning file is the store envelope `{version, data:{profile, overrides}}`: the knob goes into `data.overrides` (V: real sandbox file shape), the rest stays; the old file is the backup tag `bak-flip-<ts>`; the write is an atomic rename; the restart is the whole container (`D2_FLIP_RESTART=container`, default) or only the bot launcher (`bot`, U: not drilled). Then `--verify`: it waits (`D2_FLIP_VERIFY_S`, 240) until `ops status --why` shows the value under `knobs`; if it never does, the backup is restored and nothing is recorded. On success the row is `trial` with `trial_started`.
4. Watch (one watcher for all windows, under the process registry): `tools/agentrun.sh <agent> --ttl 90000 -- python3 tools/flip/flip.py watch`. It reads `GET /api/explain` data through `d2bot ops status --why` (`invariants`, Hard rows: potion-not-drunk, life-zero-alive, refusal-storm, xp-frozen, mode-no-exit, corpse-unresolved) every 60 s.
5. A hard counter above its baseline (taken right after the restart; a child restart inside the window rebaselines at zero) reverts: backup restored, child restarted, state `dark` with `revert_reason` and `reverted_at` in the state file, one line `REVERT ...`. Ten unreadable polls in a row count as a breach. A failed restore or restart prints `REVERT FAILED` and is retried every poll; the file is never left half set.
6. No breach to the end of the window: state `enabled` (on in the sandbox, `enabled_at`) in the state file, backup removed, line `CLOSE ...`, then a line `LEDGER git: ...` with the one-line commit that sets `enabled` in git. The coordinator then flips the code default and sets `default-on` (`ledger.py status default-on <knob>`).
7. `python3 tools/flip/flip.py status` lists open windows. Tests (fake sandbox and clock): `python3 -I tools/flip/test_flip.py`. U: not yet run against a real container.

### Hardening (card fliptool)
- Before any write `flip` runs a 20 s `docker inspect <instance>` through the transport; a wrong host or a missing docker stops it with a message naming `D2_FLIP_RUN`. The sandbox containers live in the docker of the worker host, so `D2_LABRUN_HOST` must name that host (not the NAS).
- `--reflip` accepts a row whose status is `enabled` (on in the sandbox, code default still 0); without it that row is refused.
- The watcher writes `<D2_FLIP_STATE>/watch.heartbeat` every poll; `flip` prints a WARNING when it is missing or older than 300 s. `flip.py watch --daemon` never exits; `flip.py watch --install-unit` prints a systemd user unit (`Restart=always`), `--write` writes it to `~/.config/systemd/user/d2flip-watch.service` (never enabled by the tool; generate it from the main checkout, the unit pins the script path).
- Ledger writes (card led1): `flip`, CLOSE and REVERT write the runtime state file only (see "Where the ledger lives"); the checkout is read, never written. `commit_ledger` and the watcher's ledger branches (`agent/ledger-<knob>-<date>`) are gone; `--commit-ledger` is still accepted on old command lines and does nothing. A close or revert lands even when the knob's file is unreadable (the line `LEDGER: design rows unreadable` is logged).
- `flip.py` loads `local.env` from its own checkout (it points `tools/env.sh` at it), so the systemd unit needs no `Environment=` lines; the unit also sets `WorkingDirectory` to that checkout. Generate the unit from the checkout that holds `local.env` (the main one), not from a worktree.

### D2_FLIP_RUN forms (card autorevert1, after the empty-write incident)
- `labrun <machine> --`: the usual form. labrun does NOT forward stdin, so the tool never sends the tuning file on stdin: it rides as a base64 argument (limit 100000 characters), lands in `<tuning>.new`, and size and sha256 are checked before the rename; the tool then reads the file back through the transport and compares it with the intended text BEFORE the restart. Any mismatch aborts with the container untouched and no restart.
- `ssh <host>`: works the same way (the command line is the only channel used).
- A local wrapper script that runs the rest of the line in the container is fine; it must exit non-zero when the command fails.
- Tests: `python3 -I tools/flip/test_flip.py` includes a transport that drops stdin and one that empties the payload.

See also: [autoflip](autoflip.md) (planned flips and graduation briefs).
