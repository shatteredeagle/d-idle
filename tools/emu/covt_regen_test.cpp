// SPDX-License-Identifier: GPL-3.0-or-later
// covtreasure2: the d2d side of tools/emu/covt_regen.py. d2d's own rules::mana_per_frame (components/rules/rules.hpp) with
// the class's ManaRegen as the real loader read it (load_game_data), then the cap Fight::apply_regen
// (components/game/fight/setup.cpp) puts on a frame's gain: mana = max(mana, min(max mana, mana + gain)). The cap is a
// transcription of that line (a one-frame call of the real member needs a whole Fight), the rule itself is the real function.
//   covt_regen_test classes                C class manaregen
//   covt_regen_test regen < cases          class maxmana mana bonus flat ::: delta
// Built by hand against a scratch upstream build (tools/emu/covt_build.sh).

#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <rules.hpp>

#include <algorithm>
#include <cstdio>
#include <cstdlib>
#include <exception>
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
    if (mode == "classes") {
        for (int c = 0; c < 7; ++c) std::printf("C %d %d\n", c, data->class_gains[std::size_t(c)].mana_regen);
        return 0;
    }
    long long cls = 0, max_mana = 0, mana = 0, bonus = 0, flat = 0;
    while (std::scanf("%lld %lld %lld %lld %lld ::: %*d", &cls, &max_mana, &mana, &bonus, &flat) == 5) {
        const long long gain = R::mana_per_frame(max_mana, data->class_gains[std::size_t(cls)].mana_regen, int(bonus), int(flat));
        const long long after = std::max(mana, std::min(max_mana, mana + gain));
        std::printf("%lld %lld %lld %lld %lld ::: %lld\n", cls, max_mana, mana, bonus, flat, after - mana);
    }
    return 0;
} catch (const std::exception& error) {
    std::printf("FAIL: %s\n", error.what());
    return 1;
}
