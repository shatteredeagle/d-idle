# Live-run runbook: order, rules, shared procedures (agent LIVERUNBOOK, 2026-10-29)

> status: Game.exe live-run runbook; d2d is the primary backend (ADR 0009, 2026-10-30; docs/plan/STATUS.md), Game.exe is frozen: these runs only happen for the retirement soak or an orig-decision probe (cards gameexe-retire-soak, orig-decision-probes). d2d runs use tools/d2dslot.sh and docs/d2d-bot.md.

Docs only; nothing here was run live. Written while the worker host was too loaded for new live slots (1-min load 31 at 11:11, BACKLOG guard 24). Aim:
when a slot opens, a d2-live agent can paste a brief and go. Tags: V = read from a script, table or the worker host filesystem today (path given),
S = a doc's claim not re-checked, U = not verified (no live run). Target: classic 1.14d. Companion files:
[liveruns-checks.md](liveruns-checks.md) (one runbook per check, R0..R12) and [liveruns-briefs.md](liveruns-briefs.md) (one paste-ready Agent
prompt per batch). Priorities come from [INDEX.md](INDEX.md) section 2; this file does not replace it.

## 1. State of the lab when this was written (V = read-only ssh to the worker host, 11:11 to 11:30)

- The worker host 1-min load 31/16 cores. A 2 h gate soak is ALREADY RUNNING on i4 (character Soaky, `d2bot --managed --instance i4 --goals campaign
  --stop-game --run-for 120m`, started 10:48, `d2soak` run dir `~/d2lod-live/soak/i4-20261029-104844`, unit `d2soak-i4`). It ends about 12:50.
  i4 is therefore NOT free; Game.exe runs for i4 only. i1, i2, i3, i5 have no Game.exe: every batch below starts its instance.
- `/tmp/d2lod-live.notice` absent (no restart in progress).
- Characters (save dir `~/Games/d2lod-inst/iN/proton/pfx/drive_c/users/steamuser/Saved Games/Diablo II/`; state from a copy parsed with
  `d2cheat qdump`, V; all Normal, softcore):

| inst | character (class, clvl) | state | cheat allow-list |
|---|---|---|---|
| i1 | Navi (Sorc 24) | Act 2 (difficulty byte 0x81), all 39 waypoints, Act 1 words done, Act 2 words partly set | yes |
| i1 | Palli (Pal 13) | Act 1, wp 1 | yes |
| i2 | Combi (Sorc 30) | Act 1, all waypoints, quest word 3 = 0x000c (Tools of the Trade started), word 1 = 0x8001 | yes |
| i2 | Necki (Nec 15) | Act 1, wp 1 | yes (built in and in `~/.config/d2lod/cheat-allow.txt`) |
| i2 | Druki (Dru 4), Necci (Nec 20), Pally (Pal 4), Palli (Pal 13) | Act 1 | Druki, Necci, Pally NO; Palli yes |
| i3 | Towni (Sorc 20) | Act 1 (byte 0x80), all waypoints | yes |
| i3 | Assi (Ass 4), Barby (Bar 5), Amma (Ama 20) | Act 1 | NO; Amazi (Ama 3) yes |
| i4 | Booti (Sorc 12, wp 7, word 1 = 0x9001), Soaky (Sorc 7) | Act 1 | Booti yes, Soaky NO |
| i5 | Multi (Sorc 30) | Act 1, all waypoints | yes |

The allow-list is `internal/cheat/policy.go` `Allowed` (Testy, Botty, Navi, Combi, Towni, Booti, Multi, Palli, Palli2, Amazi, Necki) plus
`D2CHEAT_ALLOW` and `~/.config/d2lod/cheat-allow.txt` (V: that file on the worker host contains only `Necki`). A cheat on any other name is refused by
the tool. Do NOT add names yourself: adding one needs the user's say-so (git history shows Necki, Amazi, Palli were added "user approved").
Consequence: the Druid and Assassin checks (R12) cannot be cheated until the user approves Druki and Assi.

## 2. Rules that bite (V from the scripts, plus AGENTS.md)

1. `tools/live.sh <agent> ...` takes the instance from `tools/instances.conf`; an UNLISTED agent name silently becomes **i0 = Testy on the
   user's desktop** (live.sh line `inst="${inst:-i0}"`). Every runbook therefore sets `D2_INSTANCE` and calls live.sh through this guard:
   ```
   export D2_INSTANCE=i2                       # the instance of the batch
   L() { : "${D2_INSTANCE:?set D2_INSTANCE}"; tools/live.sh "$@"; }
   ```
   Use a fresh agent name per batch (`lr1`, `lr2`, `lr3`): the binary is shipped to `~/d2lod-live/<agent>/<pkg>`, so two runs with the same
   agent and package at once would overwrite each other. The lock is per instance, not per agent.
2. Order of flags: `tools/live.sh [--bg] [--max SECS] [--nocap] [--nolock] <agent> <pkg> args...` (flags first). Foreground limit 15 min,
   `--bg` 60 min. Anything over about 2 min: `--bg` and poll `tools/jobs.sh tail <agent>/<ts>-<pkg> 40` every 20-30 s.
   `--bg` prints `job <agent>/<timestamp>-<pkg>`; `tools/jobs.sh list <agent>`; `tools/jobs.sh kill <job>` (a killed `god` run leaves max life
   huge: run `d2cheat restore`, section 3.6).
3. `--nolock` skips the instance lock. It exists for short read-only tools next to a run that holds the lock. The probes only read memory
   (probes.md), so R5 uses it next to a `bossrun`; never use it for anything that sends input (U that two input sources never collide).
4. Starting and stopping games: AGENTS.md says only multi/OOG tooling and the main agent start a game and the d2-live agent stops its own
   instance at the end. In every brief below "MAIN" runs `tools/game.sh start iN`, `tools/game.sh helper iN` and the first OOG login; the
   d2-live agent may use `oog leave` / `oog "start NAME"` to switch characters on the same running game (V in docs/multi.md and BOSSLIVE2) and
   runs `tools/game.sh stop iN` at the end (never i0, never another instance).
5. Load: start a batch only when `tools/devhealth.sh` shows 1-min load <= 24 (BACKLOG; the user may raise it to 28). The BOSSLIVE2 session used
   26 as the abort threshold. The soak preflight refuses above 12 (`D2_SOAK_MAXLOAD`, V `tools/soak/soak24.sh`), so the 24 h soak needs a quiet night.
6. Never screenshot the worker host's desktop: `tools/shot.sh OUT.png iN` only. Never read credentials. Do not touch `i0`, `Testy`.
7. Death: an unprotected Navi/Combi dies in seconds in Act 2/Act 1 elites (V act2.md H4, boss.md s10). Each leg is one `god` chain; after a
   death `key esc` respawns in town with 0 gold and life 0 until a belt potion is drunk (V act2.md H4), then `heal` works.

## 3. Shared procedures (copy these)

All commands run from a worktree of this repo on the head box (`cd ~/code/d2lod-wt/<you>`). `L` is the guard from rule 1.

### 3.1 Bring an instance up (MAIN)
```
tools/devhealth.sh                       # load, disk, instances, jobs; abort if load > 24 or /tmp/d2lod-live.notice exists
tools/game.sh status iN                  # expect: not running
tools/game.sh start iN && tools/game.sh helper iN
tools/mute-d2.sh                         # mandatory after every launch
L lr<N> oog "start <Char>"               # logs in (first time ever: "create <Char> sorceress"); about 15 s
L lr<N> oog who                          # expect player="<Char>" class=.. level=.. area=1
```
Expected V (docs/multi.md): the whole recover cycle is about 15 s on i5. If `oog start` fails: `tools/shot.sh /tmp/x.png iN`, read it,
then `tools/game.sh stop iN` and retry once; a second failure is a blocker (report, do not loop).

### 3.2 Switch character on a running game
```
L lr<N> oog leave "start <Other>"        # Save and Exit to the menu, then log in as Other
```
The menu is also where save edits are allowed.

### 3.3 Cheats (allow-listed names only, every cheat needs a reason; see docs/cheats.md)
```
L lr<N> d2cheat --reason "R2: Countess retest needs gold" --actor lr<N> gold 20000      # memory, game running; only if the stat exists
L lr<N> d2cheat --reason "..." --actor lr<N> level 30
L lr<N> d2cheat --reason "..." --actor lr<N> stats vit=100 ene=50 points=0
L lr<N> oog leave                                                                       # save edits need the game at the menu:
L lr<N> d2cheat --reason "..." --actor lr<N> save Combi gold=20000 maxlife=969          # keys: level exp gold stashgold statpoints skillpoints str dex vit ene maxlife maxmana skills=ID:LVL,.. waypoints=MASK difficulty= act=
L lr<N> d2cheat --reason "..." --actor lr<N> waypoints Combi all
L lr<N> d2cheat --reason "..." --actor lr<N> act Navi 3                                  # start in act 3 town
L lr<N> oog "start Combi"
L lr<N> d2cheat ledger --since 2h                                                        # read-only: what was done
```
A fresh character has no gold/exp entries in memory (`gold` fails "stat not present"): use the save edit then. Items cannot be created by any
cheat (V docs/cheats.md); potions come from `townlive`, runes/gems/Cube only from the game.

### 3.4 Read-only look
```
L lr<N> d2cheat show                     # life, mana, exp, gold, stat/skill points
L lr<N> act2live state                   # player, hostiles, pack (any act; act2live also has units, objects, presets)
tools/shot.sh /tmp/lr<N>-1.png iN        # then Read the png
```
Offline save look without the game (V, ran today): `scp` the `.d2s` to scratch and `d2cheat qdump FILE` prints class, level, difficulty bytes,
waypoint masks and every non-zero quest word.

### 3.5 Leaving the character
Standing in town, no panel open (`L lr<N> act2live key esc` twice; `L lr<N> townlive state` shows the area), cursor empty (a Staff or
item on the cursor survives the run end: put it in the pack first), nothing running (`tools/jobs.sh list lr<N>`), then:
```
tools/game.sh stop iN                    # SIGTERM Game.exe, wineserver -k, Xvfb, reaps orphan winedevice; never i0
tools/jobs.sh list lr<N> ; ssh worker.example 'pgrep -af "d2lod-live/lr<N>"'    # expect nothing of yours
tools/game.sh reap-all --dry-run         # R0: lists leftovers of i1..i9; expect none for your instance
```
Say in the report that you stopped it and that MAIN brings it back with 3.1.

### 3.6 God mode leftovers
`act2live god N ...`, `bossrun --god REASON` and `d2cheat god on` raise max life to the 21-bit limit and restore it when the run ends by itself.
After `jobs.sh kill`, SIGKILL or a crash: `L lr<N> d2cheat --reason "restore max life after killed god run" --actor lr<N> restore` (V in
act2.md Y7 / boss.md s10; `restore --maxlife N` forces a value, Combi 969, Navi 290).

### 3.7 Recording
One dated line per check into the doc named in its runbook, tags V/U, instance, character, commit hash, exact command. Cheats are already in the
ledger; mention reason and count. Update the INDEX row status only through the main agent (INDEX.md is its file); put the delta in your report.

## 4. Order and batching (minimises instance starts: 3 starts for 11 checks)

| batch | instance | checks (see liveruns-checks.md) | start cost | why this order |
|---|---|---|---|---|
| B0 | none | R0 reap dry-run, `tools/devhealth.sh` | 0 | no game; do it first, it also gates the rest |
| B1 | i2 | R2 Countess, R3 Smith, R4 Andariel, R5 probes piggyback, R6 Necki W5 | 1 start; Combi then `oog leave` + Necki | the three bosses share one bring-up (gold, potions, antidotes); probes ride the fights; Necki is a second login, not a second start |
| B2 | i1 | R7 Duriel chain, R8 Summoner, R9 rows F1 F2 F3 C1, then the Act 3 wpdump | 1 start | Navi already has Act 2 state, all waypoints, Cube and Staff; one `god` chain per leg |
| B3 | i3 | R10 NPC menus, merc rows, mercdump | 1 start (Act 1 first, then `act Towni 2/3/4/5` save edits, no restart of the game process) | Towni is the town character |
| B4 | i4 | R1 read the running 2 h gate result, then the 24 h soak; R11 mine the soak log for the Act 1 chain | 0 extra starts (the soak starts it itself) | the soak is a campaign run; wait for the gate (about 12:50), needs load <= 12 |
| B5 | any | R12 Druid and Assassin | 1 start, blocked | needs user approval of two allow-list names, or honest play |

B1 first: the Countess, Smith and Andariel rows are INDEX P0 2 to 4 and gate the alpha, and Combi (clvl 30, all waypoints) needs the least
bring-up. B2 second (P1 8 to 10). Run at most ONE live batch at a time while load > 20: each instance costs about 1.05 cores (docs/multi.md).
B4's gate and B1 can overlap only if load stays <= 24.

## 5. Not covered here (and why)
INDEX 6 (quest bit meanings, i5 save diffs), 7 (death drill), 13 to 16 (Act 3 to 5 first contact; start them from B2 once R7 to R9 are done
with `act Navi 3/4/5`, checks in [actbot.md](actbot.md)), 17 and 25 to 27 (need runes, gems or a Cube that no cheat can create), 29 to 36
(infrastructure, no character state). R0 and R1 are the only infrastructure checks included.
