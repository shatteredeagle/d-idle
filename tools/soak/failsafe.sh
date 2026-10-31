# Failsafe configuration of a soak, sourced by soak24.sh and tools/soak/failsafe_test.sh. Owner table: docs/stallfatal.md s7.
# The soak runs ONE acting failsafe per fault: the bot's own ladder and Recycler, d2bot --supervise for process death, the
# TICKWATCH watchdog (default 10 min) for a stuck loop. The sentinel and d2soak only observe. The sourcing script defines
# ok / warn / fail (one line each, fail counts).

# What soak24 start passes to d2bot for the failsafe layer. Keep this the single source: preflight prints it.
SOAK_BOT_FAILSAFE_FLAGS="--supervise"

# sentinel_verdict MODE INST: judge a sentinel mode string (internal/sentinel Mode.String: "observe-only", "act:all",
# "act:a,b", each optionally "@i4,i5" = acting only on those instances). Prints one line; returns 1 when the sentinel
# may act on INST. "observe-only@i5" still observes everywhere.
sentinel_verdict() {
  local mode="$1" inst="$2" base list
  base="${mode%%@*}"; list=""; case "$mode" in *@*) list="${mode#*@}" ;; esac
  if [ "$base" = observe-only ]; then echo "sentinel mode $mode: observes only"; return 0; fi
  if [ -n "$list" ] && ! echo ",$list," | grep -q ",$inst,"; then
    echo "sentinel mode $mode: acts only on $list, observes $inst"; return 0
  fi
  echo "sentinel mode $mode may ACT on $inst (a second restarter next to the bot): tools/sentinel.sh start (observe-only is its default) or tools/sentinel.sh stop"
  return 1
}

# failsafe_report INST SENTINEL_MODE SENTINEL_UNIT: print the effective config; uses ok/warn/fail. SENTINEL_MODE is "" when
# the sentinel did not answer, SENTINEL_UNIT its systemd state ("active", "inactive", "unchecked" when preflight was skipped).
failsafe_report() {
  local inst="$1" mode="${2:-}" unit="${3:-}" line
  echo "  note  failsafe owners (docs/stallfatal.md s7): bot ladder + Recycler act; sentinel observes; d2soak observes"
  if [ -n "$mode" ]; then
    if line="$(sentinel_verdict "$mode" "$inst")"; then ok "$line"; else fail "$line"; fi
  elif [ "$unit" = unchecked ]; then
    warn "sentinel mode not checked (--skip-preflight): it must be observe-only or stopped"
  elif [ "$unit" = active ]; then
    warn "sentinel unit is active but its API did not answer on 127.0.0.1:8895: its mode is unknown, check tools/sentinel.sh status"
  else
    ok "sentinel not running (no acting second owner)"
  fi
  case " $SOAK_BOT_FAILSAFE_FLAGS " in
    *" --supervise "*) ok "supervise: on ($SOAK_BOT_FAILSAFE_FLAGS; restarts d2bot after exit 1/2/4, at most 10 per hour; exit 3 is final)" ;;
    *) fail "supervise is off: nothing restarts d2bot after a crash or a watchdog exit" ;;
  esac
  if [ -n "${D2BOT_WATCHDOG_MIN:-}" ]; then
    warn "D2BOT_WATCHDOG_MIN=$D2BOT_WATCHDOG_MIN is set here but tools/live.sh forwards no environment: the bot uses the default"
  fi
  ok "watchdog (TICKWATCH): default 10 min without a finished loop iteration, exit 4, supervise restarts (not tunable from soak24)"
  ok "bot ladder, recycler policy and restart budgets: defaults of the shipped build (config.json not touched)"
}
