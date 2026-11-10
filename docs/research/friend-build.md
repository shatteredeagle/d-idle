# Friends beta build: a tarball with source, one command to install, a way to update

> status: PROPOSAL (2026-11-10, card friendplan, TASK-264). Docs only, nothing built or run. Answers the user decision "tarball with source plus a one-command install for one or two friends first; it must self-update or at least pull the new release". Tags: V = read in this checkout (path named), S = secondary or from a doc's own claim, U = unverified or from memory. Target: classic 1.14d; no D2R fact is used.
> Builds on: docs/dev-update.md, docs/self-update-release.md, docs/self-update-client.md, docs/self-update-drill.md, docs/research/self-update-design.md, docs/tester-licence.md, docs/tester-registry.md, docs/research/beta-readiness-review.md, docs/research/project-review-2026-11-10.md section 5. Cards: TASK-29 (pack), TASK-27 (report), TASK-48 (tour), TASK-120 (ghcr, waits), TASK-257 (legal note), TASK-121..124 (updater, done or prepared).

## 0. Answer in twelve lines

1. Most of the update machinery already exists and is the right base: signed releases (`MANIFEST.json` + Ed25519 `.sig`, `cmd/d2sign`), the client engine (`internal/selfupdate`), the boot shim with automatic rollback (`cmd/didleboot`), the watch-screen banner and Update button, and the DEV loop that proved update, tamper rejection, downgrade rejection, rollback and crash rollback in 5 to 6 seconds (V docs/dev-update.md drill 2026-11-08). The friend build should reuse them, not invent a second updater.
2. What is missing for a friend: (a) a PACK that carries the image without a registry (`docker save` tarball plus source), (b) a single `install.sh` and a small `./didle` wrapper, (c) a release FEED the friend's container can read before the public repo and the licence note are cleared, (d) a beta public key baked into the pack's image, (e) a way for the friend to report a bad update (TASK-27 plus one line).
3. Recommendation: option (b) in-place self-update with signature verification and rollback (already built, never run on a real feed), fronted by option (c) the banner (also built: it only shows "Update available" and acts on a click), plus a `./didle update` command that does the same thing from the terminal. Option (a) "pull the latest release" is the SAME engine pointed at a feed URL, not a separate mechanism.
4. The feed during the held period is a static HTTPS directory the user controls (the same layout as a GitHub release, signed with a beta key). After TASK-257 is read and approved, the feed flips to GitHub Releases of the public repo with no client change. The only code change needed is letting a non-dev image read one more trusted feed URL (section 5.4).
5. Never replace saves: the app layer lives under `/data/didle/releases/<ver>` and is swapped by a symlink; the hero, ladder, token and wizard state in `/data` are never in a release (V docs/self-update-client.md, docs/dev-update.md volume marker survived).
6. Host support v1: Linux x86-64 with Docker or Podman is the supported target; Windows through WSL2 Ubuntu is "works if the friend is Docker-literate" and must be tried before promising; macOS is not v1 (Apple silicon would run the amd64 image under emulation, U, probably too slow).
7. Effort: 6 PRs, about 26 to 34 agent-hours, plus user steps (key, one environment, one read of the legal note) of about 1 hour. Section 9.

## 1. What the friend downloads: tarball layout

One file, `didle-friends-<ver>.tar.gz`, built by `tools/tester-pack.sh` (TASK-29 owns it; its AC 1 already lists most of this, V backlog/tasks/task-29). Proposed layout (the `bin/`, `src/` and `licenses/` rows are the additions to TASK-29):

```
didle-friends-<ver>/
  README.txt              the ten lines: prerequisites, ./install.sh, where the page is, how to update, how to report
  install.sh              the one command (section 2)
  didle                   wrapper: status | update | rollback | report | token | stop | start | uninstall
  compose.yml             compose.tester.yml with the image tag set and the update env (section 5.4)
  image/didle-tester-<ver>.tar.gz     `docker save | gzip -n` of the image (about 410 MB uncompressed, S docs/tester-registry.md; gz size U)
  pubkeys/beta.pub        the same key the image bakes, so the friend can verify the manifest by hand
  src/
    d-idle-<commit>.tar.gz           `git archive` of this repo at the image commit, after `leakcheck -mode fail` (S docs/tester-licence.md section 1)
    d2d-didle-<commit12>.tar.gz      the fork source at the pinned ref plus the vendored-inputs list with hashes
    BUILD.txt                        how to rebuild the image from src/ (`context.sh`, `docker build`; 370 s cold on 16 cores, V docs/tester-registry.md)
  licenses/               GPL-3.0 text, NOTICE, SOURCE.txt with the written offer, THIRD_PARTY.md, LEGAL.md of upstream (S docs/tester-licence.md section 1)
  guide/tester-guide.md   TASK-29 AC 2 and 3: the install route, the MPQ list, troubleshooting, known weaknesses, what to send back, the first-stranger checklist
  RELEASE_NOTES.md        the CHANGELOG sections of this release (rule: every release has CHANGELOG notes)
  SHA256SUMS              hashes of every file above, and SHA256SUMS.sig (Ed25519, same beta key) so a friend can check the pack itself
```

Rules (all existing rules, restated as pack checks):
- No Blizzard file, ever: no `*.mpq`, `game.exe`, `d2*.dll`, `*.d2s`, no CD key. The workflow already scans the built image for these (V docs/tester-registry.md section 1); the pack script repeats the scan over the whole pack tree and a test checks the file list (V TASK-29 AC 1). The friend brings their own 1.14d files; the compose file mounts the folder read-only at `/game` (V `deploy/d2d-container/compose.tester.yml`: `${D2_GAME_DIR}:/game:ro`).
- Source is in the box (GPL): the image holds the GPL d2d fork binary, so the pack carries the corresponding source and the written offer. This is the cheap way to meet the distribution duty for a closed circle of friends; it does not decide the public-image question (TASK-257). U: whether a source tarball in the same pack fully satisfies GPLv3 section 6 for the Ubuntu base layers; docs/tester-licence.md section 1 proposes a pointer to the distribution's source repositories plus `dpkg -l`, still marked U (S).
- The tarball is not public. The user hands it over by any private route (LocalSend, a share link, a USB stick). That keeps TASK-120 and the licence decision out of the critical path, as the user decided.
- No secret inside: no token, no private key, no hostname, no home path (`leakcheck`, V AGENTS.md). The beta PUBLIC key is the only key material.
- The pack is reproducible in the way the app assets are (sorted names, mtime 0, `gzip -n`, V docs/self-update-release.md), so a friend and the publisher can compare hashes.

## 2. The one-command install and the first-run checks

`./install.sh [--game-dir PATH]` (bash, POSIX tools, `docker`). It never uses sudo and never touches anything outside its own directory, the Docker image and one named volume. Steps, each printing one line "ok" or "STOP: reason + what to do":

| # | Check or action | Fails with |
|---|---|---|
| 1 | `docker --version` and `docker compose version` (v2) work; user can reach the daemon; host is x86-64 (`uname -m`) | "Docker is not installed or not running", "this build is amd64 only" (S design decision 5) |
| 2 | `sha256sum -c SHA256SUMS` (the manifest signature of each later release is verified again by the image itself) | "the download is damaged; fetch it again" |
| 3 | Licence note shown, `y` required (stored as `.licence-ack` with the version): GPL source offer, "bring your own game", "no Blizzard data included, Diablo II is a trademark", "nothing is sent anywhere except the update check to ..." (section 5.5). `--yes` is refused unless the ack file exists | the text of `licenses/` |
| 4 | Game dir: given or asked. Required files by name: `d2data.mpq`, `d2exp.mpq`, `d2char.mpq`, `Patch_D2.mpq` (V docs/tester-live.md, docs/first-session.md "Do this first"); sizes non-zero; the folder is read, never written (mounted `:ro`). The 1.14d version verdict is the wizard's data check (V first-session.md), not the script's | "missing: <names>; point --game-dir at your own Diablo II folder" |
| 5 | Disk free at least 3 GB (image 410 MB, volume, update copies; U number) and ports 8891 and 8892 free on loopback | the port or space message |
| 6 | `docker load -i image/...`, verify the loaded image has the label `io.didle.image.abi` and the version equals the pack's | "image does not match the pack" |
| 7 | write `.env` (D2_IMAGE, D2_GAME_DIR, D2_WATCH_ORIGIN=`http://localhost:8891`, D2BOT_UPDATE_*), `docker compose up -d` | compose output |
| 8 | wait up to 120 s for the bot health URL; then print the version line, the setup/watch URL, the token ONCE (also recoverable with `./didle token`), "warm-up is 1 to 3 minutes" (V first-session.md) | `./didle report` hint |

The "version line" is `d.idle <ver> (image <abi>, d2d <commit12>, channel beta)`; it must also be in the watch header (TASK-47 owns the page; beta review section 7 asks for it, V). First-run checks are exactly steps 1 to 8 plus the wizard's own data check; nothing else is needed before the user sees the wizard.

`./didle` subcommands are thin shells over `docker compose` and `docker exec`: `status` (version line, update state, last health), `update`, `rollback`, `report` (TASK-27's exec script, writes the scrubbed tarball to the current directory and prints what to send), `token`, `stop` (the 45 s grace saves the hero, V compose), `start`, `uninstall` (removes container and image; the volume is kept unless `--purge-data` and a typed confirmation).

Windows via WSL2 (U): the same scripts run inside an Ubuntu WSL distro with Docker Desktop's WSL integration or Docker Engine in WSL. Bind-mounting the game folder from `/mnt/c/...` is slow and sometimes denies metadata; the guide says "copy the four MPQ files into your WSL home first". Nobody has tried this; the first Windows friend is the test. A PowerShell installer is NOT v1.

## 3. Update model: the three options weighed

| Option | What it is | Security | Friend effort | Build cost | Verdict |
|---|---|---|---|---|---|
| (a) pull the latest release | `./didle update`: fetch the feed's release list, download assets, verify, apply | same as (b) if it reuses the manifest and signature; unsafe if it is a bare `curl | tar` | one command | low if it shells into the existing engine | not separate: this is the CLI front of (b) |
| (b) in-place self-update | the built engine: check, download, verify signature and hashes, unpack into `releases/<ver>`, swap `current`, restart through the shim, health probation, automatic rollback | Ed25519 pinned key; signature checked before anything is believed; hash and size caps; safe unpack; downgrade floor; revocation list; no unsigned code ever runs (V docs/self-update-client.md, docs/dev-update.md fault table) | click Update or `./didle update` | engine done offline (S: unit tests), DEV drill done live (V); never against a real feed or a hero in a fight (U) | RECOMMENDED |
| (c) banner only | "Update available: X, what changed" with no install; the friend re-runs a command | no code path at all; smallest attack surface | two commands, friend must read the banner | the banner exists (V update.js, wording "Update available: X"); the "what changed" text comes from the signed `notes` | keep as the default front end of (b): it already requires a click for download and a second click for install (V design 3.6, client doc) |

Recommendation (one choice): ship (b) with (c) as its entry. The friend sees the banner, clicks Download then Install, or types `./didle update`, which POSTs to the same two routes with the token (so there is one code path, one set of checks). Nothing downloads or installs without a human action; the daily check only asks for the version and the signed manifest (V design decision 4: ON by default, version-only, one off switch). Why not (c) alone: a friend who must run `docker load` of a new 400 MB image for every bot fix will not update, and the maintainer's own findings ("fix lands, friend plays the old bug") would repeat. Why not unattended auto-install: the restart interrupts a hero (the `IdleSafe` hook is nil today, so Apply can interrupt a fight, U docs/self-update-client.md); a click keeps the friend in control.

What can NOT self-update, and falls back to a new pack: the base image (OS, libs), the shim, the entrypoint's outer shell, the baked public key, the ABI label. The updater already reports "image rebuild needed" (V design 3.9). For the beta this means: most weeks are app releases; a rare "pack 2" release is a new tarball and `./install.sh` again (the volume is kept).

## 4. Where the feed lives and why not GitHub first

The built client talks to `api.github.com`, `github.com` and `*.githubusercontent.com` only; a different base URL is honoured ONLY when the image is a DEV image (`DIDLE_DEV=1` gates `D2BOT_UPDATE_URL` and `D2BOT_UPDATE_PUBKEY`, V cmd/d2bot/update.go and docs/dev-update.md). That is correct for the player image and a gap for this plan.

| Feed | Pros | Cons |
|---|---|---|
| GitHub Releases of the public repo (design 3.2, tag `app-vX.Y.Z`, the inert workflow `.github/workflows/app-release.yml`, V) | no new infrastructure; the signed-asset format and the CI job exist; free bandwidth; later the normal path | the release assets contain the d2d GPL binary and the data overlay: attaching them to a PUBLIC release IS publication, the thing TASK-120 and TASK-257 hold back; the user has not read the legal note |
| A static directory the user controls (same file layout as a release: `MANIFEST.json`, `.sig`, tarballs, optional `REVOKED`) behind HTTPS | no publication decision; invisible to strangers; trivially revocable by deleting files; the bytes are still signed, so the host is untrusted | needs a host that is not a dev box (not a dev workstation; a home server) and a hostname the friends can reach (a tunnel); the client needs a small allow-list change (5.4); the friend's IP is visible to the user's host (say so, same as GitHub seeing it) |
| A signed manifest URL only (assets mirrored elsewhere) | one small file to host | same code change, more moving parts, no benefit |

Recommendation: the user-controlled static directory for the held period, in exactly the GitHub release layout (so the engine, tests and `tools/release/app-assets.sh` are unchanged); flip to GitHub Releases after TASK-257 and the user's go. The feed host is untrusted by design: it can serve nothing the key did not sign, and cannot make the client accept an older release (the `seq` and version floor, V docs/dev-update.md "rollback attack" row).

## 5. Security model

### 5.1 What is signed and by whom

Release assets are as in docs/self-update-release.md: `MANIFEST.json` (version, monotonic `seq`, `channel`, `min_image`, `compat`, file hashes, optional `revoke`) and `MANIFEST.json.sig`, an Ed25519 signature over the exact manifest bytes with the stdlib `d2sign` (V). The pack's image bakes up to two public keys from `deploy/d2d-container/pubkeys/` (V update_image_test.go, boot shim reads them). The friend build uses a BETA key: not the DEV key (dev key stays on the worker host and is never the release key, V docs/dev-update.md), and ideally the same key that the later public release uses, so friends do not re-key. U: whether one key for beta and stable is acceptable; a leaked beta key then also signs stable. Proposal: one key now, a second key slot ("next") reserved for rotation (the format supports two, V).

### 5.2 Key handling without secrets in the repo

- Generated by `d2sign keygen` on the user's own machine; only `beta.pub` is committed (to `deploy/d2d-container/pubkeys/`); the private file is 0600 and never printed (V d2sign tests refuse a group-readable key).
- Existing user decision (2026-11-07): the private key is a secret of the GitHub environment `app-release` with the user as required reviewer; no release signs without an approval click; tag rulesets on `app-v*` (V docs/research/self-update-design.md Decisions, docs/self-update-release.md "Turning it on", TASK-124). That decision stands. Interim until the environment exists (nothing has run on GitHub, U): the user runs the release script locally; the private key lives in a file outside any repo or in a password manager, never in a worktree, an env var in shell history, or a CI variable of a PR workflow. This interim is a procedure, not a design change; tag it open (question 3).
- Residual risk the user already accepted: a compromised account with environment approval can ship signed code (design 3.12). Mitigation for friends: the first apply of each release shows the notes and file list the reviewer saw; the maintainer's own container takes the release first (canary, design 3.12).

### 5.3 No remote code execution from an unauthenticated feed

The feed is untrusted: (1) the signature is checked before any field is believed; (2) the host allow-list is enforced on redirects; (3) archives are unpacked by code that refuses absolute paths, `..`, links, device nodes and size overruns; (4) the release is never executed until the shim verified the stored manifest against a baked key again at selection time; (5) compatibility pairs and ABI are checked before download; (6) a version at or below the floor, or on the revocation list, is refused. All V as unit tests or the DEV fault drill (docs/self-update-client.md, docs/self-update-drill.md, docs/dev-update.md fault table: bad signature, no signature, tampered manifest, tampered tarball, downgrade all REJECTED).

### 5.4 The one code change the plan needs (small, explicit)

Let a non-DEV image read ONE extra feed URL and keep the key pinned: add `D2BOT_UPDATE_FEED` (default empty = GitHub as today). When set it replaces the default host in the allow-list (never adds a wildcard), must be `https://`, and the trusted keys are still only the baked ones (the DEV key is not accepted outside DEV). The pack's compose sets it to the user's feed. Because the key is baked, a friend who edits `.env` to point at another feed gains nothing: the new feed cannot produce a valid signature. This is a design change the user should confirm (question 2); it touches `cmd/d2bot/update.go` and `internal/selfupdate/fetch.go` (not owned by this card). The release-list JSON may be served as a static file in the same shape as the GitHub API response (the dev server already does, V dev-update.md "`<URL>/repos/<repo>/releases`").

### 5.5 Privacy

No telemetry. The only outbound request is the daily update check (and downloads on a click) to the feed host, which sees the friend's IP and the `didle-updater/<ver>` User-Agent (V design 3.13). The install note says so; `D2BOT_UPDATE_CHECK=off` turns it off, `manual` keeps the button (V docs/dev-update.md).

## 6. Versions, channels, rollback

Versions: strict `x.y.z` in the manifest (V). Proposed lines:
- DEV: `0.0.<yymmddHHMMSS>`, signed with the DEV key, LAN server, DEV image only (V today).
- BETA: `0.<minor>.<patch>` with `channel: "beta"`, built from a tagged commit, signed with the beta key, served from the feed. Tag `app-v0.N.P`. Image releases (new pack) bump the pack version and the image ABI label.
- STABLE: the same format later with `channel: "stable"` (design 3.5 already lists stable/beta as P3, U).

The `channel` field is required in the manifest and stored (V manifest.go) but I found no client code that filters by it (U: `grep channel internal/selfupdate` shows only the schema). v1 proposal: the pack's env sets `D2BOT_UPDATE_CHANNEL=beta` and the client ignores manifests with another channel; if that is not built the beta feed simply contains beta releases only. Do not build channel switching UI for two friends.

Rollback: automatic (the shim, a failed health probation or crash loop; the version is put on a bad list and never retried, V drills) and manual (`./didle rollback`, the engine's Rollback, V `ROLLED-BACK ... in 2s`; keeps the last two releases). The container, volume and saves are untouched in every case (V marker file survived every step).

Restart that saves the hero: apply stops the bot with SIGTERM, the same path as `docker stop`, so the character is saved before the new release starts (S docs/self-update-client.md; V in the DEV drill only for the wizard state, a hero in a fight was NOT drilled, U). Therefore the apply drill with a live hero is a gate for this plan (section 10, drill D4). Until it passes, the guide says "update when the hero is standing in town".

## 7. How a friend reports a bad update

1. The page shows the banner "Update X failed and was rolled back" and the bad list (V design 3.8). The friend runs `./didle report` (TASK-27: scrubbed tarball of bot log, d2d log, minute lines, deaths, ladder journal, update state and `image.json`, with tokens, key-shaped text, paths and hostnames removed; V TASK-27 AC 1) and sends the file by the same private route as the pack.
2. The report must contain: the version line before and after, `state.json` (current, previous, pending, bad, floors), the rollback marker, the last 200 lines of the shim and bot logs. Add these to TASK-27's list (one line in its brief).
3. A bad release that did not crash but plays badly (the health check cannot see this): the friend runs `./didle rollback` and reports. The publisher can ban the version for every friend by publishing a signed `REVOKED` next to the latest manifest (V design 3.3: a revoked version is refused and a floor can be raised), which takes effect on the next check.
4. If the key is ever suspected stolen: the only fix is a new pack with a new baked key (V design 3.3). Say so in the guide; two friends can be told in a message.

## 8. How the publisher cuts a release

Reuse `tools/release/app-assets.sh` and `d2sign` (V). One workflow, one approval:

1. Land the CHANGELOG section for the version (RULE: every release has notes; `RELEASE_NOTES.md` is generated from the first two sections of CHANGELOG, V docs/release.md; the release job fails when the section is missing).
2. Tag `app-vX.Y.Z` (protected tag) or dispatch the workflow with the version. Job `build`: gate (`go vet`, short tests on the worker host or the hosted runner, `leakcheck -mode fail`), `app-assets.sh` with reproducible tarballs, no secrets (V draft, inert).
3. Job `sign` in environment `app-release`: the reviewer reads the `d2sign summary` text in the job summary (file names, sizes, sources) and approves; the job signs and verifies against the pinned public key (V draft).
4. Job `publish`: GitHub Releases after the legal go; during the held period the same assets are copied to the user's feed directory by a documented script (`tools/release/feed-publish.sh`, new, small: refuse unless `d2sign verify` passes with the baked key).
5. Canary: the maintainer's own container updates first (`dev-update.sh` style, or the real feed), 10 minutes of play, then the friends are told. A new PACK (rare) is built by `tools/tester-pack.sh` from the same tag and handed over privately.
6. Release notes are the signed `notes` field plus RELEASE_NOTES.md; each friend sees them in the banner.

Today `app-release.yml` is a draft, dispatch only and its d2d build step deliberately fails until wired to the Dockerfile builder stage (V docs/self-update-release.md); finishing that is PR 3 below.

## 9. Host support, v1

| Host | v1 | Notes |
|---|---|---|
| Linux x86-64, Docker Engine + compose v2 (or rootless Podman, U) | supported | the image has run on Linux for a one-hour realtime session with 0 stalls and 0 deaths (V docs/tester-live.md); a second-machine run exists (V first-session.md). `cap_drop: ALL`, `no-new-privileges`, 3 GB memory, 2 CPUs are in the compose file (V) |
| Windows 10 or 11 | best effort via WSL2 + Docker | not tried (U); copy MPQs into WSL first; no PowerShell installer in v1. CPU load matters: the sandbox shows several traced calls per frame and a laptop may feel it (U, beta review section 7) |
| macOS | not v1 | the image is amd64 only; Apple silicon would emulate (U, likely too slow); no arm64 build has ever run (V design 2.2) |

Ask the two friends their hardware before sending anything (design decision 5, still the open question).

## 10. Effort (PRs and agent-hours)

| PR | What | Owner card | Agent-hours |
|---|---|---|---|
| 1 | `tools/tester-pack.sh` + `install.sh` + `didle` wrapper + pack file-list test + licence text embed | TASK-29 (+ the install script) | 6 to 8 |
| 2 | `D2BOT_UPDATE_FEED` (non-dev feed, https only, keys unchanged), `D2BOT_UPDATE_CHANNEL`, tests incl. an attempt to add the DEV key outside DEV | new card (container/bot) | 4 to 6 |
| 3 | finish `app-release.yml` (d2d stage wired to the Dockerfile builder), `feed-publish.sh`, release job fails without CHANGELOG notes, key setup doc | TASK-121/124 follow-up | 6 to 8 |
| 4 | `./didle update|rollback` calling the update API; update fields in the TASK-27 report; version line in the page header (TASK-47) | TASK-27 + small card | 3 to 4 |
| 5 | guide and first-stranger checklist incl. update and report steps | TASK-29 | 3 to 4 |
| 6 | drills D1..D5 below on the worker host (no game files shipped, hero from the lab) | live-agent | 4 to 6 |

Total about 26 to 36 agent-hours over 6 PRs, 3 of them on shared surfaces (cmd/d2bot, the Dockerfile, watch) so they queue one at a time. User steps: generate and store the key (15 min), create the environment and reviewer (10 min), read the legal note, hand over the first pack (20 min). TASK-27, 29 and 48 are in the same card family and none of them is blocked by TASK-120.

Drills before the first copy leaves the machine (V = must be run, not yet run): D1 pack install from a clean directory on a clean Docker host (wrong dir, missing MPQ, port busy, bad hash); D2 update from feed A to B with a throwaway hero standing in town; D3 the DEV faults against the beta feed (bad sig, tamper, downgrade, revoked); D4 apply with a hero fighting (the hero saves, comes back, nothing lost; the open U above); D5 bad release rolls back and the friend's report contains the needed fields. A first-friend pass uses the existing first-stranger checklist (TASK-29 AC 3).

## 11. What not to do

- No unsigned or hash-only "pull" path, no `curl | sh` for the update, no auto-install without a click, no telemetry, no silent feed change.
- No Blizzard files in the pack or the feed, and no game-folder writes (the mount is read-only).
- No private hostnames, IPs or paths in the pack (leakcheck); the feed URL is a value the user fills in `.env` at pack time, it is not committed.
- Do not promise Windows or macOS before a friend has tried it.

## 12. Open questions (for the user, no decision taken here)

1. Hardware: are the one or two friends on Linux, Windows or a Mac? That sets the host table.
2. Confirm the small client change of section 5.4 (a feed URL in a non-DEV image, key still pinned)? Without it the friend build can only update from GitHub, which means publishing first.
3. Beta key custody for the interim: local signing on your machine until the GitHub environment exists, or wait for the environment?
4. One key for beta and stable, or a beta key that is retired when the public image ships?
5. Where does the feed live (a Tower directory behind a tunnel is the obvious choice) and may friends' IPs be visible there?
6. GPL: is a source tarball in the pack enough for a private circle of two friends until the legal note is read (TASK-257), or do you want the reviewer named first (S docs/tester-licence.md section 6)?

## 13. Proposed ADR text (for the coordinator or the grill session; NOT written to docs/adr or decisions.md)

> ADR (proposed): Friends beta is a private signed pack, updated through the signed-release engine.
> Status: proposed. Context: publishing the image (TASK-120) waits for the legal note (TASK-257) and the exit criteria; the user wants one or two friends to play and to receive fixes. Decision: (1) the beta is a private tarball built by `tools/tester-pack.sh` containing the saved image, the corresponding source for the GPL parts, licence files, an installer and a guide, and no Blizzard file; (2) updates use the existing signed-release engine (Ed25519 manifest, pinned baked key, boot shim rollback), entered by the page banner or `./didle update`, never silently; (3) during the held period the feed is a user-controlled static directory in the GitHub release layout, signed with the beta key, and a non-DEV image may read one `https` feed URL while its keys stay pinned; after TASK-257 the same releases go to GitHub Releases; (4) a new pack is required only when the base image, shim or baked key change; (5) v1 hosts: Linux x86-64 with Docker; Windows via WSL2 is best effort; macOS is out. Consequences: the beta needs no registry, the update path is exercised on real feeds before the public release, and the key must be generated and stored by the user before the first pack leaves the machine.
