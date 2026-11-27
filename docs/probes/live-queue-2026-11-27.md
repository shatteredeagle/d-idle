# Live queue, 2026-11-27: cards waiting only on a live run

> status: current (docs only; made by card cardfix2 on current main, no game, no lab run). Supersedes the unmerged draft of card cardfix. Sources: the cards' `result:` lines ("Remaining (live): ..."), `python3 -I tools/backlog/plan.py`, `docs/probes/lab-night-plan-2026-11-27.md` (three night runs), `docs/probes/cain-chain-flake-2026-11-27.md`, `tools/provingground/scenarios/*.json`. Tags: V = read in a card or scenario file in the tree, U = unverified (a run that has not happened). Nothing here is ticked: a box closes only after the run and its `writes after start: 0` line. Cards already Done keep their live boxes in `docs/live-checks.md`.

## A. Closed by the three night runs (names as in the night plan)

| Night run | Command shape (from the night plan) | Cards it closes or feeds |
|---|---|---|
| Run 1 (G1 + G2: `board.sm.*` (V, 7 ids in board.json), `board.park.persist-restart`, `town.plan.*` at plan 4 and 5, `town.plan.determinism`; rider `points.bank-by-default,town.resupply,cain.read`) | `D2SCEN_KNOBS=town.plan=4 tools/provingground/run.sh -id <ids> -class Paladin -seed 11`, then Sorceress seed 3 | TASK-983 AC 1 (boardSM 1 with restart drill; AC 2 flip follows), TASK-738 AC 3 (run one `board.sm.*` id twice, equal D1 and Start tick, U: needs the pair compare), TASK-972 stage 4/5 evidence (its 24 h watch is B), TASK-757 AC 3 (`broke-after-bank`, `malus-errand-gold` are not in the night chain: add to Run 1 tail, U) |
| Run 2 (G3 nav A/B, 5 areas) | `tools/provingground/knobs3way.sh <nav id> Paladin "11 3" off= grid=... gridwarp=... all=...` | flip evidence for `nav.fullGrid`, `nav.warpRead`, `quests.layoutRead`; TASK-802 and 803 are Done (all boxes ticked), this is their live read-back |
| Run 3 (Act 2 stepexec, necro stage 1, arbiter 255 Sorceress 201) | `run.sh -id drill.act2.duriel.stepexec ... -repeat 3`; `-id necro.act1-stage1 -class Necromancer -seed 11`; `featuresoak.py run ... --features arbiter-control-families-255 --classes sorceress --seeds 201 --hours 1` | TASK-976 AC 4 (`drill.act2.*.stepexec` ids now exist in stepexec-act2*.json, V; check act2chunks is on main before the run), TASK-1033 AC 5 (Teeth cast), TASK-1034 AC 2 (needs one more `--classes necromancer --seeds 201` line, U), TASK-1030/1048 "stay on" confirmation |

## B. Waiting on a run that no night run covers

- Combat delivery scenarios, fork build and a hidden slot, shape `tools/provingground/run.sh -id <scenario> -class <Paladin|Sorceress|Necromancer> -seed 11`. In combatdrills.json (V): `combat.corpse-explodes` (TASK-728 AC 1), `combat.hold-interruptible` (TASK-730 AC 2 with A1 rate), `combat.teleport-reposition` (TASK-731 AC 2, 3 with D22, row 16, A5; TASK-732 Leap Attack and Charge rows), `combat.whirlwind-clears` and its `.control` (TASK-734, also needs A6 verdict). Not in the tree yet (U, write first): `combat.hammer-in-pack` (TASK-729 AC 2, 3), `combat.point-spell-hits`, `combat.traps-stand` (TASK-727 AC 1, 2), `combat.army-holds`, `combat.pets-hold` (TASK-723 AC 2, 3). Race gate for 727 to 732: `labrun the worker host -- go test -race -count=20 <package>` (ACs 728 #4, 729 #6, 730 #5, 731 #6, 732 #4).
- Necromancer scenarios: `necro.corpse-ce5`, `necro.corpse-ce1` seeds 11 to 13 (TASK-1016 AC 2, card Done, live box in live-checks); `necro.act1-stage1` is Run 3. The `necro.drill.act2.*`, `act3`, `act4`, `act5` drills and `necro.act1-*` exist (V) and feed the class epics, not a card box today.
- Cain chain experiment (`docs/probes/cain-chain-flake-2026-11-27.md`, TASK-680 AC 3): `cain.chain` Paladin seed 11 repeated on one binary; it is the only way to settle the flake label (hypothesis 1 there says it is stale). Command shape `run.sh -id cain.chain -class Paladin -seed 11 -repeat 3` U (exact line in that doc, section on the night experiment); `cain.read` rides Run 1.
- Sandbox watch and flips (sandbox containers, not lab slots): TASK-972 AC 1, 2 (24 h watch, flip per stage), TASK-973 AC 2 (first arbiter family acting on one sandbox), TASK-981 AC 2 (replay of one real sandbox night with `-follow`), TASK-983 AC 2 flip after Run 1.
- Fork binary: TASK-986 AC 1 (town portal and points differential on the new fork binary).
- UI browser check (not a game run): TASK-424 AC 5 and 6, TASK-454 AC 2 against the two layouts that exist (cockpit, stream); `tools/uishots.sh`.
- Unscheduled live-agent cards, Proposed: TASK-246 sessions, TASK-998 flipwave2, TASK-40 tpreserve-live, TASK-56 fidprobe-vendor, TASK-105 fid-fixtures, TASK-216 ready-audit, TASK-248 awaydrill, TASK-990 ckptlive. Run commands are in the cards; no night run closes them.
- Blocked on something else, not live: TASK-777 pgledgeron (board host ingest), TASK-744, TASK-472 uib9, TASK-120 ghcrpublish (user go), TASK-762/763 (arbiter leaves shadow for the town family first).

## C. Not closable tonight

G4 invdesk (`manage.*`) has no slot in the night plan; the Akara hand-in fix is not built, so `cain.read` gives a baseline only.
