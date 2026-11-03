// SPDX-License-Identifier: GPL-3.0-or-later
// test-links: game
// verifybundle probe (scratch, not for upstream): the d2d side of vb_ground_game.py.
//   --hit ROW SKILL FRAMES CLS...  `V cls size dx dy hits` (dx, dy = unit - missile subtiles, -4..4)
//   --burst ROW SKILL TRIALS       `B trial lost0 lost1 lost2` (three monsters in one burst, resistances zeroed)
#include <ai.hpp>
#include <character.hpp>
#include <character_store.hpp>
#include <d2s_items.hpp>
#include <fight.hpp>
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <level_ids.hpp>
#include <rules.hpp>
#include <world.hpp>

#include <algorithm>
#include <array>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <string>
#include <string_view>
#include <utility>
#include <vector>

using namespace d2d::game;

int main(int argc, char** argv) {
    const std::string_view mode = argc > 1 ? argv[1] : "";
    if (mode != "--hit" && mode != "--burst") return D2D_TEST_SKIPPED;
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(fs::path(mpq_dir ? mpq_dir : ""), patch ? fs::path(patch) : fs::path{}, 0x1234);
    if (!data || !data->patched) return D2D_TEST_SKIPPED;
    d2d::rules::Rng rng(31);
    auto made = new_character(*data, 1, "Probe", false, true, rng);
    Character character;
    character.character_class = 1; character.name = "Probe";
    character.header = made.header; character.stats = made.stats; character.items = made.items;
    character.panel = panel_stats(*data, character.header, character.items, character.stats);
    World world(&*data, -1, -1);
    world.enter(character);
    Level open = *data->level(d2d::rules::level_ids::kBloodMoor);
    std::ranges::fill(open.walk, std::uint16_t(0));
    world.level = &open;
    world.fight.enter(&open);
    int at_x = -1, at_y = -1;
    for (int y = 40; y < 400 && at_x < 0; ++y)
        for (int x = 40; x < 400 && at_x < 0; ++x)
            if (!missile_rooms(*data, open, float(x) + 0.5f, float(y) + 0.5f).empty() && !missile_rooms(*data, open, float(x + 6) + 0.5f, float(y + 6) + 0.5f).empty()
                && !missile_rooms(*data, open, float(x - 6) + 0.5f, float(y - 6) + 0.5f).empty())
                at_x = x * 5 + 2, at_y = y * 5 + 2;
    world.actor->player.x = float(at_x + 60) / 5; world.actor->player.y = float(at_y + 60) / 5;
    const Crowd crowd;
    std::uint32_t now = 2000 * kTickMs;
    const std::string row_name = argv[2];
    const auto found = data->missiles.find(row_name);
    if (found == data->missiles.end()) { std::printf("NOROW %s\n", row_name.c_str()); return 0; }
    const auto& row = found->second;
    const auto& skill = *data->skills.get(data->skills.by_name.at(argv[3]));
    const auto place = [&](int cls, int sx, int sy) {
        d2d::rules::Rng spawn_rng{ 9 };
        Monster monster = make_monster(*data, cls, (float(sx) + 0.5f) / 5, (float(sy) + 0.5f) / 5, spawn_rng, 0);
        monster.stats.hit_points = monster.hit_points = 1000000;
        monster.life_frac = 0; monster.next_act = ~0u;
        world.fight.add_monster(monster);
        auto& mon = world.fight.monsters.back();
        const auto res = world.fight.target_of(world.fight.monsters.size() - 1).res;
        for (std::size_t k = 0; k < mon.boss_res.size() && k < res.size(); ++k) mon.boss_res[k] = -res[k];
    };
    const auto reset = [&] { world.fight.monsters.clear(); world.fight.missiles.clear(); world.fight.pending.clear(); world.fight.missile_frame = -1; };
    if (mode == "--hit") {
        const int frames = std::atoi(argv[4]);
        for (int k = 5; k < argc; ++k) {
            const int cls = std::atoi(argv[k]);
            for (int a = -4; a <= 4; ++a)
                for (int b = -4; b <= 4; ++b) {
                    reset();
                    place(cls, at_x, at_y);
                    world.fight.launch(row, skill, 1, (float(at_x + a) + 0.5f) / 5, (float(at_y + b) + 0.5f) / 5, 0, 0, frames + 1, now);
                    for (auto& missile : world.fight.pending) missile.fixed = 10 << 8;
                    int hits = 0;
                    for (int frame = 0; frame < frames; ++frame) {
                        now += kTickMs;
                        const int before = world.fight.monsters[0].hit_points;
                        world.fight.monsters[0].next_act = ~0u;
                        world.fight.world(true, now, 0.04f, crowd);
                        hits += (before - world.fight.monsters[0].hit_points) / 10;
                    }
                    std::printf("V %d %d %d %d %d\n", cls, data->monsters.types[std::size_t(cls)].size, -a, -b, hits);
                    now += 30 * kTickMs;
                }
        }
        return 0;
    }
    const int trials = std::atoi(argv[4]);
    const int cls = data->monsters.by_id.at("fallen1");
    for (int t = 0; t < trials; ++t) {
        reset();
        place(cls, at_x, at_y); place(cls, at_x + 1, at_y + 1); place(cls, at_x - 1, at_y + 1);
        world.fight.launch(row, skill, 10, (float(at_x) + 0.5f) / 5, (float(at_y) + 0.5f) / 5, 0, 0, 1, now);
        std::array<int, 3> lost{};
        for (int frame = 0; frame < 4; ++frame) {
            std::array<int, 3> before{};
            for (std::size_t i = 0; i < 3; ++i) { before[i] = world.fight.monsters[i].hit_points; world.fight.monsters[i].next_act = ~0u; }
            now += kTickMs;
            world.fight.world(true, now, 0.04f, crowd);
            for (std::size_t i = 0; i < 3; ++i) lost[i] += before[i] - world.fight.monsters[i].hit_points;
        }
        std::printf("B %d %d %d %d\n", t, lost[0], lost[1], lost[2]);
        now += 50 * kTickMs;
    }
    return 0;
}
