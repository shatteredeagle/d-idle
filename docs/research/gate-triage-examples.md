# gate_triage.py: examples for the five Gate 1 runs

> status: current. Tool: `tools/baseline/gate_triage.py` (card gatetriage). Tags: D = computed from the minute archives; I = hint; not verified live.

```
python3 -I tools/baseline/gate_triage.py MINUTES.jsonl[.gz] | RUN-RECORD-DIR [--log bot-0.log] [--stash-assume N] [--xp-status CONCLUSIVE]
```
Exit code 0 pass, 1 fail, 2 inconclusive (run under 60 vmin, or the portal criterion not judged because there is no log text). Rules: the soak-gates spec with the specdelta delta (belt windows count only when pack plus stash gold reaches 30; one seed inside the XP tolerance passes with a note). Stability, fault recovery and the route guard are not judged (the run record has them). Baseline 9172 XP/h and tolerance 1834 are the d2d numbers of the records; override with `--xp-base` and `--xp-tol`.

The archives keep only the minute lines and a few events, so the log part is thin here (the archive events stand in for the log, `recovery=2` is the fault log line); with `--log bot-0.log` from the worker host the counts and the 10 lines around each window fill in. The portal criterion therefore shows INCONCLUSIVE and the overall verdict of a run without other failures is 2.

## Checked against the hand findings (D)
| run | tool | hand finding (docs/research/specdelta-rejudge.md, docs/research/xphshort.md) |
|---|---|---|
| 1939 | FAIL: funded 12 min window clvl 12, pack 30..38 | belt FAIL (funded), XP +9.5% |
| 2210 | FAIL: funded 37 min window clvl 9, pack 332..366, overlaps restarts; NOKILL 11 twice over the limit | belt FAIL and NOKILL 11 > 10 |
| 0053 | INCONCLUSIVE (all pass, portal not judged): poverty window clvl 2, XP -11.0% one seed | PASSED under the new rules |
| 0338 | INCONCLUSIVE: both windows poverty because the archive has no stashGold; with `--stash-assume 1418` the clvl 11 window is funded and the run FAILS | belt FAIL (clvl 11 window, stash 1418) |
| 0628 | FAIL on XP (7060, -23.0%); the one belt window is poverty (clvl 2); Cold Plains 22 percent of the minutes at clvl >= 10 at 5.1 XP/kill | XP gap is area choice (Cold Plains 229 minutes at 3.4 XP/kill) |

XP/h differs from the records by at most 2 (10042, 9756 vs 9754, 8162, 8750 vs 8751, 7060). Kills match d2dsoakstat (increases only).

Known limits: the 0338 and 0628 archives live on their run-record branches until merged; the example for the old archives cannot see the stash (the hint line says so); NOKILL windows inside town chores (clvl 6, 13) are listed but are not belt findings; the cause hints are rules of thumb (I).

## Wiring into gate-run.sh (ops repo, separate task)
After the collect step has the run directory (it holds `result/all-minutes.log` and `logs/bot-0.log`), one line: `python3 -I "$REPO/tools/baseline/gate_triage.py" "$RUN_DIR" | tee "$RUN_DIR/result/triage.txt"` (the directory form finds both inputs; its exit code is 0/1/2 and must not abort the collect, so append `|| true`). For `--status` the same line on the partial minute file gives a live view.

## Outputs

### gate1-20261102-1939

```
GATE TRIAGE gate1-20261102-1939-minutes.jsonl.gz: FAIL   1182 vmin (19.7 h), clvl 1->15, kills 7392, deaths 0, restarts 14, xp 197999
1 VERDICT (rules: specdelta; stability/fault/route guard not judged)
  Deaths (death rule)          PASS         0 deaths in 7392 kills (0.0 per 1000)
  True XP/h vs d2d baseline    PASS         10042 vs 9172 (+9.5%, tol 1834, d2dsoak INCONCLUSIVE): at or above the baseline
  No kill and no XP <= 10      PASS         longest 7 vmin (limit 10)
  Belt: funded windows <= 5    FAIL         1 funded window(s), longest 12; 0 poverty window(s) not failing
  Town-portal trip >= 1        INCONCLUSIVE not judged: no log (pass --log bot-0.log)
2 WINDOWS (D; belt > 5 vmin, no-kill > 7 vmin; minute-line numbers)
  446-457 (12m) clvl 12 Blood Moor(2) campaign/recovery pack 30..38 stash ? pot 0..0 hpMin 49% [funded (stash unknown)]
3 XP BY BAND AND AREA SHARE (clvl >= 10)
  1-6: 117m 10806 XP/h  7-9: 128m 16624 XP/h  10-12: 315m 10505 XP/h  13+: 622m 8309 XP/h
  clvl>=10: 937 minutes, XP/kill 23.4; top areas: Stony Field(4) 34% 20.6 XP/kill; Black Marsh(6) 13% 51.7 XP/kill; Cold Plains(3) 11% 8.5 XP/kill
4 LOG TEXT (no log; non-minute lines of the archive)
  town trip emergency=true=0  shop: bought=0  walking the level chain=0  no activated waypoint=0  waypoint to town failed=0  back to field attempt=0  tome counted==0  merc hired/revived=0  safety drink portal cancelled=0  poverty 
  -- around window 446-457 (funded (stash unknown)):
     no log line within 10 vmin (the archive keeps only some events; pass --log bot-0.log)
5 LIKELY CAUSE (I: rules of thumb, check the evidence line)
  - funded window with trips not firing: window 446-457: pack+stash reached 38 (pack 30..38); no town minute inside the window (outside town by definition)
  - arrival in town without 'shop: bought': 1 of 97 town visits ended with no healing potion despite pack+stash gold >= 30; first: in town minute 246, out at 247, gold 770
  - fixed-area farming at clvl >= 12 in a low-XP area: Cold Plains(3): 106 minutes at clvl>=10, 8.5 XP/kill against 23.4 for the run (docs/research/xphshort.md)
(exit 1)
```

### gate1-20261102-2210

```
GATE TRIAGE gate1-20261102-2210-minutes.jsonl.gz: FAIL   1147 vmin (19.1 h), clvl 1->15, kills 7132, deaths 0, restarts 13, xp 186684
1 VERDICT (rules: specdelta; stability/fault/route guard not judged)
  Deaths (death rule)          PASS         0 deaths in 7132 kills (0.0 per 1000)
  True XP/h vs d2d baseline    PASS         9756 vs 9172 (+6.4%, tol 1834, d2dsoak INCONCLUSIVE): at or above the baseline
  No kill and no XP <= 10      FAIL         longest 11 vmin (limit 10)
  Belt: funded windows <= 5    FAIL         1 funded window(s), longest 37; 0 poverty window(s) not failing
  Town-portal trip >= 1        INCONCLUSIVE not judged: no log (pass --log bot-0.log)
2 WINDOWS (D; belt > 5 vmin, no-kill > 7 vmin; minute-line numbers)
  218-254 (37m) clvl 9 Dark Wood(5) campaign/farming pack 332..366 stash ? pot 0..0 hpMin 19% [funded (stash unknown)] overlaps restart@235,restart@250
  NOKILL 227-236 (10m) clvl 9 Dark Wood(5) campaign/other gold 332+0 hpMin 100% overlaps restart@235
  NOKILL 241-251 (11m) [over limit] clvl 9 Dark Wood(5) campaign/other gold 349+0 hpMin 58% overlaps restart@250
  NOKILL 408-415 (8m) clvl 11 Cold Plains(3) campaign/farming gold 66+0 hpMin 100%
  NOKILL 651-661 (11m) [over limit] clvl 13 Rogue Encampment(1) campaign/town chores gold 272+0 hpMin 54%
3 XP BY BAND AND AREA SHARE (clvl >= 10)
  1-6: 124m 10416 XP/h  7-9: 157m 13421 XP/h  10-12: 293m 11190 XP/h  13+: 573m 7906 XP/h
  clvl>=10: 866 minutes, XP/kill 23.4; top areas: Stony Field(4) 23% 19.4 XP/kill; Cold Plains(3) 21% 6.2 XP/kill; Black Marsh(6) 18% 36.3 XP/kill
4 LOG TEXT (no log; non-minute lines of the archive)
  town trip emergency=true=0  shop: bought=0  walking the level chain=0  no activated waypoint=0  waypoint to town failed=0  back to field attempt=0  tome counted==0  merc hired/revived=0  safety drink portal cancelled=0  poverty 
  -- around window 218-254 (funded (stash unknown)):
     no log line within 10 vmin (the archive keeps only some events; pass --log bot-0.log)
5 LIKELY CAUSE (I: rules of thumb, check the evidence line)
  - funded window with trips not firing: window 218-254: pack+stash reached 366 (pack 332..366); no town minute inside the window (outside town by definition)
  - fixed-area farming at clvl >= 12 in a low-XP area: Cold Plains(3): 185 minutes at clvl>=10, 6.2 XP/kill against 23.4 for the run (docs/research/xphshort.md)
(exit 1)
```

### gate1-20261103-0053

```
GATE TRIAGE gate1-20261103-0053-minutes.jsonl.gz: INCONCLUSIVE   1239 vmin (20.6 h), clvl 1->14, kills 6815, deaths 1, restarts 15, xp 168717
1 VERDICT (rules: specdelta; stability/fault/route guard not judged)
  Deaths (death rule)          PASS         1 deaths in 6815 kills (0.1 per 1000)
  True XP/h vs d2d baseline    PASS         8162 vs 9172 (-11.0%, tol 1834, d2dsoak INCONCLUSIVE): one seed, inconclusive
  No kill and no XP <= 10      PASS         longest 8 vmin (limit 10)
  Belt: funded windows <= 5    PASS         0 funded window(s), longest 0; 1 poverty window(s) not failing
  Town-portal trip >= 1        INCONCLUSIVE not judged: no log (pass --log bot-0.log)
2 WINDOWS (D; belt > 5 vmin, no-kill > 7 vmin; minute-line numbers)
  6-12 (7m) clvl 2 Blood Moor(2) campaign/recovery pack 10..16 stash ? pot 0..0 hpMin 23% [poverty (stash unknown)]
  NOKILL 603-610 (8m) clvl 12 Cold Plains(3) campaign/farming gold 279+0 hpMin 100% overlaps restart@610
  deaths: min 316 clvl 10 Cold Plains(3)
3 XP BY BAND AND AREA SHARE (clvl >= 10)
  1-6: 174m 7691 XP/h  7-9: 115m 17806 XP/h  10-12: 437m 8677 XP/h  13+: 513m 6562 XP/h
  clvl>=10: 950 minutes, XP/kill 22.6; top areas: Stony Field(4) 29% 28.0 XP/kill; Cold Plains(3) 18% 8.7 XP/kill; Blood Moor(2) 11% 1.1 XP/kill
4 LOG TEXT (no log; non-minute lines of the archive)
  town trip emergency=true=0  shop: bought=0  walking the level chain=0  no activated waypoint=0  waypoint to town failed=0  back to field attempt=0  tome counted==0  merc hired/revived=0  safety drink portal cancelled=0  poverty 
  -- around window 6-12 (poverty (stash unknown)):
     no log line within 10 vmin (the archive keeps only some events; pass --log bot-0.log)
5 LIKELY CAUSE (I: rules of thumb, check the evidence line)
  - arrival in town without 'shop: bought': 1 of 101 town visits ended with no healing potion despite pack+stash gold >= 30; first: in town minute 493, out at 494, gold 47
  - fixed-area farming at clvl >= 12 in a low-XP area: Cold Plains(3): 174 minutes at clvl>=10, 8.7 XP/kill against 22.6 for the run (docs/research/xphshort.md)
  - fixed-area farming at clvl >= 12 in a low-XP area: Blood Moor(2): 109 minutes at clvl>=10, 1.1 XP/kill against 22.6 for the run (docs/research/xphshort.md)
  - poverty only: 1 poverty window(s), gold under 30: the early game, no potion could be bought (not a bot defect by itself)
(exit 2)
```

### gate1-20261103-0338

```
GATE TRIAGE gate1-20261103-0338-minutes.jsonl.gz: INCONCLUSIVE   1205 vmin (20.1 h), clvl 1->14, kills 6751, deaths 0, restarts 13, xp 175908
1 VERDICT (rules: specdelta; stability/fault/route guard not judged)
  Deaths (death rule)          PASS         0 deaths in 6751 kills (0.0 per 1000)
  True XP/h vs d2d baseline    PASS         8750 vs 9172 (-4.6%, tol 1834, d2dsoak INCONCLUSIVE): one seed, inconclusive
  No kill and no XP <= 10      PASS         longest 7 vmin (limit 10)
  Belt: funded windows <= 5    PASS         0 funded window(s), longest 0; 2 poverty window(s) not failing
  Town-portal trip >= 1        INCONCLUSIVE not judged: no log (pass --log bot-0.log)
2 WINDOWS (D; belt > 5 vmin, no-kill > 7 vmin; minute-line numbers)
  6-12 (7m) clvl 2 Blood Moor(2) campaign/recovery pack 10..16 stash ? pot 0..0 hpMin 23% [poverty (stash unknown)]
  330-336 (7m) clvl 11 Cave 2(13) campaign/recovery pack 3..11 stash ? pot 0..0 hpMin 25% [poverty (stash unknown)]
3 XP BY BAND AND AREA SHARE (clvl >= 10)
  1-6: 176m 7618 XP/h  7-9: 123m 17111 XP/h  10-12: 310m 10509 XP/h  13+: 596m 7084 XP/h
  clvl>=10: 906 minutes, XP/kill 23.8; top areas: Stony Field(4) 25% 19.2 XP/kill; Black Marsh(6) 15% 56.6 XP/kill; Cold Plains(3) 14% 4.7 XP/kill
4 LOG TEXT (no log; non-minute lines of the archive)
  town trip emergency=true=0  shop: bought=0  walking the level chain=0  no activated waypoint=0  waypoint to town failed=0  back to field attempt=0  tome counted==0  merc hired/revived=0  safety drink portal cancelled=0  poverty 
  -- around window 6-12 (poverty (stash unknown)):
     no log line within 10 vmin (the archive keeps only some events; pass --log bot-0.log)
  -- around window 330-336 (poverty (stash unknown)):
     no log line within 10 vmin (the archive keeps only some events; pass --log bot-0.log)
5 LIKELY CAUSE (I: rules of thumb, check the evidence line)
  - fixed-area farming at clvl >= 12 in a low-XP area: Cold Plains(3): 131 minutes at clvl>=10, 4.7 XP/kill against 23.8 for the run (docs/research/xphshort.md)
  - poverty verdict rests on pack gold only: window 330-336 at clvl 11: no stashGold in this archive (rule: stash 0); rerun with --stash-assume N, e.g. the run record's stash figure (0338: clvl 11 window, stash 1418, funded)
(exit 2)
```

### gate1-20261103-0628

```
GATE TRIAGE gate1-20261103-0628-minutes.jsonl.gz: FAIL   1473 vmin (24.6 h), clvl 1->14, kills 8589, deaths 0, restarts 21, xp 173503
1 VERDICT (rules: specdelta; stability/fault/route guard not judged)
  Deaths (death rule)          PASS         0 deaths in 8589 kills (0.0 per 1000)
  True XP/h vs d2d baseline    FAIL         7060 vs 9172 (-23.0%, tol 1834, d2dsoak INCONCLUSIVE): below the baseline minus the tolerance
  No kill and no XP <= 10      PASS         longest 8 vmin (limit 10)
  Belt: funded windows <= 5    PASS         0 funded window(s), longest 0; 1 poverty window(s) not failing
  Town-portal trip >= 1        INCONCLUSIVE not judged: no log (pass --log bot-0.log)
2 WINDOWS (D; belt > 5 vmin, no-kill > 7 vmin; minute-line numbers)
  6-12 (7m) clvl 2 Blood Moor(2) campaign/recovery pack 10..16 stash 0..0 pot 0..0 hpMin 23% [poverty]
  NOKILL 151-158 (8m) clvl 6 Rogue Encampment(1) campaign/town chores gold 115+0 hpMin 100%
3 XP BY BAND AND AREA SHARE (clvl >= 10)
  1-6: 185m 7212 XP/h  7-9: 158m 12887 XP/h  10-12: 283m 11845 XP/h  13+: 847m 4619 XP/h
  clvl>=10: 1130 minutes, XP/kill 17.6; top areas: Stony Field(4) 39% 20.6 XP/kill; Cold Plains(3) 22% 5.1 XP/kill; Rogue Encampment(1) 8% 20.3 XP/kill
4 LOG TEXT (no log; non-minute lines of the archive)
  town trip emergency=true=0  shop: bought=0  walking the level chain=0  no activated waypoint=0  waypoint to town failed=0  back to field attempt=0  tome counted==0  merc hired/revived=0  safety drink portal cancelled=0  poverty 
  -- around window 6-12 (poverty):
     no log line within 10 vmin (the archive keeps only some events; pass --log bot-0.log)
5 LIKELY CAUSE (I: rules of thumb, check the evidence line)
  - fixed-area farming at clvl >= 12 in a low-XP area: Cold Plains(3): 257 minutes at clvl>=10, 5.1 XP/kill against 17.6 for the run (docs/research/xphshort.md)
  - poverty only: 1 poverty window(s), gold under 30: the early game, no potion could be bought (not a bot defect by itself)
(exit 1)
```
