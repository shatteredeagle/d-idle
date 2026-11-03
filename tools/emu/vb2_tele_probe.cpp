// SPDX-License-Identifier: GPL-3.0-or-later
// verifyb2 scratch probe (not for upstream): the REAL Level::warp_spot of a d2d build on the
// cases of vb2_tele.py (a 200 x 200 subtile grid inside a copy of a real level, walls round it,
// the case's subtiles given collision bit 1). `id found x y` in grid subtiles.
//   vb2_tele_probe CASES
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <level_ids.hpp>

#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <sstream>
#include <string>

using namespace d2d::game;

int main(int argc, char** argv) {
    if (argc < 2) return 2;
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(fs::path(mpq_dir ? mpq_dir : "/tmp/ug-emu"), patch ? fs::path(patch) : fs::path{}, 0x52);
    if (!data || !data->patched) { std::printf("FAIL: no patched data\n"); return 1; }
    constexpr int kGrid = 200, kOff = 30;
    Level field = *data->level(d2d::rules::level_ids::kBloodMoor);
    const int width = field.ds1.width() * 5, height = field.ds1.height() * 5;
    if (width < kOff + kGrid + 1 || height < kOff + kGrid + 1) { std::printf("FAIL: level %d x %d too small\n", width, height); return 1; }
    std::ifstream cases(argv[1]);
    std::string line;
    while (std::getline(cases, line)) {
        std::istringstream in(line);
        int id = 0, x = 0, y = 0, n = 0;
        if (!(in >> id >> x >> y >> n)) continue;
        field.walk.assign(std::size_t(width) * std::size_t(height), std::uint16_t(1));
        for (int row = 0; row < kGrid; ++row)
            for (int col = 0; col < kGrid; ++col) field.walk[std::size_t(kOff + row) * std::size_t(width) + std::size_t(kOff + col)] = 0;
        for (int i = 0; i < n; ++i) {
            int bx = 0, by = 0;
            in >> bx >> by;
            field.walk[std::size_t(kOff + by) * std::size_t(width) + std::size_t(kOff + bx)] = 1;
        }
        const auto spot = field.warp_spot(kOff + x, kOff + y);
        if (spot) std::printf("%d 1 %d %d\n", id, spot->first - kOff, spot->second - kOff);
        else std::printf("%d 0 - -\n", id);
    }
    return 0;
}
