#!/bin/bash
# Soak of the d2idle container (docs/nas-container.md "Running a soak on the NAS"). Needs ssh access to the docker host and
# D2FLEET_SOAK=1 in the container (template env): the d2soak sidecar inside the container does the sampling, this script only
# reads it (and, for `fault`, kills one game on purpose). Nothing here touches NAS settings.
#   tools/soak/contsoak.sh HOST [status]        one line per slot: run, verdict level, status line (exit 1 when a level is not ok)
#   tools/soak/contsoak.sh HOST report [DIR]    verdict text per slot + copy of every slot's latest run into DIR
#                                               (default ./soak-HOST-<time>/<run>/{samples.jsonl.gz,report.txt,...}); exit 1 on a FAIL
#   tools/soak/contsoak.sh HOST fault [SLOT]    fault injection: kill -9 the Game.exe of SLOT (default i1) by its WINEPREFIX
# Env: D2_CONTAINER (d2idle), D2_DOCKER_USER (the ssh user, default root), D2_SOAK_SSH (ssh command, default "ssh -o BatchMode=yes").
set -eu
host="${1:?usage: contsoak.sh HOST [status|report [DIR]|fault [SLOT]]}"
cmd="${2:-status}"
[ $# -ge 2 ] && shift 2 || shift
name="${D2_CONTAINER:-d2idle}"
SSH="${D2_SOAK_SSH:-ssh -o BatchMode=yes}"
dx() { $SSH "${D2_DOCKER_USER:-root}@$host" docker exec "$name" "$@"; }
# newest run directory per slot inside the container (names are <slot>-YYYYmmdd-HHMMSS)
RUNS='for s in 1 2 3 4 5 6 7 8 9; do d=$(ls -d /data/soak/i$s-*/ 2>/dev/null | sort | tail -1); [ -n "$d" ] && echo "${d%/}"; done; true'
case "$cmd" in
status)
  runs="$(dx sh -c "$RUNS")"
  [ -n "$runs" ] || { echo "no soak runs in $name:/data/soak (is D2FLEET_SOAK=1 set, and has a slot's bot started?)"; exit 1; }
  rc=0
  for r in $runs; do
    lvl="$(dx d2soak alerts --in "$r" 2>/dev/null | sed -n 's/.*"level": *"\([a-z]*\)".*/\1/p' | head -1)" || true
    echo "$(basename "$r") [${lvl:-?}] $(dx d2soak status --in "$r" 2>&1 | head -1)"
    [ "${lvl:-?}" = ok ] || rc=1
  done
  exit $rc ;;
report)
  out="${1:-./soak-$host-$(date +%Y%m%d-%H%M%S)}"
  runs="$(dx sh -c "$RUNS")"
  [ -n "$runs" ] || { echo "no soak runs in $name:/data/soak" >&2; exit 1; }
  rc=0
  mkdir -p "$out"
  for r in $runs; do
    b="$(basename "$r")"; mkdir -p "$out/$b"
    dx d2soak report --in "$r" >"$out/$b/report.txt" 2>&1 || rc=1
    dx d2soak report --in "$r" --json >"$out/$b/report.json" 2>/dev/null || true
    dx sh -c "cd '$r' && tar cf - ." | tar xf - -C "$out/$b" 2>/dev/null || echo "warn: could not copy $r" >&2
    [ -f "$out/$b/samples.jsonl" ] && gzip -f "$out/$b/samples.jsonl"
    echo "== $b ($([ "$rc" = 0 ] && echo PASS || echo FAIL or NO-DATA))"
    cat "$out/$b/report.txt"
  done
  echo "copied to $out"
  exit $rc ;;
fault)
  slot="${1:-i1}"
  case "$slot" in i[1-9]) ;; *) echo "slot must be i1..i9" >&2; exit 2 ;; esac
  # the slot's own prefix: its Game.exe carries WINEPREFIX=<prefix>/pfx in its environ (docs/nas-container.md)
  dx sh -c "n=0; for p in \$(pgrep -x Game.exe); do tr '\\0' '\\n' </proc/\$p/environ | grep -qx 'WINEPREFIX=/data/prefixes/$slot/pfx' && kill -9 \$p && n=\$((n+1)); done; echo killed \$n Game.exe of $slot at \$(date +%T)"
  echo "expected: bot notices the game is gone, restarts it (a game restart in the bot log minute lines), XP resumes; the run then shows one process restart" ;;
*) sed -n 2,10p "$0" >&2; exit 2 ;;
esac
