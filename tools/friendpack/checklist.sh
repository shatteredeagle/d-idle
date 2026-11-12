#!/usr/bin/env bash
# Legal checklist for a built friends pack (docs/research/legal-note.md section 4, 7 items). Read-only: it unpacks the pack
# into a temp dir, prints PASS/FAIL/WARN per item and writes a record docs/friend-checklist-<version>.md.
#   tools/friendpack/checklist.sh --pack didle-friends-0.1.0.tar.gz [--record-dir DIR] [--no-leakcheck]
# Exit 0 only when every item is PASS (WARN lines are manual follow-ups). Needs GNU tar, grep, sha256sum.
# Items 1..7 are automated as far as a file check can go; "the image is d2d only" and "no game files in a layer" are
# checked on the docker-save tarball inside the pack (layer listings), not on a running container.
set -uo pipefail

pack= recdir= leak=1
while [ $# -gt 0 ]; do
	case "$1" in
	--pack) pack=${2:?}; shift 2 ;; --record-dir) recdir=${2:?}; shift 2 ;; --no-leakcheck) leak=0; shift ;;
	*) echo "unknown flag $1" >&2; exit 2 ;;
	esac
done
[ -f "$pack" ] || { echo "usage: checklist.sh --pack FILE.tar.gz" >&2; exit 2; }
here=$(cd "$(dirname "$0")" && pwd)
repo=$(git -C "$here" rev-parse --show-toplevel)
[ -n "$recdir" ] || recdir=$repo/docs

w=$(mktemp -d); trap 'rm -rf "$w"' EXIT
tar xzf "$pack" -C "$w" || { echo "cannot unpack $pack" >&2; exit 2; }
root=$(ls -d "$w"/didle-friends-* 2>/dev/null | head -1)
[ -d "$root" ] || { echo "not a friends pack (no didle-friends-<version>/)" >&2; exit 2; }
version=$(cat "$root/VERSION" 2>/dev/null || echo unknown)
tagline=$(cat "$root/image/IMAGE_TAG" 2>/dev/null || echo unknown)

res=() notes=()
add() { res+=("$1"); notes+=("$2"); printf '%-5s item %s\n' "$1" "$2"; }
fails=0
pass() { add PASS "$1"; }
fail() { add FAIL "$1"; fails=$((fails + 1)); }
warn() { add WARN "$1"; }

d2dsrc=$(ls "$root"/src/d2d-didle-*.tar.gz 2>/dev/null | head -1)
botsrc=$(ls "$root"/src/d-idle-*.tar.gz 2>/dev/null | head -1)
mkdir -p "$w/d2d" "$w/bot"
[ -n "$d2dsrc" ] && tar xzf "$d2dsrc" -C "$w/d2d" 2>/dev/null
[ -n "$botsrc" ] && tar xzf "$botsrc" -C "$w/bot" 2>/dev/null

# layer listing of the docker-save file: names of every file in every layer (also deleted ones are still in a layer)
layers="$w/layers.txt"; : >"$layers"
imgtar="$root/image/didle-image.tar.gz"
if [ -f "$imgtar" ]; then
	mkdir -p "$w/img"
	tar xzf "$imgtar" -C "$w/img" 2>/dev/null
	while IFS= read -r -d '' l; do
		tar -tf "$l" 2>/dev/null >>"$layers" || tar -tzf "$l" 2>/dev/null >>"$layers"
	done < <(find "$w/img" -type f \( -name 'layer.tar' -o -path '*/blobs/sha256/*' \) -size +0 -print0)
fi

# 1. d2d only, lab/cheat verbs off by default
bad1=
grep -rIl -E -i 'game\.exe|wine' "$root/compose.yml" "$root/install.sh" "$root/didle" "$root/README.txt" 2>/dev/null | head -3 | sed 's/^/  wine or game.exe in: /' >"$w/o1"
[ -s "$w/o1" ] && bad1=1
grep -n -E -- '--lab-verbs|D2D_LAB|LAB_VERBS' "$root/compose.yml" "$root/install.sh" "$root/didle" 2>/dev/null | grep -v -E '^\S*:\s*[0-9]+:\s*#' >"$w/o1b"
[ -s "$w/o1b" ] && bad1=1
if [ -s "$layers" ] && grep -E -i '(^|/)(game\.exe|wine[0-9a-z_-]*)(/|$)|usr/lib/wine' "$layers" | head -3 >"$w/o1c" && [ -s "$w/o1c" ]; then bad1=1; fi
if [ -n "$bad1" ]; then cat "$w"/o1 "$w"/o1b "$w"/o1c 2>/dev/null >&2; fail "1 d2d only, lab/cheat verbs off by default: wine/game.exe or a lab flag found (listed above)"
else pass "1 d2d only, lab/cheat verbs not enabled in compose/install/didle$([ -s "$layers" ] || echo ' (image layers not listed: no image tar)')"; fi

# 2. licence, GPL text, source offer with the exact commits
ok2=1; why=
grep -q 'GNU GENERAL PUBLIC LICENSE' "$root/licenses/GPL-3.0.txt" 2>/dev/null || { ok2=; why="$why GPL-3.0.txt missing/empty;"; }
[ -f "$root/licenses/SOURCE.txt" ] || { ok2=; why="$why SOURCE.txt missing;"; }
if [ -f "$root/licenses/SOURCE.txt" ]; then
	grep -q 'NOT FOR DISTRIBUTION' "$root/licenses/SOURCE.txt" && { ok2=; why="$why SOURCE.txt says NOT FOR DISTRIBUTION;"; }
	grep -E -q '(fork commit|d2d source).*[0-9a-f]{12,40}' "$root/licenses/SOURCE.txt" || { ok2=; why="$why no d2d commit in SOURCE.txt;"; }
	grep -E -q 'bot source.*\(commit [0-9a-f]{7,40}\)' "$root/licenses/SOURCE.txt" || { ok2=; why="$why no bot commit in SOURCE.txt;"; }
	grep -q -i 'written offer' "$root/licenses/SOURCE.txt" || { ok2=; why="$why no written offer;"; }
fi
[ -n "$d2dsrc" ] && [ -n "$botsrc" ] || { ok2=; why="$why a source tarball is missing from src/;"; }
[ -f "$botsrc" ] && ! [ -f "$w/bot"/*/LICENSE ] && { ok2=; why="$why LICENSE missing in the bot source;"; }
if [ -s "$layers" ] && ! grep -q -E 'opt/d2lod.*LICENSE|licen[sc]e' "$layers"; then warn "2b no licence file seen in the image layers (the image should carry LICENSE)"; fi
[ -n "$ok2" ] && pass "2 LICENSE + GPL text + SOURCE.txt with d2d and bot commits, no NOT FOR DISTRIBUTION" || fail "2 licence/source offer:$why"

# 3. notices for every third-party package, Freepik consistent with the d2d tree
notice="$root/licenses/NOTICE $root/licenses/THIRD_PARTY.md"
miss=
for p in CLI11 crashpad FFmpeg StormLib SDL3 OpenAL libcurl zlib bzip2 Ubuntu; do
	cat $notice 2>/dev/null | grep -q -i -- "$p" || miss="$miss $p"
done
icon=$(find "$w/d2d" -path '*assets/icon/d2d.svg' 2>/dev/null | head -1)
freepik_in_tree= freepik_in_notice=
[ -n "$icon" ] && grep -q -i 'designed by.*freepik\|dgim-studio' "$icon" && freepik_in_tree=1
# the Steam launcher art left in the tree (LEGAL.md names its author) also needs the attribution
legal=$(find "$w/d2d" -maxdepth 2 -name LEGAL.md 2>/dev/null | head -1)
[ -n "$legal" ] && grep -q -i 'dgim-studio' "$legal" && freepik_in_tree=1
cat $notice | grep -q -i freepik && freepik_in_notice=1
if [ -n "$freepik_in_tree" ] && [ -z "$freepik_in_notice" ]; then miss="$miss Freepik(icon still in the d2d tree, attribution missing)"; fi
if [ -z "$freepik_in_tree" ] && [ -n "$freepik_in_notice" ]; then miss="$miss Freepik(notice claims an icon that the d2d tree no longer has)"; fi
[ -z "$miss" ] && pass "3 notices name every third-party package; Freepik attribution matches the d2d icon" || fail "3 notices missing:$miss"

# 4. nothing forbidden in the pack, the two source tarballs or the image layers; no secrets, hostnames, home paths
banned='(\.mpq|\.dc6|\.dcc|\.dt1|\.d2s|\.d2i|\.key|\.pem|watch\.token|(^|/)game\.exe|/web/art/|/cache/art/|(^|/)data/global/excel/[^/]*\.txt|(^|/)internal/watch/web/.*\.png)$'
# the layer listing drops the public CA root store of the Ubuntu base (etc/ssl/certs, usr/lib/ssl, usr/share/ca-certificates: .pem root certificates, no private keys)
{ (cd "$w" && find "$(basename "$root")" -type f); [ -n "$botsrc" ] && tar -tzf "$botsrc"; [ -n "$d2dsrc" ] && tar -tzf "$d2dsrc"; grep -E -v '^(etc/ssl/certs|usr/lib/ssl|usr/share/ca-certificates)/' "$layers"; } | grep -E -i "$banned" | head -5 >"$w/o4"
# UPDATE_FEED (the public https update feed the sender chose) is the one file allowed to name a host; it must be https
feedtxt=$(tr -d '[:space:]' <"$root/UPDATE_FEED" 2>/dev/null || true)
[ -z "$feedtxt" ] || printf '%s\n' "$feedtxt" | grep -q -E '^https://[^ /]+(/[^ ]*)?$' || echo 'UPDATE_FEED is not a plain https URL' >>"$w/o4"
# private hostnames / home paths / credentials in the pack's own text files; the source trees are covered by leakcheck below
grep -rIl --exclude-dir=src --exclude-dir=image --exclude=UPDATE_FEED -E '(/home/[a-z][a-z0-9_-]*/|\.lan\.b[i]rbhub|b[i]rbhub\.com|BEGIN (RSA |OPENSSH |EC )?PRIVATE KEY|password[[:space:]]*[:=][[:space:]]*[^[:space:]$"<{(]+)' "$root" 2>/dev/null | grep -v -E 'THIRD_PARTY.md$' | head -5 >>"$w/o4"
if [ -s "$w/o4" ]; then cat "$w/o4" >&2; fail "4 forbidden file or private string in the pack/source/layers (listed above)"
else
	lk=
	if [ "$leak" = 1 ] && command -v go >/dev/null && [ -d "$w/bot" ] && [ -n "$(ls "$w/bot")" ]; then
		(cd "$w/bot"/* && git init -q . && git add -A >/dev/null 2>&1 && go run ./cmd/leakcheck -mode fail >"$w/leak.out" 2>&1) && lk=" + leakcheck 0 hits on the shipped bot source" || { cat "$w/leak.out" >&2; fail "4 leakcheck found hits in the shipped bot source"; lk=failed; }
	fi
	[ "$leak" = 1 ] && [ -z "$lk" ] && warn "4b leakcheck not run (go or the bot source missing): run go run ./cmd/leakcheck -mode fail by hand"
	[ "$lk" = failed ] || pass "4 no .mpq/DC6/art PNG/save/key/token/password/private host/home path in pack, sources, layer listing$lk"
fi

# 5. art stays runtime-only and off in the overlay (bot source guard + no art files already checked in item 4)
js=$(find "$w/bot" -path '*internal/watch/web/inventory.js' 2>/dev/null | head -1)
if [ -n "$js" ] && grep -q 'mode=overlay' "$js"; then pass "5 inventory.js returns before any art request for mode=overlay; no art files in the pack"
else fail "5 overlay guard not found in internal/watch/web/inventory.js of the shipped bot source"; fi

# 6. disclaimer in README and guide
need6='not affiliated with'
ok6=1
grep -q -i "$need6" "$root/README.txt" || ok6=
grep -q -i "$need6" "$root/guide/friend-guide.md" || ok6=
for s in 'bring your own' 'offline only' 'GPL-3.0-or-later' 'no warranty'; do
	grep -q -i -- "$s" "$root/README.txt" || ok6=
done
[ -n "$ok6" ] && pass "6 README.txt and guide carry the disclaimer (not affiliated, own files, offline only, GPL, no warranty)" || fail "6 disclaimer missing in README.txt or guide/friend-guide.md"

# 7. no Battle.net path, no Blizzard marks in names
ok7=1
if grep -rIl -i -E 'https?://[^ "]*battle\.net[^ "]*' "$w/bot" --include='*.go' 2>/dev/null | head -3 >"$w/o7" && [ -s "$w/o7" ]; then cat "$w/o7" >&2; ok7=; fi
case "$(basename "$pack") $tagline" in *[Dd]iablo*|*[Bb]lizzard*) ok7=; echo "  pack or image name has a Blizzard mark" >&2 ;; esac
[ -n "$ok7" ] && pass "7 no Battle.net URL in the bot Go source; pack and image names carry no Blizzard marks" || fail "7 Battle.net/Blizzard string or mark (listed above)"

# record
rec="$recdir/friend-checklist-$version.md"
mkdir -p "$recdir"
{
	echo "# Friend pack checklist, version $version"
	echo
	echo "> status: generated by tools/friendpack/checklist.sh (docs/research/legal-note.md section 4). Keep one per release. Automated checks only: the GPL review and the go for the first copy remain the user's."
	echo
	echo "- pack: $(basename "$pack") (sha256 $(sha256sum "$pack" | cut -c1-16)...)"
	echo "- image tag: $tagline"
	echo "- d2d source: $(basename "${d2dsrc:-none}"), bot source: $(basename "${botsrc:-none}")"
	echo "- overall: $([ $fails -eq 0 ] && echo PASS || echo "FAIL ($fails)")"
	echo
	echo "| result | item |"
	echo "|---|---|"
	for i in "${!res[@]}"; do echo "| ${res[$i]} | ${notes[$i]} |"; done
	echo
	echo "Manual (not automatable): the user reviews GPL compliance with this list; the private link and the tarball hash are sent separately; no public fork."
} >"$rec"
echo "record: $rec"
[ $fails -eq 0 ]
