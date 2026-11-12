#!/bin/bash
# prprep (worker host): presubmit of the Cold Enchanted branch (the branch was pushed to the private remote by hand).
export PATH=$HOME/.local/bin:$PATH
cd $HOME/d2d-lab/wt-pp-cold || exit 1
git rev-parse HEAD
timeout 1500 python3 tools/presubmit.py --quick --base upstream/main > $HOME/d2d-lab/pp-cold/presubmit.txt 2>&1; echo "presubmit rc=$?"
grep -vE '^ok' $HOME/d2d-lab/pp-cold/presubmit.txt | tail -8 | cut -c1-260
