# Self-update for the tester container: registry first, in-app updater second

> status: DESIGN, 2026-11-07. Nothing is built, published or pushed by this document. User decision 2026-11-07 ("Both: registry now, in-app updater later"): friends receive the tester container ONCE, then get updates without anyone redistributing containers.
> Tags: V verified (how is named), S secondary (read in a repo doc or a vendor's docs, not re-run), U unverified. Everything about GitHub, ghcr or Docker behaviour below is S or U until the first run (the card ghcr-publish proves it).
> Related: docs/research/tester-build-plan.md (the tester slices), docs/tester-licence.md (GPL pack), docs/fork-publication/PUBLISHED.md (public fork), docs/overlay.md (table overlay), deploy/d2d-container/ (Dockerfile.d2d, entrypoint.sh, compose.tester.yml, context.sh).

## Decisions (user, popup answers 2026-11-07)

1. Signing key custody: the private key is STORED AS A CI SECRET, so releases sign automatically in GitHub Actions. This is the user's explicit departure from the recommendation in 3.3 and section 8 (offline key, CI never signs); sections 3.3, 3.12, 6 and 8 below are adjusted, the old text is kept where it explains the alternative. Default mitigations the coordinator adopts (details in 3.3 and card TASK-124): a GitHub Actions ENVIRONMENT (`app-release`) with the user as required reviewer on the signing/release job, so nothing is signed without a manual approval click; the secret scoped to that environment only, not repo-wide; branch and tag protection on `tester-v*` and `release-*`; 2FA on the account; a documented public-key rotation (an image rebuild is needed); a revocation path (signed `REVOKED` file and a monotonic minimum version).
2. Repo that owns the image and the releases: `shatteredeagle/d-idle`, the fork `d2d-didle` is a pinned input (as recommended).
3. Signing tool: plain Ed25519 with our own stdlib `cmd/d2sign` (as recommended).
4. Auto-check: ON by default, daily, version-only, never downloads without a click, one off switch (as recommended).
5. Platform: amd64 only (the user did not object).

## 0. What exists today (V: read in this checkout)

- The image is built by `deploy/d2d-container/context.sh` into a scratch dir: our Go binaries (`d2bot d2dsoak d2dview d2dsetup d2dping`, CGO off, static) land in `/opt/d2lod/`; the d2d binary (GPL fork, pinned commit) lands in `/opt/d2d/` through a multi-stage build from fork source (`Dockerfile.d2d`); licence texts land in `/usr/share/licenses/d2d/`. The entrypoint is `tini -- /opt/d2lod/entrypoint.sh`, command `tester`.
- `context.sh` takes the fork source from a PRIVATE checkout today. The same commit is now public (tag `didle-gate2-lab3`, repo `shatteredeagle/d2d-didle`, PUBLISHED.md), so a public CI can build the same bytes from public sources.
- Runtime state is on one named volume `/data` (character save under `/data/config`, ladder store, `watch.token`, `wizard.json`, `access.json`). The container runs as user `d2` with `cap_drop: ALL`, `no-new-privileges`, restart `unless-stopped`, `stop_grace_period: 45s`. `d2bot --managed` saves and quits d2d on shutdown (S docs/d2d-bot.md, docs/d2d-container-bot.md).
- The watch screen (`internal/watch`) already has a token/passcode gate (`tokenGate`, Access), a viewer listener with GET routes only and a separate answer listener for writes (V `internal/watch/serve.go` header comment). That is where an Update API belongs: on the WRITE side.
- No GitHub Actions workflow publishes anything yet (only `ci.yml`, `pr-review.yml`; V `ls .github/workflows`). No ghcr package exists (U).

## 1. Design in five lines

1. Tier 1: a tag in the PUBLIC bot repo builds `ghcr.io/shatteredeagle/didle-tester` (amd64) from the bot sources plus the public fork tag; friends update with `docker compose pull && docker compose up -d`.
2. Tier 2: the image carries a fixed, thin base (OS, libs, a tiny boot shim, a public key); the APPLICATION LAYER (d2bot/d2dview/d2dsetup, d2d, overlay rows, web assets) is a signed release set that the bot downloads from a GitHub Release into the data volume.
3. Every release carries `MANIFEST.json` plus an Ed25519 signature; the image pins the public key; hashes and compatibility pairs are checked before anything is unpacked.
4. The user presses Update on the watch screen (never silent); the bot saves the hero through the existing shutdown path, the boot shim flips a `current` symlink and starts the new release, and rolls back automatically if the release is not healthy.
5. What the app layer cannot change (OS, libs, boot shim, entrypoint, key) shows "image rebuild needed" and falls back to Tier 1.

## 2. Tier 1: the registry (do first)

### 2.1 Which repo owns the image: the bot repo `shatteredeagle/d-idle`

Decision: the image, the workflow and later the app releases live in the PUBLIC bot repo; the fork is only a source input.

- The Dockerfile, entrypoint, compose files and wizard are ours and live here; the fork repo is a clean downstream of upstream (`d2d-didle`, README-DIDLE.md only) and should stay that way, so a future upstream PR path is not muddied by container files.
- The image contains two programs under two licences (fork: GPL-3.0-or-later; ours: this repo's licence). They are separate programs in separate layers (already true, Dockerfile.d2d). The package belongs to the repo that builds it; the label `org.opencontainers.image.source` points at the bot repo and a second label names the fork commit (below).
- One owner means one release stream, one version number, one place for the source offer.
- Cost: the workflow must check out the fork at a pinned tag (`actions/checkout` with `repository: shatteredeagle/d2d-didle`, `ref: <tag>`); the pin lives in one file in the bot repo (`deploy/d2d-container/D2D_PIN`: tag + commit), replacing today's hard-coded pin in `context.sh`. The workflow fails if the tag does not resolve to the pinned commit (S: same check as `context.sh` does now).

### 2.2 Workflow (draft shape; the card writes it, nothing runs)

File `.github/workflows/tester-image.yml` in the bot repo.
- Trigger: push of a tag `tester-v*` (release) and `workflow_dispatch` with a dry-run input (build, run the checks, do NOT push). Never on every push to main (the fork build is long, U 30 to 60 min on a hosted runner; use a buildx cache).
- Permissions: `contents: read`, `packages: write` (GITHUB_TOKEN login to ghcr.io; no personal token), `id-token: write` + `attestations: write` only if build provenance is turned on (recommended: cheap, S).
- Steps: checkout bot repo; checkout fork at the pin; assemble the context (a `context.sh --public` mode that reads the fork from the workflow's checkout and fetches the vendored inputs by hash, FFmpeg 9.0.2 sha256 already pinned in `context.sh`); run `go run ./cmd/leakcheck -mode fail` and the existing `tester_image_test.go` static checks; build; run the container smoke (`smoke.sh` in `d2d` mode without game data); run the "no Blizzard data" scan on the built image (below); push; create the GitHub Release draft with notes and the source offer.
- Multi-arch: amd64 ONLY at first. d2d is x86 code built from source (so arm64 is possible in principle) but nothing arm64 has ever been run (U); a Mac friend on Apple silicon would run the amd64 image under emulation, which is probably too slow for a realtime game (U). Ask the friends' hardware before spending on arm64 (open question Q5). The image architecture is stated in the guide so nobody discovers it by a failed pull.
- Reproducibility: the d2d binary in the image is the one built in the workflow from the public tag; record its sha256 in the release notes. This is a NEW binary compared with the one the Gate 2 evidence used locally; the card must compare behaviour (same fork commit, same flags) and say so (U until built; same commit, different toolchain host).

### 2.3 Tags

| Tag | Moves? | Meaning |
|---|---|---|
| `X.Y.Z` (e.g. `0.3.0`) | never | the release; the friend can pin it |
| `latest` | on each released tester tag | what a plain `pull` gets |
| `d2d-<8 hex>` | never | the fork commit inside (several bot releases share one) |
| `sha-<8 hex>` | never | the bot commit it was built from |
| `edge` (optional, later) | on main | only for the maintainer; do not announce |

`latest` moves only when a human tags a release, never on a bare push.

### 2.4 What the image contains and what it must not

- No Blizzard data: V by construction (`.dockerignore` drops `*.mpq *.d2s *.exe`; `context.sh` aborts on an `.mpq`/`game.exe`; PUBLISHED.md outside scan found none in the fork). Add a CI check on the BUILT image: export the filesystem (`docker create` + `docker export | tar t`) and fail on `.mpq .d2s .d2x .dc6 .dcc .exe .dll` outside an allow-list of the Ubuntu base, and on the leakcheck patterns. The friend mounts their own MPQ folder read-only, as today.
- Labels: `org.opencontainers.image.source=https://github.com/shatteredeagle/d-idle`, `.version`, `.revision`, `.created`, `.licenses` (the SPDX expression for both parts), `.description`; plus `io.didle.d2d.source=https://github.com/shatteredeagle/d2d-didle`, `io.didle.d2d.commit=<40 hex>`, `io.didle.d2d.tag`. S: GitHub uses the `source` label to link the package to the repo; the package is created PRIVATE and must be set public once in the package settings (U: a one-time user action, listed in the card; the card must not claim public until checked with an unauthenticated pull).
- The GPL source offer: in the image (`/usr/share/licenses/d2d/SOURCE.txt`, already planned by TASK-28), in the labels above, and in every release body (fork repo URL, tag, commit, tarball command from PUBLISHED.md, the written offer text from docs/tester-licence.md section 4). docs/tester-licence.md stays the owner of the text; the release template copies it.

### 2.5 The friend's update, and a "what changed" note

Friend flow: the friend gets `compose.friends.yml` ONCE (attached to the first release as an asset, plus the guide). Then:
```
docker compose -f compose.friends.yml pull
docker compose -f compose.friends.yml up -d
```
`up -d` recreates the container with the new image and the same `/data` volume, so character, ladder and token survive (V design: volume is named and separate; TASK-25 AC 1 already tests "same-volume image upgrade"). "What changed": the GitHub Release body for that tag (changelog lines from CHANGELOG, a "needs your attention" section, the image digest), linked from the guide; later (Tier 2) shown in the app.

Compose options (document, default OFF where risky):
- `image: ghcr.io/shatteredeagle/didle-tester:${D2_TAG:-latest}`; the friend can pin `D2_TAG=0.3.0`.
- `pull_policy`: default `missing` (Compose default behaviour made explicit); `always` makes every `up` pull (S Compose spec). Recommend leaving it `missing` and updating on purpose.
- Watchtower-style auto-update as an OPTION: a label `com.centurylinklabs.watchtower.enable` on the service, and the friend runs a Watchtower container themselves. Risks to write into the guide: it restarts the container without asking (mid-fight, hero saved by the 45 s grace period but the run is interrupted); a bad `latest` reaches everyone at once; the Watchtower container holds the Docker socket (root-equivalent on the friend's machine); the project's maintenance status must be checked before the guide recommends it (U). Recommended stance: monitor-only mode at most, or skip it; Tier 2 is the real answer.

### 2.6 Tier 1 limits (be honest with the friend)

The friend still opens a terminal for each update (two commands) until Tier 2 lands, and the image is a few hundred MB per pull (U: layers are shared, only changed layers download; the app binaries are in a late layer so an app-only change should be small).

## 3. Tier 2: in-app updater (the real goal)

### 3.1 Layout: app layer on the data volume, not under /opt

The brief sketches `/opt/didle/releases/<ver>`. `/opt` is in the image: a `docker compose up -d` after a pull RECREATES the container and any write under `/opt` is lost, and `/opt` is not writable by user `d2`. So:

```
IMAGE (read-only, replaced by Tier 1)
  /opt/didle/base/            the release baked into the image: bin/ d2d/ overlay/ web/ MANIFEST.json(+.sig)
  /opt/didle/boot             the boot shim (Go, static): selection, rollback, supervisor handoff
  /opt/didle/pubkeys/         trusted public key(s)
VOLUME /data (survives image upgrades)
  /data/didle/releases/<ver>/ unpacked releases (kept: current + previous)
  /data/didle/current         symlink -> releases/<ver>   (flipped atomically with rename(2) of a temp symlink)
  /data/didle/state.json      {current, previous, pending, good, bad[], seq_floor, boots, last_check, etag}
  /data/didle/staging/        downloads, verified before unpack, deleted after
  /data/didle/inbox/          optional manual drop of a release set (offline update)
```
The boot shim picks the release to run: the HIGHEST version among {base, current, previous} that passes the compatibility gate (3.4) and is not on the local `bad` list. So an image upgrade with a newer baked base wins over an older downloaded release, and a downloaded release newer than the base wins over the base. This keeps Tier 1 and Tier 2 from fighting.

### 3.2 Release asset layout (GitHub Release of the bot repo, tag `app-vX.Y.Z`)

| Asset | Content |
|---|---|
| `MANIFEST.json` | the signed document (below) |
| `MANIFEST.json.sig` | Ed25519 signature over the exact bytes of MANIFEST.json |
| `didle-bin-vX.Y.Z-linux-amd64.tar.gz` | `d2bot d2dsoak d2dview d2dsetup d2dping` and the boot-visible helpers |
| `didle-d2d-vX.Y.Z-linux-amd64.tar.gz` | the d2d binary (+ `crashpad_handler` if shipped) built from the named public fork tag |
| `didle-data-vX.Y.Z.tar.gz` | overlay fragments (`excel/*.txt`, docs/overlay.md format), plan/class tables if shipped as data, web assets if not embedded |
| `SOURCE.txt` | source tag/commit for the bot and the fork, the written offer |
| `didle-source-vX.Y.Z.tar.gz` (+ fork tarball) | corresponding source of the release (GPL offer made concrete) |

`MANIFEST.json` (field names are the contract; the card for the release workflow pins them):
```
{ "schema": 1, "version": "0.4.0", "seq": 17, "issued": "2026-11-20T00:00:00Z",
  "channel": "stable",
  "notes": "what changed, plain text, short",
  "min_image": 3,                       // ABI level of the base image (label io.didle.image.abi)
  "compat": { "bot_api": 4, "d2d_ctl": 2, "data_schema": 5 },   // protocol pairs, see 3.4
  "source": { "bot": "<tag/commit>", "d2d": "<tag/commit>", "offer": "SOURCE.txt" },
  "files": [ { "name": "didle-bin-...tar.gz", "sha256": "...", "size": 12345, "dest": "bin" }, ... ],
  "revoke": { "versions": ["0.3.1"] } }  // optional, signed
```
Version numbers are monotonic semver; `seq` is a monotonic integer (anti-replay, 3.7).

Is the data tarball needed? S docs/overlay.md: overlay fragments are plain tab-separated files read at startup by the fork (`--overlay-dir`) and `internal/overlay`; shipping them as data means a table fix needs no binary change (V by design of that doc). Whether web assets are embedded in the Go binary (`go:embed` under `internal/watch/web`) is U; if embedded they ride with the bin tarball and need no separate asset.

### 3.3 Signing: recommendation

Options weighed for a one-person project:

| Option | Pro | Con |
|---|---|---|
| Static Ed25519 key, offline | Protects against a compromised GitHub account, a hijacked workflow and a rogue release. Verifier is ~30 lines of `crypto/ed25519` (our rule: standard library only, AGENTS.md). Works offline. | Key custody is on the user; loss means shipping a new image; no transparency log. |
| minisign | Mature, small, the file format is familiar. | Its default prehashed signature mode uses BLAKE2b, which is not in the Go standard library (U: legacy non-prehashed mode would be stdlib-verifiable, but then we depend on its format details for little gain). |
| cosign keyless (Sigstore, OIDC from the Actions run) | No key to keep; identity is "this workflow of this repo". | The trust root is GitHub itself: a compromised repo or workflow signs happily (it protects against a stolen artifact, not a stolen account). Verification needs Sigstore roots/Rekor (network, TUF) and a large dependency, against our stdlib-only rule. |

DECIDED (2026-11-07, user): a plain Ed25519 keypair made with a tiny tool `cmd/d2sign` (stdlib only), signing the exact bytes of `MANIFEST.json`. The PRIVATE key is a secret of a GitHub Actions ENVIRONMENT, not kept offline: the release workflow builds the assets, then a signing job in that environment runs `d2sign` and uploads the `.sig`. The earlier recommendation (key offline, CI never signs, a human signs each manifest) was NOT adopted; with a CI-held key a compromised repo or Actions token can ship an update every container accepts, so the controls below carry the trust:

- Environment `app-release` with a required reviewer (the user) on the signing/release job: no release is signed without a manual approval click. The key is an environment secret only, never a repo-wide secret, so other workflows and PR runs cannot read it.
- Tag protection (rulesets) on `tester-v*` and `release-*`; the release workflow runs only from protected tags; 2FA on the owning account; no self-hosted runner for this job.
- The key pair is generated by `cmd/d2sign keygen` on the user's own machine, the private key is pasted into the secret and the local copy is destroyed or kept encrypted as a backup; it is never in the repo.
- Rotation: the image bakes up to two public keys (current, next). A new key is introduced by an image release (Tier 1) that bakes both; after friends have pulled it, releases are signed with the new key and a later image drops the old one. Because the public key lives in the image, rotating always needs an image rebuild and a pull by the friend.
- Revocation (yanking a bad release): the updater refuses any version listed in a signed `REVOKED` file (published next to the latest manifest, signed with the same key) and any version below a monotonic minimum version carried in the signed manifest and kept in `state.json`. If the key itself is compromised, rotation (above) is the only fix and is a Tier 1 event; say so to friends.

### 3.4 Compatibility gate (checked by the boot shim AND by the updater before download)

A candidate release is runnable only if ALL hold:
- `min_image <=` the running image's ABI level (label `io.didle.image.abi`, an integer we bump when the base changes in a way the app layer depends on: new system lib, new entrypoint contract, new mount).
- `compat.bot_api`, `compat.d2d_ctl`, `compat.data_schema` match what the bundled pair expects: bot and d2d are always shipped together in one release so they cannot mismatch; the data schema integer guards the saved state (ladder store, `wizard.json`, plan tables). A release that needs a newer data schema declares a one-way migration; the updater refuses it unless the migration is part of the release and a backup of the old state file set is taken first (`/data/didle/backup/<ver>/`, small files only; the character save is copied too, it is tiny).
- the signature verifies, every file hash matches, the version is above the downgrade floor, and it is not on the `bad` or `revoke` list.
If `min_image` fails, the UI shows "This update needs a new image: run docker compose pull" and nothing else (Tier 1 path).

### 3.5 The check

- Once a day (jittered) and on demand ("Check now"). Default fetch: `https://github.com/shatteredeagle/d-idle/releases/latest/download/MANIFEST.json` and `.sig` (S: the `latest/download` redirect does not count against the REST API limits; U until tried; it needs the `latest` release to be the app release, so app and tester-image releases must use separate repositories or a naming scheme: use `releases/download/<tag>` after reading the tag from the API as the fallback). Fallback: GitHub REST `GET /repos/shatteredeagle/d-idle/releases?per_page=5` filtered for tags `app-v*`. Unauthenticated REST is limited per IP (S: 60 per hour); a daily call is far below it; send `If-None-Match` with the stored ETag so an unchanged answer costs nothing (S: a 304 is cheap; whether it counts against the unauthenticated limit is U).
- Outbound calls: ONLY `github.com` and the asset host GitHub redirects to (S: `objects.githubusercontent.com`; the card lists the observed hosts after the first run). No other host, no analytics, no crash upload (the image already sets `SENTRY_DSN=` empty, V Dockerfile).
- Offline mode: `D2BOT_UPDATE_CHECK=off` (and a switch in the wizard/settings) disables all outbound calls; the screen then shows only the running version. Manual path that works with no GitHub access at all: put a downloaded release set in `/data/didle/inbox/` (or upload it through the watch screen) and press "Apply from file"; the same verification runs.
- Failure is quiet and non-blocking: network error, 403 or rate limit shows "last check failed <time>" and retries next day; it never affects the run.

### 3.6 UI and API (watch screen)

- Read side (viewer listener, GET only, token/passcode gated like the rest): `GET /api/update/status` returns `{running, base, available: {version, notes, size, needs_image}, state: idle|checking|downloading|verifying|ready|applying|error, progress, last_check, check_enabled}`. The watch screen shows a footer line with the version and, when `available` is set, a banner "Update available: 0.4.0 + changelog" with an Update button.
- Write side (the ANSWER listener, token required, origin check like the other writes): `POST /api/update/check`, `POST /api/update/download` (stage and verify, does not restart), `POST /api/update/apply` (needs `confirm=<version>` in the body so a stale tab cannot apply a newer one by accident), `POST /api/update/cancel`, `POST /api/update/rollback` (manual, to the previous release), `POST /api/update/inbox`. Never reachable on a non-token listener; a passcode-only deployment uses the same gate as answers (V internal/watch/serve.go: the Access gate judges token or cookie before the handler).
- Never silent: nothing downloads without a click unless the user turns on "download in the background" (default off, staged only, applied only on click); apply always needs a click and shows a countdown/progress with the steps (download, verify, unpack, save hero, restart, health check).
- Progress: a server-sent stream or a polled status; the page survives the restart by reconnecting and showing "applying -> checking health -> done / rolled back".

### 3.7 The apply (in order)

1. Preconditions: the release was staged and verified (hash, signature, compat gate, floor); disk free at least 3x the release size; the bot is idle-safe: no fight, no open trade, hero in town or standing safe (the bot knows its state; U exact predicate, the updater package takes it as a callback). If not safe, the button says "Apply when safe" and the request waits (visible, cancellable), or the user may choose "apply now" which only does the shutdown path below.
2. Unpack into `/data/didle/releases/<ver>.tmp`, verify each file hash again after unpack (guards against a half-written disk), run `d2bot -version` and the release's `selftest` (cheap: manifest parse, binary starts and prints) as the invoking user, rename to `releases/<ver>`.
3. Write `state.json` `pending=<ver>`, `previous=<old current>`, `boots=0`; flip `current` by creating `current.new` and `rename(2)` over it (atomic).
4. Stop: ask the bot to run its existing shutdown path (`--managed`: save and quit d2d, V per docs/d2d-bot.md), then the bot process exits 0. The container's own exit (the entrypoint ends the container when the bot ends, V entrypoint.sh) plus `restart: unless-stopped` restarts it; the boot shim reads `state.json`, runs `current`. Chosen over an in-container supervisor restart because the entrypoint/tini is the one part that is NOT updatable, so it stays as it is; the cost is a full container restart (a few seconds; the hero is saved; the wizard is skipped because `wizard.json` says done). If `restart` is not set (a friend edited the compose file), the screen says the container stopped and `docker compose up -d` is needed (U: the guide states `restart: unless-stopped` is required).
5. Health check (3.8); on success write `good=<ver>` and prune.

### 3.8 Health check and automatic rollback

The boot shim owns this, not the new release (a broken release must not be trusted to roll itself back).
- Healthy means ALL of: (a) the bot API answers its health route within 60 s of start (loopback `127.0.0.1:8420`); (b) d2d ticks: the frame/tick counter read through the tap socket (`d2dping`, V healthcheck dials the tap) advances within 3 minutes of start; (c) the watch screen listener answers; (d) the process has not exited abnormally; (e) it stays up for N = 10 minutes with no crash loop (`boots` for this `pending` version not above 2 in that window).
- Not healthy: the shim sets `current` back to `previous`, adds the version to the local `bad` list (it is never retried automatically; a newer version clears it), restarts, and the next screen load shows "Update 0.4.0 failed and was rolled back: <reason>". Crash-loop path with no bot API at all: the shim counts boots of a `pending` release; the third boot inside the window triggers rollback before starting it.
- The previous release is by definition one that ran healthy once (it is only recorded as `previous` after it was `good`). The baked base is the last resort.
- A rollback never touches `/data` other than `state.json`, except when a migration ran: then the pre-update backup (3.4) is restored (this is why migrations are rare and one-way only with backup).

### 3.9 What cannot self-update

Base OS and system libraries, the `d2` user and mounts, the boot shim, `entrypoint.sh`, tini, the baked public key(s), the container's compose settings and Docker itself. If a release needs a change there, `min_image` is raised, the updater shows "image rebuild needed: run docker compose pull" (Tier 1). Rule to keep the app layer thin and the image stable: the entrypoint stays a launcher that only calls the boot shim; all mode logic (tester, bot, d2d) moves into the release as `bin/entry-modes` (a shell or Go file in the app layer), so changes to env handling do not need an image. The entrypoint change is a follow-up on TASK-31's files and is part of the boot card.

### 3.10 Disk usage

Keep the running release and one previous (the last-known-good); delete older ones after the new one is `good`. Staging is deleted after unpack. Expected size per release tens of MB (the Go binaries are static; d2d is the largest piece, U sizes). A cap of 600 MB for `/data/didle` is checked before download.

### 3.11 GPL

Each release carries its source tag and the offer: `SOURCE.txt` and the fork source tarball are release assets (3.2) and the fork commit/tag is in the manifest `source` block. The d2d binary is only ever built by the public workflow from a public fork tag, and its tag and sha256 are in the manifest. The in-app version screen links the repo URLs and the licence directory. docs/tester-licence.md remains the text owner. The same rule as Tier 1: the first release to a friend still needs the user's go (docs/tester-licence.md section 6, item 5) and the open Freepik-icon question applies unchanged.

### 3.12 Threat model

| Threat | With signature pinning | Without / remaining |
|---|---|---|
| Compromised GitHub account or token, malicious release uploaded | Account or token alone: the signing job needs the environment approval and the environment secret, so an upload without a signature is rejected by containers. | RESIDUAL (consequence of the CI-secret decision): a compromised repo owner, Actions token with environment access, or an approved malicious workflow run can ship an update every container accepts. Mitigations: required reviewer, environment-scoped secret, tag protection, 2FA, signed `REVOKED` file and minimum version, key rotation through a Tier 1 image. |
| Compromised CI workflow or a poisoned dependency | CI signs, but only in the approval-gated environment; the reviewer sees the job summary (file names, sizes, source commits printed by `d2sign`) before clicking approve. A poisoned dependency that reaches an approved run is signed. | A reproducible build is not claimed. |
| MITM / hostile network | TLS to GitHub plus hash plus signature; a MITM can only block (denial), not substitute. | Blocking means the friend stays on the old version; the banner shows "last check failed". |
| Replay of an old, signed, vulnerable release (downgrade) | `seq` floor and version floor stored in `state.json`; lower values refused. | The floor lives on the friend's volume: someone with write access to the volume already owns the container. |
| Rogue release by the maintainer by mistake | Auto rollback on health failure covers crashes, not subtle misbehaviour (a bot that plays badly). | The staged canary (the maintainer's own container updates first) is the process mitigation. |
| A friend's compromised browser calls the update API | Token/passcode gate plus origin check plus `confirm=<version>`; the API only applies releases that verify. | Same as any watch-screen write. |
| Malicious file in a tarball (path traversal, symlinks) | Unpack code rejects absolute paths, `..`, links and device files, and refuses to write outside the release dir (a unit test with a crafted tar). | None known. |
| Pinned key is wrong or lost | Update path stops; friends stay on the last release; Tier 1 image fixes it. | Documented, tested by the "hash/sign mismatch refused" drill. |

Signature pinning buys: nobody can push code to friends with only a GitHub account. It does not buy: protection from the maintainer's key being stolen, protection from bugs, or transparency (no log; friends trust the user).

### 3.13 Privacy

No telemetry, no id, no count of installs. The only outbound traffic is the GitHub download of the manifest and assets (GitHub sees the friend's IP and a User-Agent `didle-updater/<version>`; say so in the guide). Update state stays in `/data`. The check can be switched off (3.5).

## 4. Where the code goes (offline, no game)

| Piece | Location | Note |
|---|---|---|
| manifest types, parse, verify (hash, ed25519, floor, compat) | `internal/selfupdate/manifest.go` (stdlib) | pure functions, table tests incl. two builds' compat pairs |
| fetch (GitHub release, ETag, timeouts, size caps) | `internal/selfupdate/fetch.go` | takes an `http.RoundTripper` so tests use a local server |
| stage, unpack safely, release dir, symlink flip, prune | `internal/selfupdate/store.go` | temp-dir tests, crafted-tar test |
| state file, floors, bad list | `internal/selfupdate/state.go` | atomic write |
| engine (state machine: idle, checking, downloading, verifying, ready, applying) | `internal/selfupdate/engine.go` | callbacks: `IdleSafe()`, `Shutdown()` |
| watch routes + banner | `internal/watch/update.go` (GET status) and the answer-listener route group; page piece in `internal/watch/web/` | new files only; uses the existing gate |
| flags/env | `cmd/d2bot` (`--update-check`, `--update-repo`, env twins `D2BOT_UPDATE_*`; the repo default is a constant, the flag only for tests) | the existing flag/env-twin convention in `cmd/d2bot/ladder.go` |
| boot shim | `cmd/didleboot` (static Go) | selection, health, rollback, handoff exec |
| signing tool | `cmd/d2sign` | keygen, sign, verify, prints a review summary |
| release workflow | `.github/workflows/app-release.yml` | builds assets, writes the manifest with hashes, uploads the assets; a signing job in the approval-gated `app-release` environment signs the manifest and publishes |
| image changes | `deploy/d2d-container/Dockerfile.d2d`, `entrypoint.sh` | owned by TASK-31; changes arrive as a follow-up |

The build-flexibility rule in AGENTS.md applies: nothing here encodes a class or build; the updater moves bytes only.

## 5. Test plan

Offline (no game, no host): unit tests per file above; a fake GitHub (httptest) serving a good release, a release with a hash mismatch, one with a bad signature, one with a path-traversal tar, one with `min_image` too high, one with a lower `seq`, a 304 answer, a 403 rate limit; the state machine through every state; the boot shim with a fake clock and fake health probes: a healthy update, an update that never answers (rolls back), a crash loop (rolls back on boot 3), a missing `current` (falls back to the base).

Worker host drills (Docker only, no lab game, throwaway volume): (1) happy path: image 0.3.0 running, publish a test release (to a local file server standing in for GitHub, via an env override that is compiled out of the shipped build), press Update, character and token survive, version changes; (2) bad update: a release whose bot exits at start; it rolls back, the banner says so, the old release is running; (3) hash mismatch: refused, staging cleared, nothing swapped; (4) bad signature: refused; (5) offline: check disabled, no outbound connection (verify with a network-less compose run), inbox apply works; (6) image upgrade over a newer downloaded release: the higher version wins and the older base does not downgrade it; (7) the first-run wizard and a same-volume Tier 1 pull (TASK-25 covers the latter). Each states "verified live" only after the run.

## 6. Effort and risks

| Piece | Agent-hours | Host-hours |
|---|---|---|
| Tier 1: workflow, public context mode, compose.friends.yml, guide section, release template | 6 | 2 (dry run on the worker host; the first real run needs the user) |
| Tier 2a: manifest, d2sign, release workflow, signing doc | 8 | 1 |
| Tier 2b: updater package, engine, fetch/store, watch API and banner | 14 | 1 |
| Tier 2c: boot shim, rollback, entrypoint follow-up, tests, drills | 12 | 5 |
| Version display + check banner without apply (phase 1 below) | 4 | 0.5 |
| Total | about 40 | about 9.5 |

Risks: (1) the hosted runner may not build the fork in reasonable time or may run out of disk (FFmpeg + Qt headers; U) , fallback is a self-hosted runner, which is a security question for a public repo (never for forks' PRs), so the tag-triggered, maintainers-only workflow is the rule; (2) the restart-based apply interrupts the run: mitigate by "apply when safe"; (3) boot shim bugs are unfixable by the updater (it is in the image): keep it under 300 lines with the heaviest tests; (4) a rate limit or a GitHub outage blocks updates (manual inbox path exists); (5) the signing key is a single point and now lives in CI (see 3.3 mitigations); (6) public repo workflows run on public infrastructure: the only secret beyond GITHUB_TOKEN is the signing key, environment-scoped and approval-gated; (7) the compose-file distribution problem remains: friends need the compose file once and a way to learn it changed (a release asset, version-stamped; the guide tells them to re-download it when a release says so).

## 7. Phased rollout and the friend's experience

| Phase | What ships | What the friend does |
|---|---|---|
| P0 first friend build | Tier 1 image on ghcr, `compose.friends.yml`, guide, release notes with the source offer; the version in the watch footer | gets one compose file and one guide, runs `docker compose up -d`; later runs two commands to update; learns of updates from the user (message), not from the app |
| P1 | Check (opt-out) and banner "Update available: X, changelog" with the exact `pull` commands; no download | still two commands, but the app tells them when and what changed |
| P2 | Signed app releases, Update button (download, verify, apply, restart, health check, rollback) | clicks Update; the container restarts itself; terminal only for "image rebuild needed" |
| P3 | background staging, channel choice (stable/beta), "apply when safe", inbox UI, release canary habit | rarely thinks about it |

Recommendation: the first friend build is P0 plus the P1 version display and banner (about 4 agent-hours, no key needed, no signing); P2 follows once the first friend round shows what breaks, because the updater must not be the first thing to break a friend's character. If the user wants no terminal at all in the first round, P2 must come first and the first friend build waits for it.

## 8. Decisions (all five answered 2026-11-07; see the top)

1. Signing key custody: DECIDED, CI secret in an approval-gated environment (see Decisions at the top). The original recommendation was an offline key.
2. Which repo owns the image and the releases? Recommend: `shatteredeagle/d-idle` (the bot repo) for both, with the fork as a pinned input and its tag named in every release; not the fork repo.
3. Signing tool: Recommend a plain Ed25519 key with our own stdlib `cmd/d2sign` rather than cosign or minisign (stdlib-only rule, offline, no extra trust root).
4. Auto-check default: Recommend ON with a version-only daily check to `github.com`, never downloading without a click, with one switch to turn all outbound calls off and a line in the guide that GitHub sees the IP.
5. Friends' hardware: are any on Apple silicon or ARM? Recommend amd64 only now; arm64 only if a friend needs it (emulated play is probably too slow, U).

Also needed before the first outside copy (existing, not new): your go on the first copy (docs/tester-licence.md section 6), the GPL reviewer named or waived, and the Freepik icon question.

## 9. Cards

TASK-120 ghcrpublish (Tier 1, lead card), TASK-121 updrelease (manifest, d2sign, release workflow), TASK-122 updclient (updater package, watch UI), TASK-123 updboot (boot shim, rollback, drills), TASK-124 updkey (generate the key pair, environment secret, reviewer and tag protection; the GitHub settings clicks are the user's). Order: 120, then 124 and 121, then 122 and 123 (122 can ship the version display and banner before 121 exists).
