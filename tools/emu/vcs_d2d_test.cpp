// SPDX-License-Identifier: GPL-3.0-or-later
// verifycovstates: the d2d side of tools/emu/vcs_game.py. Written independently of the covstates probe: it uses the REAL
// loaded MonType of every MonStats row (nothing patched per case) and it sets the fight's own Rng to the attacker's seed
// that game.exe was given, so a special monster's 90 percent shrug-off is compared case by case, not by counts.
// d2d's Rng is the same LCG as game.exe's unit seed (multiplier 0x6ac690c5, low and high words; rules.hpp).
//   vcs_d2d_test --types                 -> `name boss velocity` for every loaded monster type
//   vcs_d2d_test --stun   CASES          `id name class flags ut length prior slo shi` -> `id end|-|? rnglow rnghigh`
//   vcs_d2d_test --poison CASES          `id grp rate1 len1 rate2 len2 dt`            -> `id end|- points_per_tick`
// Poison: Blow.poison = rate x len >> 8, poison_ticks = len, as rules::fire in combat.hpp builds a poison blow.
// Built by hand against a scratch upstream build (tools/emu/vcs_build.sh), never part of upstream.

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
    if (argc < 2) return 2;
    const std::string_view mode = argv[1];
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(fs::path(mpq_dir), patch ? fs::path(patch) : fs::path{}, 0x1234);
    if (!data || !data->patched) { std::printf("FAIL: no patched data\n"); return 1; }
    if (mode == "--types") {
        for (const auto& [name, index] : data->monsters.by_id)
            std::printf("%s %d %d\n", name.c_str(), data->monsters.types[index].boss_column ? 1 : 0, data->monsters.types[index].velocity);
        std::fprintf(stderr, "types %zu by_id %zu\n", data->monsters.types.size(), data->monsters.by_id.size());
        for (const char* name : { "willowisp8", "vampire9", "skmage_fire7", "overseer6", "skmage_pois7", "skeleton8", "willowisp1", "vampire1", "skeleton1" })
            if (const auto at = data->monsters.by_id.find(name); at != data->monsters.by_id.end()) {
                const auto& t = data->monsters.types[at->second];
                std::fprintf(stderr, "%s index %zu velocity %d run %d boss %d\n", name, std::size_t(at->second), t.velocity, t.run, t.boss_column ? 1 : 0);
            }
        return 0;
    }
    if (argc < 3) return 2;
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
    const std::uint32_t now = 2000 * kTickMs;
    const auto ticks = [&](std::uint32_t until, std::uint32_t from) { return until > from ? std::to_string((until - from) / kTickMs) : std::string("-"); };
    std::ifstream case_file(argv[2]);
    std::string line;
    while (std::getline(case_file, line)) {
        std::istringstream f(line);
        int id = 0;
        if (!(f >> id)) continue;
        world.fight.monsters.clear();
        if (mode == "--stun") {
            std::string name;
            int cls = 0, flags = 0, utype = 0, length = 0, prior = 0;
            std::uint32_t lo = 0, hi = 0;
            if (!(f >> name >> cls >> flags >> utype >> length >> prior >> lo >> hi) || utype != 0) continue;
            const auto type = data->monsters.by_id.find(name);
            if (type == data->monsters.by_id.end()) { std::printf("%d ? 0 0\n", id); continue; }
            d2d::rules::Rng spawn_rng{ 5 };
            Monster monster = make_monster(*data, int(type->second), world.actor->player.x, world.actor->player.y, spawn_rng, 0);
            monster.stats.hit_points = monster.hit_points = 1000000;
            monster.life_frac = 0;
            monster.boss_res = {};
            monster.stun_until = prior > 0 ? now + std::uint32_t(prior) * kTickMs : 0;
            monster.poison_until = 0;
            monster.poison_rate = 0;
            // monster data +0x16 as game.exe's spawn code sets it: champions 0xd, uniques 8, superuniques 0xa, minions 0x10
            monster.boss = flags == 0x0D ? d2d::rules::Boss::champion : flags == 0x08 ? d2d::rules::Boss::unique
                         : flags == 0x0A ? d2d::rules::Boss::superunique : flags == 0x10 ? d2d::rules::Boss::minion : d2d::rules::Boss::none;
            world.fight.add_monster(monster);
            world.rng.low = lo;
            world.rng.high = hi;
            d2d::rules::Blow blow;
            blow.hit = true;
            blow.stun_ticks = length;
            // "~": land's first two lines refuse the blow (not targetable, converted): the stun code is never reached
            const bool refused = !world.fight.monsters.back().targetable() || world.fight.monsters.back().align == 2;
            world.fight.land(0, blow, false, now, 0);
            std::printf("%d %s %u %u\n", id, refused ? "~" : ticks(world.fight.monsters.back().stun_until, now).c_str(), world.rng.low, world.rng.high);
        } else {
            std::string grp;
            int r1 = 0, t1 = 0, r2 = 0, t2 = 0, dt = 0;
            if (!(f >> grp >> r1 >> t1 >> r2 >> t2 >> dt)) continue;
            const auto fallen = data->monsters.by_id.find("fallen1");
            d2d::rules::Rng spawn_rng{ 5 };
            Monster monster = make_monster(*data, int(fallen->second), world.actor->player.x, world.actor->player.y, spawn_rng, 0);
            monster.stats.hit_points = monster.hit_points = 1000000;
            monster.life_frac = 0;
            monster.boss_res = {};
            monster.stun_until = monster.poison_until = 0;
            monster.poison_rate = 0;
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
            std::printf("%d %s %.9f\n", id, ticks(m.poison_until, now).c_str(), m.poison_rate * 40.0);
        }
    }
    return 0;
} catch (const std::exception& error) {
    std::fprintf(stderr, "%s\n", error.what());
    return 1;
}
