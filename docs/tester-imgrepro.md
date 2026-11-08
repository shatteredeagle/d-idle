# Tester image from public inputs (TASK-139)

> status: 2026-11-07. Tags: V verified (how stated), S secondary, U unverified. Closes defect D9 of `docs/tester-live.md`.

## Inputs
`deploy/d2d-container/context.sh CTX` needs only network access to these public sources; no local fork checkout, no build directory, no private remote. Every archive is checked against a sha256 pinned in the script (a mismatch aborts).

| Input | Version | Fetched from | sha256 | Licence |
|---|---|---|---|---|
| d2d fork source | tag `didle-gate2-lab3`, commit `f75816c616c8` (checked against the pin) | `https://github.com/shatteredeagle/d2d-didle` (shallow tag fetch) | commit hash (content addressed) | GPL-3.0-or-later |
| CLI11 | 2.6.2 (= commit 37bb6edc5317) | `https://github.com/CLIUtils/CLI11/archive/refs/tags/v2.6.2.tar.gz` | `c6ea6b2e5608b3ea8617999bd5f47420c71b2ebdb8dc4767c1034d1da5785711` | BSD-3-Clause |
| sentry-native with submodules (crashpad, lss) | 0.17.3 (= commit fb2a04d4a9ca) | `https://github.com/getsentry/sentry-native/releases/download/0.17.3/sentry-native.zip` | `930039d20b8f7d5520c8cb1d9911d63ce8a9e0635b56ae08256936d1f45b66ac` | MIT (crashpad Apache-2.0; third-party licences installed by the fork under `share/licenses`) |
| FFmpeg | 9.0.2 | `https://ffmpeg.org/releases/ffmpeg-9.0.2.tar.xz` | `8c3850283eb25fa026482078a04051e0be17347b09ef81a0849bec15a96e002e` | LGPL-2.1+ |

V: the versions are the fork's own pins (`CMakeLists.txt` FetchContent `GIT_TAG v2.6.2`; `apps/d2d/CMakeLists.txt` `GIT_TAG 0.17.3` and `D2D_FFMPEG_VERSION`/`D2D_FFMPEG_SHA256`), read from a fresh clone of the public tag; the two git tags resolve to the commits the old local build dir held (`git ls-remote`).

## Decisions
- FFmpeg: stays the fork's own static source build, not the distro package. The fork's options are `--disable-everything` plus the Bink demuxer and decoders only, no `--enable-gpl`, no `--enable-nonfree`: LGPL-2.1+ code, linked statically into the GPL d2d binary (under 1 MB). The distro `libav*` would add every codec and about 100 MB. The source archive is kept in the context (`vendor-archives/`, excluded from the docker context by `.dockerignore`) so the pack can ship it for the source offer.
- sentry-native: cannot be switched off by a CMake option without a fork patch (the fork builds it unconditionally unless `D2D_SENTRY_PREFIX` is set). It is compiled in but inert: crash reports are sent only when d2d is started with `--send-crash-reports` AND `SENTRY_DSN` is non-empty (V `apps/d2d/main.cpp`, `SentrySession`/`start_sentry`). The image sets `SENTRY_DSN=` empty, builds with `-DD2D_SENTRY_DSN=` (launcher only), and the entrypoint never passes the flag; `tester_image_test.go` pins all three. The tester sends nothing anywhere.
- The fetch lives in `context.sh` (host side), not in the Dockerfile, so the Dockerfile keeps its offline build stage and the archives can go into the source pack.

## Proof
V, worker host, 2026-11-07 (EST evening): a clean scratch copy of the tracked repo files, `HOME` pointed at an empty directory (so no `~/d2d-lab` checkout), `context.sh` then `nice -n 19 docker build` of throwaway tag `d2tester139-20261107`.
- Context: all four archives downloaded and sha256-checked; `SOURCE.txt` in the image lists fork URL, tag, commit and the three vendored inputs with URL and hash.
- Build: about 8 min cold on the worker host (shared with other jobs; U exact, job timestamps 19:59 start, done before 20:07 EST). Image 409 MB (the earlier image from the local build dir: 409 MB). Rebuild from cache: 15 s.
- Static self-test `DRYRUN_ONLY=1 deploy/d2d-container/smoke.sh`: `PASS (dry run only)`.
- Container start without game data (`TESTER=1`, empty `/game`): the wizard starts and prints the token URL (`/setup.html?t=...`); without the token 401, with the token 200 after the cookie redirect.
- Binary: `/opt/d2d/d2d` of the new image and of the image built from the local build dir are both 14963192 bytes and differ in 25 bytes (sha256 `2ec4f79e...` against `947c107d...`). The only strings that differ are the build timestamps (`2026-11-08T00:59:25Z` against `2026-11-06T21:59:11Z`), so the build is reproducible apart from the stamp the fork writes into the binary (V `cmp -l`, strings; the 25 bytes were not each mapped to the stamp: U).
- Cleanup: image, container, volume and scratch directory removed.

## Placeholders for `docs/tester-licence.md` (TASK-28 owner)
"Vendored inputs" row and the line `Vendored inputs (name, version, sha256)`: take the table above (CLI11 2.6.2, sentry-native 0.17.3 with submodules, FFmpeg 9.0.2 with hashes). SDL3, OpenAL Soft, StormLib, libcurl, zlib and bzip2 are Ubuntu 26.04 packages, not vendored. Note: the fork installs the sentry/crashpad third-party licence files into its install tree, but the image copies only `d2d` and `crashpad_handler`; the NOTICE should carry those licence texts (crashpad Apache-2.0, mini_chromium, lss, zlib, cpp-httplib, getopt, mpack, libunwind).

## Not public-only
Nothing remains local. The sentry-native zip and the CLI11 GitHub archive are served by GitHub; if GitHub regenerates the CLI11 tag archive the hash check fails loudly (then re-pin; the commit 37bb6edc5317 is the content anchor).
