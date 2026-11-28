# Fork rebase round 2026-11-28 (branch pushed, built, tested; nothing switched)

> status: current (2026-11-28, round `didle/rebase-20261128` of docs/process/fork-sync-cadence.md; user go for the round by popup the same day). No image was built or loaded, no container was created or restarted, no lab or sandbox binary was replaced. The SWITCH is a separate decision (last section). Tags: V = verified (how said), S = read from a tool or log, U = unverified. Previous round: docs/process/sandbox-devp30.md.

## The branch (private fork, private remote only)

- `didle/rebase-20261128`, tip `df517276aa97c59e7cac47a6e3b5475bd9e17ac2` (V: `git ls-remote <private remote> refs/heads/didle/rebase-20261128` printed that hash after the push). Two new commits:
  - `9d9bfdf24` merge of upstream main `28a6c6491` (2026-11-27) into the base, no rewrite, no force-push.
  - `df517276a` one compile fix in our own verb (below).
- Base: `didle/rebase-20261125` (`418eb9319`, the devp30 round) plus the three commits of `didle/forkcaps-20261127` (`b88f72bd6`: `cmd skillhold`, the cast caps names, their differential test), taken by fast-forward. DECISION OF THIS ROUND, to confirm: forkcaps is the newest patch branch and sits directly on the devp30 round, so the integration branch now contains it (rule 3 says the integration branch contains every patch branch). Not included: the `fidbatch-*` branches (upstream PR drafts, not lab patches).
- Drift before: 294 upstream commits behind (140 on the first-parent line), 9 of our own merged upstream commits missing from the base (Hydra heads, Bone Prison pieces, the merc/bolt/pet test skips, the radius note). After: 0 behind, 0 missing (V: `git rev-list --count HEAD..upstream/main` and `git log --author=<our account> upstream/main --not HEAD`). The alert said 15 missing and 1 conflict; measured against the newest branch on the private remote it was 9 and 0 (S: the alert's checkout had probably not seen the newest branches; U).
- Our commits come back once: none of the 9 upstream subjects exists on the fork side of the merge (V: subject search in `upstream/main..base`), so nothing is carried twice.
- What the fork still carries over upstream: 117 files, +9093 / -163 (V: `git diff --stat upstream/main HEAD`).

## Conflicts

| Kind | Where | Resolution |
|---|---|---|
| Text conflicts | none (V: `git merge` reported "Automatic merge went well"; 24 files were touched on both sides, all merged by themselves, among them `fight/tick.cpp`, `gamedata.hpp`, `world.hpp`, `rules.cpp`, `docs/control_channel.md`) | nothing to resolve |
| Meaning conflict, found by the compiler | our `buy` verb (`apps/d2d/devctl/world_verbs.hpp`) sent the client message with (stock index, tab, fill); upstream `47617265e` changed that message to (unit id of the stock item, fill) and lets the game find the tab | rule 4, upstream's version of the message wins: the verb now sends the stock item's unit id; its own checks (tab, index, gold, room) and its reply are unchanged; a stock item without an id is refused by name (`err ... refused`). V: first build failed on that one line ("too many initializers"), second build green, `input_verbs`, `input_verbs2_diff`, `vault_verbs` and `itemprice` pass |

Switch defaults after the merge (V: read in `apps/d2d/main.cpp`): `--lab-verbs`, `--item-verbs`, `--vault-verbs`, `--vault-pay`, `--ladder-runewords` are all `false`.

## Build (V, the builder, one job, nice 19, d2d only, 3 jobs)

- `fork-rebase-builder.sh build` with image number 31 (ref `devp31`): first run failed after 213 s on the line above; second run green in 334 s.
- Binary: 18739616 bytes, sha256 prefix `b3e3f1fbb029cfa9`, stored on the worker host under a new name next to the older fork binaries. No existing binary was replaced (V: the copy step never overwrites; the lab binary and the sandbox were not touched).

## Regression set (V, the builder, `fork-rebase-builder.sh test`, game data on the builder, one test at a time)

| Test | Result |
|---|---|
| levelmap | pass |
| levelmonsters | pass |
| levelobjects | pass |
| combat_reads | pass |
| input_items_diff | pass |
| input_verbs | pass |
| input_verbs2_diff | pass |
| lab_verbs | pass |
| vault_verbs | pass |
| item_verbs | pass |
| paygold | FAIL, FAIL, pass (3 runs). Failing lines: "digest differs in 0 line(s) only: []", "repeat leaves the digest as it was", "refusals leave the digest as it was"; the refusal and measured-run checks pass in every run. Same shape as the flake recorded for the devp29 and devp30 binaries (S: sandbox-devp30.md, the session-copy read race). Not shown to be a regression, not shown to be unchanged either: no rerun on the devp30 binary in this round (U) |
| creditgold | pass |
| itemprice | pass |
| itemdecode | pass (listed as flaky; passed first time) |
| warptriggers | pass |
| input_cast_tp_diff (forkcaps) | pass |
| smoke_d2d (run last) | pass |

Not run in this round (U): upstream's own ctest suite (d2d only was built), the other fork verb tests of earlier rounds (item_colour, item_pic, labels, merc_verbs, cursor_verb, overlay, pad_use, the forkcaps hold differential), the bot-side `TestRealD2D` tests, any Proving Ground scenario, any live game.

## What changes for the bot (vs devp30)

- Verb syntax: nothing. `buy <stock> [tab] [fill]` and `cmd ... buy <stock>` take the stock index as before (V: read in the merged tree; the bot's client sends `buy <index>`).
- Behaviour of the d2d follows 294 upstream commits (S: by subject: game data stubs, combat and fight, pets, mercs, monster AI, skills incl. Hydra and Bone Prison, quests, level generation, touch and controller input, hostility between players). Stock items now carry unit ids from the moment a store opens; a buy names the item by that id. None of this was played (U).
- New since devp30 from forkcaps: `cmd skillhold`, caps names `input.use-item`, `input.cast-unit`, `input.cast-point`, `input.cast-hold` (S: docs/design/input-cast-verbs.md).

## What the user must decide before any switch

1. Confirm forkcaps belongs in the integration branch (it is in this one). If not, a branch without it is one merge away; say so.
2. Go for the image step (`fork-rebase-builder.sh image`, number 31: build and load only) in the night window. Not done here.
3. Before the image, or accept without: the rest of the fork verb tests, upstream ctest (the range is larger than 100 commits, docs/process/fork-rebase.md asks for it), the bot-side real-d2d tests and one Proving Ground scenario each for a fight row and a vendor row (the buy message changed).
4. paygold: accept as the known flake, or ask for a 3-run comparison on the devp30 binary first. It guards the vault gold write, so the comparison is cheap insurance.
5. The switch itself (lab d2d binary and the sandboxes): separate go, with backup and rollback as in docs/process/sandbox-devp30.md. The sandbox keeps running its current image until then.

## Ready-to-run checklist for the next step (no game needed for 1 to 3)

1. `FORK_DEVP=31 FORK_TESTS="item_colour_d2d item_pic_d2d labels_d2d merc_verbs_d2d cursor_verb_d2d overlay_d2d pad_use_d2d" fork-rebase-builder.sh test didle/rebase-20261128`
2. paygold three times on the devp30 binary and three more on this one; compare the failing lines.
3. Upstream ctest on the builder's long lane (tests on), then the bot-side `TestRealD2D` against a headless d2d of this binary.
4. After the user's go: `FORK_DEVP=31 fork-rebase-builder.sh image didle/rebase-20261128`, then the sandbox note for image 31.
