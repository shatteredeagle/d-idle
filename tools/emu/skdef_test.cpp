// SPDX-License-Identifier: GPL-3.0-or-later
// `--defend CASES`: a player or a hireling as the defender of a monster's hit, the cases of
// tools/emu/skdef_resist.py, one line each to diff with game.exe's: `id adj effres dealt pdmg plen`.
// effres follows d2d's code line for line (Character::panel / Fight::merc_fighter:
// clamp(sum + difficulty penalty, -100, min(75 + max stat, 95)); make_fighter: damage reduced % =
// min(stat 36, 50), no magic resistance; poison length reduction clamp(stat 110 + penalty, -100, 75)).
// dealt, pdmg, plen are the REAL rules::monster_blow (whole points) with a Fighter built that way.
// adj is "-": d2d keeps no curse on a player's or hireling's resistances.
// (Our copy; upstream path would be tests/game/test_defend.cpp.)

#include <combat.hpp>
#include <rules.hpp>

#include <algorithm>
#include <array>
#include <cstdio>
#include <exception>
#include <fstream>
#include <sstream>
#include <string>
#include <string_view>

int main(int argc, char** argv) try {
    if (argc < 3 || std::string_view(argv[1]) != "--defend") {
        std::printf("SKIP: an oracle mode, run `test_def --defend CASES` (see tools/emu/skdef_resist.py)\n");
        return 77;
    }
    std::ifstream cases(argv[2]);
    std::string line;
    while (std::getline(cases, line)) {
        std::istringstream fields(line);
        long long case_id = 0;
        int kind = 0, diff = 0, exp = 0, elem = 0, s = 0, m = 0, v = 0, p = 0, aux = 0, roll = 0, len = 0;
        if (!(fields >> case_id >> kind >> diff >> exp >> elem >> s >> m >> v >> p >> aux >> roll >> len) || elem < 0 || elem > 5 || kind > 1) continue;
        const std::int64_t penalty = exp ? std::array<std::int64_t, 3>{ 0, -40, -100 }[std::size_t(diff)] : std::array<std::int64_t, 3>{ 0, -20, -50 }[std::size_t(diff)];
        d2d::rules::Fighter defender;
        const auto cap = std::min<std::int64_t>(75 + m, 95);
        const auto resist = [&] { return int(std::clamp<std::int64_t>(s + penalty, -100, cap)); };
        std::string effres = "-";
        d2d::rules::MonStats attacker;
        attacker.to_hit = 1000000; attacker.level = 99;
        if (elem == 0) {
            defender.dr_pct = int(std::min<std::int64_t>(s, 50)); defender.dr_flat = aux;
            effres = std::to_string(defender.dr_pct);
            attacker.a1_min = attacker.a1_max = roll;
        } else if (elem == 1) {
            defender.mdr = aux;
            attacker.elements[0] = { 4, 100, roll, roll, 0, "A1" };
        } else {
            const int r = resist();
            effres = std::to_string(r);
            defender.mdr = elem == 5 ? 0 : aux;
            defender.res = { r, r, r, r };
            if (elem == 5) {
                defender.plr = int(std::clamp<std::int64_t>(aux + penalty, -100, 75));
                attacker.elements[0] = { 3, 100, roll, roll, len, "A1" };
            } else {
                attacker.elements[0] = { elem - 2, 100, roll, roll, 0, "A1" };
            }
        }
        d2d::rules::Taken taken;
        for (std::uint32_t seed = 1; seed < 5000 && !taken.hit; ++seed) {
            d2d::rules::Rng rng(seed);
            taken = d2d::rules::monster_blow(defender, 99, d2d::rules::Motion::standing, attacker, false, rng);
        }
        std::printf("%lld - %s ", case_id, effres.c_str());
        if (elem == 5) std::printf("0 %d %d\n", taken.poison, taken.poison_ticks);
        else std::printf("%d - -\n", taken.damage);
    }
    return 0;
} catch (const std::exception& error) {
    std::fprintf(stderr, "test_def: %s\n", error.what());
    return 1;
}
