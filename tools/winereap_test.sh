#!/bin/bash
# Test of tools/winereap.sh with a fake /proc and fake homes: bash tools/winereap_test.sh
set -eu
# The real /tmp/d2lod-game-<id>.lock (legacy lock) may be held by a soak on the dev box: only the lock test below takes locks.
NL=1
here="$(cd "$(dirname "$0")" && pwd)"
t="$(mktemp -d)"; trap 'rm -rf "$t"' EXIT
fail() { echo "FAIL: $*"; exit 1; }
mk() { # pid comm uid prefix
  mkdir -p "$proc/$1"; echo "$2" >"$proc/$1/comm"; printf 'Name:\t%s\nUid:\t%s\t%s\n' "$2" "$3" "$3" >"$proc/$1/status"
  [ -z "$4" ] || printf 'A=b\0WINEPREFIX=%s\0' "$4" >"$proc/$1/environ"
}
# run ID [args]: fresh env, no real HOME/XDG leakage
run() { env -i PATH="$PATH" HOME="$home" D2BOT_DATA="$data" PROC_ROOT="$proc" REAP_UID=1000 REAP_NOLOCK="${NL-1}" \
  XDG_RUNTIME_DIR="$t/run" "$@"; }
setup() { rm -rf "$t/p" "$t/h" "$t/d" "$t/run"; proc="$t/p"; home="$t/h"; data="$t/d"; mkdir -p "$proc" "$home" "$data" "$t/run"; }
basic() { # layout prefix root of i4 / i1; checks the common guards
  p4="$1"; p1="$2"
  mk 11 winedevice.exe 1000 "$p4/proton/pfx/"   # orphan of i4: reaped
  mk 12 winedevice.exe 1000 "$p1/proton/pfx/"   # other instance
  mk 13 winedevice.exe 999 "$p4/proton/pfx/"    # other user
  mk 14 services.exe 1000 "$p4/proton/pfx/"     # other program
  mk 15 winedevice.exe 1000 ""                  # no environ
  mk 16 winedevice.exe 1000 "$home/.wine"       # the user's own Wine prefix
  out="$(run bash "$here/winereap.sh" i4 --dry-run)"
  echo "$out" | grep -q "pid 11 " || fail "$layout: pid 11 not listed: $out"
  [ "$(echo "$out" | grep -c 'would kill')" = 1 ] || fail "$layout: wrong count: $out"
  mk 20 wineserver 1000 "$p1/proton/pfx/"       # other prefix's server must not block i4
  run bash "$here/winereap.sh" i4 --dry-run | grep -q "pid 11 " || fail "$layout: other server blocked"
  mk 21 wineserver 1000 "$p4/proton/pfx/"       # live server of i4: reap nothing
  run bash "$here/winereap.sh" i4 --dry-run | grep -q "is running" || fail "$layout: live server ignored"
  rm -rf "$proc/21"; mk 22 wineserver64 1000 "$p4/proton/pfx" # other server name: still blocks
  run bash "$here/winereap.sh" i4 --dry-run | grep -q "is running" || fail "$layout: wineserver64 ignored"
  rm -rf "$proc/22"
}

layout=new; setup                                 # <data>/prefixes/<id>, also the managed/fleet/container layout
mkdir -p "$data/prefixes/i0"
basic "$data/prefixes/i4" "$data/prefixes/i1"

layout=legacy; setup                              # dev era: ~/Games/d2lod exists, no prefixes/i0
mkdir -p "$home/Games/d2lod"
basic "$home/Games/d2lod-inst/i4" "$home/Games/d2lod-inst/i1"

layout=fresh-no-legacy; setup                     # neither dir: new layout by default
basic "$data/prefixes/i4" "$data/prefixes/i1"

layout=container; setup                           # D2BOT_DATA=/data style: data dir elsewhere
data="$t/container-data"; mkdir -p "$data/prefixes/i0"
basic "$data/prefixes/i4" "$data/prefixes/i1"

layout=xdg; setup                                 # no D2BOT_DATA: XDG_DATA_HOME/d2bot
run() { env -i PATH="$PATH" HOME="$home" XDG_DATA_HOME="$t/xdg" PROC_ROOT="$proc" REAP_UID=1000 REAP_NOLOCK="${NL-1}" XDG_RUNTIME_DIR="$t/run" "$@"; }
mkdir -p "$t/xdg/d2bot/prefixes/i0"
basic "$t/xdg/d2bot/prefixes/i4" "$t/xdg/d2bot/prefixes/i1"

layout=extra; setup                               # custom PrefixDir via D2_REAP_PREFIX
run() { env -i PATH="$PATH" HOME="$home" D2BOT_DATA="$data" PROC_ROOT="$proc" REAP_UID=1000 REAP_NOLOCK="${NL-1}" XDG_RUNTIME_DIR="$t/run" \
  D2_REAP_PREFIX="$t/custom/pfx:$t/other/pfx" "$@"; }
mk 31 winedevice.exe 1000 "$t/custom/pfx"
run bash "$here/winereap.sh" i4 --dry-run | grep -q "pid 31 " || fail "extra prefix not covered"

# kill path: a stub `kill` (exported function beats the builtin) logs the pid; the first kill starts a wineserver of the
# prefix, so the per-kill recheck must stop the second one
layout=recheck; setup; mkdir -p "$data/prefixes/i0"
mk 41 winedevice.exe 1000 "$data/prefixes/i4/proton/pfx"
mk 42 winedevice.exe 1000 "$data/prefixes/i4/proton/pfx"
cat >"$t/kill.sh" <<K
#!/bin/bash
echo "\$1" >>"$t/kills"
if [ ! -d "$proc/50" ]; then mkdir -p "$proc/50"; echo wineserver >"$proc/50/comm"
  printf 'WINEPREFIX=%s\\0' "$data/prefixes/i4/proton/pfx" >"$proc/50/environ"; fi
K
chmod +x "$t/kill.sh"
run env REAP_KILL="$t/kill.sh" bash "$here/winereap.sh" i4 | grep -q "appeared: stopping" || fail "recheck did not stop the kills"
[ "$(wc -l <"$t/kills")" = 1 ] || fail "expected exactly 1 kill: $(cat "$t/kills")"
rm -rf "$proc/50" "$t/kills"

# deadlock fix: a lingering wineserver kept alive only by orphans -> they are killed (TERM then KILL); a live client keeps it
layout=idle; setup; mkdir -p "$data/prefixes/i0"
px="$data/prefixes/i4/proton/pfx"
mk 61 winedevice.exe 1000 "$px"; mk 62 explorer.exe 1000 "$px"; mk 63 wineserver 1000 "$px"
mk 64 winedevice.exe 1000 "$data/prefixes/i1/proton/pfx"; mk 65 winedevice.exe 999 "$px"
run bash "$here/winereap.sh" i4 --dry-run | grep -q "kept alive only by orphans" || fail "idle dry run"
printf '#!/bin/bash\necho "$1" >>"%s/terms"\n' "$t" >"$t/term.sh"; printf '#!/bin/bash\necho "$1" >>"%s/kills9"\n' "$t" >"$t/k9.sh"; chmod +x "$t/term.sh" "$t/k9.sh"
out="$(run env REAP_KILL="$t/term.sh" REAP_KILL9="$t/k9.sh" REAP_TERMWAIT=0 bash "$here/winereap.sh" i4)"
echo "$out" | grep -q "only kept alive by orphans" || fail "idle real run: $out"
[ "$(sort "$t/terms" | tr '\n' ' ')" = "61 62 " ] || fail "term pids: $(cat "$t/terms")"
[ "$(sort "$t/kills9" | tr '\n' ' ')" = "61 62 " ] || fail "kill9 pids: $(cat "$t/kills9")"
rm -f "$t/terms" "$t/kills9"; mk 66 d2input.exe 1000 "$px"   # a live client: the server is in use
out="$(run env REAP_KILL="$t/term.sh" REAP_KILL9="$t/k9.sh" REAP_TERMWAIT=0 bash "$here/winereap.sh" i4)"
echo "$out" | grep -q "is running: nothing reaped" || fail "live client ignored: $out"
[ ! -e "$t/terms" ] || fail "killed with a live client"
rm -rf "$proc"/6?; mk 41 winedevice.exe 1000 "$px"   # back to a plain orphan for the lock test

# lock: a held instance lock makes a real reap bail out and a dry run continue
mkdir -p "$t/run/d2bot"; touch "$t/run/d2bot/game-i4.lock"
flock "$t/run/d2bot/game-i4.lock" sleep 30 & lp=$!
sleep 0.5
NL=
sed 's/flock -w 10/flock -w 1/g' "$here/winereap.sh" >"$t/wr.sh"; cp "$here/lib-paths.sh" "$t/"
run bash "$t/wr.sh" i4 2>&1 | grep -q "lock busy, nothing reaped" || fail "held lock ignored"
run bash "$t/wr.sh" i4 --dry-run 2>&1 | grep -q "would kill" || fail "dry run blocked by lock"
kill "$lp" 2>/dev/null || true
NL=1

# all: dry run only, covers several instances
run bash "$here/winereap.sh" all 2>/dev/null && fail "all without --dry-run accepted"
run bash "$here/winereap.sh" all --dry-run | grep -q "^i9: 0 orphaned" || fail "all did not reach i9"
run bash "$here/winereap.sh" i0 2>/dev/null && fail "i0 accepted"
echo OK
