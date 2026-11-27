#!/bin/bash
# The nightly Proving Ground job (card pgnightly, ADR 0027 item 4): runs the night's set from tools/provingground/nightly.json
# inside the standing window 23:00-07:00 EST, at most 3 jobs, niced, lab slots only (hidden), never the sandbox container.
#   nightly.sh            print the plan and the commands (dry run, the default)
#   nightly.sh --exec     run them; outside the window it refuses (exit 75) unless PG_NIGHTLY_FORCE=1 (coordinator only)
# Env: PG_NIGHTLY_DAY (YYYY-MM-DD, default today EST), PG_QUARANTINED (id,id), D2SCENARIO (a built d2scenario).
# The scheduler timer that calls this, and the after-push smoke stage, live in the private ops repo (docs/process/pg-ops-pgnightly.md).
set -u
here=$(cd "$(dirname "$0")" && pwd); repo=$(cd "$here/../.." && pwd); cd "$repo" || exit 2
if [ -n "${D2SCENARIO:-}" ]; then ds=("$D2SCENARIO"); else ds=(go run ./cmd/d2scenario); fi
day=${PG_NIGHTLY_DAY:-$(TZ=EST5 date +%Y-%m-%d)}
flags=(-day "$day" -quarantined "${PG_QUARANTINED:-}")
"${ds[@]}" nightly check "${flags[@]}" || exit 2
cmds=$("${ds[@]}" nightly plan "${flags[@]}" -format cmds) || exit 2
if [ "${1:-}" != --exec ]; then
  "${ds[@]}" nightly plan "${flags[@]}" -format text
  echo "--- commands (job, then run.sh arguments)"; echo "$cmds"; exit 0
fi
hm=$(TZ=EST5 date +%H%M)
if [ "${PG_NIGHTLY_FORCE:-}" != 1 ] && [ "$hm" -lt 2300 ] && [ "$hm" -ge 0700 ]; then
  echo "nightly: $hm EST is outside 23:00-07:00 EST; the second worker is not allowed now (ADR 0027 item 4)" >&2; exit 75
fi
export PG_EVENT=nightly
rc=0
while IFS= read -r line; do
  [ -n "$line" ] || continue
  job=${line%% *}; args=${line#* }
  hm=$(TZ=EST5 date +%H%M)
  if [ "${PG_NIGHTLY_FORCE:-}" != 1 ] && [ "$hm" -ge 0700 ] && [ "$hm" -lt 2300 ]; then echo "nightly: window over, stopping before: $line" >&2; rc=75; break; fi
  # the job's wall cap comes from the plan file (never repeated here), cut to the minutes left before 07:00 EST
  cap=$(sed -n "s/.*\"name\": \"$job\", \"max_wall_min\": \([0-9]*\).*/\1/p" tools/provingground/nightly.json)
  : "${cap:=120}"
  left=$(( (7*60 - (10#${hm:0:2}*60 + 10#${hm:2:2}) + 1440) % 1440 ))
  [ "${PG_NIGHTLY_FORCE:-}" = 1 ] || [ "$left" -ge "$cap" ] || cap=$left
  PG_JOB_NAME=$job nice -n 19 timeout "$((cap * 60))" "$here/run.sh" $args -parallel "${PG_NIGHTLY_PARALLEL:-2}" || { echo "nightly: job $job exit $?" >&2; rc=1; }
done <<<"$cmds"
exit $rc
