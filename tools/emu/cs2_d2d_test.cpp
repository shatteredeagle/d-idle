// SPDX-License-Identifier: GPL-3.0-or-later
// covstates2: d2d's REAL put_state (components/game/fight/states.cpp, declared in ai.hpp) on a monster's curse / cry slots,
// fed the same put sequences as tools/emu/cs2_put.py. Only the slot choice and the length floor at the call sites
// (fight.cpp: kCurse / kConfuse / kAttract -> curse else cry, max(length, 25) frames; Battle Cry max(length, 1) and the
// "already running" guard; Howl's Terror into curse) are transcribed here.
//   cs2_d2d_test cases.tsv > d2d.tsv
// A case line: index k sid level length ... (k puts). Output: index acc(0/1 per put) then state:skill:level:remaining per slot.
#include <algorithm>
#include <cstdint>
#include <cstdio>
#include <fstream>
#include <iostream>
#include <map>
#include <sstream>
#include <string>
#include <vector>

#include "ai.hpp"
#include "gamedata.hpp"

using namespace d2d::game;

namespace {
struct Row { int state; bool curse; };
// skill -> (state, goes to the curse slot)
const std::map<int, Row> kSkills = {
    {66, {9, true}}, {71, {23, true}}, {72, {19, true}}, {76, {55, true}}, {77, {56, true}}, {81, {59, true}}, {82, {58, true}},
    {86, {57, true}}, {87, {60, true}}, {91, {61, true}}, {309, {113, true}}, {310, {114, true}}, {130, {56, true}},
    {137, {27, false}}, {146, {89, false}}, {8, {17, false}}, {17, {87, false}},
};
}  // namespace

int main(int argc, char** argv) {
    if (argc < 2) return 2;
    std::ifstream in(argv[1]);
    GameData game_data;
    std::string line;
    const std::uint32_t now = 100000;
    while (std::getline(in, line)) {
        std::istringstream ss(line);
        int idx, k;
        if (!(ss >> idx >> k)) continue;
        Monster monster;
        std::string acc;
        for (int i = 0; i < k; ++i) {
            int sid, lvl, len;
            ss >> sid >> lvl >> len;
            const Row row = kSkills.at(sid);
            auto& slot = row.curse ? monster.curse : monster.cry;
            const int floor = sid == 146 ? 1 : 25;
            const std::uint32_t until = now + std::uint32_t(std::max(len, floor)) * 40;
            bool ok;
            if (sid == 146 && monster.cry.skill == 146 && now < monster.cry.until) ok = false;   // fight.cpp:2924 guard
            else ok = put_state(game_data, monster, slot, { sid, lvl, until, false }, now);
            acc += ok ? '1' : '0';
        }
        std::vector<std::string> nodes;
        for (const auto* s : { &monster.curse, &monster.cry })
            if (s->skill >= 0) {
                std::ostringstream o;
                o << kSkills.at(s->skill).state << ':' << s->skill << ':' << s->level << ':' << (s->until - now) / 40;
                nodes.push_back(o.str());
            }
        std::sort(nodes.begin(), nodes.end(), [](const std::string& a, const std::string& b) {
            return std::stoi(a.substr(0, a.find(':'))) < std::stoi(b.substr(0, b.find(':')));
        });
        std::cout << idx << ' ' << acc;
        for (const auto& n : nodes) std::cout << ' ' << n;
        std::cout << '\n';
    }
    return 0;
}
