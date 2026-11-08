#!/bin/sh
# Assemble the docker build context for Dockerfile.d2d in a scratch dir (never inside this repo).
#   deploy/d2d-container/context.sh CTX [REF]
# REF defaults to the pinned fork ref below (the binary the Gate 2 evidence used). A different REF is allowed for experiments
# and is printed in SOURCE.txt, but it is not the pinned build.
# Every input is public: the d2d source is the tag didle-gate2-lab3 of the published fork, the vendored inputs are release
# archives pinned by sha256 below. Nothing comes from a local build directory.
# Env: FORK_URL=https://github.com/shatteredeagle/d2d-didle  (default) git URL or local path to take the d2d source from
#      VENDOR_CACHE=dir  keep the downloaded archives between runs (default $TMPDIR/d2d-vendor-cache); every file is re-hashed
#      PREBUILT=path to a d2d binary: skip the source stage (build with --build-arg D2D_SRC=prebuilt)
# The fork source stays in CTX (a build artifact); this repo never holds it (GPL, docs/d2d-container/licensing.md).
# No *.mpq or Blizzard file may enter CTX: .dockerignore excludes them and the script fails if one is found.
set -eu
ctx=${1:?usage: context.sh CTX [REF]}
# Pin: branch d2d.lab3-labverbs on the private fork remote, commit f75816c616c8 (= the Gate 2 lab2-freezeresume ee5e06d8ef99 plus the
# --lab-verbs gate; the shipped tester d2d runs WITHOUT --lab-verbs, docs/d2d-container/README.md).
pin_ref=lab3-labverbs
pin_commit=f75816c616c8253930d88277b90936e3eabe3bc8
ref=${2:-$pin_commit}
fork_url=${FORK_URL:-https://github.com/shatteredeagle/d2d-didle}
pin_tag=didle-gate2-lab3
repo=$(cd "$(dirname "$0")/../.." && pwd)
# Vendored inputs (versions are what the fork pins in its CMake files: CLI11 GIT_TAG v2.6.2, sentry-native GIT_TAG 0.17.3,
# D2D_FFMPEG_VERSION/D2D_FFMPEG_SHA256). The archives are fetched here, hash-checked, and unpacked into vendor/.
cli11_url=https://github.com/CLIUtils/CLI11/archive/refs/tags/v2.6.2.tar.gz # = commit 37bb6edc5317, BSD-3-Clause
cli11_sha=c6ea6b2e5608b3ea8617999bd5f47420c71b2ebdb8dc4767c1034d1da5785711
sentry_url=https://github.com/getsentry/sentry-native/releases/download/0.17.3/sentry-native.zip # = commit fb2a04d4a9ca, with submodules, MIT
sentry_sha=930039d20b8f7d5520c8cb1d9911d63ce8a9e0635b56ae08256936d1f45b66ac
ffurl=https://ffmpeg.org/releases/ffmpeg-9.0.2.tar.xz # LGPL-2.1+ (no --enable-gpl: the fork builds a Bink-only static decoder)
ffsha=8c3850283eb25fa026482078a04051e0be17347b09ef81a0849bec15a96e002e

fetch() { # url sha256 -> verified path in the cache
	f=${VENDOR_CACHE:-${TMPDIR:-/tmp}/d2d-vendor-cache}/$(basename "$1")
	mkdir -p "$(dirname "$f")"
	if ! { [ -f "$f" ] && echo "$2  $f" | sha256sum -c - >/dev/null 2>&1; }; then
		curl -fsSL --retry 5 --retry-all-errors --connect-timeout 20 -o "$f.part" "$1" && mv "$f.part" "$f" || { echo "context.sh: download of $1 failed" >&2; exit 1; }
	fi
	echo "$2  $f" | sha256sum -c - >&2 || { echo "context.sh: sha256 mismatch for $1" >&2; exit 1; }
	echo "$f"
}

mkdir -p "$ctx/bin" "$ctx/licenses" "$ctx/vendor" "$ctx/src"
cp "$repo/deploy/d2d-container/Dockerfile.d2d" "$ctx/Dockerfile"
cp "$repo/deploy/d2d-container/entrypoint.sh" "$ctx/entrypoint.sh"
printf '*.mpq\n*.MPQ\n*.d2s\n*.exe\n.git\nvendor-archives\n' > "$ctx/.dockerignore"

# our Go side (CGO off: static, runs on any libc)
(cd "$repo" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o "$ctx/bin/" \
	./cmd/d2bot ./cmd/d2dsoak ./cmd/d2dview ./cmd/d2dsetup ./deploy/d2d-container/d2dping)

if [ -n "${PREBUILT:-}" ]; then
	cp "$PREBUILT" "$ctx/bin/d2d"
	echo "prebuilt d2d: $PREBUILT (sha256 $(sha256sum "$ctx/bin/d2d" | cut -d' ' -f1))" > "$ctx/licenses/SOURCE.txt"
else
	srcrepo=$(mktemp -d)
	trap 'rm -rf "$srcrepo"' EXIT
	git init -q "$srcrepo"
	if [ "$ref" = "$pin_commit" ]; then
		git -C "$srcrepo" fetch -q --depth 1 "$fork_url" "refs/tags/$pin_tag"
		ref=FETCH_HEAD
	else
		git -C "$srcrepo" fetch -q --tags "$fork_url" "+refs/heads/*:refs/remotes/o/*"
	fi
	commit=$(git -C "$srcrepo" rev-parse "$ref^{commit}")
	[ "$ref" != FETCH_HEAD ] || [ "$commit" = "$pin_commit" ] || { echo "context.sh: tag $pin_tag is $commit, not the pin $pin_commit" >&2; exit 1; }
	git -C "$srcrepo" archive --prefix=src/ "$commit" | tar x -C "$ctx"
	[ "$ref" != FETCH_HEAD ] || ref="$pin_ref"
	echo "$commit" | cut -c1-8 > "$ctx/src/VERSION"
	mkdir -p "$ctx/vendor-archives" "$ctx/vendor/cli11" "$ctx/vendor/sentry-native"
	a=$(fetch "$cli11_url" "$cli11_sha"); cp "$a" "$ctx/vendor-archives/"
	tar xzf "$a" -C "$ctx/vendor/cli11" --strip-components=1
	a=$(fetch "$sentry_url" "$sentry_sha"); cp "$a" "$ctx/vendor-archives/"
	if command -v unzip >/dev/null; then unzip -q -o "$a" -d "$ctx/vendor/sentry-native"; else python3 -m zipfile -e "$a" "$ctx/vendor/sentry-native"; fi
	a=$(fetch "$ffurl" "$ffsha"); cp "$a" "$ctx/vendor-archives/"
	tar xJf "$a" -C "$ctx/vendor"
	cp "$ctx/src/LICENSE" "$ctx/licenses/GPL-3.0.txt"
	cp "$ctx/src/LEGAL.md" "$ctx/licenses/d2d-LEGAL.md"
	cat > "$ctx/licenses/SOURCE.txt" <<EOT
This image contains d2d, a GPL-3.0-or-later program (upstream https://github.com/psi29a/d2decomped) with local patches.
Fork: $fork_url, tag $pin_tag ($ref), commit $commit.
Vendored build inputs (public release archives, sha256 checked by the build):
  CLI11 2.6.2 (BSD-3-Clause)        $cli11_url  sha256 $cli11_sha
  sentry-native 0.17.3 with submodules, crashpad (MIT, Apache-2.0, see licenses in the d2d install tree; crash reporting is OFF: no DSN, no --send-crash-reports)
                                    $sentry_url  sha256 $sentry_sha
  FFmpeg 9.0.2 (LGPL-2.1+, static, Bink demuxer and decoders only, no GPL parts)  $ffurl  sha256 $ffsha
Build recipe: Dockerfile and entrypoint in this context (deploy/d2d-container in the d2lod-bot repo).
NOT FOR DISTRIBUTION until the corresponding source is offered to every recipient (docs/d2d-container/licensing.md).
EOT
fi

if find "$ctx" \( -iname '*.mpq' -o -iname 'game.exe' \) | grep -q .; then
	echo "context.sh: game data found in $ctx, abort" >&2
	exit 1
fi
echo "context ready: $ctx ($(du -sh "$ctx" | cut -f1))"
