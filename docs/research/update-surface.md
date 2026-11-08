# Update surface: what can be hot-updated without a new image

> status: DESIGN REVIEW, 2026-11-08 (TASK: none; review brief of the coordinator, doc only). Nothing here is built. Proposed ADR text is in the coordinator's report, not in docs/adr.
> Tags: V = verified (read in this checkout at the named file and line, or run read-only against the image on the worker host: `docker history`, `docker image inspect`, `docker save` into a scratch dir that was deleted afterwards); S = secondary (vendor practice or a doc, not re-run here); U = unverified. Opinions are marked "Opinion".
> Builds on: docs/research/self-update-design.md (Tier 1/2 design), docs/self-update-release.md (format), docs/self-update-client.md (updater), docs/self-update-drill.md (shim), docs/dev-update.md (dev loop).

## 0. Answer in one paragraph

Most of the image is already, or can cheaply become, a signed release asset: every Go binary, the launcher script, all data tables and all web pages (they are `go:embed`ed into the Go binaries, V), and the d2d binary (the shim already points `D2D_BIN` at a release's `d2d/d2d`, V). What must stay baked is small and well-defined: the base distro with glibc, the shared libraries d2d links against, tini, the boot shim, the trust root and the container contract (entrypoint line, user, volume, mounts, ports, healthcheck). The present code has one design flaw that defeats the goal: the compatibility gate compares a release's protocol and data-schema numbers with constants COMPILED INTO the frozen shim, so any release that bumps them is skipped forever, i.e. needs a new image (finding H1). The recommendation: replace "image ABI" with a declared PLATFORM ABI (min/max) that is the only thing the shim gates on, make protocol pairs intra-release, add data-schema migrations with a snapshot and restore, and cut ONE "platform 2" image that carries every remaining baked change (stage-0 shim, root key with delegated signing keys, platform file, shim-routed healthcheck) BEFORE the first image goes to friends. After that only a platform major needs a pull.

## 1. What the image really contains (V)

Correction to the brief's premise: the d2d image has NO Xvfb and NO ffmpeg runtime (V, `deploy/d2d-container/Dockerfile.d2d` stage 2 installs only `libsdl3-0 libopenal1 libcurl4t64 libstorm9 zlib1g libbz2-1.0 tini ca-certificates`). FFmpeg 9.0.2 is compiled statically INTO d2d (Bink decoders only, `context.sh` line 29, `-DD2D_FFMPEG_SOURCE_DIR`), SDL runs with `SDL_VIDEODRIVER=dummy`, audio with `ALSOFT_DRIVERS=null`, and the watch video is the tap's JPEG stream (`d2dview ... -jpeg`, `entrypoint.sh`). Xvfb lives only in the retired Wine image `deploy/container/`. So the "hard layer" is the shared libraries of d2d, not a display stack.

Layer sizes of `d2dbot:dev` (built 2026-11-08, 421 MB on disk). Raw = tar size; gz = `gzip -6`; zst = `zstd -19` (V, measured on the worker host from `docker save`):

| # | Layer | Content | raw | gz | zst |
|---|---|---|---|---|---|
| 1 | base | ubuntu:26.04 rock (glibc, coreutils, perl, apt, libstdc++) | 113.0 MB | 39.5 MB | 28.9 MB |
| 2 | apt | SDL3, openal, curl, storm, zlib, bz2, tini, ca-certs, AND their dependency tree | 229.1 MB | 75.7 MB | 57.3 MB |
| 3 | d2d | `/opt/d2d/d2d` (15.0 MB, FFmpeg static inside) | 15.0 MB | 5.7 MB | 4.5 MB |
| 4 | licences | `/usr/share/licenses/d2d` | 4 KB | 0.4 KB | 0.3 KB |
| 5 | Go tools | `d2bot` 28.6, `d2dview` 13.5, `d2dsetup` 13.1, `d2dsoak` 6.6, `d2dping` 4.5 MB (all static, `CGO_ENABLED=0`) | 66.4 MB | 36.5 MB | 32.0 MB |
| 6 | launcher | `/opt/d2lod/entrypoint.sh` | 8 KB | 3.6 KB | 3.4 KB |
| 7 | shim | `/opt/didle/boot` (static Go) | 11.0 MB | 6.3 MB | 5.8 MB |
| 8 | keys | `/opt/didle/pubkeys/` (empty today) | 0 | 0 | 0 |

Inside layer 2, `libLLVM.so.21.1` (138.6 MB) and `libgallium-26.0.8` (45.6 MB) are 184 MB of the 229 MB: the Mesa GL stack that `libsdl3-0` pulls in although SDL runs with the dummy video driver (V file list; that it is a hard dependency even with `--no-install-recommends` is S, inferred from the install line).

d2d's dynamic needs (V, `readelf -d`, `objdump -T`): `libSDL3.so.0 libopenal.so.1 libstorm.so.9 libz.so.1 libcurl.so.4 libstdc++.so.6 libm libgcc_s libc ld-linux`; highest glibc symbol version `GLIBC_2.43` (= the 26.04 base); `RPATH` is `:::::::::::` (empty components; finding L1).

Data and web assets are not separate files (V, `grep go:embed`): `data/unlocks/ladder.json`, `data/pacts/pacts.json`, `internal/plandata/*.json`, `internal/combat/classdata/*.json`, `internal/senses/adjacency.json`, the decide/loot/gear/belt/currency tables, `internal/watch/web`, `web/` (index, setup.html), `cmd/d2dsetup` page and `tables.json`, `cmd/d2dview/web` are embedded. They ride the Go binaries and are therefore already hot-updatable through the `bin` asset. The only non-embedded table set is the d2d overlay (`data/overlay/didle-v1`), read by d2d through `--overlay-dir` / `D2_OVERLAY_DIR`; the launcher does not set it today (V `entrypoint.sh`), so the `data` destination of a release has no runtime reader (as docs/dev-update.md says).

Delta measurement (V): two real builds of the same day, `d2dbot:play-20261108` vs `d2dbot:dev`, per file with `zstd -19 --long=27 --patch-from=<old>`:

| File | full zst | patch | patch/full |
|---|---|---|---|
| d2bot | 13.7 MB | 8.4 MB | 61 % |
| d2dview | 7.0 MB | 4.3 MB | 62 % |
| d2dsetup | 6.5 MB | 2.2 MB | 33 % |
| d2dping | 2.5 MB | 578 B | unchanged |
| d2d | byte-identical (only the layer digest differed: file mtimes) | | |

So the cheap win is NOT binary deltas but "do not ship unchanged files": a bot-only update today ships a 36.5 MB gz tarball of five binaries; shipping only changed files is about 15 MB gz; a binary patch on top saves another 40 %. Go binaries change widely between builds (addresses shift), so patches stay large.

## 2. Findings on the current code (ranked)

| Id | Rank | Finding (evidence) | Recommendation |
|---|---|---|---|
| H1 | high | The gate (`internal/selfupdate/engine.go:252-259`) requires `compat.bot_api <= local.BotAPI`, `d2d_ctl <= local.D2DCtl`, `data_schema == local.DataSchema`, where `local` is `selfupdate.BuildCompat` of the CHECKING binary (`cmd/didleboot/main.go:65`, `cmd/d2bot/update.go:57`, V). In the shim that constant is frozen with the image. A release that raises any pair (a new tap verb, a new volume format) is skipped by the shim on every boot and the base runs; the updater in the running bot refuses it too. In effect every protocol or data change needs an image, which is the opposite of the goal. The pairs also answer the wrong question: bot and d2d ship in the SAME release, so their pairing is decided at build time, not by the install. | The shim gates only on the platform (min/max, section 3.1), the signature, the bad/revoke lists and "the volume's data schema is readable or migratable by this release" (3.3). Protocol pairs move to a build-time check of the release builder plus a start-time self-check of the new bot against its own d2d. Ship this in platform 2 (the shim must change). |
| H2 | high | No migration or backup path exists (V: no `backup` in `internal/selfupdate` or `cmd/didleboot`; `data_schema` is an exact match). The volume holds about 15 state file kinds (`tuning.json`, `goals.json`, `merc.json`, `ledger.jsonl`, `session.jsonl`, `decisions.jsonl`, `summary.json`, `hall`/`bestiary`/`alerts`/`deaths`/`hardcore`/`visit`/`proposals`/`stamp` JSON, `access.json`, `wizard.json`, the d2s saves); only `access.json` and `wizard.json` carry a `schema` int and `session.jsonl` a `v` (V, grep). A rollback after a release rewrote one of them hands the old release a file it may misread. | A data-schema registry (one integer per file kind plus one volume integer), forward migrations shipped IN the release and run by the shim before the launcher, a snapshot of the small state files and the saves before the migration, restored on rollback. Card updmigrate. |
| H3 | high | Everything that makes the update surface wider sits in the frozen shim (gate, keys, probes, selection). Each such fix after friends hold an image costs them one more pull. | Batch every baked change into one "platform 2" image and make it the FIRST image a friend receives (section 6 ordering). |
| M1 | medium | The platform level is read from the environment (`D2BOT_IMAGE_ABI`, `cmd/didleboot/main.go:53`, `cmd/d2bot/update.go:50`; ENV in the Dockerfile). A compose `environment:` line overrides it, so a user can make the shim run a release the image cannot run (crash, then rollback; not a security hole, the volume owner owns the container anyway). | Read the platform from a baked read-only file `/opt/didle/platform.json` (level, glibc, lib sonames, base digest); keep the env only as a display copy. |
| M2 | medium | The health probes and the Docker HEALTHCHECK use the BAKED `d2dping` (`cmd/didleboot/main.go:102`; `HEALTHCHECK ... /opt/d2lod/d2dping ping`). The tap verbs `ping` and `clock` and the tap socket path are therefore part of the platform contract, unstated. `clock` in realtime mode is still U (docs/self-update-drill.md). | Either declare `ping`/`clock`/socket path as frozen platform contract, or (better, platform 2) make the HEALTHCHECK `/opt/didle/boot health`, which runs the CURRENT release's `d2dping` (falls back to the baked one). |
| M3 | medium | A release is a set already (one directory, one manifest, one atomic symlink flip, V `store.go`, `boot.go:146`), and the shim already prefers a release's d2d (`cmd/didleboot/real.go:65`) and its launcher (`boot.go:150-154`). What is missing for d2d: the release builder takes `--d2d-bin PATH` from anywhere (`tools/release/app-assets.sh`), with no check that the binary's NEEDED libraries and highest glibc symbol exist on the target platform; nothing snapshots the saves; `IdleSafe` is nil, so an apply can cut a fight (docs/self-update-client.md). | Build d2d for releases only in the platform builder stage (same base digest as the image), record NEEDED and max GLIBC in the manifest, verify against `platform.json` at release build and at boot. Card updset. |
| M4 | medium | `channel` is in the manifest but nothing reads it (V, no use in `engine.go`/`fetch.go`); `DiskFree` is nil in `cmd/d2bot` (V `engine.go:50`). | Channel filter plus per-channel floors (3.6); a statfs-based DiskFree (stdlib `syscall.Statfs`). |
| M5 | medium | Key rotation needs an image (design 3.3, V `boot.go` verifies stored releases against baked keys only). With the signing key held in CI, a key compromise is fixed only by a pull on every friend's machine. | Baked offline ROOT key that signs delegations to CI signing keys (3.7). Platform 2. |
| L1 | low | d2d has `RPATH` of empty components (V). glibc reads an empty RPATH element as the current directory (S, glibc dynamic-loader behaviour); d2d inherits the cwd of the launcher, `/` today (no WORKDIR), so nothing writable is searched now. | In the fork build set `RUNPATH=$ORIGIN/../lib` (needed anyway for P3) or skip RPATH; a release-builder check rejects empty RPATH elements. |
| L2 | low | `d2dsoak` (6.6 MB, a soak test tool) ships in the tester image and in every bin asset. | Leave it out of the tester/release set. |
| L3 | low | Five static Go binaries each carry their own Go runtime and the embedded tables (d2bot alone 28.6 MB). | Opinion: a later multi-call binary (one executable, argv[0] dispatch) would cut the bin asset roughly in half (U, not measured). Not worth it before deltas. |

## 3. The compatibility contract

### 3.1 Platform ABI (replaces "image ABI")

The image is a PLATFORM with an integer major `P` (today 1). It is written once into `/opt/didle/platform.json` (read-only, baked) and the label `io.didle.platform=P`:

```
{ "platform": 2, "base": "ubuntu:26.04@sha256:...", "glibc": "2.43",
  "libs": ["libSDL3.so.0","libopenal.so.1","libstorm.so.9","libcurl.so.4","libz.so.1","libstdc++.so.6"],
  "contract": { "uid": 1000, "data": "/data", "game": "/game:ro", "ports": [8891, 8892],
                "tap_verbs": ["ping","clock"], "stage0": 1, "root_keys": ["<fp>"] } }
```

Every release manifest declares `"platform": {"min": 2, "max": 2}` (replaces `min_image`; max is new). The updater and the shim refuse a release outside the range with the plain message "this release needs image platform 3: run `docker compose pull && docker compose up -d`" (the existing `NeedsImage` banner path, V `engine.go:198`). A max bound matters because a newer platform may drop a library an old release needs; the shim then runs the newest release whose range includes the platform, else the base.

What counts as BREAKING (= new platform major, a pull):

| Change | Why it cannot ride a release |
|---|---|
| base distro or glibc version raise | d2d is dynamically linked; a d2d built against a newer glibc does not start on the old one (V: needs GLIBC_2.43) |
| add, remove or re-soname a shared library d2d links (unless the library moved into the release, P3) | the release cannot install system packages (not root, image read-only) |
| tini, the stage-0 shim, the root key | they verify and start everything else |
| the container contract: ENTRYPOINT/CMD, USER/uid, VOLUME `/data`, the `/game` read-only mount, the exposed ports, HEALTHCHECK command, `cap_drop`/`no-new-privileges`, required env names that the shim reads | Docker reads them from the image and the compose file before any of our code runs |
| a compose file change that is not additive with a default | the friend holds a copy of the compose file |
| a volume layout change without a shipped migration | an old release (rollback) and the shim must find their files |
| a new kernel capability or device (GPU, audio device, seccomp change) | host and Docker policy, outside the container |
| a new signature algorithm or a manifest envelope stage-0 cannot parse | stage-0 is the verifier |

NOT breaking (a release may do it): any Go binary, the launcher script and its mode logic, d2d itself (same platform libs), tap/ctl protocol changes (bot and d2d move together), all tables and web pages, config defaults, the setup wizard, the update logic, the update source and host allow-list, data formats WITH a migration, stage-1 shim after P4.

### 3.2 Protocol pairs inside a release

`bot_api` (the watch API spoken to pages of the same release: always equal, so it needs no runtime check) and `d2d_ctl` stay in the manifest for information. The release builder checks that the bot's supported d2d_ctl range contains the shipped d2d's version (`d2d --ctl-version` or the banner verb; U which verb exists). At start the bot checks the live d2d answers its range and exits with a clear code otherwise, which the shim's probation turns into a rollback. The only cross-release protocol is the frozen platform part (tap `ping`/`clock`, socket path), see M2.

### 3.3 Data schema and migrations

- The volume carries `/data/didle/schema.json`: `{ "volume": N, "files": { "tuning": 1, "ledger": 2, ... } }`. A missing file means "N = 1, every kind 1" (today's volumes).
- A manifest declares `"data": { "reads": [1, 3], "writes": 3, "migrations": ["1-2", "2-3"] }`. The shim accepts a release when the volume's N is inside `reads`, or below and a chain of shipped migrations reaches `writes`.
- Migrations are code in the release (`d2bot migrate --from N --to M --data /data`), forward only, idempotent, run by the shim after the swap and BEFORE the launcher, under a timeout, never while the bot runs.
- Before the first migration the shim copies the small state files and the d2s saves to `/data/didle/backup/<from-version>/` (the saves are kilobytes; the design already asked for this in 3.4). Rollback across a migration restores that snapshot, then flips the link; the release that wrote newer data is marked bad. Opinion: refuse rollback silently past a migration only if the snapshot is gone, and say so on the banner.
- Saves: the d2s format is the game's, so a d2d update does not change it; the exception is d.idle-mode custom items (metacurrency design) if they ever touch the save. Count such a change as a data migration of kind `save`.
- Owner: each package that owns a file kind owns its migration and its schema number (one table, `internal/selfupdate/dataschema.go` or a small new package); a test fails when a struct with a `json` tag in that table changes without a schema bump (golden of the field list; Opinion).

### 3.4 Volume formats today (V)

`access.json` and `wizard.json` carry `schema: 1`, `session.jsonl` carries `v`; the rest carry none. Platform-2-day value: all of them are "kind version 1" by definition, no rewrite needed.

## 4. Release mechanics

- Set and atomicity (V today): one release directory `releases/<ver>/{bin,d2d,data,boot}` with one signed manifest, flipped by one `rename(2)` of the `current` symlink. Bot, d2d, launcher, tables and web are one unit and roll back together. Keep it that way; do not introduce per-component versions.
- Apply order (proposed, extends design 3.7): verify set, unpack into `releases/<v>`, reuse unchanged files from `current` by hash (hard link on the same volume), mark pending, wait for a safe point (`IdleSafe`: hero in town, no panel, not mid-transaction; a timeout plus the existing Force for a stuck hero), stop the launcher the normal way (the bot saves and quits d2d through the tap, then the tap stops: V `entrypoint.sh` `stop()`), shim snapshots data, runs migrations, flips, starts the launcher, probation, `good` then prune; any failure restores the snapshot and the link and marks the version bad.
- d2d teardown: d2d is a child of the managed bot, so "restart d2d" is "restart the launcher"; nothing new is needed beyond the safe point and the save snapshot. d2s write atomicity of the fork is U; the snapshot covers it.
- Disk layout and GC: `/data/didle/{releases/<v>, current, state.json, schema.json, staging/, inbox/, backup/<v>/, keys/, boot/{a,b}}`. Keep current, previous, the base needs no copy (it is in the image); keep the last two backups; staging deleted after unpack; cap 600 MB (design 3.10); the cap check needs a real `DiskFree` (M4). With hard-linked unchanged files a second release costs only what changed.
- Offline / air-gapped: the inbox exists with the same verification (V docs/self-update-client.md). Add a single-file bundle `didle-<v>.bundle` (tar of manifest, sig, assets, delegation, REVOKED) so a friend copies one file into the volume.
- Update source: repo, base URL and host allow-list live in d2bot (`internal/selfupdate/fetch.go:18,64`, V), i.e. in the release, so moving the source is a release. Trust does not depend on the source (signatures), so a mirror is safe; the allow-list exists only to limit where the client talks.
- Channels: `stable` and `dev`. The client setting picks the channel; floors (`seq`, version) are kept per channel; dev releases are signed by a DEV delegation that the root key grants with `channels: ["dev"]` only (3.7); switching dev to stable moves to the highest stable release whose data `reads` range includes the volume, else stays and says why.
- Deltas: phase 1 is file reuse by hash (no new format, a manifest field per file). Phase 2 (optional) is binary patches. The Go standard library has no zstd or xz encoder or decoder; it has `compress/bzip2` (decode only), which is enough to APPLY bsdiff-4 patches (bspatch is about 100 lines). Patch size of bsdiff on our binaries is U; zstd patch-from gave 33 to 62 % (V above). Ship a patch only "from the previous stable release", always with the full file as fallback.

## 5. The shim itself: A/B stage-1 under an immutable stage-0

How real systems do it (S, vendor documentation, not re-read here): Android A/B and ChromeOS keep two slots with per-slot metadata (priority, tries remaining, successful flag); the updater writes the inactive slot, the bootloader decrements tries before booting it and falls back when tries reach zero; the running system marks the slot successful after its own health check. U-Boot `bootcount`/`bootlimit`/`altbootcmd` (used by Mender, RAUC, SWUpdate) and systemd-boot's boot counting (`+tries` in the entry name, blessed by `boot-complete.target`) are the same idea. Common to all: a tiny immutable first stage that only verifies, counts and chooses; two updatable slots; a "blessed" mark written by the new code itself; automatic fallback.

Applied here (proposed for platform 2):
- stage-0 = `/opt/didle/boot` reduced to: read `platform.json`, load root key and delegations, read `/data/didle/boot/state.json`, choose slot A or B (signed, platform range OK, tries left, not bad), decrement tries, `syscall.Exec` the slot binary with `--stage1` (same PID under tini, no fork). If neither slot qualifies it runs its own built-in copy of the stage-1 logic (today's didleboot), so a broken update can never brick the container. Target under 400 lines, the heaviest tests in the repo.
- stage-1 = the current shim logic (selection, probation, migrations, rollback, `health` subcommand) shipped as release asset dest `boot`. It blesses its own slot after it has started a release that passed probation (or after 15 min up with the base). Two failed starts drain the tries and stage-0 goes back to the other slot.
- Why not exec-replace a running shim in place: the shim is the parent of the launcher and holds its process group; an in-place exec would have to hand over child state through the environment. Updating stage-1 only at the next launcher restart (which every apply already does) avoids that: after the swap, stage-1 exits its supervision loop after the launcher stopped and re-execs stage-0. Opinion: this is enough; no live handover.
- Value check (Opinion): with H1 fixed and the policy in data, stage-1 changes become rare. P4 is still worth it because the shim is the one piece whose bug otherwise forces a pull on every friend, but it is last in the order of work, not in the order of the image: stage-0 must be IN platform 2.

## 6. Security without rebuilds

- Root key (offline, user-held, baked in platform 2) signs a `KEYS.json` delegation: `{ "seq": N, "keys": [{ "fp": ..., "channels": ["stable"], "expires": ... }], "revoked": [fp...] }`. CI signing keys and the dev key are delegated, never baked. Stage-0 and the updater accept a manifest only through a valid, unexpired delegation chain with a monotonic `seq` (stored floor). This is the TUF root/targets split in its smallest form (S).
- CI key compromise: the user signs a new `KEYS.json` that revokes the old key fingerprint and delegates a new one; it is published with the next release and in the inbox bundle; no pull. Root key loss or compromise: a new platform image (pull), as today for any key.
- `REVOKED` and the version floor stay as they are (V docs/self-update-release.md).
- This departs from the 2026-11-07 decision only by ADDING an offline root key; the CI-held signing key stays as decided (user decision 1 of docs/research/self-update-design.md). Needs the user's go (proposed ADR below in the report).

## 7. Dev versus production

Today a DEV image (`DIDLE_DEV=1`) trusts a key and a URL from the environment (V docs/dev-update.md). Keep the flag (a production container can never be pointed at a LAN URL by an env line), but in platform 2 let the dev loop use the same delegation mechanism (dev key delegated for channel `dev` only), so the dev loop exercises the production verification path every day. The shorter probation of the dev policy stays.

## 8. The friend's flow after platform 2

1. Once: pull the registry image `platform 2` and the compose file (docs/tester-registry.md); start; setup wizard.
2. Every update: the watch banner "Update available: X" with notes, Download, Install at the next safe point; the hero is saved, the container stays up, about a minute of restart (U for d2d warm-up). Typical download: 5 to 40 MB (section 9).
3. Rarely: "This release needs image platform 3" with the two commands; the volume, hero and token survive (the shim on the new image picks the newest release in range, design 3.1).
4. Offline friend: drop a `.bundle` into the volume and press Inbox.

## 9. What a typical update downloads (V sizes from section 1, estimates marked)

| Update | Today (full set, gz) | With file reuse | With patches (est., U) |
|---|---|---|---|
| table or balance change (embedded table, one binary) | 36.5 MB | about 15 MB (d2bot only) | about 9 MB |
| bot code change (d2bot, d2dview) | 36.5 MB | about 22 MB | about 13 MB |
| d2d fix | + 5.7 MB | 5.7 MB | about 3 MB |
| launcher or wizard text only | 36.5 MB | KB (launcher) / about 6.5 MB (d2dsetup) | KB / about 2 MB |
| platform pull (once per major) | about 115 MB gz (layers 1, 2, 7) | | about 45 MB after P3 slims layer 2 (U) |

Optional, Opinion: an override directory for the embedded tables (a release `data/tables/<name>.json` wins over the embedded copy when its schema validates) would make balance changes kilobytes. It costs a second source of truth per table; do it only for tables that change weekly (ladder, pacts, tuning defaults).

## 10. Recommended architecture

1. The image is a PLATFORM: base, glibc, d2d's system libraries, tini, stage-0 shim, root key, `platform.json`, container contract. Frozen per platform major.
2. Everything the project writes is a RELEASE SET in the volume: Go tools (with embedded tables and web), launcher, d2d, overlay, stage-1 shim, migrations; one manifest, one signature, one symlink flip.
3. The shim gates on platform range, signature chain, bad/revoke lists and data-schema readability only; protocol pairs are build-time facts of a set.
4. Data formats carry versions; migrations ship in the release and run before the launcher, behind a snapshot that rollback restores.
5. Trust is a baked offline root key delegating to CI and dev keys; rotation and revocation are signed files, not images.
6. Health checks run the current release's tools through the shim; only `ping`/`clock` on the tap are platform contract.
7. Downloads reuse unchanged files by hash; patches later; offline via one bundle file.
8. Platform 2 is cut once with all baked changes and is the first image a friend receives.

## 11. Phased plan and proposed cards

Sizes: S under 4 agent-hours, M 4 to 12, L over 12. Owns are proposals; the coordinator assigns.

P0 (done, V): bot-only signed release through the shim, dev loop (TASK-121, 122, 123, 192).

P1: d2d, launcher and overlay in the same signed set (no shim change; works on platform 1).
- `updset: release set carries d2d built on the platform` (M). Owns `tools/release/`, `deploy/d2d-container/` (builder stage target only). AC: release d2d is built in the Dockerfile builder stage from the pinned fork tag; the builder records NEEDED libs and max GLIBC in SOURCE.txt and fails when either exceeds the platform list; `tools/dev-update.sh --with-d2d PATH` publishes and the drill on a throwaway container shows the release's `D2D_BIN` in `/proc/*/environ`. Risks: fork build time in CI; d2d warm-up longer than the dev 15 s probation.
- `updsafe: safe point and save snapshot before apply` (M). Owns `cmd/d2bot/update.go`, `internal/selfupdate`. AC: `IdleSafe` true only in town with no panel and no transaction, tested for two classes (build-flexibility rule); saves and state files copied to `backup/<ver>` before the swap; DiskFree via statfs. Risks: a hero that never reaches town (Force with a timeout).
- `updprobe: d2dping clock in realtime mode` (S, live, worker host throwaway container). AC: the verb answers and advances in the tester profile, or the probe is replaced; result recorded in docs/self-update-drill.md.
- `updslim: drop d2dsoak from the tester set; reject empty RPATH elements` (S).

P2: the contract (needs the platform 2 image for the shim part).
- `updcompat: platform range replaces image ABI; pairs become build-time` (M). Owns `internal/selfupdate`, `cmd/didleboot`, `cmd/d2sign`. AC: manifest `platform{min,max}`, shim reads `/opt/didle/platform.json` not the env, message names the needed platform, old manifests with `min_image` still parse (schema 2 with a schema-1 reader); a release bumping `d2d_ctl` runs on the same image (test). Risks: two manifest schemas in the field during the switch.
- `updmigrate: data-schema registry, migrations, snapshot restore on rollback` (L). Owns `internal/selfupdate`, `cmd/didleboot`, a new `internal/dataschema`. AC: registry of every volume file kind with version; `d2bot migrate` runs forward chains; rollback across a migration restores the snapshot (test with a fake release pair); golden test fails on an unversioned struct change. Risks: a missed file kind; migrations of append-only jsonl logs (prefer "new readers read old lines" over rewriting).
- `updplatform2: the platform 2 image` (S, plus the user's go for a friend-visible image). Owns `deploy/d2d-container/`. AC: `platform.json`, label `io.didle.platform=2`, HEALTHCHECK via `/opt/didle/boot health`, stage-0 shim, root key file; static tests updated. Risks: cutting it before P4 and the key work are merged means a platform 3 soon after.

P3: platform policy and a slim, relocatable d2d.
- `updreloc: d2d with bundled SDL3/openal/storm under release lib/, RUNPATH $ORIGIN/../lib` (L). Owns the fork build recipe (fork-branch exclusive, pushed the same day per AGENTS.md) and `deploy/d2d-container/`. AC: d2d starts with the release lib dir and no SDL3/openal/storm packages in the platform; platform layer 2 under 60 MB raw (U target); a library fix ships as a release. Risks: SDL3 built without video backends (U), licence notices for bundled LGPL libs (docs/tester-licence.md), glibc floor still the platform's.
- `updpolicy: platform bump rules and checklist` (S, docs plus a CI check that compares the Dockerfile's base digest and apt list with `platform.json`).

P4: shim self-update and trust without rebuilds.
- `updstage0: stage-0 / stage-1 A/B boot slots` (L). Owns `cmd/didleboot`. AC: stage-0 verifies, counts tries, execs slot A or B, falls back to its built-in stage-1; a stage-1 that crashes twice is abandoned; fake-clock tests and a throwaway-container drill. Risks: the stage-0 code is the new unfixable piece; keep it tiny.
- `updkeys: offline root key and signed delegations` (M). Owns `cmd/d2sign`, `internal/selfupdate`. AC: `d2sign delegate`, chain verification with seq floor and expiry, dev key limited to channel dev, revocation of a CI key without an image (test). Risks: the user must hold an offline key (a decision; ADR).
- `upddelta: file reuse by hash, optional bspatch` (M). AC: manifest lists per-file hashes; unchanged files are hard-linked from `current`; measured download for a bot-only change. Risks: hard links across a volume that is a bind mount on some hosts (fallback copy).
- `updchannel: channel filter, per-channel floors, single-file bundle` (S).

Ordering note (H3): updcompat, updstage0's stage-0 part, updkeys' root-key part and updplatform2 go into ONE image. If the first friend image ships before them, friends pull twice.

## 12. What still requires a new image after P4

| Item | Reason |
|---|---|
| glibc / base distro upgrade | d2d and the bundled libraries are linked against it; Go binaries are static and do not care |
| a new system library the release cannot carry (GL, audio device, a library needing root to install) | the release is unprivileged, the image read-only |
| tini, stage-0 | they run before any verified code |
| root key loss or compromise | it is the trust anchor |
| ENTRYPOINT/CMD, user, volume path, `/game` mount, ports, HEALTHCHECK command, capabilities, a non-additive compose change | Docker and the friend's compose file read them before our code runs |
| a volume layout change that cannot be migrated forward | rollback and stage-0 must find their files |
| a new signature scheme or manifest envelope stage-0 cannot read | stage-0 is the verifier |
| CPU architecture (arm64) | a second platform line, not an update |

## 13. Unknowns (U)

- Whether `libsdl3-0` hard-depends on Mesa on 26.04 (S from the file list); whether SDL3 builds usefully without video backends for d2d.
- bsdiff patch sizes on our binaries; Go has no stdlib zstd, so patches need bspatch+bzip2 or a dependency (AGENTS.md: stdlib only).
- `d2dping clock` in realtime mode (open since TASK-123).
- d2d's save write atomicity.
- d2d warm-up time after a restart with a big hero (affects probation windows).
- Hard links in the data volume on every Docker storage setup.
- GitHub-side behaviour (rate limits, ETag) is still untested live (docs/self-update-client.md).
