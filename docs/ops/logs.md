# Where the logs are (sandbox container and dev)

> status: current (card logsink). Code: `internal/logx/sink.go`, `cmd/d2bot/main.go` (newLogger), `cmd/d2bot/opscmd.go` (`ops log`).

Read the bot's log without `docker logs`:

    docker exec <container> d2bot ops log --tail 200 --grep 'inv=|tick'

`--file PATH` or `D2BOT_LOG` override the default `/data/instances/main/logs/bot.log`. It reads the file and its rotations, needs no token and no operator socket, and works while the bot is down.

| log | where | what |
|---|---|---|
| bot operational log (JSON lines: `tick`, `inv=`, `quest board`, `task step`, `combat`, `nav.trace`) | stdout of `d2bot` AND `<data>/instances/<inst>/logs/bot.log` (+ `.1` to `.5`, 20 MiB each) | written by `newLogger` through `logx.Tee`; `D2BOT_LOG_FILE=0` turns the file off |
| game process (d2d) | `<data>/instances/<inst>/d2d/d2d.out` (rotated by the launcher) and `d2d/config/d2d/d2d.log` | V read in the running sandbox |
| picture tap | `/tmp/d2dview.log` in the container (lost on restart) | listen address and errors |
| invariant monitor and deaths | `<inst>/incidents/*.json`, `deaths.jsonl` | one file per incident |
| launcher (`/opt/didle/boot`) | container stdout/stderr, same pipes as `d2bot` | release start lines |
| `docker logs` | docker's json-file copy of container stdout | NOT reliable, see below |

## Why `docker logs` went silent (2026-11-27, V)

On the two sandbox containers `docker logs` ended at 2026-11-25 14:15Z (Doofer) and 2026-11-26 15:50Z (Questy) although the bots ran on and `d2bot` was restarted since. V: the container's json-file driver is the default (`LogConfig {json-file, {}}`, no `max-size`), the bot and the launcher share the same stdout pipes (`/proc/<pid>/fd/1`) so they still write, and the docker daemon journal shows a stream of `Error decoding log file: invalid character '\x00' looking for beginning of value`. The json-file log has a NUL-filled region (the host disk was at 99 percent; the real cause of the damage is U), and the reader stops there, so every later line exists in the file but `docker logs` cannot decode it. A container restart keeps appending to the same damaged file. The bot, the shim and the entrypoint were not at fault; they had no file of their own, so nothing else held the lines.

Fix in this repo: the bot now writes the same lines to `bot.log` itself (best effort per sink: a broken stdout cannot stop the file and a full disk cannot stop stdout). Recreating a container (or `docker logs` plus a `max-size` log option in the compose file, a deploy decision) also clears the damaged docker file. U: not verified live; needs a sandbox release that carries this change, then `d2bot ops log --tail 5` shows lines.
