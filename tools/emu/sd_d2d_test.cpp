// SPDX-License-Identifier: GPL-3.0-or-later
// seeddist: the d2d side of tools/emu/sd_game.py. Calls d2d's own compiled code on the real loaded tables:
// make_monster (life, on the world rng, as Fight::rooms_up does), monster_look / monster_variant /
// monster_props (the unit seed as spawn_monsters draws it), monster_blow (to-hit + physical roll, as the AI
// swings), tc_upgrade + roll_drops (as Loot::drop). Built by hand against a scratch upstream build
// (tools/emu/sd_build.sh), never part of upstream. Output: one line per case on stdout.
//   sd_d2d_test --life  ID N              N monsters in a row off the world rng {0x2545f491} (stream A)
//   sd_d2d_test --lifeb ID GAMES PER ADV  GAMES fresh world rngs {0x2545f491}, game g advanced by ADV * g draws first, PER monsters
//                                       (stream B: ADV large = disjoint stretches; ADV 0 = every game starts alike)
//   sd_d2d_test --swing ID DEF DLVL GAMES PER   GAMES x PER monsters, 100 swings each at a defender (DEF, level DLVL); "hit phys"
//   sd_d2d_test --drop  ID MF K GAMES PER       first drop of GAMES x PER monsters (unit seeds off game streams), K extra seed
//                                       draws (0, or 1000 + n: a draw count 0..n from an independent stream), MF magic find
//   sd_d2d_test --dropfile ID MF FILE   the same roll on the unit seed states "low high level" read from FILE (control)
//   sd_d2d_test --info ID               id level tc1 tc4 tcquestid
// The game stream: {map seed, 666}, 4 steps (regions, objects, sunitproxy, quests: Spawning's order), then one step per unit
// made, with a gap model per game: even games gap 1, odd games a gap of 1..5 steps (units, items, missiles made in between).

#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <monsters.hpp>
#include <combat.hpp>
#include <drops.hpp>
#include <ai.hpp>

#include <algorithm>
#include <cstdio>
#include <cstdlib>
#include <exception>
#include <filesystem>
#include <fstream>
#include <string>
#include <string_view>

using namespace d2d::game;
namespace R = d2d::rules;

static std::string cat(const std::vector<R::Drop>& drops) {
    if (drops.empty()) return "-";
    std::string out;
    for (const auto& d : drops) {
        if (!out.empty()) out += ",";
        out += d.code + ":" + std::to_string(d.quality);
        if (d.mul) out += "*" + std::to_string(d.mul);
    }
    return out;
}

int main(int argc, char** argv) try {
    if (argc < 3) return 2;
    const std::string_view mode = argv[1];
    const std::string id = argv[2];
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(fs::path(mpq_dir), patch ? fs::path(patch) : fs::path{}, 0x1234);
    if (!data || !data->patched) { std::printf("FAIL: no patched data\n"); return 1; }
    const auto found = data->monsters.by_id.find(id);
    if (found == data->monsters.by_id.end()) { std::printf("FAIL: no row %s\n", id.c_str()); return 1; }
    const int row = int(found->second);
    const auto& type_info = data->monsters.types[std::size_t(row)];
    auto arg = [&](int i) { return std::atoll(argv[i]); };

    if (mode == "--info") {
        std::printf("%s %d %s %s %d\n", id.c_str(), type_info.level[0], type_info.diff[0].treasure_class.c_str(), type_info.tc_quest[0].c_str(), int(type_info.tc_quest_id));
        return 0;
    }
    if (mode == "--life" || mode == "--lifeb") {
        const long long games = mode == "--life" ? 1 : arg(3), per = mode == "--life" ? arg(3) : arg(4);
        for (long long g = 0; g < games; ++g) {
            R::Rng world{ 0x2545f491u };
            if (mode == "--lifeb") for (long long k = 0; k < arg(5) * g; ++k) world.next();
            for (long long i = 0; i < per; ++i) {
                auto monster = make_monster(*data, row, 100.5f, 100.5f, world, 0, true, 1);
                std::printf("%d\n", monster.stats.hit_points);
            }
        }
        return 0;
    }
    // unit seeds off a game stream
    auto unit_supply = [&](long long game_index, auto&& body) {
        R::Rng game{ std::uint32_t(0x51ed270bu * std::uint32_t(game_index + 1) + 12345u + std::uint32_t(row) * 0x1000193u) };
        for (int k = 0; k < 4; ++k) game.next();
        R::Rng gaps{ std::uint32_t(0x9e3779b1u * std::uint32_t(game_index + 7)) };
        return std::pair{ game, gaps };
    };
    if (mode == "--swing" || mode == "--drop") {
        const bool swing = mode == "--swing";
        const long long games = arg(5), per = arg(6);
        std::vector<R::Components> sets(3);
        for (long long g = 0; g < games; ++g) {
            auto [game, gaps] = unit_supply(g, 0);
            R::Rng world{ 0x2545f491u };
            R::Rng extra{ std::uint32_t(0x85ebca6bu * std::uint32_t(g + 3)) };
            for (long long i = 0; i < per; ++i) {
                if (g % 2 == 1) for (int skip = int(gaps(5)); skip > 0; --skip) game.next();   // other units made in between
                R::Rng unit{ game.next() };
                auto monster = make_monster(*data, row, 100.5f, 100.5f, world, 0, true, 1);
                (void)R::monster_look(&sets, type_info.choices, unit);
                (void)R::monster_variant(row, unit);
                (void)R::monster_props(data->rules, type_info, 0, unit);
                if (swing) {
                    R::Fighter defender;
                    defender.defense = int(arg(3));
                    for (int s = 0; s < 100; ++s) {
                        const auto taken = R::monster_blow(defender, int(arg(4)), R::Motion::standing, monster.stats, false, unit);
                        std::printf("%d %d\n", taken.hit ? 1 : 0, taken.hit ? taken.physical : 0);
                    }
                    continue;
                }
                const long long k_mode = arg(4);
                if (k_mode >= 1000) { for (long long n = extra(int(k_mode - 1000) + 1); n > 0; --n) unit.next(); }
                else for (long long n = 0; n < k_mode; ++n) unit.next();
                std::vector<R::Drop> drops;
                const auto rolled = R::tc_upgrade(data->rules, argc > 7 ? std::string(argv[7]) : type_info.diff[0].treasure_class, 0);
                R::roll_drops(data->rules, rolled, monster.stats.level, unit, drops, 1, int(arg(3)), 6, 0, 1);
                std::printf("%s\n", cat(drops).c_str());
            }
        }
        return 0;
    }
    if (mode == "--lifectl") {     // life on the unit seed states "low high" read from FILE (argv[3])
        std::ifstream in(argv[3]);
        unsigned long long low = 0, high = 0;
        int level = 0;
        while (in >> low >> high >> level) {
            R::Rng unit;
            unit.low = std::uint32_t(low); unit.high = std::uint32_t(high);
            const auto stats = R::monster_stats(data->monsters, row, 0, unit);
            std::printf("%d %08x\n", stats.hit_points, unit.low);
        }
        return 0;
    }
    if (mode == "--dropfile") {
        std::ifstream in(argv[4]);
        unsigned long long low = 0, high = 0;
        int level = 0;
        while (in >> low >> high >> level) {
            R::Rng unit;
            unit.low = std::uint32_t(low); unit.high = std::uint32_t(high);
            std::vector<R::Drop> drops;
            const auto rolled = R::tc_upgrade(data->rules, argc > 5 ? std::string(argv[5]) : type_info.diff[0].treasure_class, 0);
            R::roll_drops(data->rules, rolled, level, unit, drops, 1, int(arg(3)), 6, 0, 1);
            std::printf("%s %08x\n", cat(drops).c_str(), unit.low);
        }
        return 0;
    }
    return 2;
} catch (const std::exception& error) {
    std::printf("FAIL: %s\n", error.what());
    return 1;
}
