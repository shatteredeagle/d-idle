"""decomp6: write the tooling files of the fork branch fid/cold-enchanted-length from our umod_stats files.
python3 sk6_cold_tool.py SRC_DIR TREE   (SRC_DIR holds umod_stats.py and umod_stats_harness.cpp)"""
import sys
src, tree = sys.argv[1], sys.argv[2]
py = open(src + "/umod_stats.py").read()
py = py.replace("(agent DECOMPMODS)", "")
py = py.replace("    ./umod_stats_harness < game.tsv > d2d.tsv               (d2d's rules::boss_stats on the same cases)\n",
                "    build/tests/test_umod_stats --umod-stats game.tsv > d2d.tsv     (d2d's rules::boss_stats on the same cases)\n")
open(tree + "/tools/emu/umod_stats.py", "w").write(py)
cpp = open(src + "/umod_stats_harness.cpp").read()
body = cpp[cpp.index("int main() {"):]
head = """// SPDX-License-Identifier: GPL-3.0-or-later
// test-links: rules
// `--umod-stats CASES`: d2d's champion / unique / minion stat mods on the cases
// tools/emu/umod_stats.py writes (game.exe's FUN_005a2120 and the mods' inits).
// Calls rules::boss_stats (components/rules/uniques.hpp) and applies it the way
// game::make_boss does (components/game/ai.cpp), on the case's fixed base (level,
// exp 1000, life 1000, defence 100, the row's resistances), so each field compares
// with game.exe's. A minion gets no mods, as d2d tags it (objgroups.cpp).
//
//   uv run python umod_stats.py 100000 1 > game.tsv
//   build/tests/test_umod_stats --umod-stats game.tsv > d2d.tsv
//   uv run python umod_stats.py --compare game.tsv d2d.tsv
//
// Fields d2d does not model print `*`: armor% (d2d folds it into def, the effective
// defence; --compare folds game.exe's the same way) and regen (no monster regeneration).
#include "uniques.hpp"

#include <algorithm>
#include <array>
#include <cstdio>
#include <cstring>
#include <sstream>
#include <string>
#include <vector>

"""
body = body.replace("int main() {", "int main(int argc, char** argv) {\n    if (argc < 3 || std::strcmp(argv[1], \"--umod-stats\") != 0) { std::printf(\"SKIP: needs --umod-stats CASES\\n\"); return D2D_TEST_SKIPPED; }\n    std::FILE* cases = std::fopen(argv[2], \"r\");\n    if (cases == nullptr) { std::fprintf(stderr, \"cannot open %s\\n\", argv[2]); return 1; }", 1)
body = body.replace("std::fgets(buf, sizeof buf, stdin)", "std::fgets(buf, sizeof buf, cases)")
body = body.rstrip()
assert body.endswith("}")
body = body[:-1] + "    std::fclose(cases);\n    return 0;\n}\n"
open(tree + "/tests/monsters/test_umod_stats.cpp", "w").write(head + body)
