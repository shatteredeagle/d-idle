// SPDX-License-Identifier: GPL-3.0-or-later
// verifyb2 scratch probe (not for upstream): the player's mana check and payment through the
// REAL Fight entry points, in the order world/commands.cpp picks them for a skill used on a
// spot with no live unit: self_cast -> Fight::cast, missile or spot skill -> cast_missile,
// summon skill -> cast_summon. Melee (start_swing) is not driven. The character is of the
// skill's own class (charclass), the Sorceress for a row with none.
//   vb2_mana_probe CASES   (lines `id skill lvl mana256`) -> `id check paid` | `id -1 -1`
#include <character.hpp>
#include <character_store.hpp>
#include <fight.hpp>
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <level_ids.hpp>
#include <rules.hpp>
#include <world.hpp>

#include <algorithm>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <map>
#include <memory>
#include <sstream>
#include <string>
#include <type_traits>

using namespace d2d::game;

namespace {
struct Probe {
    const GameData& data;
    d2d::rules::Rng rng{ 1913 };
    Character character;
    std::unique_ptr<World> world;
    Level field;
    std::uint32_t now = 500 * kTickMs;
    int level = 1;
    Probe(const GameData& game_data, int cls) : data(game_data) {
        auto made = new_character(data, cls, "Probe", false, true, rng);
        character.character_class = cls;
        character.name = "Probe";
        character.header = made.header;
        character.stats = made.stats;
        character.items = made.items;
        character.panel = panel_stats(data, character.header, character.items, character.stats);
        world = std::make_unique<World>(&data, -1, -1);
        world->enter(character);
        field = *data.level(d2d::rules::level_ids::kBloodMoor);
        std::ranges::fill(field.walk, std::uint16_t(0));
        world->level = &field;
        world->fight.enter(&field);
        world->fight.skill_level = [this](int) { return level; };
        world->fight.skill_base = world->fight.skill_level;
    }
    auto& mana() { return world->actor->character.stats.values[d2d::d2s::kMana]; }
    bool use(int id, int lvl, long long m) {
        auto& fight = world->fight;
        auto* actor = world->actor;
        level = lvl;
        mana() = static_cast<std::remove_reference_t<decltype(mana())>>(m);
        actor->pmode = -1; actor->swing_skill = 0; actor->attack_mon = -1; actor->attack_skill = 0;
        now += 60 * kTickMs;
        const auto* skill = data.skills.get(id);
        if (!skill) return false;
        const float x = actor->player.x + 2, y = actor->player.y + 1;
        if (self_cast(*skill)) return fight.cast(id, now);
        if (fight.missile_skill(*skill) || Fight::spot_skill(*skill)) return fight.cast_missile(id, x, y, now);
        if (fight.summon_skill(*skill)) return fight.cast_summon(id, x, y, now);
        return false;
    }
};
}  // namespace

int main(int argc, char** argv) {
    if (argc < 2) return 2;
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(fs::path(mpq_dir ? mpq_dir : "/tmp/ug-emu"), patch ? fs::path(patch) : fs::path{}, 0x51);
    if (!data || !data->patched) { std::printf("FAIL: no patched data\n"); return 1; }
    std::map<int, std::unique_ptr<Probe>> probes;
    std::ifstream cases(argv[1]);
    std::string line;
    int last_skill = -1, last_lvl = -1;
    bool usable = false;
    Probe* probe = nullptr;
    while (std::getline(cases, line)) {
        std::istringstream in(line);
        long long id = 0, m = 0;
        int sid = 0, lvl = 0;
        if (!(in >> id >> sid >> lvl >> m)) continue;
        const auto* skill = data->skills.get(sid);
        static const std::map<std::string, int> codes{ { "ama", 0 }, { "sor", 1 }, { "nec", 2 }, { "pal", 3 }, { "bar", 4 }, { "dru", 5 }, { "ass", 6 } };
        const auto code = skill ? codes.find(skill->cls) : codes.end();
        const int cls = code != codes.end() ? code->second : 1;
        auto& slot = probes[cls];
        if (!slot) slot = std::make_unique<Probe>(*data, cls);
        probe = slot.get();
        if (sid != last_skill || lvl != last_lvl) {
            last_skill = sid; last_lvl = lvl;
            usable = probe->use(sid, lvl, 0x7fffff);
        }
        if (!usable) { std::printf("%lld -1 -1\n", id); continue; }
        const bool began = probe->use(sid, lvl, m);
        std::printf("%lld %d %lld\n", id, began ? 1 : 0, began ? m - static_cast<long long>(probe->mana()) : 0LL);
    }
    return 0;
}
