# Play session on the LAN (TASK-43)

> status: current. Offline parts verified (V: `sh tools/play-session-test.sh`); the live run is NOT verified (U, checklist below).

One command brings up a hidden instance and starts the bot with `-ladder -watch -watch-video`, so a friend on the LAN can watch the game and answer the bot's questions from another device (watch screen: [research/watch-screen-design.md](research/watch-screen-design.md), video: [video-feed.md](video-feed.md)).

## Use
```
tools/play-session.sh --dry-run start      # print the exact commands, run nothing
tools/play-session.sh start                # start the session (default instance i4)
tools/play-session.sh --instance i3 --port 8891 start
tools/play-session.sh stop
```
- It prints the watch URL (`http://<worker host>:<port>/watch`) and the token FILE PATH, never the token. The token file is made on the worker host with mode 0600 before the bot starts, so the bot never prints a fresh token. Read the file there and add `?t=<token>` to the URL to share the link. The answer route listens on port + 1.
- Host, user and ports come from `tools/env.sh` / `local.env` (`D2_WORKER_HOST`, `D2_WORKER_USER`); nothing is hard-coded.
- Without the private ops checkout (instance bring-up and the live runner live there, [process/public-split.md](process/public-split.md)) it prints one line and exits 2. Only hidden instances i1 to i5 are accepted, never i0.

## Stop and cleanup
1. `tools/jobs.sh list`, then `tools/jobs.sh kill <job>` for the bot job (the session has a 60 minute cap).
2. `tools/play-session.sh --instance iN stop` (runs `tools/game.sh stop iN`; prints the same steps).
3. Rotate the token: delete the token file on the worker host; the next start makes a new one. Check `tools/jobs.sh list` is empty of your job and no `d2lod-live/<agent>` process is left.

## Live check (ready to run, not run)
1. `tools/play-session.sh --dry-run start`; read the commands.
2. Check the worker host capacity (`labrun <host> -- top -bn2 -d1`), the instance is free, `/tmp/d2lod-live.notice` absent.
3. `tools/play-session.sh start`; from another LAN device open the URL with the token: the screen loads, video shows, a decision can be answered on the answer port.
4. Confirm the token never appears in the script output or `ps`; stop and clean up as above.

Test: `sh tools/play-session-test.sh` (dry-run output and the refusal). It is not yet listed in `tools/ci/scripts.sh` (not owned by this card).
