#!/bin/bash
# prprep (worker host): rebase local/fid-load-gold-clamp onto upstream main and re-run its differential, with the claim and with the claim's components reverted.
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
HERE=$(cd "$(dirname "$0")" && pwd)
F=$HOME/d2d-lab/fork/src; T=$HOME/d2d-lab/wt-pp-gold; B=$HOME/d2d-lab/build-pp-gold; O=$HOME/d2d-lab/pp-gold; mkdir -p $O
cd $F
[ -d $T ] || { git worktree add -b fid/load-gold-clamp-r1 $T upstream/main 2>&1 | tail -1; cd $T; git -c user.name=Claude -c user.email=noreply@anthropic.com cherry-pick 42d40b09 >/dev/null 2>&1 || { echo "CONFLICT"; git status --short | head; exit 1; }; }
cd $T; git log --format='%h %an | %s' -2
bash $HERE/pp_build.sh gold test_game mpq-cat | tail -2
run() { # label
  cd $T/tools/emu
  uv run python fl_gold.py gen > $O/cases.tsv; uv run python fl_gold.py game $O/cases.tsv > $O/game.tsv
  (cd $B && tests/test_game --load-gold $O/cases.tsv 2>/dev/null | grep -E '^[A-Za-z_][A-Za-z_0-9]* -?[0-9]+( -?[0-9]+){1}$' > $O/d2d-$1.tsv)
  echo "$1: cases $(wc -l < $O/cases.tsv) game $(wc -l < $O/game.tsv) d2d $(wc -l < $O/d2d-$1.tsv) mismatches $(diff $O/game.tsv $O/d2d-$1.tsv | grep -c '^<')"
}
run after
cd $T; git diff HEAD~1 HEAD -- components > $O/claim.patch; git apply -R $O/claim.patch && bash $HERE/pp_build.sh gold test_game | tail -1
run before
git apply $O/claim.patch; bash $HERE/pp_build.sh gold test_game | tail -1
cd $B && timeout 900 tests/test_game > $O/tg.out 2>&1; echo "test_game rc=$?"
cd $T; python3 tools/pr_guard.py upstream/main HEAD 2>&1 | tail -2
git rev-parse HEAD
