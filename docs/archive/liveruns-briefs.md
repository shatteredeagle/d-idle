# Agent briefs for the live batches (paste as the `prompt` of an Agent call with `subagent_type: "d2-live"`)

Companion of [liveruns.md](../liveruns.md) (rules, procedures) and [liveruns-checks.md](../liveruns-checks.md) (runbooks R0..R12). The d2-live agent
type already carries the standing procedure (worktree, status board, tags, commit trailer, report); these briefs add name, branch, instance,
the steps and the stop rules. MAIN = the launching agent. Launch a batch only when `tools/devhealth.sh` shows 1-min load <= 24, no
`/tmp/d2lod-live.notice`, and the instance is free (`tools/game.sh status iN`). Before pasting, MAIN does the bring-up of liveruns.md 3.1 for the
instance (start, helper, mute, first login) and says so in the brief, because AGENTS.md reserves launches to MAIN. Never run two batches on
one instance. Owned files of every batch: the doc named in each runbook's "record" line (one new dated section each), nothing else.

## Brief B0 (no game, any agent; or MAIN itself, 2 minutes)
```
Agent LRB0, branch agent/lrb0 (worktree ~/code/d2lod-wt/lrb0 from master). No game, docs only. Run R0 of docs/liveruns-checks.md: tools/devhealth.sh and
tools/game.sh reap-all --dry-run, compare the candidates with ps on the worker host (ssh worker.example 'ps -eo pid,args | grep -E "winedevice|wineserver"'). i4 is running
a 2 h soak (do not touch it); no candidate may be a pid of a live instance. Record one dated line in docs/leakhunt.md. Report under 150 words, commit hash.
```

## Brief B1: i2, Combi bosses + probes + Necki (about 90 min)
```
Agent LR2, branch agent/lr2 (worktree ~/code/d2lod-wt/lr2 from master), instance i2, characters Combi then Necki. MAIN has started i2 (game.sh start i2, helper, mute) and
logged Combi in. Read docs/liveruns.md (rules 1 to 7 and procedures 3.1 to 3.7) and docs/liveruns-checks.md R2 to R6, docs/boss.md s7.3, s10, s11, docs/probes.md s2 and s4, docs/summon.md W5/W6.
FIRST: export D2_INSTANCE=i2 and define L() { : "${D2_INSTANCE:?}"; tools/live.sh "$@"; } (an unlisted agent name otherwise drives i0 = Testy). Use agent name lr2 for runs.
Goal, in this order, one dated result each:
 1. R2 Countess without --nokite (bossrun --god "reason" --timeout 14m countess): grep the log for "pinned: evasive moves go nowhere"; pass = killed=true, exit 0.
 2. R5 piggyback: while R2/R3/R4 fight, start probes monflags and missiledump with --nolock and agent name lr2p; interpret the flag bits against tools/shot.sh screenshots.
 3. R3 Smith: bossrun smith; check the log for NO "preset 1/754 not found" and a "boss view: unique" for The Smith near (15119,4389); fallback steps in R3.
 4. R4 Andariel: town chores for potions, townlive buy 148 yps yps for antidotes, bossrun andariel; record A1 to A5 values.
 5. R6 Necki W5 (oog leave "start Necki"; save edit level 30 and skills 70/74/95 with a ledger reason; necrolive bind/farm/census). Necki is allow-listed, Druki and the others are NOT.
Cheats only through cmd/d2cheat with a reason; allow-list names only (Combi, Necki here). Stop rules: life < 25% twice, three hops fail, run over its limit, character dies, load > 26 for 5 minutes:
stop, record, move on; at most one retry per check. Run anything over 2 minutes with --bg and poll tools/jobs.sh tail every 20-30 s. After a killed god run use d2cheat restore.
Leave Combi/Necki standing in town, panels closed, then tools/game.sh stop i2 and check nothing of yours remains (tools/jobs.sh list lr2; pgrep -af d2lod-live/lr2). Record sections in docs/boss.md ("12. LIVERUNBOOK")
and docs/summon.md, flip the s11 checks, tag V only what you saw. Commit often. Report under 250 words: per check pass/fail with the numbers, what you did to the characters, how to bring i2 back.
```

## Brief B2: i1, Navi Act 2 chain (about 2 h)
```
Agent LR1, branch agent/lr1 (worktree ~/code/d2lod-wt/lr1 from master), instance i1, character Navi (Sorc 24, Act 2, all waypoints, Cube, Horadric Staff). MAIN has started i1 and logged Navi in.
Read docs/liveruns.md (rules, 3.1 to 3.7), docs/liveruns-checks.md R7 to R9, docs/act2.md (Live results 3 to 6 and "Next live chain"). FIRST: export D2_INSTANCE=i1 and define
L() { : "${D2_INSTANCE:?}"; tools/live.sh "$@"; } ; agent name lr1.
Goal: R7 (wp 46, exitto 69, probefar, tpto the orifice platform, place hst 152, then the portal into Duriel's Lair, Duriel, Tyrael), R8 (Summoner via Palace Cellar portal 298), R9 (the Act 2 rows not already
done per Navi's quest words), then wpdump (probes wpdump) and `d2cheat act Navi 3` first contact only if R7 to R9 are done and time remains.
Each leg is ONE act2live god chain (Navi dies in seconds otherwise; V). Keep the Staff off the cursor at run ends. Cheats only through d2cheat with reasons (Navi is allow-listed); save edits need oog leave / oog "start Navi".
Stop rules: two deaths in a leg, three failed hops, a run over its limit, load > 26 for 5 minutes. After a killed god run: d2cheat restore (Navi max life 290). Use --bg for anything over 2 minutes.
Leave Navi in Lut Gholein, panels closed, cursor empty; tools/game.sh stop i1; verify nothing of yours runs. Record in docs/act2.md "Live results 7" (V only what you saw, with area ids and coordinates).
Commit often. Report under 250 words.
```

## Brief B3: i3, Towni NPC menus and merc (about 1.5 h)
```
Agent LR3, branch agent/lr3 (worktree ~/code/d2lod-wt/lr3 from master), instance i3, character Towni (Sorc 20, all waypoints, Act 1). MAIN has started i3 and logged Towni in.
Read docs/liveruns.md and docs/liveruns-checks.md R10, docs/actbot.md checklist 1 and 2, docs/probes.md s3. FIRST: export D2_INSTANCE=i3 and define L() { : "${D2_INSTANCE:?}"; tools/live.sh "$@"; } ; agent name lr3.
Goal: record the NPC menu rows (townlive menu CLASS, screenshot each with tools/shot.sh OUT.png i3) and hire rows (townlive hire ROW) for Act 1 (Akara 148, Charsi 154, Kashya 150, Gheed 147), then for Acts 2 to 5 via
d2cheat act Towni N with the game at the menu (oog leave, edit, oog "start Towni"; no game restart). Run probes mercdump with --nolock while a dialog is open. Gold only through d2cheat with a reason
(Towni is allow-listed; Assi, Barby, Amma, Soaky are NOT). Do not hire more than needed (one hire per act; the gold cheat is logged).
Stop rules: any death, a panel that cannot be closed (Esc, then shot and report), load > 26. Leave Towni in a town, no panel; tools/game.sh stop i3. Record in docs/actbot.md (checklist 1, 2) and docs/merc-farm.md;
list every row list that differs from internal/quest/act*_menus.go as a fix request (you do not edit Go). Commit often. Report under 250 words.
```

## Brief B4: i4, soak gate verdict and 24 h (MAIN or a soak agent, not before about 12:50)
```
Agent LRS, branch agent/lrs. No new game start while the gate soak owns i4. Step 1 (read only): fetch the 2 h gate verdict (~/d2lod-live/soak/i4-20261029-104844/report.txt, tools/soak/soak.sh report) and
compare with docs/soak24.md s3; list failures with the fix section from s5. Step 2 only if PASS and the load is <= 12: follow docs/soak24.md s2 and docs/liveruns-checks.md R1 (preflight, start i4, watch every 10 minutes; do not touch
the game, no manual restart: it voids the run). Step 3: grep the bot log for the Act 1 chain (R11). Hand-off per soak24.md s6. Report under 300 words.
```
