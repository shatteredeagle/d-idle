#!/bin/sh
# Environment for the skab_* oracles on the worker host (source it).
# D2 = an upstream d2decomped checkout used for emu.py / life_apply.py / disasm.py.
export D2=${D2:-$HOME/d2d-lab/pr/skresistab-d2d}
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
export D2_BUILD=${D2_BUILD:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream/build}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
export PYTHONPATH=$D2/tools/emu:${PYTHONPATH:-}
PY=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python
