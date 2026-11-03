// SPDX-License-Identifier: GPL-3.0-or-later
// covmissiles2: `--skill-missile CASES`: what d2d's andariel_missile (ai.cpp) makes of a skill-sourced monster missile row, with the REAL
// loader, the REAL rules::row_damage and the REAL ai.hpp missile_blow (the two calls andariel_missile makes: poison = row_damage(row, lvl),
// stats = missile_blow(row, poison, monster level)), then the real rules::monster_blow on the produced stats against a plain defender.
// A case line: `case row level` (tab separated). Output per case:
//   case amin amax | E type min max dur (per element of mode A2) | T to_hit collide_kill pierce size next_hit | R range_frames | B poison ticks
// amin / amax = MonStats a2_min / a2_max (whole points); R = max(range, 1) (andariel_missile: now + R x 40 ms); B = lowest and highest Taken.poison (per frame) over 200 rng seeds
// and poison_ticks of the real monster_blow (a plain defender, to-hit forced), `B 0 0 0` when none.

#include <ai.hpp>
#include <combat.hpp>
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <monsters.hpp>
#include <rules.hpp>

#include <algorithm>
#include <cassert>
#include <cstdio>
#include <cstdlib>
#include <exception>
#include <filesystem>
#include <fstream>
#include <sstream>
#include <string>
#include <string_view>

using namespace d2d::game;

int main(int argc, char** argv) try {
    if (argc < 3 || std::string_view(argv[1]) != "--skill-missile") {
        std::printf("SKIP: an oracle mode, run `cm2_probe --skill-missile CASES` (see tools/emu/cm2_run.sh)\n");
        return D2D_TEST_SKIPPED;
    }
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* home = std::getenv("HOME");
    const fs::path data_dir = mpq_dir ? fs::path(mpq_dir) : fs::path(home ? home : "") / "Workspace/private/diablo2";
    if (!fs::exists(data_dir / "d2data.mpq")) { std::printf("SKIP: no d2data.mpq in %s\n", data_dir.string().c_str()); return D2D_TEST_SKIPPED; }
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(data_dir, patch ? fs::path(patch) : fs::path{}, 0x1234);
    assert(data && data->patched);

    std::ifstream cases(argv[2]);
    std::string line;
    while (std::getline(cases, line)) {
        std::istringstream fields(line);
        int case_id = 0, lvl = 0;
        std::string row;
        std::getline(fields, row, '\t');
        case_id = std::atoi(row.c_str());
        std::getline(fields, row, '\t');
        fields >> lvl;
        const auto found = data->missiles.find(row);
        if (found == data->missiles.end()) { std::printf("%d -\n", case_id); continue; }
        const auto& info = found->second;
        const auto poison = d2d::rules::row_damage(info.etype, info.emin, info.emax, info.emin_lev, info.emax_lev, info.hitshift, info.elen, info.elen_lev, lvl);
        const auto stats = missile_blow(info, poison, 40);
        std::printf("%d %d %d |", case_id, stats.a2_min, stats.a2_max);
        for (const auto& element : stats.elements)
            if (element.type >= 0 && element.mode == "A2") std::printf(" E %d %d %d %d", element.type, element.min, element.max, element.dur);
        std::printf(" | T %d %d %d %d %d | R %d |", int(info.to_hit), int(info.collide_kill), int(info.pierce), info.size, int(info.next_hit), std::max(info.range, 1));
        d2d::rules::MonStats shot = stats;
        shot.level = 99; shot.to_hit = 1000000;
        int low = 1 << 30, high = 0, ticks = 0, hits = 0;
        for (unsigned seed = 1; seed <= 200; ++seed) {            // the poison of one blow varies with the seed: its range over 200 seeds
            d2d::rules::Rng rng(seed);
            d2d::rules::Fighter defender;
            d2d::rules::Taken taken;
            for (int i = 0; i < 50 && !taken.hit; ++i) taken = d2d::rules::monster_blow(defender, 1, d2d::rules::Motion::standing, shot, true, rng, true);
            if (!taken.hit) continue;
            ++hits; low = std::min(low, taken.poison); high = std::max(high, taken.poison); ticks = taken.poison_ticks;
        }
        std::printf(" B %d %d %d\n", hits ? low : 0, high, ticks);
    }
    return 0;
} catch (const std::exception& error) {
    std::fprintf(stderr, "cm2_probe: %s\n", error.what());
    return 1;
}
