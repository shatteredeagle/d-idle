// SPDX-License-Identifier: GPL-3.0-or-later
// test-links: game
// Verifier probe (agents verifyphys, verifyplayerroll), dropped into
// tests/game/ of a scratch d2d worktree only.
//   --physroll CASES  each case of physroll_game.py (id res life r1 r2 ...)
//       through the REAL missile_blow (physical only, roll fixed as lo = hi)
//       and the REAL hurt() on a fallen1: life in 256ths after each hit.
//   --elemroll CASES  each case of elemroll_game.py (id elo ehi seedlo
//       seedhi): the REAL missile_blow, fire at res 0 (so damage * 256 + frac
//       is the roll), on an Rng set to the seed: `id roll low high`.
//   --pairs CLS...    the REAL missile_damage of every skill of those classes
//       with an element 0..4, levels 1..40, synergy levels 0 / lvl / 20,
//       masteries 0 / 37 / 140: `cls skill etype elo ehi`.

#include <ai.hpp>
#include <combat.hpp>
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <rules.hpp>
#include <skills.hpp>
#include <d2s_items.hpp>

#include <algorithm>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <sstream>
#include <string>
#include <string_view>
#include <vector>

using namespace d2d::game;
namespace fs = std::filesystem;

int main(int argc, char** argv) {
    if (argc < 3) { std::fprintf(stderr, "usage: --physroll CASES | --elemroll CASES | --pairs CLS...\n"); return 2; }
    const std::string_view mode = argv[1];
    if (mode == "--elemroll") {
        std::ifstream in(argv[2]);
        std::string line;
        while (std::getline(in, line)) {
            std::istringstream f(line);
            long long id, lo, hi;
            unsigned long long sl, sh;
            if (!(f >> id >> lo >> hi >> sl >> sh)) continue;
            d2d::rules::Rng r;
            r.low = std::uint32_t(sl);
            r.high = std::uint32_t(sh);
            const auto blow = d2d::rules::missile_blow({ .etype = 0, .elo = int(lo), .ehi = int(hi) }, d2d::rules::Target{}, {}, r);
            std::printf("%lld %lld %u %u\n", id, (long long)blow.damage * 256 + blow.frac, r.low, r.high);
        }
        return 0;
    }
    const char* mpq = std::getenv("D2_MPQ_DIR");
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    if (!mpq) { std::fprintf(stderr, "D2_MPQ_DIR unset\n"); return 2; }
    auto data = load_game_data(fs::path(mpq), patch ? fs::path(patch) : fs::path{}, 0x1234);
    d2d::rules::Rng rng{ 7 };
    if (mode == "--pairs") {
        for (int a = 2; a < argc; ++a)
            for (const auto& skill : data->skills.rows) {
                if (skill.cls != argv[a] || skill.etype < 0 || skill.etype > 4) continue;
                for (int lvl = 1; lvl <= 40; ++lvl)
                    for (const int syn : { 0, lvl, 20 })
                        for (const int mastery : { 0, 37, 140 }) {
                            d2d::rules::CalcEnv env;
                            env.clvl = std::min(99, lvl * 2 + 10);
                            env.base_level = env.level = [&](int other) { return other == skill.id ? lvl : syn; };
                            env.stat = [&](int stat) { return stat >= d2d::d2s::kPassiveFireMastery && stat <= d2d::d2s::kPassivePoisonMastery ? mastery : 0; };
                            const auto d = d2d::rules::missile_damage(data->skills, skill, env, lvl);
                            std::printf("%s\t%s\t%d\t%d\t%d\n", argv[a], skill.name.c_str(), d.etype, d.elo, d.ehi);
                        }
            }
        return 0;
    }
    if (mode != "--physroll") return 2;
    const auto fallen = data->monsters.by_id.find("fallen1");
    if (fallen == data->monsters.by_id.end()) return 3;
    const Monster made = make_monster(*data, fallen->second, 10, 10, rng, 0);
    std::ifstream in(argv[2]);
    std::string line;
    while (std::getline(in, line)) {
        std::istringstream f(line);
        long long id, res, life;
        if (!(f >> id >> res >> life)) continue;
        Monster m = made;
        m.stats.hit_points = m.hit_points = int(life);
        m.life_frac = 0;
        d2d::rules::Target target;
        target.res[0] = int(res);
        std::printf("%lld", id);
        long long r;
        while (f >> r) {
            const auto blow = d2d::rules::missile_blow({ .phys_lo = int(r), .phys_hi = int(r) }, target, {}, rng);
            hurt(*data, m, blow.damage, 0, blow.frac);
            const long long now = m.alive() ? (long long)m.hit_points * 256 - m.life_frac : 0;
            std::printf(" %lld", now);
            if (now == 0) break;
        }
        std::printf("\n");
    }
    return 0;
}
