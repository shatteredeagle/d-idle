// SPDX-License-Identifier: GPL-3.0-or-later
// decomp12 scratch probe (not for upstream): d2d's REAL warp arrival pieces on upstream main.
//   d12_arrive_probe synth CASES            vb2_tele.py cases (id x y n {bx by}...): `id found x y` for
//                                           A = Level::nearest_free as World::arrive calls it (centre of the unit subtile)
//                                           B = Level::warp_spot (the proven FUN_0064e7b0 rule)
//   d12_arrive_probe levels SEED OUTDIR     every warp of every level that loads for SEED: writes
//                                           OUTDIR/cases.tsv (a 200 x 200 subtile window round the BACK warp's unit, blocked =
//                                           mask 0x0c09 or off the level), OUTDIR/A.tsv, OUTDIR/B.tsv (answers in window
//                                           subtiles) and OUTDIR/trigger.tsv (the click-box measure of the arrival).
// The click measure uses World::warp_clicked and World::warp_close (the real functions): from the arrival spot and from the
// spot after the ExitWalk, 16 directions x distances 2 4 8 12 16 20 subtiles, how many clicks are inside the return warp's
// Select box (a fresh Move there sets take_warp in World::walk_to) and whether the hero is within reach (< 5) of the unit.
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <level_ids.hpp>
#include <world.hpp>

#include <cmath>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <set>
#include <sstream>
#include <string>

using namespace d2d::game;
namespace fs = std::filesystem;

static auto load(unsigned seed) {
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    return load_game_data(fs::path(mpq_dir ? mpq_dir : "/tmp/ug-emu"), patch ? fs::path(patch) : fs::path{}, seed);
}

static int sub(float cells) { return int(std::floor(cells * 5)); }

static int synth(const char* path) {
    auto data = load(0x52);
    if (!data || !data->patched) { std::printf("FAIL: no patched data\n"); return 1; }
    constexpr int kGrid = 200, kOff = 30;
    Level field = *data->level(d2d::rules::level_ids::kBloodMoor);
    const int width = field.ds1.width() * 5, height = field.ds1.height() * 5;
    std::ifstream cases(path);
    std::string line;
    std::ofstream fa("A.tsv"), fb("B.tsv");
    while (std::getline(cases, line)) {
        std::istringstream in(line);
        int id = 0, x = 0, y = 0, n = 0;
        if (!(in >> id >> x >> y >> n)) continue;
        field.walk.assign(std::size_t(width) * std::size_t(height), std::uint16_t(1));
        for (int row = 0; row < kGrid; ++row)
            for (int col = 0; col < kGrid; ++col) field.walk[std::size_t(kOff + row) * std::size_t(width) + std::size_t(kOff + col)] = 0;
        for (int i = 0; i < n; ++i) { int bx = 0, by = 0; in >> bx >> by; field.walk[std::size_t(kOff + by) * std::size_t(width) + std::size_t(kOff + bx)] = 1; }
        const auto a = field.nearest_free((float(kOff + x) + 0.5f) / 5, (float(kOff + y) + 0.5f) / 5);
        const int ax = sub(a.first) - kOff, ay = sub(a.second) - kOff;
        const bool a_ok = ax >= 0 && ay >= 0 && ax < kGrid && ay < kGrid && !field.unit_blocked(a.first, a.second);
        if (a_ok) fa << id << " 1 " << ax << " " << ay << "\n"; else fa << id << " 0 - -\n";
        const auto b = field.warp_spot(kOff + x, kOff + y);
        if (b) fb << id << " 1 " << b->first - kOff << " " << b->second - kOff << "\n"; else fb << id << " 0 - -\n";
    }
    return 0;
}

static int levels(unsigned seed, const fs::path& out) {
    auto data = load(seed);
    if (!data || !data->patched) { std::printf("FAIL: no patched data\n"); return 1; }
    fs::create_directories(out);
    std::ofstream cases(out / "cases.tsv"), fa(out / "A.tsv"), fb(out / "B.tsv"), fc(out / "C.tsv"), trig(out / "trigger.tsv");
    trig << "seed\tfrom\tto\tslot\tunit_x\tunit_y\tfree_at_unit\tA_dx\tA_dy\tB_dx\tB_dy\texit_dx\texit_dy\tclose_A\tclose_exit\tin_box_A\tin_box_exit\tclicks_in_box_A\tclicks_in_box_exit\tclicks\n";
    constexpr int kGrid = 200, kHalf = 100;
    int cid = 0;
    for (int from = 1; from <= 39; ++from) {
        const Level* source = data->level(from);
        if (!source) continue;
        for (const auto& warp : source->warps) {
            const Level* dest = data->level(warp.destination);
            if (!dest || dest->ds1.width() == 0) continue;
            auto back = std::ranges::find_if(dest->warps, [&](const Level::Warp& other) { return other.destination == from && other.pair == warp.pair; });
            if (back == dest->warps.end())
                for (auto it = dest->warps.begin(); it != dest->warps.end(); ++it)
                    if (it->destination == from && (back == dest->warps.end() || it->slot < back->slot)) back = it;
            if (back == dest->warps.end()) continue;
            const int ux = sub(back->unit_x), uy = sub(back->unit_y);
            const int ox = ux - kHalf, oy = uy - kHalf;
            std::set<std::pair<int, int>> cells;
            for (int row = 0; row < kGrid; ++row)
                for (int col = 0; col < kGrid; ++col)
                    if (dest->blocked((float(ox + col) + 0.5f) / 5, (float(oy + row) + 0.5f) / 5, 0x0c09)) cells.insert({ col, row });
            cases << cid << " " << kHalf << " " << kHalf << " " << cells.size();
            for (const auto& c : cells) cases << " " << c.first << " " << c.second;
            cases << "\n";
            const auto a = dest->nearest_free(back->unit_x, back->unit_y);
            const int ax = sub(a.first), ay = sub(a.second);
            fa << cid << " 1 " << ax - ox << " " << ay - oy << "\n";
            const auto b = dest->warp_spot(ux, uy);
            if (b) fb << cid << " 1 " << b->first - ox << " " << b->second - oy << "\n"; else fb << cid << " 0 - -\n";
            {   // C = World::warp_arrival (the fix candidate); absent in the unfixed tree: build with -DD12_FIXED
#ifdef D12_FIXED
                const auto c = World::warp_arrival(*dest, back->unit_x, back->unit_y);
                fc << cid << " 1 " << sub(c.first) - ox << " " << sub(c.second) - oy << "\n";
#else
                fc << cid << " 0 - -\n";
#endif
            }
            // the walk of ExitWalk (cells) from the arrival spot, then the trigger measure
            const float exit_x = a.first + back->exit_x, exit_y = a.second + back->exit_y;
            const bool free_unit = !dest->unit_blocked(back->unit_x, back->unit_y);
            int in_a = 0, in_e = 0, total = 0;
            for (int dir = 0; dir < 16; ++dir)
                for (int dist : { 2, 4, 8, 12, 16, 20 }) {
                    const float dx = std::cos(float(dir) * 3.14159265f / 8) * float(dist) / 5, dy = std::sin(float(dir) * 3.14159265f / 8) * float(dist) / 5;
                    in_a += World::warp_clicked(*back, a.first + dx, a.second + dy);
                    in_e += World::warp_clicked(*back, exit_x + dx, exit_y + dy);
                    ++total;
                }
            trig << seed << "\t" << from << "\t" << warp.destination << "\t" << back->slot << "\t" << ux << "\t" << uy << "\t" << free_unit
                 << "\t" << ax - ux << "\t" << ay - uy << "\t" << (b ? b->first - ux : 99) << "\t" << (b ? b->second - uy : 99)
                 << "\t" << sub(exit_x) - ux << "\t" << sub(exit_y) - uy
                 << "\t" << World::warp_close(*back, a.first, a.second) << "\t" << World::warp_close(*back, exit_x, exit_y)
                 << "\t" << World::warp_clicked(*back, a.first, a.second) << "\t" << World::warp_clicked(*back, exit_x, exit_y)
                 << "\t" << in_a << "\t" << in_e << "\t" << total << "\n";
            ++cid;
        }
    }
    std::printf("seed %u: %d warp arrivals\n", seed, cid);
    return 0;
}

int main(int argc, char** argv) {
    if (argc >= 3 && std::string(argv[1]) == "synth") return synth(argv[2]);
    if (argc >= 4 && std::string(argv[1]) == "levels") return levels(unsigned(std::strtoul(argv[2], nullptr, 10)), argv[3]);
    return 2;
}
