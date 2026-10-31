#!/bin/sh
# Assemble the docker build context for Dockerfile.d2d in a scratch dir (never inside this repo).
#   deploy/d2d-container/context.sh CTX [REF]
# Env: FORK=~/d2d-lab/fork (private fork checkout: src/ with its build/_deps from an earlier build)
#      PREBUILT=path to a d2d binary: skip the source stage (build with --build-arg D2D_SRC=prebuilt)
# The private fork source/binary stays in CTX (a local build artifact); this repo never holds it (GPL, docs/d2d-container/licensing.md).
# No *.mpq or Blizzard file may enter CTX: .dockerignore excludes them and the script fails if one is found.
set -eu
ctx=${1:?usage: context.sh CTX [REF]}
ref=${2:-lab5}
fork=${FORK:-$HOME/d2d-lab/fork}
repo=$(cd "$(dirname "$0")/../.." && pwd)
ffsha=8c3850283eb25fa026482078a04051e0be17347b09ef81a0849bec15a96e002e # ffmpeg-9.0.2.tar.xz (docs/d2d-container/design.md 1.1)

mkdir -p "$ctx/bin" "$ctx/licenses" "$ctx/vendor" "$ctx/src"
cp "$repo/deploy/d2d-container/Dockerfile.d2d" "$ctx/Dockerfile"
cp "$repo/deploy/d2d-container/entrypoint.sh" "$ctx/entrypoint.sh"
printf '*.mpq\n*.MPQ\n*.d2s\n*.exe\n.git\n' > "$ctx/.dockerignore"

# our Go side (CGO off: static, runs on any libc)
(cd "$repo" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o "$ctx/bin/" \
	./cmd/d2bot ./cmd/d2dsoak ./cmd/d2dview ./deploy/d2d-container/d2dping)

if [ -n "${PREBUILT:-}" ]; then
	cp "$PREBUILT" "$ctx/bin/d2d"
	echo "prebuilt d2d: $PREBUILT (sha256 $(sha256sum "$ctx/bin/d2d" | cut -d' ' -f1))" > "$ctx/licenses/SOURCE.txt"
else
	git -C "$fork/src" archive --prefix=src/ "$ref" | tar x -C "$ctx"
	commit=$(git -C "$fork/src" rev-parse "$ref")
	echo "$commit" | cut -c1-8 > "$ctx/src/VERSION"
	deps=$fork/src/build/_deps
	for p in cli11-src:cli11 sentry_native-src:sentry-native; do
		rm -rf "$ctx/vendor/${p#*:}"
		mkdir -p "$ctx/vendor/${p#*:}"
		(cd "$deps/${p%%:*}" && tar c --exclude=.git .) | tar x -C "$ctx/vendor/${p#*:}"
	done
	tb=$fork/src/build/apps/d2d/ffmpeg_build-prefix/src/ffmpeg-9.0.2.tar.xz
	echo "$ffsha  $tb" | sha256sum -c - >&2
	tar xJf "$tb" -C "$ctx/vendor"
	cp "$fork/src/LICENSE" "$ctx/licenses/GPL-3.0.txt"
	cp "$fork/src/LEGAL.md" "$ctx/licenses/d2d-LEGAL.md"
	cat > "$ctx/licenses/SOURCE.txt" <<EOT
This image contains d2d, a GPL-3.0-or-later program (upstream https://github.com/psi29a/d2decomped) with local patches.
Fork: private repo <owner>/d2d-fork, ref $ref, commit $commit.
Vendored build inputs: CLI11 37bb6edc, sentry-native fb2a04d4 (with submodules), FFmpeg 9.0.2 (sha256 $ffsha).
Build recipe: Dockerfile and entrypoint in this context (deploy/d2d-container in the d2lod-bot repo).
NOT FOR DISTRIBUTION until the corresponding source is offered to every recipient (docs/d2d-container/licensing.md).
EOT
fi

if find "$ctx" \( -iname '*.mpq' -o -iname 'game.exe' \) | grep -q .; then
	echo "context.sh: game data found in $ctx, abort" >&2
	exit 1
fi
echo "context ready: $ctx ($(du -sh "$ctx" | cut -f1))"
