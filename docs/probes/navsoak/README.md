# navsoak: long hidden soak on main with the NAVSTALL fix (2026-10-31)

> status: handoff for the analysis agent. Soak started 2026-10-31 02:29:33 local (worker host clock), hard end 06:29:33 (240 wall minutes; registry TTL 15000 s).

## What it is
- Goal: confirm the combat livelock (nav stall, "combat crowd: stepping away" + "combat move failed ... nav: no path") is gone and see how far the Act 1 campaign gets. The earlier stream soak (Viewy, seed 5) hung on it, probably on a pre-NAVSTALL binary (unverified).
- Built from origin/main 279a163, which contains the NAVSTALL merge 85b429d (`git merge-base --is-ancestor 85b429d origin/main` true) and the watchdog string `combat no progress, holding` in internal/combat/fight.go. Bot binary `bin/d2bot.navsoak` (sha256 prefix 51c9bb0c07da), d2d `d2d.lab6`.
- Fresh uncheated Sorceress "Navy", seed 5 (same as Viewy), `--backend d2d --managed --d2d-class 4 --toggle autoloot=off,autoparty=off --d2d-args "--lockstep" --goals campaign --run-for <rest of 240m>`, `D2D_RUN=on` (run policy), nice 10, no viewer/tap, no stream. Lockstep pace is about 18x virtual (6.4 virtual h after 21 wall min).
- Started as `D2D_AGENT=navsoak bin/d2dslot.sh run -w 600 -t 15000 -- bash bin/navsoak-sup.sh` (slot 0, registered under agentrun as agent `navsoak`).
- Supervisor `bin/navsoak-sup.sh`: if the bot exits (watchdog "runner loop made no progress, exiting for a restart" or any crash) it restarts, up to 3 times, with `--d2d-class -1 --clean-start` (reload the saved Navy, fresh game), one line per restart in the supervisor log, and exits at the time limit.

## Where (on the worker host, relative to `~/d2d-lab/navsoak/`)
- `logs/supervisor.log` (start, launches, restarts with reason, end)
- `logs/bot-0.log` (first run), `bot-1.log` .. `bot-3.log` after restarts; the last line is `exit <code>`
- `data/` the bot data dir (character saves, state); `bin/` the shipped binaries and scripts

## Analysis (read-only; run on the worker host or via ssh)
```
cd ~/d2d-lab/navsoak
cat logs/supervisor.log
bin/agentreap.sh --list                                   # still running?
grep -c 'combat no progress, holding' logs/bot-*.log      # watchdog storm check (0 so far)
grep -c 'stepping away' logs/bot-*.log
grep '"msg":"minute"' logs/bot-0.log | tail -3            # clvl, xp, kills, deaths, area, step
grep '"msg":"pace"' logs/bot-0.log | tail -1              # virtual vs wall
grep -E 'died|death|failsafe|panic|runner loop made no progress' logs/bot-*.log | tail
```
Minute lines also give potions, gold, restarts, hpMinPct. Compare with docs/d2d-soak-full-fix.md.

## Stop
`tools/agentreap.sh navsoak` (on the worker host; takes the supervisor, bot and d2d down together and releases slot 0).

## First 21 minutes (at 02:50, V)
Virtual 6.4 h, clvl 8, 1153 kills, 1 death, 0 "combat no progress, holding", 86 "stepping away" lines, log advancing (13.8k lines), registry entry alive, no supervisor restart yet (the minute line's `restarts` field counts in-game restarts, not supervisor ones).
