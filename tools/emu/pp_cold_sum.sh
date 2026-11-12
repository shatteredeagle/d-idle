#!/bin/bash
export PATH=$HOME/.local/bin:$PATH
O=$HOME/d2d-lab/pp-cold; cd $HOME/d2d-lab/wt-pp-cold/tools/emu
mode=$1
uv run python umod_stats.py --compare $O/game-1-100000.tsv $O/$mode.d2d 2>&1 | grep -v WARNING > $O/$mode.full
head -30 $O/$mode.full
echo ---; grep -c cold_len $O/$mode.full
