// stateev_diff (job stateevdiff): the REAL d2d Fight::absorb (Energy Shield block) and the real rules::monster_blow
// (the resistance step), linked from a scratch upstream build's libgame.a, called on one single-element blow a case.
// World set up as tools/emu/v4_probe.cpp does (a new Barbarian, World::enter, 25 ticks); stateev_diff_build.sh links it.
// stdin, one case a line: "elem res whole eslvl mana256"
//   elem 0 phys, 1 magic, 2 fire, 3 lightning, 4 cold, 5 poison (game.exe order of tools/emu/ser_events.py ELEM)
//   res  the defender's resistance % of that element (phys: Fighter::dr_pct), whole  the raw blow in points
//   eslvl 0: no Energy Shield; mana256: the defender's mana in 256ths
// stdout: "pre post mana256 espct per16": pre = foe.damage after monster_blow (resisted, whole points), post = what
//   Fight::absorb returns (the life lost), mana after, and the Energy Shield calc1 (clamped 0..95) / calc2 used.
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
    const auto states = hero.self_states;
    const std::uint32_t now = 30 * kTick;
    int elem, res, whole, eslvl;
    long long mana256;
    while (std::scanf("%d %d %d %d %lld", &elem, &res, &whole, &eslvl, &mana256) == 5) {
        d2d::rules::MonStats att{};
        att.level = 1;
        for (auto& e : att.elements) e.type = -1;
        Foe foe = fight.player_foe(now);
        foe.fighter.dr_pct = 0; foe.fighter.dr_flat = 0; foe.fighter.mdr = 0;
        for (auto& r : foe.fighter.res) r = 0;
        // monster_blow's element types: 0 fire, 1 lightning, 2 cold, 3 poison, 4 magic; physical is a1_min / a1_max
        if (elem == 0) { att.a1_min = att.a1_max = whole; foe.fighter.dr_pct = res; }
        else {
            const int type = elem == 2 ? 0 : elem == 3 ? 1 : elem == 4 ? 2 : elem == 5 ? 3 : 4;
            att.elements[0] = { type, 100, whole, whole, 25, "A1" };
            if (type < 4) foe.fighter.res[std::size_t(type)] = res;
        }
        d2d::rules::Rng rng(12345);
        // running defender: the hit roll is skipped (monster_blow: running && !missile); no block / dodge set
        const auto taken = d2d::rules::monster_blow(foe.fighter, 1, d2d::rules::Motion::running, att, false, rng, false, true, true);   // exact: no hit roll, no 1..2 point physical roll from an empty range
        foe.take(taken);
        const int pre = foe.damage;
        int espct = 0, per16 = 0;
        hero.self_states = states;
        hero.absorb_pool = 0; hero.absorb_skill = -1;
        if (eslvl > 0) {
            hero.self_states.push_back({ es_id, eslvl, ~0u });
            const auto* skill = data->skills.get(es_id);
            espct = std::clamp(fight.calc(*skill, skill->calc[0], eslvl), 0, 95);
            per16 = std::max(fight.calc(*skill, skill->calc[1], eslvl), 1);
        }
        hero.character.stats.values[d2d::d2s::kMana] = mana256;
        hero.self_hurt = 0;
        hero.pmode = -1;
        const int post = fight.absorb(foe, now);
        std::printf("%d %d %lld %d %d\n", pre, post, static_cast<long long>(hero.character.stats.values[d2d::d2s::kMana]), espct, per16);
    }
    return 0;
}
