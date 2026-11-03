// verifydmg4 (fourth independent check of DMGROLL r4 9ea2afe9): the REAL Fight::absorb and Fight::hurt_side, compiled
// in the r4 tree's libgame.a, called on one monster blow a case (v4_build.sh links this file against that library).
// The world is set up as tests/game/test_game.cpp does (a new Barbarian, World::enter, 25 ticks).
// stdin, one case a line: "min max seedlo seedhi pct flat pool256 exact eslvl mana256" (eslvl 0: no Energy Shield).
// stdout: "life256 pool256 seedlo seedhi mana256 espct per16 hurt_life256 hurt_pool256 hurt_mana256"
//   life256 / pool256 / mana256: after Fight::absorb (life256 = what it returns << 8 + foe.frac);
//   espct / per16: the Energy Shield calc1 (clamped 0..95, as states.cpp) and calc2 the real function used;
//   hurt_*: the same case through Fight::hurt_side (the player's life stat lost, the pool and the mana after).
// The monster_blow is r4's header (rules/combat.hpp), as ai.cpp calls it; Foe::take builds the blow (ai.hpp).
#include <ai.hpp>
#include <character.hpp>
#include <combat.hpp>
#include <d2s.hpp>
#include <fight.hpp>
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <rules.hpp>
#include <world.hpp>

#include <algorithm>
#include <array>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <filesystem>

namespace fs = std::filesystem;
using namespace d2d::game;

int main() {
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    if (!mpq_dir || !fs::exists(fs::path(mpq_dir) / "d2data.mpq")) { std::fprintf(stderr, "no d2data.mpq\n"); return 2; }
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(fs::path(mpq_dir), patch ? fs::path(patch) : fs::path{}, 0x1234);
    d2d::rules::Rng made_rng(7);
    auto made = new_character(*data, 4, "Headless", false, true, made_rng);
    Character character;
    character.character_class = 4;
    character.name = "Headless";
    character.header = made.header;
    character.stats = made.stats;
    character.items = made.items;
    character.panel = panel_stats(*data, character.header, character.items, character.stats);
    World world(&*data, -1, -1);
    world.enter(character);
    constexpr std::uint32_t kTick = 40;
    for (std::uint32_t tick = 1; tick <= 25; ++tick) world.tick({}, tick * kTick, (tick - 1) * kTick);
    auto& fight = world.fight;
    auto& hero = *world.actor;
    int es_id = -1;
    for (int id = 0; id < 400 && es_id < 0; ++id)
        if (const auto* skill = data->skills.get(id); skill && skill->aurastate == "energyshield") es_id = id;
    if (es_id < 0) { std::fprintf(stderr, "no energyshield skill\n"); return 2; }
    std::fprintf(stderr, "energy shield skill %d, bone armor 68 aurastate %s\n", es_id, std::string(data->skills.get(68)->aurastate).c_str());
    const auto states = hero.self_states;
    const std::uint32_t now = 30 * kTick;
    int mn, mx, pct, flat, pool256, exact, eslvl;
    unsigned lo, hi;
    long long mana256;
    while (std::scanf("%d %d %u %u %d %d %d %d %d %lld", &mn, &mx, &lo, &hi, &pct, &flat, &pool256, &exact, &eslvl, &mana256) == 10) {
        d2d::rules::MonStats att{};
        att.a1_min = mn;
        att.a1_max = mx;
        att.level = 1;
        for (auto& e : att.elements) e.type = -1;
        Foe foe = fight.player_foe(now);
        foe.fighter.dr_pct = pct;
        foe.fighter.dr_flat = flat;
        d2d::rules::Rng rng;
        rng.low = lo;
        rng.high = hi;
        const auto taken = d2d::rules::monster_blow(foe.fighter, 1, d2d::rules::Motion::standing, att, false, rng, false, true, exact != 0);
        foe.take(taken);
        int espct = 0, per16 = 0;
        const auto arm = [&] {
            hero.self_states = states;
            hero.absorb_pool = 0; hero.absorb_frac = 0; hero.absorb_skill = -1;
            if (pool256 > 0) {
                hero.absorb_skill = 68;
                hero.self_states.push_back({ 68, 1, ~0u });
                hero.absorb_pool = pool256 >> 8; hero.absorb_frac = pool256 & 255;
            }
            if (eslvl > 0) {
                hero.self_states.push_back({ es_id, eslvl, ~0u });
                const auto* skill = data->skills.get(es_id);
                espct = std::clamp(fight.calc(*skill, skill->calc[0], eslvl), 0, 95);
                per16 = std::max(fight.calc(*skill, skill->calc[1], eslvl), 1);
            }
            hero.character.stats.values[d2d::d2s::kMana] = mana256;
            hero.self_hurt = 0;
            hero.pmode = -1;
        };
        arm();
        Foe direct = foe;
        const int whole = fight.absorb(direct, now);
        const long long life_a = (static_cast<long long>(whole) << 8) + direct.frac;
        const long long pool_a = static_cast<long long>(hero.absorb_pool) * 256 + hero.absorb_frac;
        const long long mana_a = hero.character.stats.values[d2d::d2s::kMana];
        arm();
        auto side = std::array{ foe, fight.merc_foe(now) };
        side[1].hits = 0; side[1].damage = 0; side[1].frac = 0; side[1].blows.clear();
        const std::int64_t full = std::int64_t(1000000) << 8;
        hero.character.stats.values[d2d::d2s::kLife] = full;
        fight.hurt_side(side, now);
        const long long life_b = full - hero.character.stats.values[d2d::d2s::kLife];
        const long long pool_b = static_cast<long long>(hero.absorb_pool) * 256 + hero.absorb_frac;
        const long long mana_b = hero.character.stats.values[d2d::d2s::kMana];
        std::printf("%lld %lld %u %u %lld %d %d %lld %lld %lld\n", life_a, pool_a, rng.low, rng.high, mana_a, espct, per16, life_b, pool_b, mana_b);
    }
    return 0;
}
