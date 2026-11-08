# Self-update boot shim and drills (Tier 2, part 3)

> status: CURRENT for the code; the container drills below are a CHECKLIST that has NOT been run (no image was built, no Docker run, no game). Design: docs/research/self-update-design.md 3.1, 3.8, 3.9, 5. Task: TASK-123 (updboot).
> Tags: V = proven by the offline tests named here; U = unverified.

## What changed in the image

- `cmd/didleboot` (static Go, about 350 lines with its probes) is copied to `/opt/didle/boot` and is the process under tini: `ENTRYPOINT ["/usr/bin/tini", "--", "/opt/didle/boot", "--launcher", "/opt/d2lod/entrypoint.sh"]`, `CMD ["bot"]`. Public keys (`deploy/d2d-container/pubkeys/*.pub`, up to two) are copied to `/opt/didle/pubkeys/`; none exists in the repo until the key setup (TASK-124).
- `Dockerfile.d2d` gains `ARG APP_VERSION` (becomes `DIDLE_BASE_VERSION`), `ENV D2BOT_IMAGE_ABI=1` and the label `io.didle.image.abi="1"` (same number, checked by a static test). `context.sh` builds the shim and copies the keys.
- `entrypoint.sh` is NOT changed: it remains the launcher and the owner of all mode logic. A downloaded release brings its own copy as `bin/entrypoint.sh` (the release builder ships it), so mode logic updates without an image; a release without one uses the baked launcher. This is the design 3.9 rule in its smallest form; moving more logic out of the image later needs no further shim change. (AC 2 asked for a patch for the TASK-31 owners; the change is small and the existing static tests pass, so it is applied here and the owner list widened.)

## The shim (V: `go test ./cmd/didleboot`, fake clock, fake probes, plus real shell launchers)

1. Select: the highest version among `current`, `previous` and the baked base that is not on the bad list, whose stored `MANIFEST.json` verifies against a baked key, passes the compatibility gate (`selfupdate.Gate`: image ABI, protocol pairs, data schema) and has `bin/d2bot`. The base wins a tie and is the last resort, so an image upgrade over an older downloaded release runs the base, and a missing `current` runs the base.
2. Start the release launcher with its `bin` and `d2d` directories first on PATH (`D2D_BIN`, `DIDLE_RELEASE`). The shim is the parent so a container stop (TERM from tini) is forwarded and waited for (45 s, then KILL).
3. Probation applies only to a pending release (`state.json` pending equals the selected version): boot counter +1; the third boot rolls back before starting (crash loop). Healthy means: the bot API `/healthz` answers within 60 s; d2d's counter (read with `d2dping clock` through the tap) advances within 3 min; in the tester profile the watch screen answers; and the release stays up for 10 minutes. Then it becomes `good` and older releases are pruned. A start error, an exit during probation or a failed rule makes the shim point `current` back at `previous` (or remove the link for the base), put the version on the bad list, write `rollback.json` (the watch banner shows it) and start the older release in the same container without a restart.
4. `image.json` records ABI, base version, key fingerprints, what changed since the last start (abi, base, keys), stored releases it had to skip and `rebuild_needed` when a stored release cannot run on this image (needs a newer ABI, signed by a key that is no longer baked, incompatible). The updater's `needs_image` banner covers the not-yet-downloaded case.

Facts the tests prove (V): selection and ties; gate, bad-list, missing-dir and rotated-key skips; healthy update becomes good only after the window; never-answering API, frozen counter and silent watch screen roll back with a marker; crash loop on boot 3; start failure; missing `current`; confirmed release exit passes through; a stop request is not judged; a stubborn child is killed after the grace; an applied-but-not-run release is not blamed. `drill_test.go` runs the whole chain with the real updater (inbox, signature, hashes, unpack, swap) and the shim: good update, and a bad update that rolls back to the base and is refused if offered again. `real_test.go` runs real shell launchers through `realMain`: exit code and argument pass-through, `--launcher`, a pending release that exits is rolled back and the base runs.

Not proven: `d2dping clock` in realtime mode (the verb is documented for the lockstep fork, U; if the realtime build answers `err unknown verb` the counter rule fails every update, so check this FIRST in drill 1), the 10-minute window against a real bot, TERM forwarding through real tini, a Docker build of the changed Dockerfile.

## Worker host drills (checklist, NOT run)

Docker only, throwaway volume, no game instance of the lab, no player data. Build with `deploy/d2d-container/context.sh` (the d2d stage is slow, a `PREBUILT` d2d is fine here) and a test key pair from `d2sign keygen` whose `.pub` is placed in `deploy/d2d-container/pubkeys/` for the build only (never commit it). Serve a release set built by `tools/release/app-assets.sh` and signed with `d2sign sign` from a local file server; the updater has no test override for the repo URL in the shipped binary, so drill it through the inbox (`/data/didle/inbox/`, same verification) or with the test hook of the fork-side harness. Record `writes after start: 0` for every drill (no game memory is written; these drills never start a hero).

1. Image 0.3.0 starts, `image.json` exists, `d2dping clock` answers in the tester profile (decides the counter rule). Press Check/Inbox, Download, Install: container restarts, `docker exec ... didleboot -select` shows `release=0.4.0`, version in the footer changes, after 10 minutes `state.json` shows `good=0.4.0`, character and token survive.
2. Bad update: a release whose launcher exits at start. Expect rollback within seconds, banner "Update 0.4.0 failed and was rolled back", `bad` list holds 0.4.0, the old release runs.
3. Hash mismatch: edit one asset byte after signing. Download refuses, staging cleared, nothing swapped.
4. Bad signature: sign with another key. Check refuses, no ETag stored.
5. Offline: `D2BOT_UPDATE_CHECK=off` and `docker network disconnect`. No outbound connection (`ss -tn` inside shows none), inbox apply still works.
6. Image upgrade over a newer downloaded release: run image 0.3.0 with downloaded 0.5.0, then start image 0.6.0: the base wins, `image.json` shows `changed: [base]`; start image 0.4.0 over downloaded 0.5.0: 0.5.0 wins.
7. Crash loop: a pending release that survives 30 s each time: third boot rolls back before it starts (`boots` in `state.json`).
8. `docker stop` during probation: the hero is saved, the release is NOT marked bad.

## Open points

- `IdleSafe` is nil in `cmd/d2bot` (U what "safe" means), so Apply can interrupt a fight; the hero is saved by the shutdown path.
- A pending release whose bot exits on its own with code 0 inside ten minutes (for example `D2BOT_RUN_HOURS` tiny) is rolled back; documented behaviour, rare.
- Key rotation, the first signed release and the GitHub settings are TASK-124 (user steps).
