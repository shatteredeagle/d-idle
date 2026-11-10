# Reset the play sandbox to a fresh hero

> status: CURRENT. Task: TASK-263 (devreset). Dev tooling for a DEV container such as `d2play-20261108` (docs/dev-update.md). Tags: V = run in the scratch drill below (2026-11-10, worker host, a throwaway container with fake state, no game); S = read from the entrypoint and the container, not run; U = unverified.

## Commands (run from the repo, they go to the worker host through labrun)

```
tools/dev-reset.sh --to d2play-20261108 --dry-run                        # show the plan, change nothing
tools/dev-reset.sh --to d2play-20261108 --yes                            # fresh hero, same name and class
tools/dev-reset.sh --to d2play-20261108 --class paladin --name Newbie --yes
tools/dev-sandbox.sh list    --to d2play-20261108
tools/dev-sandbox.sh restore --to d2play-20261108 --from last --yes      # or --from <stamp>; --dry-run works too
```

`dev-reset.sh` is only a thin entry for `dev-sandbox.sh reset`. Without `--yes` (or `--dry-run`) nothing runs. Classes are the ids of `cmd/d2dsetup/tables.json` (the wizard's table); an unknown class or a bad name (2 to 15 letters, `-` `_`) is an error. `--class` and `--name` default to the current `wizard.env` values.

## What reset does

1. `docker stop -t 60` (the bot saves the hero through the normal TERM path first). (S)
2. Archives, outside the volume, `instances/` (hero save and key, `goals.json`, `state/` with inventory-ui, goldfarm-ui, ladder, currency, tuning, deaths, d2d logs), `config/` (bare d2d saves and log) and `wizard.env` into `~/play-archive-<suffix>/<stamp>/` on the worker host (mode 700): `state.tgz`, `state.tgz.sha256`, `files.txt`, `meta.txt`. The archive is read back and counted before anything is removed. (V)
3. Removes those paths from the volume and writes a new `wizard.env` (name, d2d class index, goal). The entrypoint finds no save and creates the hero by its normal path (`--d2d-class N`, no lab verbs, no memory write, no seeded items). (V for the files; the hero creation itself is the entrypoint's existing path, S)
4. `docker start`, waits up to 240 s for the new save file, prints `ARCHIVE`, `HERO`, `VERSION`, `URL` and the restore command.

Never touched: the data volume itself (it is not deleted), `access.json`, `watch.token` (never read), `didle/` (releases, so the dev updater and rollback keep working), `cache/`, `wizard.json`, the read-only game mount, the container and its image.

## Restore

`restore` checks the archive checksum, archives the CURRENT state first as `<stamp>-pre-restore` (so a restore is itself undoable; the last line prints the undo command), wipes the same paths and extracts the archive. `--from last` is the newest directory under the archive root. (V)

## Safety rules

Nothing is deleted without an archive that was read back first. Archives are never pruned by the tools; remove old ones by hand. Do not run it against a container you cannot lose a session on: the bot restarts, the first hour starts again. Provenance: a reset hero has zero setup writes, so it is a normal fresh character (V by construction: no lab verbs in this image); log it in docs/probes/test-characters.md only if you use it as evidence.

## Extension point

One entry script, one function per operation. To add an operation (pause/resume, choose a build, restart only the bot): add `cmd_<name>` to `tools/dev-sandbox-host.sh` (it already has the helpers `vrun`, `stopc`, `startc`, `make_archive`, `report`) and add the name to the `case` in `tools/dev-sandbox.sh`. Keep one flag per option. The script cannot be driven from the watch UI: the container has no docker access (V: its only mounts are the data volume and the read-only game directory), so a UI button would need a host-side hook that does not exist; not built.

## Drill (V)

`tools/dev-sandbox-test.sh` on the worker host (`labrun orion -- bash tools/dev-sandbox-test.sh`, about 2 minutes because the stand-in container ignores TERM): scratch container and volume, fake save, bot state and `access.json`. Dry run changes nothing; no `--yes` refuses; an unknown class errors; reset archives and leaves only `wizard.env`, `access.json`, `didle/`; list shows the archive; restore brings the fake state back and archives the fresh state as pre-restore. Not run: a reset on the real sandbox, the hero-created wait, the `VERSION`/`URL` lines against the real container (U).
