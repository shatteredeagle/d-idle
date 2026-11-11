#!/bin/bash
# verifydmg4: read-only look at the r4 commit through a private clone in $D2_SCRATCH (the fork's own repo is never
# written), then export base 3e073888 and r4 9ea2afe9 (components/, tests/ and the build files) into $D2_SCRATCH/t_LABEL.
#   tools/emu/run-on-worker.sh v4_git.sh
set -u
S=${D2_SCRATCH:?}
C=$S/clone
[ -d "$C" ] || git clone --quiet --no-checkout "$HOME/d2d-lab/fork/src" "$C"
git -C "$C" fetch -q origin local/batch2-monsterdmg-r4 2>&1
H=9ea2afe9; B=3e073888
echo "== ids"; git -C "$C" rev-parse $H $B $H^ 2>&1
echo "== count"; git -C "$C" rev-list --count $B..$H
echo "== log"; git -C "$C" log --format='A=%an <%ae>%nC=%cn <%ce>%nP=%P%n---%n%B---' $B..$H
echo "== lengths"; git -C "$C" log -1 --format=%B $H | awk '{print length($0) ": " $0}'
echo "== stat"; git -C "$C" diff --stat $B $H
git -C "$C" diff $B $H > "$S/r4.diff"; echo "diff lines: $(wc -l < "$S/r4.diff")"
git -C "$C" diff 2df7d1eb $H > "$S/r3r4.diff"; echo "r3..r4 diff lines: $(wc -l < "$S/r3r4.diff")"
echo "== pr178"; git -C "$C" fetch -q origin pull/178/head 2>&1; git -C "$C" rev-parse 2597875 FETCH_HEAD 2>&1
echo "== merge-tree"; git -C "$C" merge-tree --write-tree 2597875 $H > "$S/mt.txt" 2>&1; echo "merge-tree rc=$?"; head -5 "$S/mt.txt"
echo "== remote has head"; git -C "$HOME/d2d-lab/fork/src" branch -r --contains $H 2>&1 | head
for p in base:3e073888 r4:9ea2afe9; do
	l=${p%%:*}; h=${p#*:}
	[ -d "$S/t_$l" ] || { mkdir -p "$S/t_$l" && git -C "$C" archive "$h" | tar -x -C "$S/t_$l"; }
	echo "exported t_$l"
done
exit 0
