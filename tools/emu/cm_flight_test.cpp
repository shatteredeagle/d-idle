// SPDX-License-Identifier: GPL-3.0-or-later
// covmissiles: `--flight CASES`: d2d's missile row values for (name, level) cases, from the REAL loader and the REAL rules header.
// Each input line `name level`; output `name level velocity range max_vel accel activate`, with
// velocity = rules::missile_velocity(Vel, VelLev, level), range = rules::missile_range(Range, LevRange, level),
// max_vel = MaxVel << 8 (what rules::MissileFlight keeps), accel, activate (the loaded MissileInfo). A row d2d does not load prints
// `name level -` (GameData::missiles holds only the rows skills, monsters, traps and bosses use: gamedata_load.cpp).
// With `--flags NAMES` prints `name to_hit collide_kill pierce size next_hit return_fire` of each loaded row (one per line of NAMES).

#include <combat.hpp>
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <missiles.hpp>
#include <rules.hpp>

#include <cassert>
#include <cstdio>
#include <cstdlib>
#include <exception>
#include <filesystem>
#include <fstream>
#include <sstream>
#include <string>
#include <string_view>

using namespace d2d::game;

int main(int argc, char** argv) try {
    if (argc < 3 || (std::string_view(argv[1]) != "--flight" && std::string_view(argv[1]) != "--flags")) {
        std::printf("SKIP: an oracle mode, run `cm_flight_test --flight CASES` (see tools/emu/cm_flight.py)\n");
        return D2D_TEST_SKIPPED;
    }
    const bool flags = std::string_view(argv[1]) == "--flags";
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* home = std::getenv("HOME");
    const fs::path data_dir = mpq_dir ? fs::path(mpq_dir) : fs::path(home ? home : "") / "Workspace/private/diablo2";
    if (!fs::exists(data_dir / "d2data.mpq")) { std::printf("SKIP: no d2data.mpq in %s\n", data_dir.string().c_str()); return D2D_TEST_SKIPPED; }
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(data_dir, patch ? fs::path(patch) : fs::path{}, 0x1234);
    assert(data && data->patched);

    std::ifstream cases(argv[2]);
    std::string line;
    while (std::getline(cases, line)) {
        const auto tab = line.find('\t');
        const std::string name = line.substr(0, tab);
        const int level = tab == std::string::npos ? 0 : std::atoi(line.c_str() + tab + 1);
        const auto found = data->missiles.find(name);
        if (flags) {
            if (found == data->missiles.end()) { std::printf("%s\t0\t-\n", name.c_str()); continue; }
            const auto& m = found->second;
            std::printf("%s\t0\t%d\t%d\t%d\t%d\t%d\t%d\n", name.c_str(), int(m.to_hit), int(m.collide_kill), int(m.pierce), m.size, int(m.next_hit), int(m.return_fire));
            continue;
        }
        if (found == data->missiles.end()) { std::printf("%s\t%d\t-\n", name.c_str(), level); continue; }
        const auto& m = found->second;
        std::printf("%s\t%d\t%d\t%d\t%d\t%d\t%d\n", name.c_str(), level, d2d::rules::missile_velocity(m.vel, m.vel_lev, level),
                    d2d::rules::missile_range(m.range, m.lev_range, level), m.max_vel << 8, m.accel, m.activate);
    }
    return 0;
} catch (const std::exception& error) {
    std::fprintf(stderr, "cm_flight_test: %s\n", error.what());
    return 1;
}
