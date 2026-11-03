// SPDX-License-Identifier: GPL-3.0-or-later
// seedspawn: d2d's REAL make_monster (components/game/ai.cpp) fed one world stream, as d2d spawns a level's monsters:
// reads game.exe's class ids (MonStats ids) one per line from stdin, in the order game.exe made them; for each makes
// the monster on ONE persistent `Rng{0x2545f491}` (the World's rng, world.hpp) and prints
//   class row min_hp max_hp life
// (hit points as d2d rolled them at Normal, one player). Built by ss_life_build.sh.
#include <ai.hpp>
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <monsters.hpp>

#include <cstdio>
#include <cstdlib>
#include <exception>
#include <filesystem>
#include <iostream>

namespace fs = std::filesystem;

int main() try {
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = d2d::game::load_game_data(fs::path(mpq_dir), patch ? fs::path(patch) : fs::path{}, 0x1234);
    if (!data || !data->patched) { std::printf("FAIL: no patched data\n"); return 1; }
    d2d::rules::Rng world{ 0x2545f491u };
    std::size_t id;
    while (std::cin >> id) {
        if (id >= data->mon_bin.size()) { std::printf("%zu -1 0 0 0\n", id); continue; }
        const int row = int(data->mon_bin[id]);
        const auto range = d2d::rules::monster_ranges(data->monsters, row, 0);
        const auto monster = d2d::game::make_monster(*data, row, 10.5f, 10.5f, world, 0, true, 1);
        std::printf("%zu %d %d %d %d\n", id, row, range.min_hp, range.max_hp, monster.stats.hit_points);
    }
    return 0;
} catch (const std::exception& error) {
    std::printf("FAIL: %s\n", error.what());
    return 1;
}
