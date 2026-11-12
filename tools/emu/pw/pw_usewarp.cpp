// SPDX-License-Identifier: GPL-3.0-or-later
// prwarp probe (not for upstream; written by the verifier prverify3): the FULL warp path. For every warp of every level FROM..TO that d2d builds
// for SEED, a World walks the player onto the warp's unit, sets take_warp and calls World::use_warp() (which calls
// World::arrive). Writes, per arrival, into OUTDIR:
//   pre.tsv   vb2_tele.py cases: 200 x 200 subtile window of the destination's collision round the back unit, taken
//             just BEFORE use_warp (blocked = mask 0x0c09 or off the level)
//   post.tsv  the same window taken AFTER use_warp (objects, doors, rooms the arrival made)
//   final.tsv the hero's subtile after use_warp in window subtiles (`id 1 x y`), as vb2_tele.py answers
//   meta.tsv  seed from to slot unit subtile, level reached, pre-spot free?, hero moved by arrive's second nearest_free?
#include <character.hpp>
#include <character_store.hpp>
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <level_ids.hpp>
#include <world.hpp>

#include <cmath>
#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <string>
#include <vector>

using namespace d2d::game;
namespace fs = std::filesystem;

static int sub(float cells) { return int(std::floor(cells * 5)); }

static void window(std::ofstream& out, const Level& level, int cid, int ux, int uy) {
    constexpr int kGrid = 200, kHalf = 100;
    std::vector<std::pair<int, int>> cells;
    for (int row = 0; row < kGrid; ++row)
        for (int col = 0; col < kGrid; ++col)
            if (level.blocked((float(ux - kHalf + col) + 0.5f) / 5, (float(uy - kHalf + row) + 0.5f) / 5, 0x0c09)) cells.push_back({ col, row });
    out << cid << " " << kHalf << " " << kHalf << " " << cells.size();
    for (const auto& cell : cells) out << " " << cell.first << " " << cell.second;
    out << "\n";
}

int main(int argc, char** argv) try {
    if (argc < 5) { std::fprintf(stderr, "usage: SEED FROM TO OUTDIR\n"); return 2; }
    const unsigned seed = unsigned(std::strtoul(argv[1], nullptr, 10));
    const int first = std::atoi(argv[2]), last = std::atoi(argv[3]);
    const fs::path out = argv[4];
    fs::create_directories(out);
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(fs::path(mpq_dir ? mpq_dir : "/tmp/ug-emu"), patch ? fs::path(patch) : fs::path{}, seed);
    if (!data || !data->patched) { std::printf("FAIL: no patched data\n"); return 1; }
    d2d::rules::Rng rng(7);
    constexpr int kSorceress = 1;
    auto made = new_character(*data, kSorceress, "Probe", false, true, rng);
    Character character;
    character.character_class = kSorceress;
    character.name = "Probe";
    character.header = made.header;
    character.stats = made.stats;
    character.items = made.items;
    character.panel = panel_stats(*data, character.header, character.items, character.stats);
    World world(&*data, -1, -1);
    world.enter(character);
    std::ofstream pre(out / "pre.tsv"), post(out / "post.tsv"), fin(out / "final.tsv"), meta(out / "meta.tsv");
    meta << "cid\tseed\tfrom\tto\tslot\tunit_x\tunit_y\treached\tunit_free_pre\tfinal_dx\tfinal_dy\n";
    int cid = 0, skipped = 0;
    for (int from = first; from <= last; ++from) {
        const Level* source = data->level(from);
        if (!source || source->ds1.width() == 0) continue;
        for (std::size_t index = 0; index < source->warps.size(); ++index) {
            const Level::Warp warp = source->warps[index];
            const Level* dest = data->level(warp.destination);
            if (!dest || dest->ds1.width() == 0) { ++skipped; continue; }
            auto back = std::ranges::find_if(dest->warps, [&](const Level::Warp& other) { return other.destination == from && other.pair == warp.pair; });
            if (back == dest->warps.end())
                for (auto it = dest->warps.begin(); it != dest->warps.end(); ++it)
                    if (it->destination == from && (back == dest->warps.end() || it->slot < back->slot)) back = it;
            if (back == dest->warps.end()) { ++skipped; continue; }
            // onto the source level, standing on the warp's unit
            if (world.level != source) world.arrive(source, warp.unit_x, warp.unit_y, "probe");
            world.actor->player.x = warp.unit_x; world.actor->player.y = warp.unit_y;
            world.actor->player.walking = false; world.actor->player.path.clear();
            const int ux = sub(back->unit_x), uy = sub(back->unit_y);
            window(pre, *dest, cid, ux, uy);
            const bool unit_free = !dest->unit_blocked(back->unit_x, back->unit_y);
            world.actor->take_warp = int(index);
            world.use_warp();
            const bool reached = world.level == dest;
            window(post, *dest, cid, ux, uy);
            const int hx = sub(world.actor->player.x), hy = sub(world.actor->player.y);
            if (reached) fin << cid << " 1 " << hx - ux + 100 << " " << hy - uy + 100 << "\n"; else fin << cid << " 0 - -\n";
            meta << cid << "\t" << seed << "\t" << from << "\t" << warp.destination << "\t" << back->slot << "\t" << ux << "\t" << uy << "\t" << reached
                 << "\t" << unit_free << "\t" << hx - ux << "\t" << hy - uy << "\n";
            ++cid;
        }
    }
    std::printf("seed %u levels %d..%d: %d warp arrivals, %d skipped (destination not built or no way back)\n", seed, first, last, cid, skipped);
    return 0;
} catch (const std::exception& error) {
    std::fprintf(stderr, "FAIL: %s\n", error.what());
    return 1;
}
