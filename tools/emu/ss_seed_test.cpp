// SPDX-License-Identifier: GPL-3.0-or-later
// seedspawn: d2d's unit seed of every monster a level's population makes: the seed it starts from (the game seed's step,
// spawn.seed) and the seed spawn_monsters leaves it with (Monster::seed: after the look, the variant and the props).
//   ss_seed_test FIRST-LAST LEVEL        one block per seed:  `seed N` then `mon X,Y: type@start>end ...` per populated room
// Same room key (level-relative tiles) and creation order within a room as drlg-dump's `monsters` and monsters.py.
#include <ai.hpp>
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <monsters.hpp>

#include <cstdio>
#include <cstdlib>
#include <exception>
#include <filesystem>
#include <span>
#include <string>

namespace fs = std::filesystem;

int main(int argc, char** argv) try {
    if (argc < 3) return 2;
    const std::string range = argv[1];
    const int id = std::atoi(argv[2]);
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = d2d::game::load_game_data(fs::path(mpq_dir), patch ? fs::path(patch) : fs::path{}, 0);
    if (!data) return 1;
    const auto dash = range.find('-');
    const auto first = std::uint32_t(std::stoul(range.substr(0, dash), nullptr, 0));
    const auto last = dash == std::string::npos ? first : std::uint32_t(std::stoul(range.substr(dash + 1), nullptr, 0));
    for (auto seed = first;; ++seed) {
        d2d::game::set_map_seed(*data, seed, 0);
        const auto* level = data->level(id);
        auto spawning = d2d::game::start_spawning(*data, 0);
        const auto order = d2d::game::populate_level(*data, spawning, *level);
        const auto& spawns = spawning.levels[level].spawns;
        const auto& region = spawning.regions[std::size_t(level->id)];
        d2d::rules::Rng world{ 0x2545f491u };
        std::printf("seed %u\n", seed);
        for (std::size_t k = 0; k < order.size(); ++k) {
            const auto [room, from] = order[k];
            const std::size_t to = k + 1 < order.size() ? order[k + 1].second : spawns.size();
            if (from == to) continue;
            std::printf("mon %d,%d:", level->rooms[room].x, level->rooms[room].y);
            const auto made = d2d::game::spawn_monsters(*data, std::span(spawns).subspan(from, to - from), region, world, 0, 1);
            for (std::size_t i = from, m = 0; i < to; ++i) {
                const auto& spawn = spawns[i];
                if (spawn.type < 0 || std::size_t(spawn.type) >= data->mon_npc.size()) continue;   // spawn_monsters skips it too
                std::printf(" %d@%08x>%08x.%08x%s", spawn.type, spawn.seed, made[m].seed.low, made[m].seed.high, spawn.boss != d2d::rules::Boss::none ? "B" : "");
                ++m;
            }
            std::printf("\n");
        }
        if (seed == last) break;
    }
    return 0;
} catch (const std::exception& error) {
    std::printf("FAIL: %s\n", error.what());
    return 1;
}
