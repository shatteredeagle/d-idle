// SPDX-License-Identifier: GPL-3.0-or-later
// test-links: game
// Verifier probe (agent palverify), dropped into tests/game/ of a scratch copy
// only. `--hammer CASES`: the cases of hammer_game.py, one line each:
// `id lo hi life targetable` from the REAL rules::missile_damage, (head only)
// rules::aura_influence / rules::influenced with the hammer's Param1, the
// Missiles.txt row's dmg_func / dmg_par1 / dmg_par2 as Fight::strike sets them,
// Monster::target() of make_monster(row) for the undead / demon flags, then
// rules::missile_blow on the case's seed and hurt(). Build with -DPV_HEAD on the
// head tree. `--flags`: `flag row undead demon` for every MonStats row.
// `--conc`: Concentration's damagepercent (as Fight::fire evaluates it) at 1..40.

#include <ai.hpp>
#include <combat.hpp>
#include <d2s_items.hpp>
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <rules.hpp>
#include <skills.hpp>

#include <algorithm>
#include <cassert>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <exception>
#include <filesystem>
#include <fstream>
#include <map>
#include <sstream>
#include <string>
#include <string_view>
#include <type_traits>

using namespace d2d::game;

int main(int argc, char** argv) try {
    if (argc < 2) { std::printf("SKIP: oracle mode\n"); return D2D_TEST_SKIPPED; }
    const std::string_view mode(argv[1]);
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    if (!mpq_dir) return D2D_TEST_SKIPPED;
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(fs::path(mpq_dir), patch ? fs::path(patch) : fs::path{}, 0x1234);
    assert(data && data->patched);
    const auto& names = data->skills.by_name;
    const int hammer_id = names.at("Blessed Hammer"), aim_id = names.at("Blessed Aim"), vigor_id = names.at("Vigor");
    const auto* hammer = data->skills.get(hammer_id);
    if (mode == "--flags") {
        int dmg_rows = 0;
        for (const auto& [name, info] : data->missiles)
            if (info.skill == "Blessed Hammer" || name == "blessedhammer") std::printf("# missile %s skill '%s' etype %d\n", name.c_str(), info.skill.c_str(), info.etype);
#ifdef PV_HEAD
        for (const auto& [name, info] : data->missiles)
            if (info.dmg_func != 0) { ++dmg_rows; std::printf("# dmg row %s func %d par %d %d\n", name.c_str(), info.dmg_func, info.dmg_par1, info.dmg_par2); }
#endif
        std::printf("# dmg rows %d, hammer id %d par0 %d etype %d\n", dmg_rows, hammer_id, hammer->par[0], hammer->etype);
        for (std::size_t i = 0; i < data->monsters.types.size(); ++i) {
            const auto& t = data->monsters.types[i];
            std::printf("flag %zu %d %d\n", i, t.undead ? 1 : 0, t.demon ? 1 : 0);
        }
        return 0;
    }
    if (mode == "--conc") {
        const auto* conc = data->skills.get(names.at("Concentration"));
        d2d::rules::CalcEnv env;
        for (int lvl = 1; lvl <= 40; ++lvl) {
            env.base_level = env.level = [&](int s) { return s == conc->id ? lvl : 0; };
            env.stat = [](int) { return 0; };
            for (std::size_t i = 0; i < conc->aurastat.size(); ++i)
                if (conc->aurastat[i] == d2d::d2s::kDamagePercent)
                    std::printf("conc %d state '%s' damagepercent %d\n", lvl, conc->aurastate.c_str(), d2d::rules::eval_calc(data->skills, conc->aura_calc[i], env, conc->id, lvl));
        }
        return 0;
    }
    if (mode != "--hammer" || argc < 3) return 2;
    const std::remove_cvref_t<decltype(data->missiles)>::mapped_type* row = nullptr;
    for (const auto& [name, info] : data->missiles) if (name == "blessedhammer") row = &info;
    assert(row);
    std::ifstream cases(argv[2]);
    std::string line;
    int skipped = 0;
    const bool predraw = std::getenv("PV_PREDRAW") && std::string_view(std::getenv("PV_PREDRAW")) == "1";
    while (std::getline(cases, line)) {
        std::istringstream in(line);
        long long id, lvl, clvl, aim, vigor, pct, mon, res, life, sl, sh;
        in >> id >> lvl >> clvl >> aim >> vigor >> pct >> mon >> res >> life >> sl >> sh;
        if (mon < 0 || std::size_t(mon) >= data->monsters.types.size()) { ++skipped; continue; }
        std::map<int, int> levels{ { hammer_id, int(lvl) }, { aim_id, int(aim) }, { vigor_id, int(vigor) } };
        d2d::rules::CalcEnv env;
        env.clvl = int(clvl);
        env.base_level = env.level = [&](int s) { const auto f = levels.find(s); return f == levels.end() ? 0 : f->second; };
        env.stat = [](int) { return 0; };
        auto damage = d2d::rules::missile_damage(data->skills, *hammer, env, int(lvl));
#ifdef PV_HEAD
        damage = d2d::rules::influenced(damage, d2d::rules::aura_influence(int(pct), hammer->par[0]));
        if (row->dmg_func == 5) { damage.undead_pct = row->dmg_par1; damage.demon_pct = row->dmg_par2; }
#endif
        d2d::rules::Rng made_rng{ 3 };
        Monster monster = make_monster(*data, int(mon), 10, 10, made_rng, 0);
        monster.stats.hit_points = monster.hit_points = int(life);
        monster.life_frac = 0;
        auto target = monster.target(*data);
        for (auto& r : target.res) r = 0;
        target.res[1] = int(res);                                   // magic (kRes[4])
        std::array<int, 4> pierces{};
        d2d::rules::Rng rng;
        rng.low = std::uint32_t(sl);
        rng.high = std::uint32_t(sh);
        // PV_PREDRAW=1: advance the stream by the draw FUN_005a89a0 makes first (both stats over 0, rand(max - min)).
        if (predraw && damage.elo > 0 && damage.ehi > 0) (void)rng(std::uint32_t(std::max(damage.elo, damage.ehi) - std::min(damage.elo, damage.ehi)));
        const auto blow = d2d::rules::missile_blow(damage, target, pierces, rng);
        const bool targetable = monster.targetable();       // not targetable: d2d's hurt() refuses, no hit lands in play (out of domain)
        hurt(*data, monster, blow.damage, 0, blow.frac);
        std::printf("%lld %d %d %d %d\n", id, damage.elo, damage.ehi, monster.alive() ? monster.hit_points * 256 - monster.life_frac : 0, targetable ? 1 : 0);
    }
    std::fprintf(stderr, "# skipped %d\n", skipped);
    return 0;
} catch (const std::exception& error) {
    std::fprintf(stderr, "test_palprobe: %s\n", error.what());
    return 1;
}
