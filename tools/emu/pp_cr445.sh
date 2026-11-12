#!/bin/bash
# prpost2 (worker host): CodeRabbit's four comments on the Cold Enchanted PR: fix, rebuild, re-run the differential (numbers must not change), presubmit, tests.
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
G="git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com"
HERE=$(cd "$(dirname "$0")" && pwd)
T=$HOME/d2d-lab/wt-pp-cold5; B=$HOME/d2d-lab/build-pp-cold5; O=$HOME/d2d-lab/pp-cold5
cd $T; git checkout -q fid/cold-enchanted-length-r5; git log --format=%h -1
python3 - <<'PY'
p="tests/monsters/test_umod_stats.cpp"; s=open(p).read()
a="int main(int argc, char** argv) {\n"; assert a in s
s=s.replace(a,"int main(int argc, char** argv) try {\n")
a="    std::fclose(cases);\n    return 0;\n}\n"; assert a in s
s=s.replace(a,"    std::fclose(cases);\n    return 0;\n} catch (const std::exception& error) {\n    std::fprintf(stderr, \"%s\\n\", error.what());\n    return 1;\n}\n")
s=s.replace('#include "montypes.hpp"\n','#include "combat.hpp"\n#include "montypes.hpp"\n')
s=s.replace("#include <cstdio>\n","#include <cstdint>\n#include <cstdio>\n#include <exception>\n",1)
a="""        int hit_points = 1000, armor_class = 100, exp = 1000;
        hit_points += hit_points * boss.hp_pct / 100;
        hit_points = std::max(hit_points + hit_points * boss.life_after_pct / 100, 1);
"""
assert a in s
s=s.replace(a,"""        int armor_class = 100, exp = 1000;
        int life = 1000 * 256;                                // 256ths through both steps, as make_boss (ai.cpp)
        life = int(std::uint32_t(life) + std::uint32_t(d2d::rules::mul_div(life, boss.hp_pct, 100)));
        life = int(std::uint32_t(life) + std::uint32_t(d2d::rules::mul_div(life, boss.life_after_pct, 100)));
        life = std::max(life, 256);
""")
a="exp, hit_points << 8, boss.dmg_pct"; assert a in s
s=s.replace(a,"exp, life, boss.dmg_pct")
open(p,"w").write(s)
PY
grep -n "hit_points" tests/monsters/test_umod_stats.cpp
nice -n 19 cmake --build $B -j8 --target test_umod_stats 2>&1 | grep -E "error|warning: unused" | head -5; echo "build rc=${PIPESTATUS[0]}"
cd tools/emu
for seed in 1 9; do
  $B/tests/test_umod_stats --umod-stats $O/game-$seed.tsv > $O/cr-$seed.d2d && uv run python umod_stats.py --compare $O/game-$seed.tsv $O/cr-$seed.d2d 2>&1 | grep -v WARNING > $O/cr-$seed.full
  echo "seed $seed: $(head -1 $O/cr-$seed.full); life256 groups: $(grep -c ' life256 ' $O/cr-$seed.full)"; python3 $HERE/pp_cold_count.py $O/cr-$seed.full
  diff <(grep -v '^cases' $O/claim-$seed.full) <(grep -v '^cases' $O/cr-$seed.full) > $O/cr-$seed.diff; echo "full compare listing differs from the verified claim run: $(wc -l < $O/cr-$seed.diff) lines"
done
