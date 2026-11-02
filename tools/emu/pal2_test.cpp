// SPDX-License-Identifier: GPL-3.0-or-later
// palskill (card palskill): the d2d side of tools/emu/pal2.py. Built by hand against the libraries of a scratch
// upstream build (tools/emu/pal2_build.sh), never part of upstream.
//   --calc CASES   C lines: `C id c1 c2 c3 c4 tohit` (eval_calc of the row's four calc slots, skill_tohit) on the loaded patched table
//   --blow CASES   T lines: `T id hit_smite hit_plain` (player_blow with swing.smite, then without)
//                  S lines: `S id hit phys` (player_blow with swing.smite on a shield's item_base row)

#include <combat.hpp>
#include <d2s_items.hpp>
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <rules.hpp>
#include <skills.hpp>

#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <exception>
#include <filesystem>
#include <fstream>
#include <map>
#include <optional>
#include <sstream>
#include <string>
#include <string_view>

using namespace d2d::rules;
namespace fs = std::filesystem;

int main(int argc, char** argv) try {
    if (argc < 3) return 2;
    const std::string_view mode = argv[1];
    namespace s = d2d::d2s;
    std::optional<d2d::game::GameData> data;
    if (mode == "--calc") {
        const char* mpq_dir = std::getenv("D2_MPQ_DIR");
        const char* patch = std::getenv("D2_PATCH_INSTALLER");
        data = d2d::game::load_game_data(fs::path(mpq_dir), patch ? fs::path(patch) : fs::path{}, 0x1234);
        if (!data || !data->patched) { std::printf("FAIL: no patched data\n"); return 1; }
    }
    Tables tables;
    d2d::d2s::Item weapon, shield;
    weapon.code = "tst";
    shield.code = "tsh";
    std::ifstream cases(argv[2]);
    std::string line;
    while (std::getline(cases, line)) {
        std::istringstream in(line);
        std::string tag;
        long long id = 0;
        if (!(in >> tag >> id)) continue;
        if (tag == "C" && mode == "--calc") {
            int skill_id = 0, lvl = 0, clvl = 0, count = 0;
            in >> skill_id >> lvl >> clvl >> count;
            std::map<int, int> levels;
            for (int k = 0; k < count; ++k) { int other = 0, level = 0; in >> other >> level; levels[other] = level; }
            levels[skill_id] = lvl;
            CalcEnv env;
            env.clvl = clvl;
            env.base_level = env.level = [&](int other) { const auto found = levels.find(other); return found == levels.end() ? 0 : found->second; };
            env.stat = [&](int) { return 0; };
            const auto* skill = data->skills.get(skill_id);
            if (!skill) continue;
            std::printf("C %lld", id);
            for (std::size_t k = 0; k < 4; ++k) std::printf(" %d", eval_calc(data->skills, skill->calc[k], env, skill_id, lvl));
            std::printf(" %d\n", skill_tohit(data->skills, *skill, env, lvl));
            continue;
        }
        if (mode != "--blow") continue;
        StatSum sum{}, wsum{};
        d2d::d2s::Stats stats;
        Target target;
        if (tag == "T") {
            std::uint32_t s0 = 0, s1 = 0;
            long long ar = 0, pct = 0, alvl = 1, dlvl = 1, armed = 0, def = 0, mdef = 0;
            if (!(in >> s0 >> s1 >> ar >> pct >> alvl >> dlvl >> armed >> def >> mdef)) continue;
            tables.item_base["tst"] = { .mindam = 1, .maxdam = 2 };
            stats.values[s::kLevel] = alvl; stats.values[s::kDex] = 7; stats.values[s::kStr] = 10;
            sum[s::kToHit] = ar; sum[s::kToHitPercent] = pct;
            const auto fighter = make_fighter(tables, armed ? &weapon : nullptr, nullptr, sum, wsum, stats, {}, 0, {});
            target.armor_class = int(def + mdef); target.level = int(dlvl);
            int hit[2];
            for (int with_smite = 1; with_smite >= 0; --with_smite) {
                Swing swing;
                swing.smite = with_smite != 0;
                Rng rng; rng.low = s0; rng.high = s1;
                hit[1 - with_smite] = player_blow(fighter, target, int(alvl), rng, swing).hit ? 1 : 0;
            }
            std::printf("T %lld %d %d\n", id, hit[0], hit[1]);
            continue;
        }
        if (tag == "S") {
            std::uint32_t d0 = 0, d1 = 0;
            long long bm = 0, bM = 0, hl = 0, hh = 0, ed = 0, n111 = 0, dp = 0, str = 0, dex = 0, sb = 0, db = 0, off = 0;
            if (!(in >> d0 >> d1 >> bm >> bM >> hl >> hh >> ed >> n111 >> dp >> str >> dex >> sb >> db >> off)) continue;
            tables.item_base["tsh"] = { .mindam = int(bm), .maxdam = int(bM), .str_bonus = int(sb), .dex_bonus = int(db) };
            stats.values[s::kLevel] = 1; stats.values[s::kStr] = str; stats.values[s::kDex] = dex;
            sum[s::kMaxDamagePercent] = sum[s::kMinDamagePercent] = off;
            sum[s::kNormalDamage] = n111; sum[s::kDamagePercent] = dp;
            const auto fighter = make_fighter(tables, nullptr, &shield, sum, wsum, stats, {}, 0, {});
            target.armor_class = 0; target.level = 1;
            target.hit_points = target.max_hp = 1 << 30;
            Swing swing;
            swing.smite = true; swing.ed_pct = int(ed); swing.skill_lo = int(hl); swing.skill_hi = int(hh);
            Rng rng; rng.low = d0; rng.high = d1;
            const auto blow = player_blow(fighter, target, 1, rng, swing);
            std::printf("S %lld %d %d\n", id, blow.hit ? 1 : 0, blow.phys);
        }
    }
    return 0;
} catch (const std::exception& error) {
    std::fprintf(stderr, "pal2_test: %s\n", error.what());
    return 1;
}
