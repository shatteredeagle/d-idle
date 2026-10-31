# WINECLEAN: prefix hygiene (agent wineclean)

Defects found in G1 attempts 1-3 (docs/g1-log.md), all fixed offline; none exercised on a live game.

- Reaper deadlock (V: The worker host had 5-7 orphaned winedevice.exe up to 11 h old, `game.sh stop` reported them and killed none).
  The reaper refused while ANY wineserver of the prefix ran, but the lingering server is kept alive by those orphans.
  Now (`internal/launch/reap.go`, `tools/winereap.sh`): if the server is still there after 10 s and every other process of the
  prefix (exact WINEPREFIX, our uid) is in the orphan set {winedevice, explorer, services, rpcss, plugplay, svchost} the orphans are
  SIGTERMed, then SIGKILLed after 3 s. Any other client (d2input.exe, Game.exe) keeps the old refusal. Dry run only reports.
- `launch.CleanPrefix` / `Game.PrepareLaunch`: with no Game.exe of the instance running (i1..i9 only, never i0, D2BOT_REAP=0 turns
  it off): `wineserver -k`, wait <= 10 s, SIGTERM then SIGKILL of the orphan set. It ends the instance's helper too, so d2bot calls it
  before `helper.Ensure` at managed start and in the recycler's Start.
- `Game.Launch` refuses to launch while a network mount is hung and, after the process appears, requires the Diablo II window
  (helper `info` ok) within 45 s (U: typical time not measured) else stops the game and fails with the reason.
- Hung mounts: The NAS rebooting made /mnt/media 'Host is down' and Wine stalled at startup (V, coordinator finding). `soak24.sh
  preflight` FAILs with "mount X is hung (host down): Wine will stall" (`tools/soak/mounts.sh`); Launch errors carry the same text.
  The earlier theory that a stale wineserver caused the 'no window' failures was wrong; the stale wineserver is a separate defect.
- `soak24.sh stop <iN>` waits (90 s) for the bot to exit, then runs `tools/game.sh stop <iN>` (`tools/soak/stopclean.sh`). It never
  cleans under a live bot. `soak.sh stop` only stops the read-only sampler and is unchanged.
- Dropped by the coordinator: an extra retry before fatal.
