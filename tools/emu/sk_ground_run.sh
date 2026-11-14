#!/bin/bash
# SKGROUND helper (the worker host): run an sk_ground script or disasm.py from
# upstream's tools/emu in the scratch checkout, with the emulator's data dir.
#   tools/emu/sk_ground_run.sh UPSTREAM PY_SCRIPT args...
# UPSTREAM = the scratch upstream checkout; the script is copied there first.
set -e
up=$1; shift; s=$1; shift
here=$(cd "$(dirname "$0")" && pwd)
[ -f "$here/$s" ] && cp "$here/$s" "$up/tools/emu/$s"
cd "$up/tools/emu"
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
exec nice "${PY:-python3}" "$s" "$@"
