// SPDX-License-Identifier: GPL-3.0-or-later
// covmonskills2: the d2d side of tools/emu/cmk_slots.py. For every MonStats row d2d parsed (load_game_data on the real tables), prints
// the Skill1..8 slots as d2d holds them and d2d's own rules::monster_skill_level at the three difficulties, plus the map from
// game.exe's MonStats id (hcIdx) to d2d's row. Built by hand against a scratch upstream build (tools/emu/cmk_build.sh); never part of upstream.
//   cmk_slots_test --bin     id row                 (game.exe unit class id -> d2d row)
//   cmk_slots_test --slots   row id slot name mode lvl skill_id L0 L1 L2   (one line per non-empty slot; skill_id -1 when the name is unknown)

#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <monsters.hpp>

#include <cstdio>
#include <cstdlib>
#include <exception>
#include <filesystem>
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
    namespace R = d2d::rules;
    if (mode == "--bin") {
        for (std::size_t id = 0; id < data->mon_bin.size(); ++id) std::printf("bin %zu %u\n", id, unsigned(data->mon_bin[id]));
        return 0;
    }
    const auto& types = data->monsters.types;
    for (std::size_t row = 0; row < types.size(); ++row) {
        const auto& type = types[row];
        for (std::size_t slot = 0; slot < type.skill.size(); ++slot) {
            if (type.skill[slot].empty()) continue;
            const auto found = data->skills.by_name.find(type.skill[slot]);
            const int sid = found == data->skills.by_name.end() ? -1 : int(found->second);
            std::string shown = type.skill[slot];
            for (auto& c : shown) if (c == ' ') c = '_';
            std::printf("slot %zu %s %zu %s %s %d %d %d %d %d\n", row, type.id.c_str(), slot + 1, shown.c_str(),
                        type.sk_mode[slot].empty() ? "-" : type.sk_mode[slot].c_str(), type.sk_lvl[slot], sid,
                        R::monster_skill_level(type.sk_lvl[slot], 0), R::monster_skill_level(type.sk_lvl[slot], 1), R::monster_skill_level(type.sk_lvl[slot], 2));
        }
    }
    return 0;
} catch (const std::exception& error) {
    std::printf("FAIL: %s\n", error.what());
    return 1;
}
