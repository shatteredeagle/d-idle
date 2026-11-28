# Character file backups, startup repair and the disk guard

> status: current (card savesafe). Code: `internal/savesafe/`, `cmd/d2bot/savesafewire.go`. NOT verified live (unit tests with built character file images for two classes). Fork side: [docs/design/fork-save-atomic.md](../design/fork-save-atomic.md).

Why: d2d writes `<name>.d2s` in place (U, see the fork request), so a save on a full disk can leave a 0 byte file. On 2026-11-28 that cost a hero. The bot now keeps its own copies. Always on for a managed d2d, no knob.

## What the bot does

| Part | Rule |
|---|---|
| Rolling backup | Every 10 minutes of wall time and at every arrival in town the bot copies `<name>.d2s` plus `.key`, `.map`, `.ma0`.. and any `*.d2x` / `*.sss` (shared stash, name U) from the d2d save folder. |
| Only good files | The live file must have the signature `0xaa55aa55`, at least 765 bytes, a size field equal to its length and a correct checksum. A file that fails (also one d2d is writing at that moment) is skipped and logged once: `save backup: skipped`. |
| Only changes | A file equal to the newest backup is not copied again. |
| Rotation | The newest 12 sets are kept, plus the newest set of each of the last 7 UTC days. |
| Atomic | Each file is written to a temp file in the backup folder, synced and renamed. The `.d2s` of a set is written last; a set without it does not count. |
| Low disk | Under 200 MB free (statfs) nothing is written; logged once. The planned save point is refused too (`disk low`), because a save on a full disk is what truncates the file. The bot keeps running. |
| Startup repair | Before d2d starts: when the live `<name>.d2s` is missing, 0 bytes or invalid and a valid backup exists, the bad file is kept as `<name>.d2s.broken-<unix>` and the newest valid backup takes its place. Side files that are missing or empty come back from the same set. Log line at ERROR: `save repair: restored <file> from <backup>`. |
| Notices | A repair of the last 7 days and free space under 2 GB show in `d2bot ops status --why` (`notices`) and as a red bar at the top of the viewer (`/api/notices`). |

Progress between the backup and the damage is lost: at most about ten minutes of play or one trip.

## Where the files are

- Live: `<data>/instances/<id>/d2d/config/d2d/save/<name>.d2s` (sandbox container: `/data/instances/main/d2d/config/d2d/save/`).
- Backups: `<data>/instances/<id>/state/savebackups/<name>.<unix>.d2s`, side files as `<name>.<unix>.key` (`.map`, `.ma0`) and `<name>.<unix>.shared-<file>`.
- Last repair record: `savebackups/last-repair.json`.

`<unix>` is the copy time in seconds; `date -u -d @<unix>` prints it.

## Check (read only)

```
d2bot ops save check                      # every character, sandbox default paths
d2bot ops save check --char <name> --save <save folder> --state <state folder>
```

It prints one `live` line (path, size, `valid` or `INVALID: reason`), one `backup` line per set (newest first) and the notices. It needs no running bot and changes nothing. Exit 0: every live file is valid; 1: one is not; 2: usage.

## Restore by hand

Use this for an older backup than the newest, or when the bot is not to be started.

1. Stop the bot and d2d (stop the container). Never copy over a file a running d2d may write.
2. Run `d2bot ops save check --char <name>` and pick a `backup` line marked `valid`.
3. In the save folder keep the current file: `mv <name>.d2s <name>.d2s.broken-manual`.
4. Copy the backup: `cp <state>/savebackups/<name>.<unix>.d2s <save>/<name>.d2s`. When `.key` or `.map` are missing or empty, copy `<name>.<unix>.key` and `.map` the same way.
5. Run the check again (the `live` line must say `valid`), then start the container.

## Starting a hero over on purpose

The repair restores a missing file. To really start a new hero under the same name, move `savebackups/<name>.*` away before deleting the live file.

## Free space

`ops status --why` warns under 2 GB free below the state folder. Under 200 MB the bot stops writing backups and stops planned save points until space is back. Free space first; on a build host the Go build cache is the usual cause (`go clean -cache`).

## Limits and dead ends

- The copy is made while d2d may be in a game. The checksum catches a half-written file; d2d writing the same valid bytes twice is harmless.
- The item section is not parsed: a file with a valid checksum and wrong content passes.
- Only the managed d2d path is wired (`runD2D`); the Wine path keeps `internal/saves` snapshots (`cmd/d2saves`).
- A repair restores only the active hero (`--char`); other heroes in the folder are reported by `ops save check`, not repaired.
