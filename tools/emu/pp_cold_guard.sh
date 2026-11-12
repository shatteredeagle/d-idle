#!/bin/bash
export PATH=$HOME/.local/bin:$PATH
cd $HOME/d2d-lab/wt-pp-cold
python3 tools/pr_guard.py upstream/main HEAD 2>&1 | tail -6
timeout 1500 python3 tools/presubmit.py --quick --base upstream/main 2>&1 | grep -vE '^ok' | tail -10 | cut -c1-250
