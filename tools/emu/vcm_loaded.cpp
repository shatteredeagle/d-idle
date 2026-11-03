// SPDX-License-Identifier: GPL-3.0-or-later
// verifycovmissiles: every missile row the REAL d2d loader (load_game_data) holds, with the columns the loader keeps.
// Output, one line per loaded row (map order): name vel vel_lev max_vel accel activate range lev_range size collide_kill pierce
// next_hit next_delay to_hit always_explode hit_func srv_do emin emax etype
#include <gamedata.hpp>
#include <gamedata_load.hpp>

#include <cassert>
#include <cstdio>
#include <cstdlib>
#include <exception>
#include <filesystem>

using namespace d2d::game;

int main() try {
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* home = std::getenv("HOME");
    const fs::path data_dir = mpq_dir ? fs::path(mpq_dir) : fs::path(home ? home : "") / "Workspace/private/diablo2";
    if (!fs::exists(data_dir / "d2data.mpq")) { std::printf("SKIP: no d2data.mpq\n"); return D2D_TEST_SKIPPED; }
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(data_dir, patch ? fs::path(patch) : fs::path{}, 0x1234);
    assert(data && data->patched);
    for (const auto& [name, m] : data->missiles)
        std::printf("%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n", name.c_str(), m.vel, m.vel_lev, m.max_vel,
                    m.accel, m.activate, m.range, m.lev_range, m.size, int(m.collide_kill), int(m.pierce), int(m.next_hit), m.next_delay,
                    int(m.to_hit), int(m.always_explode), m.hit_func, m.srv_do, m.emin, m.emax, m.etype);
    return 0;
} catch (const std::exception& error) {
    std::fprintf(stderr, "vcm_loaded: %s\n", error.what());
    return 1;
}
