# Roadmap refresh 2026-11-28: every row of roadmap-fable.md against main

> status: current (card roadfresh, TASK-1123; docs and one read-only script, no game run). Consumer: the coordinator's page `docs/roadmap-fable.md` and whoever refreshes it next. Checked against `git log origin/main` at 01b93bd88.

Tags: V = read in this tree (a merge subject, a file, a card, the ledger); S = another doc's claim; U = not known from the repo.

## Why the page went stale three times

1. The page is edited by hand at merge time and a card merged under another branch name leaves no trace a search for the card id finds: `dlvwave3` carried five dlv cards, `townrest` two town cards, `pgcards2` the nightly and earned cards, `townwire`, `smetown` and `townapply` finished rows whose own cards stayed open (V merge subjects).
2. Card status lags or leads: town12 (TASK-765) reads In Progress with a NOT DONE result although the SME run is merged; inputl5 (TASK-989) has both boxes ticked and reads Proposed; nine dlv cards moved from Proposed to In Progress after their code merged (V card files).
3. Cells cited `docs/knobs.md` by line number; 11 of 18 citations pointed at another knob or an empty line (V, each line read). They now cite the knob by name.
4. The refresh passes moved the rows they knew about and left the `not started` rows alone, so those were the stale ones.

## Method

For every table row: the card ids of its card cell were looked up in `backlog/tasks/` (status, ticked and open boxes, `result:`), in the merge subjects of `origin/main`, and the files the row names were checked to exist (V for the rows moved below). A row counts as merged only when its code and tests are on `origin/main`; branches of `git branch --no-merged origin/main` are named as in flight and never as evidence. Ledger words come from `data/darklaunch.json` and `python3 -I tools/darklaunch/report.py`; knob defaults from `docs/knobs.md` (all defaults quoted on the page match, V). Lab results come from `docs/probes/lab-night-results-2026-11-28.md` (S for this card: read, not rerun; every setup there is CHEATED, mechanics only).

## Rows whose STATE moved

| row | was | now | evidence (V) |
|---|---|---|---|
| R9 difficulty hop | built-dark | trial | ledger `bot.diffReadyV2` trial since 2026-11-28T03:41Z; `Merge agent/diffgo` 9c6c61d38 |
| R12 Barbarian | not started | in flight | Leap rows in `dlvwave3` 555650c52, Whirlwind driver `dlvwhirl` 27324ad8f, `classapply` 83a010d59, scenarios `whirlscen`; epic card has no box ticked |
| R14 advisor | built-dark | trial | ledger `items.advisor` trial since 2026-11-28T03:41Z |
| PG5 pgclock | in flight | enabled | card Done, all boxes; `Merge agent/pgclock2` 2a0dbd60d |
| PG15 pgnightly | not started | in flight | `Merge agent/pgcards2` a65a48c0a, `tools/provingground/nightly.json`; card Done, repo side; no timer on any host (S) |
| PG16 pgearned | not started | in flight | same merge, `tools/provingground/earned/milestones.json`; card Proposed, one of three boxes |
| DLV3 press, Teleport, Charge, Leap, Whirlwind | not started | in flight | `internal/combat/press.go`, `move.go`, `whirl.go`; `dlvwave3`, `dlvwhirl`; four cards In Progress |
| BSMc bsmchk | not started | enabled | `internal/questspec/boardsm.go` rules 12 to 14 with tests; `Merge agent/bsmchk` 3392b8986; card Done |
| BSMd bsmdef | not started | in flight | `Merge agent/bsmprep` d894a57bf; card In Progress, no box |
| TOWN7 short-gold table | not started | built-dark | `internal/supply/priority.json`; `Merge agent/town7` d60306160; card Done |
| TOWN8 stage 5 route | not started | built-dark | `internal/town/route.go`; same merge; card Done |
| TOWN9 /api/townplan | not started | enabled | `internal/watch/townplanapi.go`; `townrest` b197ab183, `townwire` 5757b6333; both cards Done; not verified live |
| TOWN10 upkeep source | not started | built-dark | `internal/arbiter/upkeep.go`; `townrest`, `townwire`; card In Progress |
| TOWN11 cleanup | not started | in flight | `Merge agent/town11` 0ae6b91df (seam only, nothing deleted) |
| TOWN12 SME review | not started | enabled | `Merge agent/smetown` db2f6c04f, `townapply` 90dde084a |

Rows whose STATE stayed but whose text was wrong and is fixed: R5 (said rules 12 to 14 were not built), DLV2 (said aim-corpse and stance-pack had no merged commit), DLV1, R7, R11, TOWN6, BSM, PG10 (said cards Proposed that are In Progress or Done), R6 (pointed at the wrong dark row), F (the matrix card is Deferred), P5, P6, P9, R8, R10, Q5, MAP2, MAP3 (lab night results added). The "Next up", "Flips ready", "Built but dark" (two rows added), "Order of work" and plan snapshot sections were rewritten for today.

Tally after the refresh, from `python3 -I tools/roadmap/check.py` (run it for the current one): 84 rows; enabled 33, trial 12, built-dark 14, in flight 21, not started 3 (R13 Acts 3 to 5, PG11 mgonejob, DLVc dlv-cleanup), retired 1; 0 warnings. Before the refresh the same check gave not started 15, in flight 16 and nine warnings.

## What the check found that is not this page's to fix

- town12 (TASK-765): result says NOT DONE, the SME run is merged. inputl5 (TASK-989): both boxes ticked, status Proposed. pgfix (TASK-1093): merged, Proposed, no result (S handoff item 12). navbarracks (TASK-1102): `Merge agent/nf1` names it, card Proposed. The coordinator closes or corrects cards.
- TOWN7 against the lab night: `town.plan.gold-short-priorities` was RED and the scenario note says the priority table is not built, while town7 is merged and Done. One of the two is stale (U); townnight (TASK-1108) is the card to say which.
- T4: the ledger criterion of `bot.tickLogSec` cannot hold as worded (S `docs/probes/needs-owner-2026-11-28.md`); an owner row, not edited here. `agent/grad1` (not on main) graduates the knob, which will retire this row's knob cell when it merges.
- The review of 2026-11-28 (`docs/design/fable-review-2026-11-28/`, thirteen pages) has no section on the roadmap page. Its fix cards fix1 to fix8 are merged. It needs rows of its own, written by someone who reads the thirteen pages; proposed as one docs card, not done here.

## The drift check

`tools/roadmap/check.py` (stdlib, read-only, exit 0 unless `--strict`; test `python3 -I tools/roadmap/test_check.py`). It reads every table of the page with a `card` and a `STATE` column and warns:

- S1: STATE `not started` while a card of the row is Done or `Merge agent/<card>` is on `origin/main`.
- S2: STATE `not started` or `in flight` while every card named is Done with every acceptance box ticked. A Done card with an open box (a live check) keeps the row quiet, and so does `(open: why)` in the card cell.
- S3: the card cell names no known card, so the row cannot be checked.

Options: `--rows` (per row: cards found, Done, landed, and the tally of states), `--summary` (one line for CI), `--ref`, `--repo`. Limits (V, by construction): it does not see a card merged under another branch name while that card is not Done, a row marked `in flight` whose code is all merged but whose cards are open, or stale evidence text. The rule text is in the page's "How to keep this file true".

### Proposed wiring (not done: `tools/ci/check.sh` is outside this card's owns)

Next to the backlog drift line of `tools/ci/check.sh` (a warning only, like `drift.py --summary`):

```sh
# --- roadmap drift as a warning only (tools/roadmap/check.py exits 0) ---
python3 -I tools/roadmap/check.py --summary || true
python3 -I tools/roadmap/test_check.py >/dev/null 2>&1 || { echo "FAIL tools/roadmap/test_check.py"; fail=1; }
```

And one line for the merge step (ops repo): print `check.py --summary` in the queue result of every merge, so the coordinator sees the warning in the tick that caused it.
