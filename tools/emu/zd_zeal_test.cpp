// SPDX-License-Identifier: GPL-3.0-or-later
// test-links: game
// ZEALDIFF (decomp13, TASK-330 d2d side): drive the REAL d2d Fight with a Paladin Zeal swing on a passive dummy and print, per tick,
// the strike edge (actor->pstruck rising) and the mode edges. Usage: zd_zeal_test SKILL LEVEL [IAS [WSM [TICKS [WEAPON]]]]
//   WEAPON = an item code put in the main hand before the character enters (picks the swing animation: sword class, thrust, staff ...).
//   prints `E <tick> <what> <mode> <pstruck> <kicks_left>`; `W <weapon code> <prate>`; `A <frames> <action ms>`; `S gap N`; `M <mana before> <mana after>`.
// The game.exe side is tools/emu/d10_roll.py (FUN_00553b10 natively).
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
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <exception>
#include <filesystem>
#include <string>
#include <string_view>
using namespace d2d::game;

int main(int argc, char** argv) try {
    if (argc < 3) { std::printf("SKIP: zd_zeal_test SKILL LEVEL [IAS [WSM [TICKS]]]\n"); return D2D_TEST_SKIPPED; }
    const int skill_id = std::atoi(argv[1]), level = std::atoi(argv[2]);
    const int ias = argc > 3 ? std::atoi(argv[3]) : 0, wsm = argc > 4 ? std::atoi(argv[4]) : 0;
    const int ticks = argc > 5 ? std::atoi(argv[5]) : 200;
    const std::string weapon = argc > 6 ? argv[6] : "";
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* home = std::getenv("HOME");
    const fs::path data_dir = mpq_dir ? fs::path(mpq_dir) : fs::path(home ? home : "") / "Workspace/private/diablo2";
    if (!fs::exists(data_dir / "d2data.mpq")) { std::printf("SKIP: no d2data.mpq\n"); return D2D_TEST_SKIPPED; }
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(data_dir, patch ? fs::path(patch) : fs::path{}, 0x1234);
    assert(data && data->patched);
    d2d::rules::Rng rng(7);
    constexpr int kPaladin = 3;
    auto made = new_character(*data, kPaladin, "Headless", false, true, rng);
    Character character;
    character.character_class = kPaladin;
    character.name = "Headless";
    character.header = made.header;
    character.stats = made.stats;
    character.items = made.items;
    if (!weapon.empty()) for (auto& item : character.items) if (item.location == 1 && item.slot == 4) item.code = weapon;   // slot 4 = right hand (the starter ssd)
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
            if (!missile_rooms(*data, open, float(x) + 0.5f, float(y) + 0.5f).empty()) at_x = x * 5 + 2, at_y = y * 5 + 2;
    assert(at_x >= 0);
    world.actor->player.x = (float(at_x) + 0.5f) / 5;
    world.actor->player.y = (float(at_y) + 0.5f) / 5;
    const Crowd crowd;
    std::uint32_t now = 1000 * kTickMs;
    world.fight.skill_level = [&](int id) { return id == skill_id ? level : 0; };
    for (const auto& item : world.actor->character.items) if (item.location == 1) std::printf("W %s slot %d\n", item.code.c_str(), item.slot);
    Monster monster;
    {
        d2d::rules::Rng spawn_rng{ 5 };
        const auto found = data->monsters.by_id.find("zombie1");
        assert(found != data->monsters.by_id.end());
        monster = make_monster(*data, found->second, (float(at_x + 2) + 0.5f) / 5, (float(at_y) + 0.5f) / 5, spawn_rng, 0);
        monster.stats.hit_points = monster.hit_points = 1000000;
        monster.life_frac = 0;
        monster.next_act = ~0u;
        world.fight.add_monster(monster);
    }
    auto& a = *world.actor;
    a.attack_skill = skill_id;
    a.attack_mon = 0;
    a.player_combat.ias = ias; a.player_combat.wsm = wsm;   // the swing length is attack_ticks(frames, speed, ias, wsm) (set_pmode)
    std::printf("R ias %d wsm %d rate %d\n", ias, wsm, d2d::rules::attack_speed(256, ias, wsm, 0));
    const int mana0 = a.character.stats.values[d2d::d2s::kMana];
    int last_mode = -9; bool last_struck = false; int last_kicks = -9; int n_strikes = 0; int first = -1, prev = -1;
    for (int t = 0; t < ticks; ++t) {
        now += kTickMs;
        world.fight.monsters[0].next_act = ~0u;
        world.fight.world(true, now, 0.04f, crowd);
        (void)world.fight.engage(now);                 // World::tick order: engage (starts the swing), then player_modes
        (void)world.fight.player_modes(true, now, 0.04f);
        if (a.pmode != last_mode && a.pmode >= 0 && last_mode < 0) std::printf("A %d %d\n", int(world.fight.player_anim(a.pmode).frames), int(world.fight.player_anim(a.pmode).action_ms()));
        if (a.pmode != last_mode || a.pstruck != last_struck || a.kicks_left != last_kicks) {
            std::printf("E %d %s mode %d struck %d kicks_left %d prate %.3f life %d\n", t, (a.pstruck && !last_struck) ? "STRIKE" : "edge", a.pmode, int(a.pstruck), a.kicks_left, a.prate, world.fight.monsters[0].hit_points);
            if (a.pstruck && !last_struck) { ++n_strikes; if (first < 0) first = t; if (prev >= 0) std::printf("S gap %d\n", t - prev); prev = t; }
            last_mode = a.pmode; last_struck = a.pstruck; last_kicks = a.kicks_left;
        }
    }
    std::printf("M %d %d strikes %d first %d\n", mana0, a.character.stats.values[d2d::d2s::kMana], n_strikes, first);
    return 0;
} catch (const std::exception& e) { std::fprintf(stderr, "zd: %s\n", e.what()); return 1; }
