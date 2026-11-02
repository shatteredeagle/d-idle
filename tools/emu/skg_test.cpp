// SPDX-License-Identifier: GPL-3.0-or-later
// test-links: game
// skground: Meteor's blast area through the REAL Fight::area (our copy of a test; upstream path would be tests/game/).
//   `--blast CASES`: for each `K id kind r cx cy ux uy` line of tools/emu/skg_blast_diff.py gen (milli-subtiles): one hostile
//     size-1 monster with a million life at the unit's float spot, a `meteorcenter` missile whose damage is fixed at 10,
//     Fight::area at the centre's float spot with radius r; prints `D id hit` (1 when the monster lost life).
//     game.exe's side: skg_blast_diff.py game (FUN_0056bad0 on integer subtiles, V run).

#include <ai.hpp>
#include <character.hpp>
#include <character_store.hpp>
#include <combat.hpp>
#include <d2s_items.hpp>
#include <fight.hpp>
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <level_ids.hpp>
#include <rules.hpp>
#include <world.hpp>

#include <algorithm>
#include <array>
#include <cassert>
#include <cmath>
#include <cstddef>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <exception>
#include <filesystem>
#include <fstream>
#include <sstream>
#include <string>
#include <string_view>
#include <utility>
#include <vector>

using namespace d2d::game;

int main(int argc, char** argv) try {
    const std::string_view mode = argc > 1 ? argv[1] : "";
    if (mode != "--blast") { std::printf("SKIP: needs --blast CASES (see tools/emu/skg_blast_diff.py)\n"); return D2D_TEST_SKIPPED; }
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* home = std::getenv("HOME");
    const fs::path data_dir = mpq_dir ? fs::path(mpq_dir) : fs::path(home ? home : "") / "Workspace/private/diablo2";
    if (!fs::exists(data_dir / "d2data.mpq")) { std::printf("SKIP: no d2data.mpq in %s\n", data_dir.string().c_str()); return D2D_TEST_SKIPPED; }
    const char* patch = std::getenv("D2_PATCH_INSTALLER");

    auto data = load_game_data(data_dir, patch ? fs::path(patch) : fs::path{}, 0x1234);
    assert(data && data->town.id == 1 && !data->town.walk.empty());
    if (!data->patched) { std::printf("SKIP: needs the 1.14d data (D2_PATCH_INSTALLER)\n"); return D2D_TEST_SKIPPED; }
    d2d::rules::Rng rng(7);
    constexpr int kSorceress = 1;
    auto made = new_character(*data, kSorceress, "Headless", false, true, rng);
    Character character;
    character.character_class = kSorceress;
    character.name = "Headless";
    character.header = made.header;
    character.stats = made.stats;
    character.items = made.items;
    character.panel = panel_stats(*data, character.header, character.items, character.stats);
    World world(&*data, -1, -1);
    world.enter(character);
    Level open = *data->level(d2d::rules::level_ids::kBloodMoor);
    std::ranges::fill(open.walk, std::uint16_t(0));
    world.level = &open;
    world.fight.enter(&open);
    const int at_x = 1000, at_y = 1000;   // the base subtile of every case
    world.actor->player.x = (float(at_x) + 0.5f) / 5;
    world.actor->player.y = (float(at_y) + 0.5f) / 5;
    std::uint32_t now = 1000 * kTickMs;
    const auto& skill = *data->skills.get(data->skills.by_name.at("Meteor"));
    const auto& row = data->missiles.at("meteorcenter");

    int kind_row = -1;
    for (const char* name : { "maggotegg1", "fallen1" })
        if (const auto found = data->monsters.by_id.find(name); found != data->monsters.by_id.end() && kind_row < 0 && data->monsters.types[found->second].size == 1) kind_row = int(found->second);
    if (kind_row < 0) { std::printf("SKIP: no size-1 row\n"); return D2D_TEST_SKIPPED; }

    world.fight.monsters.clear(); world.fight.missiles.clear(); world.fight.pending.clear();
    world.fight.missile_frame = -1;
    d2d::rules::Rng spawn_rng{ 5 };
    Monster monster = make_monster(*data, kind_row, (float(at_x) + 0.5f) / 5, (float(at_y) + 0.5f) / 5, spawn_rng, 0);
    monster.stats.hit_points = monster.hit_points = 1000000;
    monster.life_frac = 0;
    monster.next_act = ~0u;
    world.fight.add_monster(monster);
    {
        auto& mon = world.fight.monsters[0];
        mon.boss_res = {};
        const auto res = world.fight.target_of(0).res;
        for (std::size_t k = 0; k < mon.boss_res.size() && k < res.size(); ++k) mon.boss_res[k] = -res[k];
    }
    Missile blast = world.fight.launch(row, skill, 1, (float(at_x) + 0.5f) / 5, (float(at_y) + 0.5f) / 5, 0, 0, 60, now);
    blast.fixed = 10 << 8;

    std::ifstream cases(argc > 2 ? argv[2] : "");
    std::string line;
    long n = 0, hits = 0;
    while (std::getline(cases, line)) {
        std::istringstream fields(line);
        std::string tag, kind;
        int id = 0, r = 0, cx = 0, cy = 0, ux = 0, uy = 0;
        if (!(fields >> tag >> id >> kind >> r >> cx >> cy >> ux >> uy) || tag != "K") continue;
        auto& mon = world.fight.monsters[0];
        mon.hit_points = 1000000;
        mon.next_act = ~0u;
        mon.unit.x = (float(at_x) + float(ux) / 1000.0f) / 5;
        mon.unit.y = (float(at_y) + float(uy) / 1000.0f) / 5;
        const float centre_x = (float(at_x) + float(cx) / 1000.0f) / 5;
        const float centre_y = (float(at_y) + float(cy) / 1000.0f) / 5;
        now += kTickMs;
        world.fight.area(blast, centre_x, centre_y, r, now);
        const int hit = world.fight.monsters[0].hit_points < 1000000 ? 1 : 0;
        std::printf("D %d %d\n", id, hit);
        ++n; hits += hit;
    }
    std::fprintf(stderr, "d2d: %ld cases, %ld hits\n", n, hits);
    return 0;
} catch (const std::exception& error) {
    std::fprintf(stderr, "%s\n", error.what());
    return 1;
}
