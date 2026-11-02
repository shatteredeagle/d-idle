"""monsterpace: build a counting replay program from the replay block of d2d's tests/test_monsters.cpp (rules::mon_think on game.exe's
recorded runs), over tests/mp_cases.inc / mp_names.inc written by mp_ais.py. Nothing of upstream is copied into the repo: the
block is cut out of the scratch tree's file at build time and patched.
usage: python3 mp_patch.py <d2d source tree> <scratch dir>   writes <scratch dir>/mp_replay.cpp
Replay additions over the first pass (mai_patch.py): the case line's unused stat words carry `dying` (Fallen), the target offset
(off_x / off_y, which ai.cpp feeds and the case line does not carry) is rebuilt from the spots, the Blood Raven's home point rides in
free spot 0 (home_dist), her flag lives in AI control word 3 in game.exe and in the command word in d2d (mapped), and the tally is per
AI plus the Fallen's dying / command regime."""
import sys

root, out_dir = sys.argv[1], sys.argv[2]
t = open(root + "/tests/monsters/test_monster_ai.cpp").read()
import os
HAS_TARGET_HOME = "target_home_dist" in open(os.environ.get("D2D_TREE", root) + "/components/rules/monsters.hpp").read()   # the headers the replay builds against (D2D_TREE), the test text from `root`
head = t[:t.index("using namespace d2d::rules;") + len("using namespace d2d::rules;")]
head = head.replace("#include <array>", "#include <array>\n#include <fstream>\n#include <map>", 1)
a = t.index("            // Act 2's monster thinks (rules::mon_think")
b = t.index("            // The AI toggle (FUN_005b0e00)")
block = t[a:b]
a2 = block.index('#include "mon_ais_cases.inc"')
b2 = block.index('#include "mon_ais_act1_cases.inc"') + len('#include "mon_ais_act1_cases.inc"')
block = block[:a2] + '#include "mp_cases_stub.inc"' + block[b2:]
block = block.replace('#include "mon_ais_names.inc"', '#include "mp_names.inc"')
# the cases are read at run time from the file named by argv[1] (a 100,000-class corpus does not compile as a literal array)
ka=block.index("static const std::string_view kCases[] = {")
kb=block.index("};", ka)+2
block=block[:ka]+'''std::vector<std::string> kCaseText;
                {
                    std::ifstream in(argv[1]);
                    for (std::string raw; std::getline(in, raw);)
                        if (raw.size() > 3 && raw[0] == '"') kCaseText.push_back(raw.substr(1, raw.rfind('"') - 1));
                }
                const std::vector<std::string_view> kCases(kCaseText.begin(), kCaseText.end());'''+block[kb:]
block = block.replace("                int checked = 0;\n                for (const std::string_view case_line : kCases) {",
                      "                int checked = 0;\n                std::map<std::string, std::array<int, 2>> tally;\n                std::map<std::string, int> pace_bad;\n                for (const std::string_view case_line : kCases) {", 1)
for old, new in (
    ("auto away = [&](int reach, bool) {\n                        assert(asked < asks.size() && asks[asked].first == reach);",
     "bool away_bad = false;\n                    auto away = [&](int reach, bool) {\n                        if (!(asked < asks.size() && asks[asked].first == reach)) { away_bad = true; return false; }"),
    ("for (const auto& effect : effects)                     // the toggle back",
     "if (rally) effects.push_back(d2d::rules::ThinkEffect{ d2d::rules::ThinkEffect::rally, 0, 0 });   // the leader's rally, as game.exe's effect\n                    for (const auto& effect : effects)                     // the toggle back"),
    ("assert(asked == asks.size());", "bool bad = asked != asks.size() || away_bad;"),
    ("assert(effects.size() == want_effects.size());", "if (effects.size() != want_effects.size()) bad = true;"),
    ("for (std::size_t k = 0; k < effects.size(); ++k)\n                        if (int(effects[k].kind) != want_effects[k][0]", "for (std::size_t k = 0; k < std::min(effects.size(), want_effects.size()); ++k)\n                        if (int(effects[k].kind) != want_effects[k][0]"),
    ("for (std::size_t k = 0; k < effects.size(); ++k)\n                        assert(int(effects[k].kind) == want_effects[k][0] && effects[k].id == want_effects[k][1] && effects[k].value == want_effects[k][2]);",
     "for (std::size_t k = 0; k < std::min(effects.size(), want_effects.size()); ++k)\n                        if (!(int(effects[k].kind) == want_effects[k][0] && effects[k].id == want_effects[k][1] && effects[k].value == want_effects[k][2])) bad = true;"),
    ("assert(same);\n                    ++checked;",
     "if (!same) bad = true;\n                    std::string key(ai_name);\n                    if (ai_name == \"Fallen\") key += std::string(input.dying ? \"+dying\" : \"\") + (cmd0 ? \"+cmd\" + std::to_string(cmd0) : \"\");\n"
     "                    ++tally[key][bad ? 1 : 0]; if (pace != want[7]) ++pace_bad[key]; ++checked;"),
    ("assert(checked > 300);\n            }\n",
     "(void)checked; int total = 0, differ = 0;\n                for (const auto& [name, count] : tally) { std::printf(\"mp %s %d %d pace %d\\n\", name.c_str(), count[0] + count[1], count[1], pace_bad[name]); total += count[0] + count[1]; differ += count[1]; }\n                std::printf(\"mp TOTAL %d %d\\n\", total, differ);\n                return differ != 0;\n            }\n"),
    # mai2: dying rides in the unused stat word, the offsets are rebuilt, Raven's home is free spot 0, her flag is control word 3
    ("input.regen = read(); (void)read(); (void)read(); (void)read();", "input.regen = read(); (void)read(); const int dying_flag = read(); (void)read();"),
    ("input.same_room = read() != 0; input.can_land = read() != 0; input.victim = read() != 0; input.grouped = read() != 0;",
     "input.same_room = read() != 0; input.can_land = read() != 0; input.victim = read() != 0; input.grouped = read() != 0;\n"
     "                    input.off_x = input.target_x - input.self_x; input.off_y = input.target_y - input.self_y;"),
    ("int cast_word = 0;",
     "input.dying = (ai_of(which) == \"Fallen\") && dying_flag != 0;\n"
     "                    if (ai_of(which) == \"BloodRaven\") { const int home_x = input.free_spot[0] ? input.free_spot[0]->first : input.self_x, home_y = input.free_spot[0] ? input.free_spot[0]->second : input.self_y;\n"
     "                        const int hx = home_x - input.self_x, hy = home_y - input.self_y; input.home_dist = less_two ? d2d::rules::point_apart(hx, hy, 2) : d2d::rules::ai_distance(hx, hy);\n"
     + ("                        input.target_home_dist = d2d::rules::point_apart(home_x - input.target_x, home_y - input.target_y, 2); }\n" if HAS_TARGET_HOME else "                        }\n") + "                    int cast_word = 0;"),
    ("const auto ai_name = think.name;", "const auto ai_name = think.name;\n                    const int cmd0 = command;\n                    if (ai_name == \"BloodRaven\") command = ctrl[2];"),
    (": d2d::rules::mon_think(ai_name, input, seed, away);",
     ": d2d::rules::mon_think(ai_name, input, seed, away);\n"
     "                    if (ai_name == \"BloodRaven\") ctrl[2] = command;\n"
     "                    if (input.dying) effects.push_back(d2d::rules::ThinkEffect{ d2d::rules::ThinkEffect::command_done, 0, 0 });"),
):
    assert old in block, old[:70]
    block = block.replace(old, new, 1)
open(out_dir + "/mp_replay.cpp", "w").write(head + "\n\nint main(int argc, char** argv) {\n    const bool less_two = argc > 2;   // any argument: the home distance as game.exe takes it (FUN_005dc480, less 2), else as ai.cpp:897 feeds it\n" + block + "    return 0;\n}\n")
