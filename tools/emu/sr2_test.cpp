// SPDX-License-Identifier: GPL-3.0-or-later
// test-links: game
// SORCREST2: D2 of docs/research/sor-rest-trace.md on the real d2d Fight: a burst (Fire Ball row `fireball`, 47) takes
// in a monster standing BETWEEN subtile centres. One standing missile at the centre of subtile (0, 0); one hostile
// monster of each size 1..3 at the float subtile position (dx + f/5, dy + g/5), dx, dy in -7..7, f, g in 0..4.
//   usage: test_sr2 ROW SKILL LEVEL [RADIUS=4]
// Per case `R size dx dy f g d2d game`: d2d = the monster lost life; game = game.exe's rule on the unit's INTEGER subtile
// (dx, dy): dx^2 + dy^2 <= r^2 (V run: tools/emu/sk_rest_burst.py, 3,375 cases, 0 differ). Last line `# ... differ N`.

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
#include <string>
#include <string_view>
#include <utility>

using namespace d2d::game;

int main(int argc, char** argv) try {
    if (argc < 4) {
        std::printf("SKIP: usage ROW SKILL LEVEL [RADIUS] (see tools/emu/sr2_test.cpp)\n");
        return D2D_TEST_SKIPPED;
    }
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
    int at_x = -1, at_y = -1;
    for (int y = 20; y < 400 && at_x < 0; ++y)
        for (int x = 20; x < 400 && at_x < 0; ++x)
            if (std::ranges::all_of(std::array<std::pair<int, int>, 5>{ { { 0, 0 }, { 8, 0 }, { -8, 0 }, { 0, 8 }, { 0, -8 } } },
                                    [&](auto off) { return !missile_rooms(*data, open, float(x + off.first) + 0.5f, float(y + off.second) + 0.5f).empty(); }))
                at_x = x * 5 + 2, at_y = y * 5 + 2;
    assert(at_x >= 0);
    world.actor->player.x = (float(at_x) + 0.5f) / 5;
    world.actor->player.y = (float(at_y) + 0.5f) / 5;
    const Crowd crowd;
    std::uint32_t now = 1000 * kTickMs;

    const std::string row_name = argv[1];
    const auto& skill = *data->skills.get(std::atoi(argv[2]));
    const auto& missile_row = data->missiles.at(row_name);
    const int level = std::atoi(argv[3]);
    const int r = argc > 4 ? std::atoi(argv[4]) : 4;

    std::array<int, 4> kind{ -1, -1, -1, -1 };
    for (const char* name : { "maggotegg1", "fallen1", "zombie1", "bighead1", "brute2", "wendigo1", "flyingscimitar" })
        if (const auto found = data->monsters.by_id.find(name); found != data->monsters.by_id.end())
            if (const int size = data->monsters.types[found->second].size; size >= 1 && size <= 3 && kind[std::size_t(size)] < 0) kind[std::size_t(size)] = int(found->second);
    for (std::size_t row = 0; row < data->monsters.types.size(); ++row)
        if (const int size = data->monsters.types[row].size; size >= 1 && size <= 3 && kind[std::size_t(size)] < 0) kind[std::size_t(size)] = int(row);

    long cases = 0, differ = 0, in_d2d = 0;
    for (int size = 1; size <= 3; ++size) {
        if (kind[std::size_t(size)] < 0) { std::printf("SKIP: no size %d row\n", size); return D2D_TEST_SKIPPED; }
        for (int dx = -7; dx <= 7; ++dx)
            for (int dy = -7; dy <= 7; ++dy)
                for (int f = 0; f < 5; ++f)
                    for (int g = 0; g < 5; ++g) {
                        world.fight.monsters.clear();
                        world.fight.missiles.clear();
                        world.fight.pending.clear();
                        world.fight.missile_frame = -1;
                        d2d::rules::Rng spawn_rng{ 5 };
                        Monster monster = make_monster(*data, std::size_t(kind[std::size_t(size)]), (float(at_x + dx) + float(f) / 5) / 5, (float(at_y + dy) + float(g) / 5) / 5, spawn_rng, 0);
                        monster.stats.hit_points = monster.hit_points = 1000000;
                        monster.life_frac = 0;
                        monster.next_act = ~0u;
                        world.fight.add_monster(monster);
                        auto& mon = world.fight.monsters.back();
                        mon.boss_res = {};
                        const auto res = world.fight.target_of(0).res;
                        for (std::size_t k = 0; k < mon.boss_res.size() && k < res.size(); ++k) mon.boss_res[k] = -res[k];
                        world.fight.launch(missile_row, skill, level, (float(at_x) + 0.5f) / 5, (float(at_y) + 0.5f) / 5, 0, 0, 1, now);
                        for (auto& missile : world.fight.pending) missile.fixed = 10 << 8;
                        const int before = world.fight.monsters[0].hit_points;
                        for (int frame = 1; frame <= 3; ++frame) {
                            now += kTickMs;
                            world.fight.monsters[0].next_act = ~0u;
                            world.fight.world(true, now, 0.04f, crowd);
                        }
                        const int d2d = before != world.fight.monsters[0].hit_points ? 1 : 0;
                        const int game = dx * dx + dy * dy <= r * r ? 1 : 0;
                        ++cases;
                        in_d2d += d2d;
                        differ += d2d != game;
                        std::printf("R %d %d %d %d %d %d %d\n", size, dx, dy, f, g, d2d, game);
                        now += 20 * kTickMs;
                    }
    }
    std::printf("# radius %d cases %ld d2d_hits %ld differ %ld\n", r, cases, in_d2d, differ);
    return 0;
} catch (const std::exception& error) {
    std::fprintf(stderr, "%s\n", error.what());
    return 1;
}
