// SPDX-License-Identifier: GPL-3.0-or-later
// skdruid: the d2d side of the Druid calc differential (tools/emu/skdruid_calc.py).
// Identical to nec_test.cpp (necskill); built by hand against a scratch upstream build.
//   --calc CASES   `id calc1..4 petmax sumsk1..5 delay range len perdelay a1..a6 p1..p5` (x: no stat)
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <skills.hpp>

#include <algorithm>
#include <array>
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

using namespace d2d::game;

int main(int argc, char** argv) try {
    if (argc < 2) return 2;
    const std::string_view mode = argv[1];
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(fs::path(mpq_dir), patch ? fs::path(patch) : fs::path{}, 0x1234);
    if (!data || !data->patched) { std::printf("FAIL: no patched data\n"); return 1; }
    std::ifstream cases(argv[2]);
    std::string line;
    while (std::getline(cases, line)) {
        std::istringstream fields(line);
        int id = 0, skill_id = 0, lvl = 0, clvl = 0, count = 0;
        std::array<int, 4> masteries{};
        fields >> id >> skill_id >> lvl >> clvl >> masteries[0] >> masteries[1] >> masteries[2] >> masteries[3] >> count;
        std::map<int, int> levels;
        for (int k = 0; k < count; ++k) { int other = 0, level = 0; fields >> other >> level; levels[other] = level; }
        levels[skill_id] = lvl;
        d2d::rules::CalcEnv env;
        env.clvl = clvl;
        env.base_level = env.level = [&](int other) { const auto found = levels.find(other); return found == levels.end() ? 0 : found->second; };
        env.stat = [&](int stat) { return stat >= d2d::d2s::kPassiveFireMastery && stat < d2d::d2s::kPassiveFireMastery + 4 ? masteries[std::size_t(stat - d2d::d2s::kPassiveFireMastery)] : 0; };
        const auto* skill = data->skills.get(skill_id);
        if (!skill) continue;
        if (mode == "--calc") {
            const auto value = [&](const d2d::rules::Calc& calc) { return d2d::rules::eval_calc(data->skills, calc, env, skill_id, lvl); };
            const auto named = [&](const std::string& name) {
                if (name.empty()) return false;
                if (data->skills.by_name.contains(name)) return true;
                for (const auto& [skill_name, row] : data->skills.by_name)
                    if (std::ranges::equal(skill_name, name, [](char left, char right) { return std::tolower(static_cast<unsigned char>(left)) == std::tolower(static_cast<unsigned char>(right)); })) return true;
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
    }
    return 0;
} catch (const std::exception& error) {
    std::printf("FAIL: %s\n", error.what());
    return 1;
}
