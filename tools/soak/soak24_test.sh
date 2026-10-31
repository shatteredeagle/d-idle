#!/bin/bash
# Offline test of the --for duration parsing of soak24.sh: bash tools/soak/soak24_test.sh
# Only rejected durations are run through soak24.sh itself (they exit before any ssh); a valid one would start a bot.
set -u
here="$(cd "$(dirname "$0")" && pwd)"
. "$here/duration.sh"
bad=0
fail() { echo "FAIL: $*"; bad=$((bad + 1)); }
ok_case() { got="$(soak_secs "$1")" && [ "$got" = "$2" ] || fail "soak_secs '$1': got '${got:-}' want $2"; }
bad_case() { out="$(soak_secs "$1")" && fail "soak_secs '$1' must be rejected, printed '$out'"; [ -z "$out" ] || fail "soak_secs '$1' printed '$out'"; return 0; }
ok_case 24h 86400
ok_case 2h 7200
ok_case 90m 5400
ok_case 1h30m 5400          # the review-evening L15 case: used to be an arithmetic error
ok_case 45s 45
ok_case 1h2m3s 3723
ok_case 2m30s 150
ok_case 5400 5400
ok_case 08m 480             # leading zero is decimal, not octal
ok_case 01h09m 4140
for g in "" garbage h m 1x 1h30 1.5h -1h 1h-30m 30m1h 1H 1h30mm "1 h" "1h 30m" 0 0h 0m0s 1h30m5 '$(id)' '1h;ls'; do bad_case "$g"; done
# soak24.sh refuses a bad --for with exit 2 before any ssh or preflight
for g in garbage 1h30 0m; do
  out="$("$here/soak24.sh" start i4 --for "$g" --skip-preflight 2>&1 </dev/null)"; rc=$?
  { [ "$rc" = 2 ] && echo "$out" | grep -q "must be a duration"; } || fail "soak24.sh --for $g: rc=$rc out=$out"
done
# tools/stamp.sh: env override, git describe, and the no-.git content-hash fallback
repo="$(cd "$here/../.." && pwd)"
[ "$(D2_VERSION=abc1234 "$repo/tools/stamp.sh")" = abc1234 ] || fail "stamp.sh D2_VERSION override"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/tools" "$tmp/cmd/x" "$tmp/internal"; cp "$repo/tools/stamp.sh" "$tmp/tools/"; echo 'package main' > "$tmp/cmd/x/a.go"; : > "$tmp/go.mod"
v1="$("$tmp/tools/stamp.sh")"; echo 'package main // x' > "$tmp/cmd/x/a.go"; v2="$("$tmp/tools/stamp.sh")"
case "$v1" in tree-????????) ;; *) fail "stamp.sh no-git form: $v1" ;; esac
[ "$v1" != "$v2" ] || fail "stamp.sh content hash did not change with the tree"
# mounts.sh: a network mount that does not answer is named, a healthy one and non-network ones are not
. "$here/mounts.sh"
mf="$tmp/mounts"; printf 'proc /proc proc rw 0 0\n//t/ok %s cifs rw 0 0\n//t/gone /nonexistent/mnt\\040x cifs rw 0 0\n/dev/x /nonexistent/ext ext4 rw 0 0\n' "$tmp" >"$mf"
[ "$(hung_mounts "$mf")" = "/nonexistent/mnt x" ] || fail "hung_mounts: '$(hung_mounts "$mf")'"
# stopclean.sh: cleans after the bot is gone, never while it lives
. "$here/stopclean.sh"
repo="$tmp"; printf '#!/bin/bash\necho "$@" >>%s/gamecalls\n' "$tmp" >"$tmp/game.sh"; chmod +x "$tmp/game.sh"
export SOAK_GAME_SH="$tmp/game.sh" SOAK_STOP_POLL=0 SOAK_STOP_WAIT=4
ssh_() { return 1; }          # pgrep finds no bot
soak_stop_clean i4 >/dev/null; [ "$(cat "$tmp/gamecalls" 2>/dev/null)" = "stop i4" ] || fail "stopclean did not run game.sh stop i4"
rm -f "$tmp/gamecalls"; ssh_() { return 0; }   # bot never goes away
soak_stop_clean i4 >/dev/null 2>&1 && fail "stopclean claimed success with a live bot"; [ ! -e "$tmp/gamecalls" ] || fail "cleaned under a live bot"
# watchfmt.py: status text, level, and each alert with its since time and new codes
wf="$(printf 'soak 3h0m: lvl 12\nlive 60m: clvl 12, 8400 xp/h, deaths 1, recovery 3.2%%, top state farming 62%%\nunit: active\n{\n  "updated": "2026-10-30T12:00:00Z",\n  "level": "crit",\n  "alerts": [\n    {"level":"crit","code":"death_streak","msg":"3 deaths in 20m","since":"2026-10-30T11:42:10Z"},\n    {"level":"warn","code":"state_dominant","msg":"x"}\n  ]\n}\n' | python3 "$here/watchfmt.py")"
echo "$wf" | grep -q '^live 60m: clvl 12, 8400 xp/h' || fail "watchfmt: live summary line: $wf"
echo "$wf" | grep -q '^level: crit$' || fail "watchfmt: level: $wf"
echo "$wf" | grep -q 'ALERT crit death_streak (since 11:42): 3 deaths in 20m' || fail "watchfmt: alert with since: $wf"
echo "$wf" | grep -q 'ALERT warn state_dominant: x' || fail "watchfmt: alert without since: $wf"
# layout.sh: per-instance soak paths, ports and bot patterns; the old shared layout is still matched for a running i4
. "$here/layout.sh"
[ "$(soak_new_dir i5)" = $HOME/d2lod-live/soak24-i5 ] || fail "soak_new_dir i5: $(soak_new_dir i5)"
[ "$(soak_old_dir)" = $HOME/d2lod-live/soak24 ] || fail "soak_old_dir"
[ "$(D2_SOAK_ROOT=/x soak_new_dir i3)" = /x/soak24-i3 ] || fail "D2_SOAK_ROOT override"
[ "$(soak_port i4)" = 8420 ] && [ "$(soak_port i5)" = 8445 ] && [ "$(soak_port i3)" = 8443 ] || fail "soak_port"
[ "$(D2_BOT_PORT=9000 soak_port i5)" = 9000 ] || fail "D2_BOT_PORT override"
oldcmd="$HOME/d2lod-live/soak24/d2bot --managed --instance i4 --char Soaky --addr 127.0.0.1:8420"
newcmd="$HOME/d2lod-live/soak24-i5/d2bot --managed --instance i5 --char Fresha --addr 127.0.0.1:8445"
echo "$oldcmd" | grep -qE "$(soak_bot_pat i4 old)" || fail "old pattern must match the running i4 bot"
echo "$oldcmd" | grep -qE "$(soak_bot_pat i5 old)" && fail "old pattern for i5 must not match the i4 bot"
echo "$oldcmd" | grep -qE "$(soak_bot_pat i4 new)" && fail "new pattern for i4 must not match the old-layout bot"
echo "$newcmd" | grep -qE "$(soak_bot_pat i5 new)" || fail "new pattern must match the i5 bot"
echo "$newcmd" | grep -qE "$(soak_bot_pat i4 new)" && fail "new pattern for i4 must not match the i5 bot"
echo "$newcmd" | grep -qE "$(soak_bot_pat i5 old)" && fail "old pattern must not match a new-layout bot"
# stop i5 must not touch the old shared bot: drive soak24.sh stop with a stub ssh that records what it is asked to run
stub="$tmp/stub"; mkdir -p "$stub"
printf '#!/bin/bash\nshift; echo "$*" >>%s/ssh.log; case "$*" in *"pgrep -af"*) exit 1;; esac\n' "$tmp" >"$stub/ssh"; printf '#!/bin/bash\nexit 0\n' >"$stub/scp"; chmod +x "$stub/ssh" "$stub/scp"
printf '#!/bin/bash\nexit 0\n' >"$tmp/game0.sh"; chmod +x "$tmp/game0.sh"
SOAK_GAME_SH="$tmp/game0.sh" SOAK_STOP_WAIT=0 PATH="$stub:$PATH" "$here/soak24.sh" stop i5 >/dev/null 2>&1 </dev/null
grep -q "pkill -TERM -f '^$HOME/d2lod-live/soak24-i5/d2bot '" "$tmp/ssh.log" || fail "stop i5 must pkill the per-instance binary: $(cat "$tmp/ssh.log")"
grep -q "soak24/d2bot .*--instance i5" "$tmp/ssh.log" || fail "stop i5 old-layout fallback must pin --instance i5"
grep -q "d2soak-i5.service" "$tmp/ssh.log" || fail "stop i5 must stop only d2soak-i5"
grep -q "d2soak-\*" "$tmp/ssh.log" && fail "stop i5 must not stop every d2soak unit"
[ -z "${DBG:-}" ] && rm -f "$tmp/ssh.log" || cat "$tmp/ssh.log"
[ "$bad" = 0 ] && echo "soak24_test: PASS" || { echo "soak24_test: $bad FAILED"; exit 1; }
