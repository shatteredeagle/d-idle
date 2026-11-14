#!/bin/sh
# necskill oracle env on the worker host (source it). The upstream emu modules
# (emu.py, skill_tips.py, skill_damage.py) come from the palskill read checkout.
export D2=${D2:-$HOME/d2d-lab/pr/palskill}
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
export D2_BUILD=${D2_BUILD:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream/build}
HERE=${HERE:-$PWD}
export PYTHONPATH=$HERE:$D2/tools/emu:${PYTHONPATH:-}
PY=${PY:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python}
