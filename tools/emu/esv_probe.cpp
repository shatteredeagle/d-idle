// SPDX-License-Identifier: GPL-3.0-or-later
// esverify d2d probe: one frame of monster blows on a player with (or without) Energy Shield through the REAL d2d code
// (rules::monster_blow, Foe::take, Fight::absorb) of whichever d2d tree it is compiled against (base or head).
//   esv_probe CASES.tsv > out.tsv        (D2_MPQ_DIR, D2_PATCH_INSTALLER: the 1.14d data)
// A case line (tab separated):
//   id cls es_level mana256 dr_pct dr_flat mdr res_fire res_light res_cold nblows {phys fire light cold magic} x nblows
// Whole-point rolls (min == max, `exact`: no draw, no 256ths added). es_level 0: no Energy Shield state.
// Out: `id life256 mana256` (life lost: the returned whole points x 256 + foe.frac), and `#calc level pct per16` lines.
#include <ai.hpp>
#include <character.hpp>
#include <character_store.hpp>
#include <combat.hpp>
#include <fight.hpp>
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <rules.hpp>
#include <world.hpp>

#include <algorithm>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <fstream>
#include <sstream>
#include <string>
#include <vector>

using namespace d2d::game;
namespace rules = d2d::rules;

int main(int argc, char** argv) {
    if (argc < 2) { std::fprintf(stderr, "usage: esv_probe CASES.tsv\n"); return 2; }
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    if (!mpq_dir || !patch) { std::fprintf(stderr, "D2_MPQ_DIR and D2_PATCH_INSTALLER needed\n"); return 2; }
    auto data = load_game_data(fs::path(mpq_dir), fs::path(patch), 0x5eed);
    if (!data || !data->patched) { std::fprintf(stderr, "no patched 1.14d data\n"); return 2; }
    int es_id = -1;
    for (int id = 0; id < 400 && es_id < 0; ++id)
        if (const auto* skill = data->skills.get(id); skill && skill->aurastate == "energyshield") es_id = id;
    if (es_id < 0) { std::fprintf(stderr, "no energyshield skill\n"); return 2; }

    std::ifstream in(argv[1]);
    std::string line;
    int built_cls = -1;
    std::unique_ptr<World> world;
    Character character;
    constexpr std::uint32_t kTick = 40;
    std::vector<Fight::SelfState> base_states;
    while (std::getline(in, line)) {
        if (line.empty() || line[0] == '#') continue;
        std::istringstream f(line);
        long long id = 0, mana = 0;
        int cls = 0, level = 0, dr_pct = 0, dr_flat = 0, mdr = 0, nblows = 0;
        std::array<int, 3> res{};
        f >> id >> cls >> level >> mana >> dr_pct >> dr_flat >> mdr >> res[0] >> res[1] >> res[2] >> nblows;
        if (cls != built_cls) {
            rules::Rng made_rng(11);
            auto made = new_character(*data, cls, "Probe", false, true, made_rng);
            character = Character{};
            character.character_class = cls;
            character.name = "Probe";
            character.header = made.header;
            character.stats = made.stats;
            character.items = made.items;
            character.panel = panel_stats(*data, character.header, character.items, character.stats);
            world = std::make_unique<World>(&*data, -1, -1);
            world->enter(character);
            for (std::uint32_t tick = 1; tick <= 25; ++tick) world->tick({}, tick * kTick, (tick - 1) * kTick);
            base_states = world->actor->self_states;
            built_cls = cls;
            const auto* es = data->skills.get(es_id);
            for (const int lvl : { 1, 2, 5, 10, 18, 20, 25, 30, 40, 50, 60, 99 })
                std::printf("#calc cls %d level %d pct %d per16 %d\n", cls, lvl, world->fight.calc(*es, es->calc[0], lvl), world->fight.calc(*es, es->calc[1], lvl));
        }
        auto& fight = world->fight;
        auto& hero = *world->actor;
        const std::uint32_t now = 30 * kTick;
        Foe foe = fight.player_foe(now);
        foe.fighter.dr_pct = dr_pct; foe.fighter.dr_flat = dr_flat; foe.fighter.mdr = mdr;
        for (auto& r : foe.fighter.res) r = 0;
        for (int k = 0; k < 3; ++k) foe.fighter.res[std::size_t(k)] = res[std::size_t(k)];
        for (int b = 0; b < nblows; ++b) {
            int phys = 0, fire = 0, light = 0, cold = 0, magic = 0;
            f >> phys >> fire >> light >> cold >> magic;
            rules::MonStats attacker;
            attacker.level = 1;
            for (auto& element : attacker.elements) element.type = -1;
            attacker.a1_min = attacker.a1_max = phys;
            std::size_t slot = 0;
            // monster_blow element types: 0 fire, 1 lightning, 2 cold, 4 magic (3 is poison)
            for (const auto& [type, roll] : { std::pair{ 0, fire }, std::pair{ 1, light }, std::pair{ 2, cold }, std::pair{ 4, magic } })
                if (roll > 0 && slot < attacker.elements.size()) attacker.elements[slot++] = { type, 100, roll, roll, 25, "A1" };
            rules::Rng rng(12345 + b);
            foe.take(rules::monster_blow(foe.fighter, 1, rules::Motion::running, attacker, false, rng, false, true, true));
        }
        hero.self_states = base_states;
        hero.absorb_pool = 0; hero.absorb_skill = -1; hero.absorb_frac = 0;
        if (level > 0) hero.self_states.push_back({ es_id, level, ~0u });
        hero.character.stats.values[d2d::d2s::kMana] = mana;
        hero.self_hurt = 0;
        hero.pmode = -1;
        const int lost = fight.absorb(foe, now);
        std::printf("%lld\t%lld\t%lld\n", id, static_cast<long long>(lost) * 256 + foe.frac, static_cast<long long>(hero.character.stats.values[d2d::d2s::kMana]));
    }
    return 0;
}
