#!/bin/bash
# verifyb2 (the worker host): a throwaway merge commit of pull/178 and one branch head in the
# verifier's own clone (never pushed), printed for vb2_build.sh.
#   vb2_merge.sh HEAD_COMMIT
set -e
V=$HOME/d2d-lab/vb2
tree=$(git -C "$V/src" merge-tree --write-tree pr178 "$1")
c=$(git -C "$V/src" -c user.name=verifier -c user.email=verifier@invalid commit-tree "$tree" -p pr178 -p "$1" -m "throwaway merge")
echo "merge $c"
