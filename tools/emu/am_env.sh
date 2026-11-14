#!/bin/bash
# areamon (decomp18): worker-host wrapper. Sources the emulator image env ($D2_EMU_DIR, default ~/d2d-lab/emu)
# and runs a tools/emu script with the image's venv and upstream emulator modules.
#   bash tools/emu/am_env.sh table NAME        # print one patched 1.14d excel table (mpq-cat)
#   bash tools/emu/am_env.sh py SCRIPT args... # run tools/emu/SCRIPT (ms_spawn.py, am_*.py) under nice
set -e
here=$(cd "$(dirname "$0")" && pwd)
. "${D2_EMU_DIR:-$HOME/d2d-lab/emu}/env.sh"
cmd=$1; shift
case $cmd in
table)
  t=${1:?table}; case $t in *[!A-Za-z0-9_]*) echo bad name >&2; exit 2;; esac
  S=${D2_SCRATCH:-/tmp/d2agent-$USER}/amlink; mkdir -p "$S"
  for f in "$D2_EMU_DIR"/mpq/*.mpq; do ln -sf "$f" "$S/$(basename "$f")"; done
  ln -sf "$D2_EMU_DIR/mpq/Patch_D2.mpq" "$S/patch_d2.mpq"
  exec "$MPQCAT" "$S" "data\\global\\excel\\$t.txt" ;;
py)
  s=$1; shift
  export PYTHONPATH=$here:$EMU:${PYTHONPATH:-}
  [ -f "$here/$s" ] && s=$here/$s
  cd "$EMU"; exec nice "$PY" "$s" "$@" ;;
*) echo "usage: table|py" >&2; exit 2;;
esac
