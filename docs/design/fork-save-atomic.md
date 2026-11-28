# Fork request: write the character file atomically

> status: request to the d2d fork, not built (card savesafe). Nothing in the fork was changed by this card. The bot side that limits the damage until then is in [docs/ops/save-backups.md](../ops/save-backups.md).

## What happened (incident 2026-11-28)

The root disk of the worker host filled. At the next save the sandbox's d2d left the character file `<name>.d2s` at 0 bytes. The only `.bak` next to it was a level 1 file from days before. The hero's progress was lost.

Tags: V = seen on the damaged files after the incident (0 byte `.d2s`, old `.bak`). U = the fork's save code path was not read for this card; "written in place" is the conclusion from the 0 byte file, to be confirmed in the fork source by the card that takes this request.

## Why the bot cannot fix this alone

The bot now keeps rolling copies and repairs at startup (card savesafe). A copy is at most ten minutes or one town arrival old, so a truncated save still loses that much. Only the writer of the file can make the write itself safe.

## Requests

1. **Temp file, then rename.** Write the new character image to `<name>.d2s.tmp` in the same folder, never to `<name>.d2s` directly. Check every write and the close for errors. Replace the live file only by `rename(2)` after the temp file is complete.
2. **fsync before the rename.** `fsync` the temp file before the rename and `fsync` the folder after it, so a crash or power loss right after the save cannot leave a renamed but empty file.
3. **A failed write leaves the old file.** On any error (ENOSPC, EIO, short write) remove the temp file, keep the live file untouched, and report the failure on the control socket and in the log (one line naming the file and the errno). The save verb returns an error; the game keeps running.
4. **Keep the last good file.** Before the rename, move the current live file to `<name>.d2s.bak` only when it passes the load check (signature, size field, checksum). A file that fails the check never replaces a good `.bak`.
5. **Same rule for the side files.** `.key`, `.map`, `.ma0`.. and a shared stash file are written the same way (temp, fsync, rename).
6. **Refuse to load a bad file loudly.** A zero-length or checksum-failing `<name>.d2s` at startup is an error that names the file. It is never treated as "no save" and never overwritten by a newly created hero of the same name.
7. **Report free space.** Optional: the status reply carries the free bytes of the save folder, so the bot can read it without its own statfs.

## Acceptance for the fork card

- A test fills a small tmpfs, triggers a save and finds the previous `<name>.d2s` byte-identical and loadable.
- A test kills the process between write and rename and finds either the old or the new file, both valid.
- Starting with a 0 byte `<name>.d2s` exits with an error; no new hero is written.

## When this lands

The bot's rolling backup and startup repair stay (they also cover a deleted or damaged volume). The planned save point refusal on a low disk (`bot.Deps.DiskLow`) can then be relaxed; that is a decision for the card that verifies requests 1 to 3 live.
