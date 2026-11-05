// SPDX-License-Identifier: GPL-3.0-or-later
// covtreasure2: the d2d side of tools/emu/covt_xp.py. Calls d2d's own rules::kill_award (components/rules/combat.hpp) on the
// ratios, shift and max level the real loader read from Experience.txt (load_game_data).
//   covt_xp_test ratios               R level ratio (level 0 = the shift)
//   covt_xp_test award < cases        exp clvl mlvl award   (add_pct 0, one player)
// Built by hand against a scratch upstream build (tools/emu/covt_build.sh), never part of upstream.

#include <combat.hpp>
#include <gamedata.hpp>
#include <gamedata_load.hpp>

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
    if (mode == "ratios") {
        std::printf("R 0 %d\n", data->exp_ratio_shift);
        for (int level = 1; level < 100; ++level)
            std::printf("R %d %d\n", level, std::size_t(level) < data->exp_ratio.size() ? data->exp_ratio[std::size_t(level)] : -1);
        std::printf("max_level %d size %zu\n", data->max_level, data->exp_ratio.size());
        return 0;
    }
    int exp = 0, clvl = 0, mlvl = 0;
    while (std::scanf("%d %d %d %*d", &exp, &clvl, &mlvl) == 3)
        std::printf("%d %d %d %lld\n", exp, clvl, mlvl,
                    static_cast<long long>(R::kill_award(exp, clvl, mlvl, 0, data->exp_ratio, data->exp_ratio_shift, data->max_level)));
    return 0;
} catch (const std::exception& error) {
    std::printf("FAIL: %s\n", error.what());
    return 1;
}
