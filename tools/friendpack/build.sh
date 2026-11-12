#!/usr/bin/env bash
# Build the private friends pack (docs/friend-guide.md, docs/research/friend-build.md section 1). It never publishes
# and never touches the real signing key; it writes ONE tarball and a checksum file next to it.
#   tools/friendpack/build.sh --version 0.1.0 --image NAME:TAG --d2d-src FILE --d2d-commit SHA --out DIR
#       [--image-tar FILE]   a ready `docker save` file (default: docker save NAME:TAG | gzip -n)
#       [--pubkey FILE]      public key shown in the pack (default: every key baked into the image tree, pubkeys/*.pub; with this flag only FILE, as beta.pub)
#       [--feed URL]         https update feed written into the pack (empty: the image default)
#       [--bot-ref REF]      commit of this repo to ship as source (default HEAD)
#       [--force]            do not fail the build when tools/friendpack/checklist.sh reports a FAIL (it still warns)
# The pack itself is checked by SHA256SUMS; releases the bot downloads later are signed (cmd/d2sign) and checked in the image.
# Reproducible: sorted names, zero mtime, root owner, gzip -n. Needs GNU tar, git, sha256sum.
set -euo pipefail

version= image= imgtar= d2dsrc= d2dcommit= out= pubkey= feed= botref=HEAD force=
while [ $# -gt 0 ]; do
	case "$1" in
	--version) version=${2:?}; shift 2 ;; --image) image=${2:?}; shift 2 ;; --image-tar) imgtar=${2:?}; shift 2 ;;
	--d2d-src) d2dsrc=${2:?}; shift 2 ;; --d2d-commit) d2dcommit=${2:?}; shift 2 ;; --out) out=${2:?}; shift 2 ;;
	--pubkey) pubkey=${2:?}; shift 2 ;; --feed) feed=${2:-}; shift 2 ;; --bot-ref) botref=${2:?}; shift 2 ;; --force) force=1; shift ;;
	*) echo "unknown flag $1" >&2; exit 2 ;;
	esac
done
for v in version image d2dsrc d2dcommit out; do
	[ -n "${!v}" ] || { echo "--${v/d2dsrc/d2d-src} is required" >&2; exit 2; }
done
[[ $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "version must be x.y.z" >&2; exit 2; }
[[ $d2dcommit =~ ^[0-9a-f]{12,40}$ ]] || { echo "--d2d-commit must be a hex commit id" >&2; exit 2; }
[ -f "$d2dsrc" ] || { echo "d2d source tarball not found: $d2dsrc" >&2; exit 2; }
if [ -n "$feed" ]; then [[ $feed == https://* ]] || { echo "--feed must be https://" >&2; exit 2; }; fi

here=$(cd "$(dirname "$0")" && pwd)
repo=$(git -C "$here" rev-parse --show-toplevel)
if [ -n "$pubkey" ]; then pubkeys=("$pubkey"); else pubkeys=("$repo"/deploy/d2d-container/pubkeys/*.pub); fi
for k in "${pubkeys[@]}"; do
	[ -f "$k" ] || { echo "public key not found: $k" >&2; exit 2; }
	if grep -q 'PRIVATE' "$k"; then echo "that is not a public key file" >&2; exit 2; fi
done
botcommit=$(git -C "$repo" rev-parse --short=12 "$botref")
d2dshort=${d2dcommit:0:12}

name=didle-friends-$version
mkdir -p "$out"
out=$(cd "$out" && pwd)
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
root=$stage/$name
mkdir -p "$root"/{image,pubkeys,src,licenses,guide}

# the files that run
install -m 0755 "$here/install.sh" "$here/didle" "$root/"
install -m 0644 "$here/compose.yml" "$here/README.txt" "$root/"
echo "$version" >"$root/VERSION"
echo "$image" >"$root/image/IMAGE_TAG"
printf '%s\n' "$feed" >"$root/UPDATE_FEED"
if [ -n "$pubkey" ]; then
	install -m 0644 "$pubkey" "$root/pubkeys/beta.pub"
else
	for k in "${pubkeys[@]}"; do install -m 0644 "$k" "$root/pubkeys/$(basename "$k")"; done
fi
install -m 0644 "$here/licence-note.txt" "$root/licenses/licence-note.txt"
install -m 0644 "$repo/docs/friend-guide.md" "$root/guide/friend-guide.md"

# the image
if [ -n "$imgtar" ]; then
	[ -f "$imgtar" ] || { echo "image tar not found: $imgtar" >&2; exit 2; }
	case "$imgtar" in *.gz) cp "$imgtar" "$root/image/didle-image.tar.gz" ;; *) gzip -nc "$imgtar" >"$root/image/didle-image.tar.gz" ;; esac
else
	docker save "$image" | gzip -n >"$root/image/didle-image.tar.gz"
fi

# source (GPL): this repo at the shipped commit, and the d2d fork tarball as given
botname=d-idle-$botcommit
git -C "$repo" archive --format=tar --prefix="$botname/" "$botref" | gzip -n >"$root/src/$botname.tar.gz"
cp "$d2dsrc" "$root/src/d2d-didle-$d2dshort.tar.gz"
cat >"$root/src/BUILD.txt" <<BUILD
How to rebuild the image from the source in this folder
1. tar xzf $botname.tar.gz ; cd $botname
2. deploy/d2d-container/context.sh CTX   (assembles the build context; the d2d source tarball of this folder is the
   d2d input, see docs/d2d-container/README.md in the source for the exact flags)
3. docker build -t $image CTX
The image holds the GPL program d2d (d2d-didle-$d2dshort.tar.gz) and the bot (Go, standard library only).
BUILD

# licences and notices
install -m 0644 "$repo/LICENSE" "$root/licenses/GPL-3.0.txt"
install -m 0644 "$repo/THIRD_PARTY.md" "$root/licenses/THIRD_PARTY.md"
# the Freepik line is written only while the d2d tree still carries that icon (replaced icon: no attribution needed)
freepik=
iconpath=$(tar -tzf "$d2dsrc" 2>/dev/null | grep -E 'assets/icon/d2d\.svg$' | head -1 || true)
if [ -n "$iconpath" ] && tar -xzOf "$d2dsrc" "$iconpath" 2>/dev/null | grep -q -i freepik; then freepik=1; fi
{
cat <<'NOTICE'
d2d (d.idle fork)
Based on d2decomped by psi29a and contributors, GPL-3.0-or-later.
Modifications: lockstep clock, dump and lab verbs (off unless --lab-verbs), table overlay (--overlay-dir).
d.idle bot (Go, standard library only): GPL-3.0-or-later, source in src/.
Not affiliated with or endorsed by Blizzard Entertainment. Diablo II is a trademark of Blizzard Entertainment.
No Blizzard game data is included; you supply your own legally owned copy of version 1.14d.

Third-party components keep their own licences (full list: THIRD_PARTY.md in this folder):
  CLI11 (BSD-3-Clause), crashpad (Apache-2.0), FFmpeg (LGPL/GPL build as shipped in the d2d tree), StormLib (MIT),
  SDL3 (zlib), OpenAL (LGPL), libcurl (curl licence), zlib (zlib licence), bzip2 (bzip2 licence);
  Ubuntu base packages: each package's copyright file is under /usr/share/doc/<package> in the image.
NOTICE
if [ -n "$freepik" ]; then
	echo 'The d2d window icon (goat skull and pentagram) is "Designed by dgim-studio / Freepik", used under the Freepik licence.'
fi
} >"$root/licenses/NOTICE"
d2dsha=$(sha256sum "$d2dsrc" | cut -d' ' -f1)
cat >"$root/licenses/SOURCE.txt" <<SOURCE
This pack contains the program d2d, built from the d.idle fork of d2decomped, and the bot that drives it.
  d2d source : src/d2d-didle-$d2dshort.tar.gz  (fork commit $d2dcommit, sha256 $d2dsha)
  bot source : src/$botname.tar.gz  (commit $botcommit)
Build recipe: src/BUILD.txt. The programs are licensed GPL-3.0-or-later; the licence text is in GPL-3.0.txt.
Third-party libraries built into d2d keep their own licences (THIRD_PARTY.md). The Ubuntu base layers of the image are
covered by their distribution's source repositories for the package versions in the image
(docker run --rm --entrypoint dpkg $image -l lists them).
Written offer: if any source named here is missing from the pack, ask the person who sent it; a copy will be sent
on request for at least three years from the date you received it, at no more than the cost of the medium.
SOURCE
# release notes: the first two CHANGELOG sections
awk '/^## /{n++} n<=2' "$repo/CHANGELOG.md" >"$root/RELEASE_NOTES.md"

# game, art and secret guard over the pack tree and the listings of both source tarballs
banned='(\.mpq|\.dc6|\.dcc|\.dt1|\.d2s|\.d2i|\.key|\.pem|watch\.token|(^|/)game\.exe|(^|/)d2[a-z0-9_]*\.dll|/web/art/|/cache/art/|(^|/)data/global/excel/[^/]*\.txt|(^|/)internal/watch/web/.*\.png)$'
listing() {
	(cd "$stage" && find "$name" -type f)
	tar -tzf "$root/src/$botname.tar.gz"
	tar -tzf "$root/src/d2d-didle-$d2dshort.tar.gz"
}
if listing | grep -Ei "$banned" >&2; then
	echo "STOP: game, art or secret files in the pack (listed above)" >&2
	exit 1
fi

# checksums (d2sign signs release manifests only; a signed SHA256SUMS is not part of this version)
(cd "$root" && find . -type f ! -name SHA256SUMS ! -name SHA256SUMS.sig | LC_ALL=C sort | sed 's#^\./##' | xargs sha256sum >SHA256SUMS)
tar --sort=name --mtime=@0 --owner=0 --group=0 --numeric-owner -C "$stage" -cf - "$name" | gzip -n >"$out/$name.tar.gz"
(cd "$out" && sha256sum "$name.tar.gz" >"$name.tar.gz.sha256")
echo "pack: $out/$name.tar.gz"
cat "$out/$name.tar.gz.sha256"

# legal checklist (docs/research/legal-note.md section 4); the record goes next to the pack
if ! "$here/checklist.sh" --pack "$out/$name.tar.gz" --record-dir "$out" ${CHECKLIST_FLAGS:-}; then
	if [ -n "$force" ]; then echo "WARNING: the legal checklist reports FAIL (continuing because of --force)" >&2
	else echo "WARNING: the legal checklist reports FAIL; do not send this pack (re-run with --force to silence)" >&2; exit 3; fi
fi
