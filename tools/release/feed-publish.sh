#!/usr/bin/env bash
# Publish a signed app release to a static update feed (docs/friend-feed.md). Host-agnostic: it writes a feed tree
# into a directory, checks it with the real update client and the pinned public key, and only then runs YOUR upload
# step. No host, credential or key is stored here; a private key path is only ever passed to d2sign.
#   tools/release/feed-publish.sh --release DIR --base-url https://feed.example/d2 --out DIR
#       [--pub FILE]        pinned public key (default deploy/d2d-container/pubkeys/current.pub)
#       [--sign-key FILE]   sign MANIFEST.json with d2sign first when MANIFEST.json.sig is missing
#       [--repo owner/name] API path inside the tree (default: the client's default repo)
#       [--upload]          after a green verify, run the upload step below
#       [--withdraw X.Y.Z]  instead of publishing: hide that release (files stay), verify, optionally upload
# Upload step, first match wins (settings via the environment or the gitignored local.env, see tools/env.sh):
#   D2_FEED_UPLOAD_CMD   a command run as: sh -c "$D2_FEED_UPLOAD_CMD" feed-upload TREE_DIR   (your hook)
#   D2_FEED_RSYNC_DEST   an rsync/ssh destination such as user@host:/srv/feed/   (assets first, then the list; no --delete)
# Idempotent: publishing the same release again changes nothing; old releases and their files stay for rollback.
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd)
repo_root=$(cd "$here/../.." && pwd)
# shellcheck disable=SC1091
. "$repo_root/tools/env.sh"

release= base=${D2_FEED_BASE_URL:-} out= pub=${D2_FEED_PUBKEY:-} signkey= repo= upload=0 withdraw=
while [ $# -gt 0 ]; do
	case "$1" in
	--release) release=${2:?}; shift 2 ;; --base-url) base=${2:?}; shift 2 ;; --out) out=${2:?}; shift 2 ;;
	--pub) pub=${2:?}; shift 2 ;; --sign-key) signkey=${2:?}; shift 2 ;; --repo) repo=${2:?}; shift 2 ;;
	--upload) upload=1; shift ;; --withdraw) withdraw=${2:?}; shift 2 ;;
	*) echo "unknown flag $1" >&2; exit 2 ;;
	esac
done
[ -n "$base" ] && [ -n "$out" ] || { echo "--base-url (or D2_FEED_BASE_URL) and --out are required" >&2; exit 2; }
[ -n "$pub" ] || pub=$repo_root/deploy/d2d-container/pubkeys/current.pub
repoflag=()
[ -z "$repo" ] || repoflag=(-repo "$repo")
d2feed() { (cd "$repo_root" && go run ./cmd/d2feed "$@"); }

if [ -n "$withdraw" ]; then
	d2feed withdraw -out "$out" -version "$withdraw" ${repoflag[@]+"${repoflag[@]}"}
else
	[ -n "$release" ] || { echo "--release is required" >&2; exit 2; }
	if [ ! -f "$release/MANIFEST.json.sig" ]; then
		[ -n "$signkey" ] || { echo "STOP: $release/MANIFEST.json.sig is missing; sign first (--sign-key FILE, or d2sign sign)" >&2; exit 1; }
		(cd "$repo_root" && go run ./cmd/d2sign sign -key "$signkey" -in "$release/MANIFEST.json")
	fi
	d2feed build -release "$release" -pub "$pub" -base-url "$base" -out "$out" ${repoflag[@]+"${repoflag[@]}"}
fi
d2feed verify -out "$out" -pub "$pub" -base-url "$base" ${repoflag[@]+"${repoflag[@]}"}

if [ "$upload" = 0 ]; then
	echo "tree ready in $out (verified, not uploaded). Upload it, or rerun with --upload."
	exit 0
fi
if [ -n "${D2_FEED_UPLOAD_CMD:-}" ]; then
	sh -c "$D2_FEED_UPLOAD_CMD" feed-upload "$out"
elif [ -n "${D2_FEED_RSYNC_DEST:-}" ]; then
	rsync -a "$out/dl/" "${D2_FEED_RSYNC_DEST%/}/dl/"
	rsync -a "$out/repos/" "${D2_FEED_RSYNC_DEST%/}/repos/"
else
	echo "STOP: --upload needs D2_FEED_UPLOAD_CMD or D2_FEED_RSYNC_DEST (tools/env.sh, local.env)" >&2
	exit 1
fi
echo "uploaded. Check from a friend's side: ./didle update"
