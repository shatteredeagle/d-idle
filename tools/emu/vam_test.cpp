// SPDX-License-Identifier: GPL-3.0-or-later
// verifyskamazon: the d2d side of the independent Amazon calc check (tools/emu/vam_calc.py): d2d's eval_calc with
// base level != level and every stat nonzero (same H as vam_calc.py).
//   --calc CASES   `id calc1..4 petmax delay range len perdelay a1..a6 p1..p5`
//   --dump         the compiled code of every calc cell of Skills 6..35
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <rules.hpp>
#include <skills.hpp>

#include <algorithm>
#include <cctype>
#include <cstdio>
#include <cstdlib>
#include <exception>
#include <filesystem>
#include <fstream>
#include <map>
#include <sstream>
#include <string>
#include <string_view>
#include <utility>

using namespace d2d::game;

static int H(int seed, int stat) { return (seed * 131 + stat * 17) % 89 + 1; }

int main(int argc, char** argv) try {
    if (argc < 2) return 2;
    const std::string_view mode = argv[1];
    if (mode == "--roll") {   // `lo hi min max` -> `result lo' hi'` with d2d's own Rng::range (rules.hpp)
        std::ifstream cs(argv[2]);
        std::uint32_t lo = 0, hi = 0;
        int mn = 0, mx = 0;
        while (cs >> lo >> hi >> mn >> mx) {
            d2d::rules::Rng rng;
            rng.low = lo; rng.high = hi;
            const int v = rng.range(mn, mx);
            std::printf("%d %u %u\n", v, rng.low, rng.high);
        }
        return 0;
    }
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(fs::path(mpq_dir), patch ? fs::path(patch) : fs::path{}, 0x1234);
    if (!data || !data->patched) { std::printf("FAIL: no patched data\n"); return 1; }
    if (mode == "--dump") {
        for (int s = 6; s < 36; ++s) {
            const auto* k = data->skills.get(s);
            if (!k) { std::printf("row %d missing\n", s); continue; }
            const auto dump = [&](const char* name, const d2d::rules::Calc& c) {
                if (c.empty()) return;
                std::printf("%d %s:", s, name);
                for (const auto& i : c.code) std::printf(" %d/%d", int(i.opcode), i.operand1);
                std::printf("\n");
            };
            for (int j = 0; j < 4; ++j) dump(("calc" + std::to_string(j + 1)).c_str(), k->calc[j]);
            dump("petmax", k->petmax); dump("delay", k->delay); dump("range", k->aurarange); dump("len", k->auralen);
            dump("perdelay", k->perdelay_calc);
            for (int j = 0; j < 6; ++j) { std::printf("%d aurastat%d id %d\n", s, j + 1, int(k->aurastat[j])); dump(("aura" + std::to_string(j + 1)).c_str(), k->aura_calc[j]); }
            for (int j = 0; j < 5; ++j) dump(("passive" + std::to_string(j + 1)).c_str(), k->passive_calc[j]);
        }
        return 0;
    }
    std::ifstream cases(argv[2]);
    std::string line;
    while (std::getline(cases, line)) {
        std::istringstream f(line);
        int id = 0, skill_id = 0, lvl = 0, blvl = 0, clvl = 0, seed = 0, n = 0;
        f >> id >> skill_id >> lvl >> blvl >> clvl >> seed >> n;
        std::map<int, std::pair<int, int>> lv;
        for (int k = 0; k < n; ++k) { int o = 0, l = 0, b = 0; f >> o >> l >> b; lv[o] = { l, b }; }
        lv[skill_id] = { lvl, blvl };
        d2d::rules::CalcEnv env;
        env.clvl = clvl;
        env.level = [&](int o) { const auto it = lv.find(o); return it == lv.end() ? 0 : it->second.first; };
        env.base_level = [&](int o) { const auto it = lv.find(o); return it == lv.end() ? 0 : it->second.second; };
        env.stat = [&](int st) { return st == 12 ? clvl : H(seed, st); };
        const auto* k = data->skills.get(skill_id);
        if (!k) continue;
        const auto v = [&](const d2d::rules::Calc& c) { return d2d::rules::eval_calc(data->skills, c, env, skill_id, lvl); };
        std::printf("%d", id);
        for (int j = 0; j < 4; ++j) std::printf(" %d", v(k->calc[j]));
        std::printf(" %d", v(k->petmax));
        for (std::size_t j = 0; j < 5; ++j) {
            bool named = false;
            if (!k->sumskill[j].empty())
                for (const auto& [nm, row] : data->skills.by_name)
                    if (std::ranges::equal(nm, k->sumskill[j], [](char a, char b) { return std::tolower(static_cast<unsigned char>(a)) == std::tolower(static_cast<unsigned char>(b)); })) { named = true; break; }
            if (named) std::printf(" %d", v(k->sumsk_calc[j])); else std::printf(" x");
        }
        std::printf(" %d %d %d %d", v(k->delay), v(k->aurarange), v(k->auralen), v(k->perdelay_calc));
        for (int j = 0; j < 6; ++j) { if (k->aurastat[j] >= 0) std::printf(" %d", v(k->aura_calc[j])); else std::printf(" x"); }
        for (int j = 0; j < 5; ++j) { if (k->passive_stat[j] >= 0) std::printf(" %d", v(k->passive_calc[j])); else std::printf(" x"); }
        std::printf("\n");
    }
    return 0;
} catch (const std::exception& error) {
    std::printf("FAIL: %s\n", error.what());
    return 1;
}
