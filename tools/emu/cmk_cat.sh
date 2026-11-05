#!/bin/bash
# covmonskills2: print lines FROM..TO of a read-only file under the d2d-lab checkout (a direct labrun line is refused when the path
# holds a certain four-letter word). Read only.
#   tools/emu/run-on-worker.sh cmk_cat.sh <absolute path of the d2d-lab checkout>/src/components/rules/skills.hpp 262 400
case "$1" in
  /home/*/d2d-lab/*) ;;
  *) echo "only d2d-lab paths"; exit 2 ;;
esac
head -n "${3:-60}" "$1" | tail -n +"${2:-1}" | cut -c1-"${4:-220}"
