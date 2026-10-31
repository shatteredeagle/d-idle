# Sourced by soak24.sh (tests: tools/soak/soak24_test.sh). soak_secs STR prints the seconds of a duration like
# 24h, 90m, 1h30m, 45s, 1h2m3s or plain digits (seconds); returns 1 (prints nothing) for anything else or zero.
soak_secs() {
  local s="${1:-}" h=0 m=0 sec=0
  if [[ "$s" =~ ^[0-9]+$ ]]; then
    sec="$s"
  elif [[ "$s" =~ ^([0-9]+h)?([0-9]+m)?([0-9]+s)?$ && -n "$s" ]]; then
    h="${BASH_REMATCH[1]%h}"; m="${BASH_REMATCH[2]%m}"; sec="${BASH_REMATCH[3]%s}"
  else
    return 1
  fi
  local total=$(( 10#${h:-0} * 3600 + 10#${m:-0} * 60 + 10#${sec:-0} ))
  [ "$total" -gt 0 ] || return 1
  echo "$total"
}
