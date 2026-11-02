// SPDX-License-Identifier: GPL-3.0-or-later
// verifynmlevel: d2d side. Reads the M lines vnl_game.py wrote (game.exe's real populations) and makes each
// unit with d2d's REAL make_monster (and make_boss for a champion / unique / superunique / minion, with the
// MonUMods game.exe gave it), in the level game.exe made it in. One line out per case:
//   D lid class diff start final exp
// start = final minus make_boss's level_add. Built twice: against the base tree (make_monster has no level id:
// the level is not passed, which is the base behaviour) and with -DVNL_HEAD against the candidate.
#include <ai.hpp>
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <monsters.hpp>
#include <rules.hpp>
#include <uniques.hpp>

#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <exception>
#include <filesystem>
#include <fstream>
#include <sstream>
#include <string>
#include <vector>

using namespace d2d::game;
namespace fs = std::filesystem;

int main(int argc, char** argv) try {
    if (argc < 2) { std::printf("usage: vnl_d2d CASEFILE\n"); return 2; }
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(fs::path(mpq_dir ? mpq_dir : "/tmp/ug-emu"), patch ? fs::path(patch) : fs::path{}, 0x1234);
    if (!data || !data->patched) { std::printf("no 1.14d data\n"); return 1; }
    std::ifstream cases(argv[1]);
    std::string line;
    while (std::getline(cases, line)) {
        std::istringstream in(line);
        if (line.rfind("G ", 0) == 0) {
            // The grid: a plain unit of row `cls` made in level `lid`; its start level (stats.level, no leveladd).
            std::string grid_tag;
            int cls = 0, diff = 0, lid = 0;
            in >> grid_tag >> cls >> diff >> lid;
            if (cls < 0 || std::size_t(cls) >= data->monsters.types.size()) { std::printf("G %d %d %d no-type\n", cls, diff, lid); continue; }
            d2d::rules::Rng rng{ 5 };
#ifdef VNL_HEAD
            const Monster monster = make_monster(*data, cls, 0, 0, rng, diff, true, 1, lid);
#else
            const Monster monster = make_monster(*data, cls, 0, 0, rng, diff, true, 1);
#endif
            std::printf("G %d %d %d %d\n", cls, diff, lid, monster.stats.level);
            continue;
        }
        std::string tag, mod_list;
        long long seed = 0;
        int diff = 0, lid = 0, cls = 0, col = 0, start = 0, final_level = 0, flags = 0, tflags = 0;
        if (!(in >> tag >> seed >> diff >> lid >> cls >> col >> start >> final_level >> flags >> mod_list >> tflags) || tag != "M") continue;
        if (cls < 0 || std::size_t(cls) >= data->monsters.types.size()) { std::printf("D %d %d %d no-type\n", lid, cls, diff); continue; }
        std::vector<int> mods;
        if (mod_list != "-") {
            std::istringstream parts(mod_list);
            for (std::string part; std::getline(parts, part, '.');) if (std::stoi(part)) mods.push_back(std::stoi(part));
        }
        // MonsterData +0x16 as game.exe left it: 1 rolled, 2 superunique, 4 champion, 8 unique, 0x10 minion (a champion is
        // 1|4|8, a superunique 1|2|8). 8 alone is an act boss (FUN_005b1cf0): d2d makes it plain, no make_boss.
        const auto kind = tflags & 2 ? d2d::rules::Boss::superunique : tflags & 4 ? d2d::rules::Boss::champion
                        : tflags & 0x10 ? d2d::rules::Boss::minion : (tflags & 9) == 9 ? d2d::rules::Boss::unique : d2d::rules::Boss::none;
        const bool boss = kind != d2d::rules::Boss::none;
        d2d::rules::Rng rng{ std::uint32_t(seed) ^ std::uint32_t(cls * 7919 + lid) };
#ifdef VNL_HEAD
        Monster monster = make_monster(*data, cls, 0, 0, rng, diff, !boss, 1, lid);
#else
        Monster monster = make_monster(*data, cls, 0, 0, rng, diff, !boss, 1);
#endif
        int add = 0;
        if (boss) {
            add = d2d::rules::boss_stats(data->umods, data->monsters.types[std::size_t(cls)], kind, mods, diff).level_add;
            make_boss(*data, monster, kind, mods, -1, 0, diff, rng, 1);
        }
        std::printf("D %d %d %d %d %d %d\n", lid, cls, diff, monster.stats.level - add, monster.stats.level, monster.stats.exp);
    }
    return 0;
} catch (const std::exception& error) {
    std::fprintf(stderr, "vnl_d2d: %s\n", error.what());
    return 1;
}
