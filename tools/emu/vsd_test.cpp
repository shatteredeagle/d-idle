// SPDX-License-Identifier: GPL-3.0-or-later
// verifyskdruid: the d2d side of tools/emu/vsd_calc.py. Like skdruid_test.cpp, but a case gives every other
// skill a BASE level and an item BONUS, so CalcEnv::base_level (blvl) and CalcEnv::level (lvl) differ.
//   vsd_test CASES   (case: id skill lvl clvl n (skill base bonus)*n; output as skdruid_test --calc)
// Built by hand against a scratch upstream build (vsd_run.sh), never part of upstream.
#include <gamedata.hpp>
#include <gamedata_load.hpp>
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

using namespace d2d::game;

int main(int argc, char** argv) try {
    if (argc < 2) return 2;
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(fs::path(mpq_dir), patch ? fs::path(patch) : fs::path{}, 0x1234);
    if (!data || !data->patched) { std::printf("FAIL: no patched data\n"); return 1; }
    std::ifstream cases(argv[1]);
    std::string line;
    while (std::getline(cases, line)) {
        std::istringstream fields(line);
        int id = 0, skill_id = 0, lvl = 0, clvl = 0, count = 0;
        fields >> id >> skill_id >> lvl >> clvl >> count;
        std::map<int, int> base, total;
        for (int k = 0; k < count; ++k) {
            int other = 0, b = 0, x = 0;
            fields >> other >> b >> x;
            base[other] = b;
            total[other] = b + x;
        }
        base[skill_id] = total[skill_id] = lvl;
        d2d::rules::CalcEnv env;
        env.clvl = clvl;
        env.base_level = [&](int other) { const auto f = base.find(other); return f == base.end() ? 0 : f->second; };
        env.level = [&](int other) { const auto f = total.find(other); return f == total.end() ? 0 : f->second; };
        env.stat = [](int) { return 0; };
        const auto* skill = data->skills.get(skill_id);
        if (!skill) continue;
        const auto value = [&](const d2d::rules::Calc& calc) { return d2d::rules::eval_calc(data->skills, calc, env, skill_id, lvl); };
        const auto named = [&](const std::string& name) {
            if (name.empty()) return false;
            if (data->skills.by_name.contains(name)) return true;
            for (const auto& [skill_name, row] : data->skills.by_name)
                if (std::ranges::equal(skill_name, name, [](char l, char r) { return std::tolower(static_cast<unsigned char>(l)) == std::tolower(static_cast<unsigned char>(r)); })) return true;
            return false;
        };
        std::printf("%d", id);
        for (std::size_t k = 0; k < 4; ++k) std::printf(" %d", value(skill->calc[k]));
        std::printf(" %d", value(skill->petmax));
        for (std::size_t k = 0; k < 5; ++k) { if (named(skill->sumskill[k])) std::printf(" %d", value(skill->sumsk_calc[k])); else std::printf(" x"); }
        std::printf(" %d %d %d %d", value(skill->delay), value(skill->aurarange), value(skill->auralen), value(skill->perdelay_calc));
        for (std::size_t k = 0; k < 6; ++k) { if (skill->aurastat[k] >= 0) std::printf(" %d", value(skill->aura_calc[k])); else std::printf(" x"); }
        for (std::size_t k = 0; k < 5; ++k) { if (skill->passive_stat[k] >= 0) std::printf(" %d", value(skill->passive_calc[k])); else std::printf(" x"); }
        std::printf("\n");
    }
    return 0;
} catch (const std::exception& error) {
    std::printf("FAIL: %s\n", error.what());
    return 1;
}
