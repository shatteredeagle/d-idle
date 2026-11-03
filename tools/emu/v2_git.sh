#!/bin/bash
# verifydmg2: read-only look at the r2 commit through a private clone in $D2_SCRATCH (the fork's own repo is never written).
#   tools/emu/run-on-worker.sh v2_git.sh
set -u
S=${D2_SCRATCH:?}
C=$S/clone
[ -d "$C" ] || git clone --quiet --no-checkout "$HOME/d2d-lab/fork/src" "$C"
H=b9ecb415; B=3e073888
echo "== ids"; git -C "$C" rev-parse $H $B 2>&1
echo "== log"; git -C "$C" log --format='A=%an <%ae>%nC=%cn <%ce>%nP=%P%n---%n%B---' $B..$H
echo "== lengths"; git -C "$C" log -1 --format=%B $H | awk '{print length($0) ": " $0}'
echo "== stat"; git -C "$C" diff --stat $B $H
git -C "$C" diff $B $H > "$S/r2.diff"; echo "diff lines: $(wc -l < "$S/r2.diff")"
echo "== pr178"; git -C "$C" cat-file -t 2597875 2>&1
git -C "$C" rev-parse 2597875 2>&1
echo "== merge-tree"; git -C "$C" merge-tree --write-tree 2597875 $H > "$S/mt.txt" 2>&1; echo "merge-tree rc=$?"; head -5 "$S/mt.txt"
echo "== remote has head"; git -C "$HOME/d2d-lab/fork/src" branch -r --contains $H 2>&1 | head
