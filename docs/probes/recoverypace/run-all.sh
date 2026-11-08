#!/bin/bash
# TASK-183: Paladin and Sorceress, old arm then new arm, 5 wall minutes (about 90 virtual), one run at a time on the worker host.
cd "$(dirname "$0")/../../.." || exit 1
for spec in "old pal-o 2 paladin-normal30-v2" "new pal-n 2 paladin-normal30-v2" "old sor-o 4 sorceress_normal30" "new sor-n 4 sorceress_normal30"; do
  set -- $spec
  docs/probes/recoverypace/run-arm.sh $1 $2 $3 7 ${RUNFOR:-5m} $4
done
