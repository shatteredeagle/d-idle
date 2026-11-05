// SPDX-License-Identifier: GPL-3.0-or-later
// COVSTATES: the d2d side of tools/emu/cs_state.py. One blow through the REAL Fight::land (the stun and poison
// parts, components/game/fight/strike.cpp) on a fallen, on the real loaded tables. Built by hand against a scratch
// upstream build (tools/emu/cs_build.sh), never part of upstream.
//   cs_d2d_test --stun   CASES   `id kind utype velocity length prior seed seed` -> `id end` (ticks from now, or -)
//   cs_d2d_test --poison CASES   `id rate1 len1 rate2 len2 dt`                    -> `id end rate`
// Stun: kinds 0 plain, 1 champion, 2 unique, 3 superunique, 4 minion, 5 boss column, 6 champion + boss column
// (Boss(kind) as the freeze oracle; kind 5 and 6 set MonType::boss_column; velocity sets MonType::velocity).
// Poison: Blow.poison = rate x len >> 8 points over len ticks (the rate is in 256ths of a point a frame).

#include <ai.hpp>
#include <character.hpp>
#include <character_store.hpp>
#include <combat.hpp>
#include <d2s_items.hpp>
#include <fight.hpp>
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <montypes.hpp>
#include <rules.hpp>
#include <world.hpp>

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

using namespace d2d::game;

int main(int argc, char** argv) try {
    if (argc < 3) return 2;
    const std::string_view mode = argv[1];
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(fs::path(mpq_dir), patch ? fs::path(patch) : fs::path{}, 0x1234);
    if (!data || !data->patched) { std::printf("FAIL: no patched data\n"); return 1; }
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
    const auto fallen = data->monsters.by_id.find("fallen1");
    assert(fallen != data->monsters.by_id.end());
    auto& type = const_cast<d2d::rules::MonType&>(data->monsters.types[fallen->second]);
    const auto real_velocity = type.velocity;
    const bool real_boss_column = type.boss_column;
    World world(&*data, -1, -1);
    world.enter(character);
    d2d::rules::Rng spawn_rng{ 5 };
    const Monster made_monster = make_monster(*data, int(fallen->second), world.actor->player.x, world.actor->player.y, spawn_rng, 0);
    const std::uint32_t now = 2000 * kTickMs;
    const auto ticks = [&](std::uint32_t until, std::uint32_t from) { return until > from ? std::to_string((until - from) / kTickMs) : std::string("-"); };
    std::ifstream case_file(argv[2]);
    std::string line;
    while (std::getline(case_file, line)) {
        std::istringstream f(line);
        int id = 0;
        if (!(f >> id)) continue;
        Monster monster = made_monster;
        monster.stats.hit_points = monster.hit_points = 1000000;
        monster.life_frac = 0;
        monster.boss_res = {};
        monster.stun_until = monster.poison_until = 0;
        monster.poison_rate = 0;
        world.fight.monsters.clear();
        if (mode == "--stun") {
            int kind = 0, utype = 0, velocity = 0, length = 0, prior = 0;
            if (!(f >> kind >> utype >> velocity >> length >> prior)) continue;
            if (utype != 0) continue;
            type.boss_column = kind == 5 || kind == 6;
            type.velocity = velocity;
            monster.boss = d2d::rules::Boss(kind == 5 ? 0 : kind == 6 ? 1 : kind);
            monster.stun_until = prior > 0 ? now + std::uint32_t(prior) * kTickMs : 0;
            world.fight.add_monster(monster);
            d2d::rules::Blow blow;
            blow.hit = true;
            blow.stun_ticks = length;
            world.fight.land(0, blow, false, now, 0);
            std::printf("%d %s\n", id, ticks(world.fight.monsters.back().stun_until, now).c_str());
            type.boss_column = real_boss_column;
            type.velocity = real_velocity;
        } else {
            int r1 = 0, t1 = 0, r2 = 0, t2 = 0, dt = 0;
            if (!(f >> r1 >> t1 >> r2 >> t2 >> dt)) continue;
            world.fight.add_monster(monster);
            const auto strike = [&](int rate, int len, std::uint32_t at) {
                d2d::rules::Blow blow;
                blow.hit = true;
                blow.poison = int((std::int64_t(rate) * len) >> 8);
                blow.poison_ticks = len;
                world.fight.land(0, blow, false, at, 0);
            };
            strike(r1, t1, now);
            strike(r2, t2, now + std::uint32_t(dt) * kTickMs);
            const auto& m = world.fight.monsters.back();
            std::printf("%d %s %lld\n", id, ticks(m.poison_until, now).c_str(), std::llround(m.poison_rate * 40.0 * 256.0));
        }
    }
    return 0;
} catch (const std::exception& error) {
    std::fprintf(stderr, "%s\n", error.what());
    return 1;
}
