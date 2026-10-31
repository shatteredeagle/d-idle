#!/bin/bash
# usage (ON THE WORKER): run.sh "ARM ..." SEED...   e.g. run.sh "off on" 1 2 3 4 5 6 7 8
# Interleaves the arms per seed (same seed, one binary, knob toggled per arm) through tools/xpgap/ab.sh on the slot pool;
# refuses fewer than 8 seeds (spec fidelity-gate "A/B protocol for bot changes"). XG_MAX caps my concurrent runs (default 3).
# Analyse the an.py output with: tools/ab/paired.py --base off --arm on FILE
set -e
[ $# -ge 2 ] || { echo "usage: run.sh \"ARM ...\" SEED..." >&2; exit 64; }
arms=$1; shift
[ $# -ge 8 ] || { echo "run.sh: $# seeds, the protocol needs at least 8 (docs/ab-design.md)" >&2; exit 64; }
[ "$(printf '%s\n' "$@" | sort | uniq -d | wc -l)" = 0 ] || { echo "run.sh: duplicate seeds" >&2; exit 64; }
here=$(cd "$(dirname "$0")" && pwd)
XG_ARMS=$arms exec "$here/../xpgap/ab.sh" "$@"
