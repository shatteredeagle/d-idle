#!/bin/bash
# Offline test of the failsafe config of soak24.sh (docs/stallfatal.md s7): bash tools/soak/failsafe_test.sh
set -u
here="$(cd "$(dirname "$0")" && pwd)"
bad=0
t_bad() { echo "FAIL: $*"; bad=$((bad + 1)); }
# the reporters failsafe.sh expects; fail lines are counted from the output
ok()   { echo "  ok    $*"; }
warn() { echo "  WARN  $*"; }
fail() { echo "  FAIL  $*"; }
. "$here/failsafe.sh"
report() { out="$(failsafe_report "$@")"; nfail=$(echo "$out" | grep -c '^  FAIL'); }
want_fail()  { report "$@"; [ "$nfail" -ge 1 ] || t_bad "expected a FAIL for: $*"; }
want_clean() { report "$@"; [ "$nfail" -eq 0 ] || t_bad "unexpected FAIL for: $* -> $out"; }

# sentinel modes (internal/sentinel Mode.String)
want_clean i4 observe-only active
want_clean i4 "observe-only@i5" active
want_clean i4 "act:all@i5" active          # acts on i5 only
want_clean i4 "" inactive                  # not running
want_fail  i4 act:all active
want_fail  i4 "act:death-overlay,esc-menu-stuck" active
want_fail  i4 "act:all@i4" active
want_fail  i4 "act:all@i4,i5" active
want_fail  i5 "act:no-progress@i4,i5" active
report i4 "" active;  echo "$out" | grep -q 'WARN.*API did not answer' || t_bad "active unit without an answer must warn"
report i4 "" unchecked; echo "$out" | grep -q 'WARN.*not checked' || t_bad "skip-preflight must warn about the sentinel"

# the printed effective config
report i4 observe-only active
for want in "failsafe owners" "sentinel mode observe-only: observes only" "supervise: on" "watchdog (TICKWATCH): default 10 min" "ladder, recycler policy"; do
  echo "$out" | grep -q -- "$want" || t_bad "report lacks '$want': $out"
done
D2BOT_WATCHDOG_MIN=1 report i4 observe-only active
echo "$out" | grep -q 'WARN.*D2BOT_WATCHDOG_MIN=1' || t_bad "a local D2BOT_WATCHDOG_MIN must warn that live.sh ignores it"
SOAK_BOT_FAILSAFE_FLAGS=""; want_fail i4 observe-only active; SOAK_BOT_FAILSAFE_FLAGS="--supervise"

# soak24.sh must pass exactly those flags to d2bot, never tune the watchdog or disable the supervisor
s24="$here/soak24.sh"
grep -q 'SOAK_BOT_FAILSAFE_FLAGS' "$s24" || t_bad "soak24.sh does not use SOAK_BOT_FAILSAFE_FLAGS"
grep -q -- '--stop-game \$SOAK_BOT_FAILSAFE_FLAGS' "$s24" || t_bad "d2bot launch line lacks the failsafe flags"
grep -q 'failsafe_report' "$s24" || t_bad "soak24.sh never prints the failsafe config"
if grep -n 'D2BOT_WATCHDOG_MIN\|--act\b\|--supervise-max 0\|D2BOT_SUPERVISE=0' "$s24" | grep -v '^[0-9]*:#'; then t_bad "soak24.sh changes the watchdog, the sentinel mode or the supervisor"; fi
[ "$SOAK_BOT_FAILSAFE_FLAGS" = "--supervise" ] || t_bad "default flags changed: $SOAK_BOT_FAILSAFE_FLAGS"
[ "$bad" = 0 ] && echo "failsafe_test: PASS" || { echo "failsafe_test: $bad FAILED"; exit 1; }
