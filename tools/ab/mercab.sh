#!/bin/bash
# usage (ON THE WORKER): mercab.sh HOURS SEED...   the merc reserve A/B, late-start variant (docs/ab-merc.md).
# Runs arm new (A, SME rule) and arm old (B, old reserve) per seed through tools/ab/mercrun.sh. The runs are dealt round-robin
# onto AB_SLOTS lanes (default "3 4"); each lane runs its queue in order, so with two lanes the two arms of a seed run side by
# side (interleaved, same load). Refuses fewer than 8 seeds unless AB_DRY=1 (harness dry run, no claim possible).
# Afterwards: mercan.py on the logs, then paired.py --base old --arm new for XP/h and kills/h.
# Wrap in agentrun.sh abmerc. AB_TAGPFX (default none) prefixes the run tags so a dry run does not mix with the real one.
set -e
[ $# -ge 2 ] || { echo "usage: mercab.sh HOURS SEED..." >&2; exit 64; }
H=$1; shift
[ "${AB_DRY:-0}" = 1 ] || [ $# -ge 8 ] || { echo "mercab.sh: $# seeds, the protocol needs at least 8 (docs/ab-design.md)" >&2; exit 64; }
[ "$(printf '%s\n' "$@" | sort | uniq -d | wc -l)" = 0 ] || { echo "mercab.sh: duplicate seeds" >&2; exit 64; }
here=$(cd "$(dirname "$0")" && pwd)
W=$HOME/d2d-lab/abmerc
slots=(${AB_SLOTS:-3 4}); n=${#slots[@]}
i=0
for s in "$@"; do for arm in new old; do
  lane=$((i % n)); i=$((i + 1))
  echo "${AB_TAGPFX}${arm}${s} $arm $s" >> "$W/lane-$$-$lane.q"
done; done
for lane in $(seq 0 $((n - 1))); do
  [ -f "$W/lane-$$-$lane.q" ] || continue
  ( while read -r tag arm seed; do "$here/mercrun.sh" "$tag" "$arm" "$seed" "$H" "${slots[$lane]}" || echo "run $tag exit $?" >&2; done < "$W/lane-$$-$lane.q" ) &
done
wait
rm -f "$W"/lane-$$-*.q
cd "$W"
logs=(); for s in "$@"; do for arm in new old; do logs+=("logs/bot-${AB_TAGPFX}${arm}${s}.log"); done; done
python3 -I "$here/mercan.py" "${logs[@]}" | tee "result-${AB_TAGPFX}$$.txt"
for m in xp_h kills_h; do python3 -I "$here/paired.py" --metric $m --base old --arm new "result-${AB_TAGPFX}$$.txt" || true; done
