// SPDX-License-Identifier: GPL-3.0-or-later
// decomp10 (stamina): the d2d side of tools/emu/d10_stam.py. d2d's own rules::stamina_drain / rules::stamina_regen
// (components/rules/combat.hpp) with RunDrain as the real loader read it, and the two lines of World::stamina_frame that wrap
// them (drain only outside town, floor at 0, stop running when 0 is reached = ret 0). The wrapping is a transcription
// (a call of the real member needs a whole World); the two rules are the real functions.
//   d10_stam_test classes               C class rundrain
//   d10_stam_test drain < cases         class town armor_speed has_armor pct cur ::: ret after
//   d10_stam_test regen < cases         mode max cur bonus ::: after
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <rules.hpp>
#include <combat.hpp>

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
        for (int c = 0; c < 7; ++c) std::printf("C %d %d\n", c, int(data->run_drain[std::size_t(c)]));
        return 0;
    }
    if (mode == "drain") {
        long long cls, town, speed, has, pct, cur;
        while (std::scanf("%lld %lld %lld %lld %lld %lld ::: %*d %*d", &cls, &town, &speed, &has, &pct, &cur) == 6) {
            long long after = cur;
            int ret = 1;
            if (!town) {
                const int drain = R::stamina_drain(int(data->run_drain[std::size_t(cls)]), has ? int(speed) : 0, int(pct));
                after = std::max<long long>(0, cur - drain);
                ret = after > 0 ? 1 : 0;
            }
            std::printf("%lld %lld %lld %lld %lld %lld ::: %d %lld\n", cls, town, speed, has, pct, cur, ret, after);
        }
        return 0;
    }
    long long m, mx, cur, bonus;
    while (std::scanf("%lld %lld %lld %lld ::: %*d", &m, &mx, &cur, &bonus) == 4)
        std::printf("%lld %lld %lld %lld ::: %lld\n", m, mx, cur, bonus, (long long)R::stamina_regen(cur, mx, int(m), int(bonus)));
    return 0;
} catch (const std::exception& error) {
    std::printf("FAIL: %s\n", error.what());
    return 1;
}
