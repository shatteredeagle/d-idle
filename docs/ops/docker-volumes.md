# Docker volumes on the worker host: safe removal

> status: current (fix4, from fable review 4 finding 2, `docs/design/fable-review-2026-11-28/04-security-privacy.md`).
> Tags: V = verified (said how), S = secondary, U = unverified. Nothing here was run against the worker host for this doc (U); the commands are standard docker CLI.

## Rules

1. Never run `docker volume prune -a` (with or without `-f`), and never `docker system prune --volumes`. They delete every volume that has no container at that second, named or not, of every project. The hero volumes survived the prune of 2026-11-27 only because their containers existed at that moment (S: review finding 2); two volumes of other projects were lost.
2. Remove a volume by name only: `docker volume rm <name>`, one name per decision.
3. A go for "the docker volumes" is not a go for a list. Print the exact list first, get the go for that list, then remove those names.
4. A volume that belongs to another project is not ours to remove: ask its owner.
5. The hero volumes `d2play-data` and `d2play-q-data` hold weeks of play. They are labelled and backed up before any removal work on the host, and they are never on a removal list.

## Procedure

Step 1, list and decide. Nothing is deleted here.

```sh
docker volume ls --format '{{.Name}}\t{{.Labels}}'
docker ps -a --format '{{.Names}}\t{{.Mounts}}'        # which container uses which volume
docker system df -v | sed -n '/VOLUME NAME/,$p'        # size and link count per volume
```

Step 2, check the hero volumes carry the keep label. A label can only be set when a volume is created, so the compose file or the create command of a sandbox must carry it (`io.didle.keep=1`); an existing unlabelled volume is recreated only by a planned copy, never during a cleanup.

```sh
for v in d2play-data d2play-q-data; do docker volume inspect -f '{{.Name}} keep={{index .Labels "io.didle.keep"}}' "$v"; done
```

If the label is missing, treat the name itself as the guard (step 4 refuses both names) and note the missing label in the report.

Step 3, back up both hero volumes before anything is removed. Read-only mount, one tar per volume, into the cold directory of `local.env` (`$D2_COLD_DIR` here stands for it).

```sh
ts=$(date -u +%Y%m%dT%H%M%SZ)
for v in d2play-data d2play-q-data; do
  docker run --rm -v "$v":/v:ro -v "$D2_COLD_DIR/volume-backups":/out busybox \
    tar -C /v -czf "/out/$v-$ts.tar.gz" .
  tar -tzf "$D2_COLD_DIR/volume-backups/$v-$ts.tar.gz" >/dev/null && echo "backup ok: $v"
done
```

Do not go on unless both lines say `backup ok`. The bot may be running during the tar; for a save that must be exact, stop the sandbox first (its own stop verb, not `docker rm`).

Step 4, remove by name. The loop refuses the hero names and any labelled volume, and `docker volume rm` itself refuses a volume that a container still uses.

```sh
for v in <name1> <name2>; do
  case $v in d2play-data|d2play-q-data) echo "refused: $v is a hero volume"; continue ;; esac
  [ "$(docker volume inspect -f '{{index .Labels "io.didle.keep"}}' "$v")" = 1 ] && { echo "refused: $v is labelled keep"; continue; }
  docker volume rm "$v"
done
```

Step 5, record what was removed (names, sizes, time, who gave the go) in the session log, and tell the owner of any other project whose volume was on the list before, not after.

## Restore

```sh
docker volume create --label io.didle.keep=1 d2play-data     # only when the volume is gone
docker run --rm -v d2play-data:/v -v "$D2_COLD_DIR/volume-backups":/in:ro busybox tar -C /v -xzf /in/<file>.tar.gz
```

## Open

- A nightly backup of the hero volumes and a reclaim helper that enforces step 4 do not exist yet (review finding 2, fix sketch b and c).
- Whether the two hero volumes carry the label today is unverified (U): run step 2.
