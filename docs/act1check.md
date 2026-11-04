# Act 1 check: log lines and report section

Logging and reporting only; no behaviour change (a test compares the reader's view with and without a logger).
Contract: `docs/contracts/act1-campaign.md` ("Act 1 cleared record", 14 rows, tiers T1 save file, T2 memory, T3 proxy).

## Log lines (writer: cmd/d2bot/milestone.go, called from the quest reader in cmd/d2bot/task.go)

- `milestone name=<den|bloodraven|cain|tower (forgotten_tower quest)|tools|andariel|act2> word=0x<hex> clvl=<n> virtual=<min>`: once per
  word, on its first memory read as Done (`quest.Interpret`). Words: den_of_evil, sisters_burial_grounds,
  search_for_cain, forgotten_tower, tools_of_the_trade, sisters_to_the_slaughter (word 6), act1_done (word 7, `act2`).
  `clvl` is the level of the last save read (V by reading the code: the memory reader has no level here; -1 = no save
  read yet), so it lags until a Save and Exit. `virtual` is minutes since the reader's first read on the env clock
  (virtual under lockstep).
- `act1: andariel killed words 6=0x<hex> 7=0x<hex>`: once, when word 6 first reads Done. UNVERIFIED (U) which of
  6 and 7 the real game sets at the kill (hops.go gates on 6, dialog.go on 7): the first uncheated run must read both.

## Parser and report

`internal/d2dsoakstat/act1check.go` (`ParseAct1`, `Act1Check.Rows/Text`), printed by `d2dsoak report LOG...` after the
skill and area sections. It also reads existing lines: `boss run boss=Andariel killed=`, `memory quest words`,
`save read` (both `normalQuestWords`), `merc: claim verified`. A log without any new line prints `n/a` on every row.

## Rows the report decides

| Row | Verdicts |
|---|---|
| 2, 3 | FAIL when the last memory read shows the word not Done; else UNKNOWN (unspent points, merc class, Kashya gold are not in the log) |
| 4 | PASS: killed=true + word 6 milestone + a `save read` with word 6 Done; FAIL: killed=true without the word; UNKNOWN otherwise |
| 8 | PASS when both andariel and act2 milestone times exist, else UNKNOWN |
| 1, 5, 6, 7, 9 | UNKNOWN always (ledger, area 40 and byte 0xA8, alive and deaths, rejoin, pack/points: live dump) |
| 10 to 13 | not printed; need their own lines |

Never marked from proxy evidence (area, kill event, item): a quest counts only through a word.

## Needs a live uncheated run

The D8.c and D8.w records used cheat-set quest words, so their milestones prove nothing about the real game; the
ledger (row 1, `d2cheat ledger --since-run`) is the check. Run a fresh character without cheats, keep the bot log, run
`d2dsoak report <log>`, and read the `act1: andariel killed` line to settle word 6 vs 7.
