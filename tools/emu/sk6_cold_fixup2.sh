#!/bin/bash
# decomp6 (worker host): second presubmit fix for fid/cold-enchanted-length (short names), squashed into commit 1; push
HERE=$(cd "$(dirname "$0")" && pwd)
. $HERE/d6_env.sh
cd $T
git switch -q fid/cold-enchanted-length
f=tests/monsters/test_umod_stats.cpp
sed -i -E 's/\bin\b( >>|\(line\)|;)/fields\1/g; s/std::istringstream in\(line\)/std::istringstream fields(line)/; s/\bin >>/fields >>/g' $f
sed -i -E 's/std::istringstream m\(mods_s\); std::string id; while \(std::getline\(m, id, /std::istringstream parts(mods_s); std::string mod; while (std::getline(parts, mod, /; s/mods.push_back\(std::stoi\(id\)\)/mods.push_back(std::stoi(mod))/' $f
sed -i -E 's/\bel\b/element_rows/g' $f
grep -n -E "\bin\b|\bel\b|\bm\b|\bid\b" $f | head
ID=(-c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com)
git "${ID[@]}" commit -q -a --fixup=HEAD~1
GIT_SEQUENCE_EDITOR=true git "${ID[@]}" rebase -q -i --autosquash upstream/main
git log --format='%h %an %s' -3
cd $B && cmake . >/dev/null 2>&1; nice -n 19 ninja -j2 test_umod_stats 2>&1 | tail -1
cd $T/tools/emu
$B/tests/test_umod_stats --umod-stats $O/umod/game.tsv > $O/umod/after2.tsv; cmp $O/umod/after.tsv $O/umod/after2.tsv && echo "same output as before the renames"
python3 $HERE/sk6_level_edge.py $O/umod/game.tsv $O/umod/after2.tsv > $O/umod/edge.txt 2>&1; cat $O/umod/edge.txt
cd $T
python3 tools/pr_guard.py upstream/main 2>&1 | tail -2
python3 tools/presubmit.py --quick --base upstream/main 2>&1 | grep -E "FAIL|WARN|presubmit:"
git push -q -f origin fid/cold-enchanted-length 2>&1 | tail -1; git ls-remote origin fid/cold-enchanted-length | cut -c1-12
