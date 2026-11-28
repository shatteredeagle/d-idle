# Fable round 2, review C: tonight's fixes for security, Proving Ground, builds as data and the pipeline

> status: REVIEW, docs only (reviewer run rv3, 2026-11-28 about 00:40 EST). No code changed, no game, no sandbox touched.
> Read at origin/main `27fe88d59` (fix4 and fix6 merged), branch `agent/fix8` at `7e5cfae44` (not merged), ops repo master `7fbab82` (mergespeed3 live), `tools/autodeploy.py` and its timer as installed.
> Tags: **V** read in the tree or run offline in this run (said how), **S** inferred from the code or from memory of the game tables, **U** not verified.
> Ops-repo files are cited as `ops:tools/<file>:<line>`.

Scope: fix4 (TASK-1100), fix6 (TASK-1098), fix8 (TASK-1097), ops mergespeed3, the autodeploy timer. Two questions per item: is the review finding closed, and what new problem did the fix bring.

## 0. Ranked findings

| # | Sev | Finding | Where |
|---|---|---|---|
| A1 | high | The "never retried" list holds one commit id. The next head of main is a different id with the same code, so a rolled-back build is deployed again about every 25 minutes | `ops:tools/autodeploy.py:169,202` |
| A2 | high | "Newest green main" is the newest head whose own CI run passed; a docs or status head passes with the heavy steps skipped while the run of the code head before it was cancelled | `ops:tools/autodeploy.py:70-85`, `.github/workflows/ci.yml:19-21,70-87` |
| A3 | high | Autodeploy knows nothing of flip windows or measured runs; the flip watcher takes a new baseline on any restart, so a window closes `enabled` over several binaries, and a knob breach is booked on the commit | `ops:tools/autodeploy.py:111-144,181`, `tools/flip/flip.py:515` |
| B1 | high | fix8: a hero without a combat profile refuses to fight but still walks; the error latches for the whole game on one bad read; the invariant is soft, so the canary watch does not see it | `agent/fix8:internal/combat/fighter.go:934-940`, `fight.go:118`, `internal/bot/invariants.go:53` |
| B2 | high | fix8: the mana need now comes from the build in the kit gate, but the shop, the chores and the belt columns still ask the class. A War Cry Barbarian in Normal is asked for mana potions it has no column and no buy rule for | `agent/fix8:internal/bot/kitgate.go:42,63`, `internal/town/shop.go:285`, `internal/town/chores.go:97`, `internal/resource/potion.go:150` |
| C1 | high | fix4: after the redirect the address is clean and the cookie has no lifetime. A bookmark or home-screen icon made from the clean address answers a bare 401 once the browser drops the session cookie | `internal/watch/serve.go:206-211` |
| D1 | medium | The smoke step in `check.sh` runs in no automatic path: the merge gate and the light CI job stop at `--hygiene-only`, and on a push to main the step's own base is HEAD | `tools/ci/check.sh:71,80-93`, `.github/workflows/ci.yml:41,87` |
| D2 | medium | "Every run recorded" compares two counts of one local spool. A double line and a refusal cancel, a second writer hides a refusal, a failed ingest to the board host still exits 0, a run cut by the nightly `timeout` is not counted at all | `cmd/d2scenario/run.go:118,183-187`, `cmd/d2scenario/ledger.go:54-61`, `tools/provingground/nightly.sh:35` |
| D3 | medium | Stability keys on the bot binary only. An edited scenario or run-time data file on the same binary is read as a flake and quarantines the key; hand runs of unmerged agent binaries can make a key `stable` | `internal/scenario/stability.go:96-101,171`, `internal/scenario/pgpage.go:59-61` |
| E1 | medium | json-list union: a top-level key changed on both sides is silently the branch's; a top-level key only main added is dropped; two rows with one name collapse to one | `ops:tools/mergeq_union.py:182-189` |
| E2 | medium | card union: a rewritten `result:` is glued behind main's ("Not verified live ... Verified live ..."); an untick by the branch is undone when main touched the same box line | `ops:tools/mergeq_union.py:246-255,312-315` |
| E3 | medium | Docs-class and scoped gates skip Go tests that read a file by path: the setup golden `.tsv` files, `profiles.json` read by the new plandata test; the flip and autoflip Python tests run in no gate | `ops:tools/merge_classify.py:14-15`, `ops:tools/gate_scope.py:24`, `tools/ci/scripts.sh` |
| C2 | medium | fix4: the flip run check reads the exec environment and PID 1 only and swallows every error; a marker exported by the entrypoint for the bot is not seen; the watcher's revert and autodeploy have no check | `tools/flip/flip.py:119-125` |
| C3 | medium | fix4: a read deletes. `Granted` drops the go when the hero name differs or is empty, and it is called from the polled offer | `internal/diffready/consent.go:98-112`, `internal/bot/diffready.go:72` |
| C4 | medium | fix4 scrubber: still leaks a lowercase or spaced key, a multi-word or unlabelled passcode; breaks JSON lines; hides commit hashes (old rule) | `deploy/d2d-container/tester-report.sh:48,52,54,57` |
| B3 | low | fix8 capability facts: the 57 filled Amazon and Assassin rows agree with the game table as I know it (S); three rows stay open and one duplicate key stays | `agent/fix8:data/capabilities/capabilities.json` |
| X | low | eleven smaller points | section 6 |

Not found, although looked for: catastrophic backtracking in the scrubber (V, section 3.4), a redirect loop in the token gate (V, read), a silently dropped ledger ROW in the adversarial merges (V, section 4.1; rows are kept or the merge fails).

## 1. Autodeploy (live since 00:19)

Facts (V): `ops:tools/autodeploy.py` read whole; user timer every 10 minutes after the last run ended; the deploy log has one deploy tonight (commit `1f5f0f8d7`, watch passed 00:34); `gh run list --branch main` read at 00:37.

### A1. High: "never retried" is by commit id, and main's head changes every few minutes

Scenario: commit X breaks the bot. Autodeploy deploys X, a hard counter rises, it rolls back and writes X into `bad`. Ten minutes later the timer runs again. Main's head is now X+1, the generated "shared files" commit or an auto-Done status commit of any other merge. X+1 is not in `bad`, it is not the deployed one, and `bot_changed(prev, X+1)` is true because the diff still holds X's change. X's code is deployed again, watched 15 minutes, rolled back again. This repeats all night, two bot restarts per round, each round with the damage the breach stands for.
Where: `ops:tools/autodeploy.py:169` (the only test is `sha in st["bad"]`), `:202` (the list holds ids, the last 20).
Fix: after a rollback set a hold that a person or the coordinator releases, or refuse every descendant of a bad commit until a later commit changes a bot path (a fix-forward candidate), and log the hold.
Test sketch: fake `git` and `gh` where the head moves from X to a docs-only child; run `main()` twice with a watch that returns a breach; assert the second run logs `held` and calls no update.

### A2. High: the green that is checked is not the green that is meant

Evidence (V, `gh run list` tonight): `829337379` (labtriage, code) run **cancelled**; `f30206f0d` (docs only, pushed three minutes later) **success**. `86d2e0bab` (fix2 code head) cancelled, the auto-Done commit after it success. The CI workflow cancels a run in progress when a newer push arrives (`ci.yml:19-21`), and the `test` job skips its heavy steps when the diff from the previous pushed head is docs or cards only (`ci.yml:70-87`). The commit deployed at 00:19 was an auto-Done status commit.
Scenario: a code merge is pushed, its auto-Done commit follows seconds later. The code run is cancelled, the status run passes in a minute with no Go test. `newest_green()` returns the status commit. The canary gets code no CI run has tested. The local merge gate did run (scoped, cached), so this is the second net missing, not both.
Where: `ops:tools/autodeploy.py:81-84`.
Fix: walk back from the candidate to the last commit with a completed full run and require every commit in between to be docs-class (the classifier exists), or stop cancelling runs on main.
Test sketch: `newest_green` over a fake run list `[docs success, code cancelled, code success]` must return the third, not the first.

### A3. High: autodeploy against an open flip window and a measured run

Facts (V): autodeploy reads no flip state and no `D2BOT_RUN`; the watcher treats a changed `since` as "the child restarted" and sets an empty baseline (`tools/flip/flip.py:515`); autodeploy counts three `since` changes in its watch as a crash loop (`autodeploy.py:133-137`) and any hard counter rise as a breach of the commit (`:139-141`).
Scenarios:
1. Window open on the canary, 2 h. With the merge rate of yesterday a bot-path commit lands every 10 to 25 minutes, so the window sees four or five binaries. It closes `enabled`. The evidence is "no hard breach on any of five builds, each for 20 minutes".
2. Autoflip flips a knob during the 15-minute watch. The knob causes a hard breach. The watcher reverts the knob and autodeploy rolls the commit back and lists it as bad. An innocent commit is out, and by A1 its code comes back with the next head anyway.
3. Flip, revert and one more restart inside one watch are three `since` changes: "crash-loop", rollback.
4. A sandbox in a soak (`D2BOT_RUN` set in the container) is restarted with a new binary. fix4 added the check to `flip.py` only.
5. Rollback after a flip (U): the tuning file keeps the flipped knob; if the previous release does not know the knob and the loader is strict, the rolled-back bot does not start, and autodeploy does not watch after a rollback.
Fix: autodeploy skips while a flip window is open on the canary or a run marker is set, and the watcher writes the build id into the window and ends it `void` when the build changed.
Test sketch: `test_flip.py` with a fake sandbox whose `since` changes mid-window must end `void`; an autodeploy test with a window file present must log `skipped (flip window)`.

### A4. Medium: rollback and watch details

- The rollback command's exit code is ignored: the log says `ROLLED-BACK` and the state says "previous" even when the command failed (`autodeploy.py:201-205`). Test: fake update that fails on `--rollback`; expect a different result word and a non-zero exit that alarms.
- No alarm: the unit has no failure hook, a rollback is a line in a file. A night of A1 rounds is seen in the morning.
- The baseline is the first status read, 60 s after the deploy (`:130-132`): a breach in the first minute is inside the baseline.
- The watch sees hard counters, panics and reachability. A parked, paused or dead hero and every soft row pass (review 5, 1.3). B1 below is such a row.
- State is autodeploy's own file: a hand deploy of a branch build to the canary is overwritten within 10 minutes, and `--rollback` then returns to that hand build while the state names another commit.
- The restart happens wherever the hero stands (U whether the update waits for town); at every bot-path merge, day and night.

## 2. fix8 (builds as data), not merged yet

### B1. High: fail closed in the field

What changed (V, branch diff): `SelectProfile` returns an error instead of the Sorceress profile; `Clear` and `Kill` return an error while `ProfileErr` is set (`fight.go:118,159`); `LoadBuild` refuses a table without `class` or `combat` (`internal/leveler/build.go:352-360` (the test at 358)); a soft invariant `no-combat-profile` counts it.
Problems:
1. Nothing parks the hero. The only reader of `ProfileErr` outside the fighter is the invariant tick (`internal/bot/invariants.go`, `invTick`). Routes and quest steps still leave town; every fight call fails; the hero walks through packs without attacking (S, not run). Review 8 fix 3 said "park with the reason".
2. The latch: `profileFor` sets the error with `CompareAndSwap(nil, &err)` (`fighter.go:937`) and only `applyClass` clears it at the next game start (`cmd/d2bot/class.go:75`), and `applyClass` returns before the clear when the snapshot is missing (`class.go:69-71`). One snapshot with a bad class value stops all fighting until the next game.
3. Soft row, 300 s gap: the flip watcher and the autodeploy watch count hard rows only, so the canary passes its 15 minutes with a hero that does not fight.
4. A table copied into a data directory before fix8 (the `dir` argument of `LoadBuild`) has no `combat` key (U whether any sandbox has one): after the deploy that hero stops fighting.
Fix before merge: hold the hero in town with the reason while `ProfileErr` is set; make the latch clear on the next good selection; make the row hard or let the watches read it.
Test sketch: runner tick in town with a fighter whose `ProfileErr` is set, for a Paladin and a Sorceress: no leave-town task is issued and the explain line names the table. Fighter test: one failing `profileFor`, then a good one, then `Clear` must fight.

### B2. High: mana need has two sources again, now inside one town visit

What changed (V): `ManaNeedFor(class, build)` returns 2 for a `mana_heavy` table in every difficulty; the kit gate, the tower route guard and the plan shadow use it. Still on the class value: `internal/town/shop.go:285`, `internal/town/chores.go:97`, `internal/resource/potion.go:150` (belt columns), `internal/combat/hard.go:59`, `internal/combat/rules.go:104`. The belt columns for a mana-heavy build still switch on only outside Normal (`manaheavy.go:67`).
Scenario (S, not run): `barbarian_warcry_endgame90` in Normal at or above the kit gate level. Class need 0: no mana column, the shop buys none. Kit gate: `minMP = kitMinMP`; the clamp at `kitgate.go:63` only applies when the mana capacity is above 0, so with no column the minimum stays. The kit is "missing mana potions 0/N" for good, which above the gate level means a town trip each time. The Hammerdin tables (class need 1, one column, shop rule for need 1) get the milder form: a minimum the buy rule may not reach.
The new test (`manaheavy_test.go`) checks the returned number for two builds, not that gate, shop and belt agree.
Fix: one function for all seven call sites (review 8 fix 9 asked for it), or keep the Normal exception until the belt columns follow.
Test sketch: for each `mana_heavy` table and each difficulty: build the belt plan, run the shop rule on an empty belt with gold, then `kitRead`; assert `missing()` has no mana line. Two classes at least (Barbarian, Paladin).

### B3. Low: capability facts from the SME docs

I compared all 57 filled Amazon and Assassin rows (required level and direct prerequisites) with the 1.14d skill table from memory (S; the repo holds no copy of that table, and the card says the same). All agree, including the two-prerequisite rows (Exploding Arrow, Guided Arrow, Charged Strike, Valkyrie, Blade Fury, Shadow Warrior).
Open, as the card's own note says: `Inferno Sentry` and `Royal Strike` have no level and no prerequisites (S: level 24 after Wake of Fire Sentry, and level 30 after Cobra Strike and Blades of Ice); Dragon Flight has no row (S: level 24 after Dragon Tail); the necromancer `Fire Golem` duplicate of `FireGolem` stays. The equality test against the real table on the worker host is not written, so review 8 C1 is closed by a copy of a copy.
Test sketch: the equality test the review asked for, skipped when the tables cache is absent, run in the nightly.

### B4. Other fix8 points

- Ratchet (`internal/plandata/consistency_test.go`): `D2_RATCHET_WRITE=1` rewrites the list from whatever is found, so "only goes down" is a comment, not a check (medium-low). Test sketch: in `check.sh`, every entry of the branch's file must be in the file at the merge base.
- The test reads `internal/combat/classdata/profiles.json` by path (line 184) without importing that package; see E3.
- Traps: one Assassin profile exists and it has the list (V), so no profile lost its ladder. A second Assassin profile added without `traps` lays none, silently ("empty = the driver is idle"). Test sketch: every profile whose layout holds a sentry skill has a non-empty `traps`.
- Review 8 B1, B2, B3, C8 (partly, see B2), C9, C12 are addressed in the branch; A1 to A4, B4 to B7, C2, C5, C6, C10, C11, C13, C14 are not in it (V, file list of the branch).

## 3. fix4 (security), merged at `f8daf5458`; it reaches the canary at the next autodeploy round

Closed as asked (V, read and tests read): finding 3 redirect, no cookie for a wrong token, two headers; finding 4 audit line, hourly budget, measured-run refusal, hero and end date; finding 5 for `flip`; finding 7 NUL runs; finding 2 as a procedure doc.
Not closed (V): finding 4 cancel route and second tap; finding 5 "same check in the watcher before a revert"; finding 7 control characters to the terminal and the rotation glob; finding 1 allowlist for the ladder files, refuse-on-survivor scan, one redactor, bundle clean-up; findings 6, 9, 10, 11, 12 (the mergeq env file is still 0664 in a 0775 directory, V `stat`); finding 8: the hub link fix is commit `a972d7b` on ops branch `agent/fix4` and is **not** in ops master (V, `git branch --merged master`), so the board still has it. Box 7 of the card (live check) is open.

### C1. High: the phone page after the redirect

Facts (V): a valid `?t=` on a page load sets `d2dview` with no `MaxAge` or `Expires` (`serve.go:206`) and answers 303 to the address without `t` (`:211`); a request without token or cookie gets the text `unauthorized` (`:202`).
Scenario (S for the browser side, not tested): the owner opens the link on the phone, lands on `/watch`, adds it to the home screen or bookmarks it. A home-screen page has its own cookie store on iOS; a session cookie also ends when the browser is closed or the tab is purged. The next tap gets a bare `unauthorized` with no hint. Before fix4 the saved address carried the token and always worked. Browsers with cookies off lose the page completely (before: page and reads worked through `?t=`).
No loop: the second request carries no `t` (V).
Fix: give the cookie a lifetime (30 days as the passcode session), answer 401 with a page that says "open the link with the token again", keep the clean address.
Test sketch: `TestTokenGateStripsTokenFromAddress`: the cookie has `MaxAge > 0`; a GET that accepts HTML with no credentials gets a body that names the token link.
Smaller (V): `ctx.token` is still read from the address (`web/shell.js:920`) and `panels/video/panel.js:193,290` and `panels/map/panel.js:110` still build `?t=`; the source guard test looks at `shell.js` only. `u.RequestURI()` for a path that starts with `//` gives a scheme-relative `Location`; it needs a valid token, so it is low.

### C2. Medium: the flip run check

`measured_run` (`tools/flip/flip.py:119-125`) runs `printenv D2BOT_RUN` in the exec environment and reads `/proc/1/environ`, with `2>/dev/null; true`.
Scenarios (S): the marker is exported by the entrypoint or a wrapper for the bot process only: PID 1's first environment does not have it, the exec environment does not have it, the answer is "" and the flip goes ahead. `/proc/1/environ` unreadable for the exec user: same answer. The docstring says "a failed read raises"; only a failed `docker exec` does.
The watcher's revert, autoflip's own path outside `flip()`, and autodeploy (A3) have no check.
`tools/flip/test_flip.py` uses a fake sandbox for it and runs in no gate (E3).
Fix: ask the bot (`ops status` carries the run) or read the environment of the bot's pid; treat "cannot tell" as a refusal.
Test sketch: a fake `_x` that returns an error text for the environ read must raise; an integration drill on a hidden sandbox with the marker exported by the entrypoint.

### C3. Medium: the per-hero go is deleted by a read

`valid` deletes the stored go when the hero name differs from the caller's, an empty name included (`consent.go:104-112`), and `Granted` is called from `DifficultyOffer`, which the page polls (`internal/bot/diffready.go:72`).
Scenarios: one snapshot with an empty player name (U whether a read can give one) erases the owner's go; a look at another hero in the same instance erases it (the review asked for that one); an old `diffgo.json` (a list of strings) fails to parse, the bot logs "unreadable" and the pending go is gone after the update (`cmd/d2bot/live.go:155`), which autodeploy applies to the canary tonight.
Also: `admit` counts refused requests (`difficultydesk.go:155`), so six taps while "not ready", or six requests by anyone with the link, lock the go for an hour; `GoTTL` is a Go variable, not table data.
Fix: `Granted` only answers; dropping happens in `Take` and at grant time; an empty hero name is "cannot tell".
Test sketch: grant for hero A; call `Granted(target, "", now)`; then `Granted(target, "A", now)` must still be true. Two classes.

### C4. Medium: scrubber, what still leaks and what it breaks

Run offline (V: `REPORT_SCRUB_ONLY=1 sh tester-report.sh` on synthetic lines, GNU sed; the image is Ubuntu, so the same sed):

| Input | Output | Kind |
|---|---|---|
| `key abcd-1234-efgh-5678` | unchanged | leak: the dashed-key rule is upper case only (`:52`) |
| `key ABCD 1234 EFGH 5678` | unchanged | leak |
| `passcode: four random words here` | `passcode: <redacted> random words here` | leak: an unquoted value ends at the first space (`:48`) |
| `passcode four random words here` | unchanged | leak: no `:` or `=` |
| `{"auth":{"mode":"token","ok":true},...}` | `{"auth":<redacted>"mode":"token",...}` | the line is no longer JSON |
| `commit <40 hex>`, `bot_sha256=<64 hex>` | `<token>` | the bundle loses the build identity (rule `:54`, older than fix4) |
| `game exe version 1.14.3.71` | `version <ip>` | over-redaction |
| `user_go=true`, `budget.token_left=2` | `<redacted>` | over-redaction of knob-like fields |
| 64 MiB of NUL bytes then a line | 64 MiB of NUL in the output | the damage fix4 handled in Go passes through the script whole |

Backtracking: lines of 300 kB (`a.` repeated, `a:` repeated) and 800 kB of mixed letters and digits all finish at once (V). No finding.
The README now lists what is not replaced, which is the honest part of the fix. The second line of defence the review asked for (refuse to write the tarball when a known shape survives) is still missing.
Test sketch: add the four leak rows to `TestScrubRemovesPrivateText`; add "every scrubbed `.jsonl` line still parses" to `TestBundle`; keep a 12-hex build id and the `build` field in `TestScrubKeepsUsefulText`.

## 4. mergespeed3 (ops, live)

Closed (V): the NUL split (`ops:tools/mergeq_union.py:63-67`), the ledger rule by `name`, the card rule, docs-class for the ledger with priority 3, the two flake names, `MERGE_GATE_CACHED=1` and `MERGEQ_NOWAIT_MAX=6` in the worker env. Review 9 5b rows 3 (retry any failed test once), 5, 7, 8, 10 are not in it.

### 4.1 Adversarial merges, run against the live code (V: Python calling `merge_card` and `resolve` on throwaway repos)

| Case | Result | Verdict |
|---|---|---|
| auto-Done on main (status, date), branch ticks a box and fills `result:` | status Done, later date, branch result, box ticked | correct |
| main reopens box 1, branch ticks box 2 | box 1 open, box 2 ticked | correct |
| quote style of `owns:` rewritten on main, branch adds an item | both items once | correct |
| `result:` folded over lines on main | conflict | correct (not silent) |
| branch deletes a ledger row, main edits it | conflict | correct |
| main deletes a row, branch appends one | git merges cleanly, both applied | correct |
| same new row on both sides, equal | one row | correct |
| both sides add a row; top-level key changed on both sides (27 to 28 and to 26) | merged, value 28 | **E1: silent, the branch wins** |
| both sides add a row; main adds a top-level key | merged, key gone | **E1: silent drop** |
| main has two rows with one name | one row kept (the later) | **E1: silent drop** |
| base "Not verified live. X.", branch "Verified live on i3. X.", main "Not verified live. X. merged 04:34" | "Not verified live. X. merged 04:34 Verified live on i3. X." | **E2: contradiction in the line the changelog is generated from** |
| base ticked, branch unticks (found false), main rewrote the same box line (`[X]`) | ticked | **E2: the untick is lost** |

E1 where: `ops:tools/mergeq_union.py:185-189` (`merged = dict(o)`; the loop runs over the branch's keys only and never reports a conflict), `:182-183` (index by name). The ledger has three top-level keys today (`version`, `note`, `items`), so today's exposure is the note; the rule also serves the scenario files. Fix: the top level through `_merge3` with conflicts reported; refuse duplicate ids.
E2 where: `:246-255` (`_join_result` appends whatever is not a prefix extension), `:312-315` (a box changed on both sides is always ticked). Fix: when the branch's result does not start with the base text, it is a rewrite: conflict, or the branch's text alone. An untick wins over a tick.
Test sketch: the five bold rows as cases in `ops:tools/tests/test_mergeq_conflict.py`.

### E3. Medium: what docs-class and scoping let through

- The ledger as docs-class is sound today: no Go file reads `data/darklaunch.json` (V, `git grep`), and the hygiene step runs the ledger check.
- `\.(md|txt|tsv|csv)$` is docs-class and ignored in a mixed diff. Since fix6, `tools/provingground/golden/setup-*.tsv` are inputs of a Go test (`TestSmokeOnShippedData`). A branch that changes only a golden file merges with no Go gate; in a mixed diff the files do not pull `cmd/d2scenario` into scope. Main then fails for the next branch that does.
- fix8's `TestBuildTablesConsistent` (package `internal/plandata`) reads `internal/combat/classdata/profiles.json` by path. A profile-only change scopes to the packages that import `classdata`; `internal/plandata` is not one. A new inconsistent profile merges green and the next plandata change is blamed (until the periodic full run).
- `.py` and `.sh` are ignored and `tools/flip/test_flip.py`, `test_autoflip.py` are in no gate (V: `tools/ci/scripts.sh` lists other suites, not these). The tools that restart sandboxes merge untested; fix4's flip change is one.
Fix: a small table "data file to test package" in the scope tool (golden files to `./cmd/d2scenario`, `profiles.json` to `./internal/plandata`), and the flip suites in `scripts.sh`.
Test sketch: `gate_scope.py` on `[tools/provingground/golden/setup-banked.tsv]` must not answer `docs`; on `[internal/combat/classdata/profiles.json]` the package list must hold `./internal/plandata`.

## 5. fix6 (Proving Ground), merged

Closed (V): chain link 1 (no default profile, 207 files migrated, golden files), link 6 (a BLOCKED line is accepted with `UNBUILT` and `missing`, refusals in a file), link 9 (the main view shows agent runs), E6 in part (streak by binary), the pack check. The migration script checks its own edit against a JSON round trip, which is good.
Not closed (V): link 4 for the real fork (the fake is a model, see D1), link 5 (`validate` and the smoke both use `hero.NameTables()`), link 7 (no nightly timer on any host, V `systemctl --user list-timers`), link 8 as an alarm (it is a page word), E1 to E5, E7 to E10, F1 to F6. E2 is now explicit: 207 scenarios say `banked`.

### D1. Medium: the smoke tier gives less than the card says

1. It does not run by itself. `tools/ci/check.sh:71` exits for `--hygiene-only` before the smoke block (`:80-93`); the merge gate and the light CI job call exactly that. The heavy CI job calls the full script on a push to main, where `git merge-base HEAD origin/main` is HEAD, the changed list is empty and the block is skipped (V, read). What runs automatically is the unit test `TestSmokeOnShippedData`, when `cmd/d2scenario` is in scope.
2. Stage B is four rows, Paladin and Sorceress. The break of 2026-11-27 came with a Necromancer change.
3. The fake accepts `cmd select` for a skill that was never learned, `cmd skillpt` with no level or prerequisite rule, `debug points` as a no-op (all tagged U in `smokegame.go`), and the prep uses the name tables, where every name resolves. A hero that binds an attack it cannot learn passes. The 2026-11-27 defect is caught (the must-fail arm exists); its neighbours are not.
4. The trigger list in `check.sh` leaves out the prep's dependencies (`internal/d2dctl`, `internal/leveler`, `internal/plandata`); the handoff doc says so itself.
Fix: move the block above line 71 or set `PG_SMOKE=1` in the CI job; one row per class; make the fake refuse `select` of a level 0 skill.
Test sketch: a CI assertion that the marker line is in the job log; a must-fail arm "select of an unlearned skill" beside `NoHands`.

### D2. Medium: results against ledger lines

`run` compares `expect` with the growth of the spool (`run.go:118,183`).
- exec-one ships its line, then dies before the result file: `run` ships a second BLOCKED line (`run.go:155-157`): plus one. A refusal elsewhere in the same job: minus one. Equal counts, exit 0 or 3.
- Two `run` calls with one output directory (a retry, a determinism pair): the other's lines count as mine.
- `PG_LEDGER_CMD` fails (board host away): the function returns true (`ledger.go:54-61`), "results N, ledger N", exit 0, the ledger on the board host has nothing. That is the failure review 6 was about.
- `nightly.sh:35` wraps the job in `timeout`: a cut job never reaches the comparison, and its running scenario has no line.
- `nightly.sh` folds exit 4 into 1.
Fix: compare run keys, not counts; count a failed ingest; write the BLOCKED line for a run when it starts and replace it at the end.
Test sketch: `record_test.go` with a fake exec-one that ships and exits without a result, plus one refused run: the exit must be 4.

### D3. Medium: stability by binary

- The key is the scenario, class and seed; "same binary" is `bot_sha256` and the d2d hash (`stability.go:96-101,171`). The ledger line has no hash of the scenario file or of run-time data. Edit an assertion of a scenario: same binary, new assertion digest, "flip", the key is quarantined and needs a release with a root-cause note. Before fix6 the commit changed with the edit and the streak just restarted. The dirty flag covers `cmd`, `internal` and `go.mod` only (`run.sh:28`).
- The main view takes every `agent/*` run (`pgpage.go:59-61`): five equal runs of an unmerged branch binary make the key `stable`, and stable survives binary changes.
- `internal/scenario/ledger.go:214` (the older report) still compares the commit, so two flip counts can disagree.
- The cell's state follows the latest run of any seed (`pgpage.go:126-133`): seed 11 BLOCKED, then seed 3 GREEN, and the row shows no `blocked`.
Fix: add the setup hash (the golden files have one) to the ledger line and to `sameBot`; count `stable` on main-ancestor binaries only.
Test sketch: `stability_test.go`: two GREEN runs on one binary with different setup hashes and digests must restart the streak, not quarantine.

## 6. Low findings

1. `ops:tools/autodeploy.py`: the canary and never-list names are defaults in code; a renamed sandbox makes it deploy nowhere and say nothing useful.
2. Docs priority 3: a steady stream of ledger and card items always runs before code; no ageing.
3. `mergeq_union`: `updated_date` is compared as text across two date formats.
4. Card rule: a card moved by a rename on main no longer matches `backlog/tasks/*.md`; it fails as before (not silent).
5. `eachLine` (`internal/logx/sink.go`): a line cut by a NUL hole is joined to the line after the hole; fine for reading, wrong for a JSON parser.
6. `difficultydesk.go`: the refusal text says "6 times" as a literal beside the constant.
7. `tester-report.sh:63-64`: the two-word home rules eat up to two words of prose after a one-word user folder.
8. fix6 golden: `scenarioHero` error ignored in `golden.go` (`h, _ :=`).
9. `smoke.go`: the marker regex in `check.sh` pins "prep 4"; a fifth row needs both edited.
10. fix8 `MercAura` now always answers `true` (`internal/bot/task.go`), so "no row for this class" can no longer be told from the default by the caller.
11. The fix4 card says "Hub link fix is in the ops repo, branch agent/fix4": true, and not merged there.

## 7. Recommended order

1. Tonight: hold in autodeploy after a rollback (A1) and skip while a flip window is open (A3). Until then a bad merge at night is redeployed until morning.
2. Before fix8 merges: park instead of walk (B1) and one mana-need function or the Normal exception back (B2).
3. Cookie lifetime and a helpful 401 (C1) before the owner next uses the phone page on the canary.
4. `newest_green` over the last fully tested commit (A2).
5. Top level of json-list through the 3-way merge, result rewrite as conflict (E1, E2); the path-read tests into the scope table (E3).
6. Smoke block above the hygiene exit; ledger comparison by key (D1, D2).
7. Merge ops `agent/fix4` (hub link).

## 8. Limits

Nothing was run against a game, a sandbox or the board. The fix8 scenarios B1 and B2 are read from the code, not run. Browser behaviour in C1 is from general knowledge. The capability rows were compared with the game table from memory. The d2d fork side and the dev-update script in the ops repo were not read beyond their usage line.
