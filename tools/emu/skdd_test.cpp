// SPDX-License-Identifier: GPL-3.0-or-later
// test-links: game
// SKDEFSHIELDD2D: a player or a hireling as the defender of a monster's hit, through d2d's REAL code
// (our copy; upstream path would be tests/game/test_defend_real.cpp). The cases are those of
// tools/emu/skdef_resist.py (`id kind diff exp elem S M v p aux roll len`), the output is the same
// line format as its `game` mode (`id adj effres dealt pdmg plen`) so that skdd_cmp.py can diff them.
//
//   kind 0 player: a real World / Fight; the case's stats are ONE worn item's props (S on the
//     resist stat, M on its max stat, aux as stat 34 / 35 / 110), the difficulty and expansion are
//     the character header's; v (an aura / curse / shrine value in full) is Fight::Boost's stat
//     (added to st_sum in Fight::update_fighters exactly like a curse's curse_stats and an aura's
//     aurastat values). Then the REAL Fight::update_fighters (panel_stats, player_fighter,
//     make_fighter, the aura-on-top loop) builds actor->player_combat.
//   kind 1 hireling: the REAL Fight::merc_fighter() with actor->merc_st.resist = a third of S and
//     the rest on a worn merc item (elements 2..5), merc_items props for the rest. v is not applied
//     (the merc's curse is handled in merc_foe, not in merc_fighter): such cases are skipped.
//   p != 0 (a monster has no pierce stat in 1.14d) and kind 2 (plain monster) are skipped.
// Then the REAL rules::monster_blow (whole points) on that Fighter, as tools/emu/skdef_test.cpp does.
//   `--defend CASES`        -> `id - effres dealt pdmg plen`  (effres: the Fighter's dr_pct or res[])
//   `--defend CASES --keep` same, and a line `#F id dr_pct dr_flat mdr res0..3 plr` on stderr

#include <ai.hpp>
#include <character.hpp>
#include <character_store.hpp>
#include <combat.hpp>
#include <d2s_items.hpp>
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
#include <exception>
#include <fstream>
#include <sstream>
#include <string>
#include <string_view>

using namespace d2d::game;

int main(int argc, char** argv) try {
    const bool shield_mode = argc >= 3 && std::string_view(argv[1]) == "--shield";
    if (argc < 3 || (std::string_view(argv[1]) != "--defend" && !shield_mode)) {
        std::printf("SKIP: an oracle mode, run `test_defend_real --defend CASES` (see tools/emu/skdd_run.sh)\n");
        return D2D_TEST_SKIPPED;
    }
    const bool keep = argc > 3 && std::string_view(argv[3]) == "--keep";
    // --curse: elem 0 cases with v = -100 (Amplify Damage, skill 66) or -50 (Decrepify, 87) are a REAL monster curse on the
    // player (Hero::curses[0]) or the hireling (curses[1]), level 5; v itself feeds game.exe only
    const bool curse_mode = argc > 3 && std::string_view(argv[3]) == "--curse";
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* home = std::getenv("HOME");
    const fs::path data_dir = mpq_dir ? fs::path(mpq_dir) : fs::path(home ? home : "") / "Workspace/private/diablo2";
    if (!fs::exists(data_dir / "d2data.mpq")) { std::printf("SKIP: no d2data.mpq in %s\n", data_dir.string().c_str()); return D2D_TEST_SKIPPED; }
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(data_dir, patch ? fs::path(patch) : fs::path{}, 0x1234);
    if (!data || !data->patched) { std::printf("SKIP: needs the 1.14d data (D2_PATCH_INSTALLER)\n"); return D2D_TEST_SKIPPED; }
    d2d::rules::Rng made_rng(7);
    constexpr int kSorceress = 1;
    auto made = new_character(*data, kSorceress, "Headless", false, true, made_rng);
    Character character;
    character.character_class = kSorceress;
    character.name = "Headless";
    character.header = made.header;
    character.stats = made.stats;
    character.items = made.items;
    character.panel = panel_stats(*data, character.header, character.items, character.stats);
    World world(&*data, -1, -1);
    world.enter(character);
    constexpr std::uint32_t now = 1000 * kTickMs;
    using namespace d2d::d2s;
    std::fprintf(stderr, "# resist_penalty %lld %lld %lld\n", (long long)data->resist_penalty[0], (long long)data->resist_penalty[1], (long long)data->resist_penalty[2]);

    const auto worn = [](std::vector<ItemProp> props) {
        Item item;
        item.code = "cap";
        item.location = item_location::kEquipped;
        item.slot = body_location::kHead;
        item.identified = true;
        item.props = std::move(props);
        return item;
    };
    if (shield_mode) {
        // `--shield CASES`: the plain Attack of a player who wears a weapon ("tst": a copy of the handaxe's info with the
        // case's base damage and bonuses) and a shield (buckler) with the case's stats, through the REAL
        // Fight::update_fighters -> player_fighter -> make_fighter, then the REAL rules::player_blow. A line
        //   id d0 d1 armed bm bM ed fmin fmax dp str dex sb db off broken
        // (ed: the weapon's own enhanced damage, stats 17 / 18 on the weapon; off, fmin, fmax, dp: stats 17 / 18, 21 / 22, 25
        // on the shield; broken: the shield's item flag 0x100) gives `M id hit phys` (phys in whole points).
        data->rules.item_info["tst"] = data->rules.item_info.at("hax");
        std::ifstream shield_cases(argv[2]);
        std::string row;
        while (std::getline(shield_cases, row)) {
            std::istringstream in(row);
            long long id = 0, armed = 0, bm = 0, bM = 0, ed = 0, fmin = 0, fmax = 0, dp = 0, str = 0, dex = 0, sb = 0, db = 0, off = 0, broken_shield = 0;
            std::uint32_t d0 = 0, d1 = 0;
            if (!(in >> id >> d0 >> d1 >> armed >> bm >> bM >> ed >> fmin >> fmax >> dp >> str >> dex >> sb >> db >> off >> broken_shield)) continue;
            data->rules.item_base["tst"] = { .mindam = int(bm), .maxdam = int(bM), .str_bonus = int(sb), .dex_bonus = int(db) };
            auto& actor = *world.actor;
            actor.character.items.clear();
            if (armed) {
                Item weapon;
                weapon.code = "tst"; weapon.location = item_location::kEquipped; weapon.slot = body_location::kRightArm; weapon.identified = true;
                if (ed) weapon.props = { { 17, 0, int(ed) }, { 18, 0, int(ed) } };
                actor.character.items.push_back(weapon);
            }
            Item shield;
            shield.code = "buc"; shield.location = item_location::kEquipped; shield.slot = body_location::kLeftArm; shield.identified = true;
            if (off) { shield.props.push_back({ 17, 0, int(off) }); shield.props.push_back({ 18, 0, int(off) }); }
            if (fmin) shield.props.push_back({ 21, 0, int(fmin) });
            if (fmax) shield.props.push_back({ 22, 0, int(fmax) });
            if (dp) shield.props.push_back({ 25, 0, int(dp) });
            shield.flags = broken_shield ? 0x100u : 0u;
            actor.character.items.push_back(shield);
            actor.character.stats.values[kLevel] = 1; actor.character.stats.values[kStr] = str; actor.character.stats.values[kDex] = dex;
            actor.boost.stats.clear(); actor.boost.until = 0;
            world.fight.update_fighters(now);
            const d2d::rules::Fighter& fighter = actor.player_combat;
            d2d::rules::Target target;
            target.armor_class = 0; target.level = 1; target.hit_points = target.max_hp = 1 << 30;
            d2d::rules::Swing swing;
            d2d::rules::Rng rng; rng.low = d0; rng.high = d1;
            const auto blow = d2d::rules::player_blow(fighter, target, 1, rng, swing);
            std::printf("M %lld %d %d\n", id, blow.hit ? 1 : 0, blow.phys);
            if (keep) std::fprintf(stderr, "#S %lld lo %d hi %d pct %d min %d max %d\n", id, fighter.phys_lo, fighter.phys_hi, fighter.phys_pct, fighter.min, fighter.max);
        }
        return 0;
    }
    std::ifstream cases(argv[2]);
    std::string line;
    while (std::getline(cases, line)) {
        std::istringstream fields(line);
        long long case_id = 0;
        int kind = 0, diff = 0, exp = 0, elem = 0, s = 0, m = 0, v = 0, p = 0, aux = 0, roll = 0, len = 0;
        if (!(fields >> case_id >> kind >> diff >> exp >> elem >> s >> m >> v >> p >> aux >> roll >> len)) continue;
        if (kind > 1 || p != 0 || elem < 0 || elem > 5) continue;
        if (curse_mode) { if (elem != 0 || (v != -100 && v != -50)) continue; }
        else if (kind == 1 && v != 0) continue;
        // the case's stats as item props: the resist stat, its max stat (elements), the flat / length reduction
        constexpr std::array<int, 6> kStat{ 36, 37, 39, 41, 43, 45 };
        std::vector<ItemProp> props;
        const int base = (kind == 1 && elem >= 2) ? s / 3 : 0;      // a hireling's own resistance (merc_st.resist, all four elements)
        props.push_back({ kStat[std::size_t(elem)], 0, s - base });
        if (elem >= 2 && m != 0) props.push_back({ kStat[std::size_t(elem)] + 1, 0, m });
        if (elem == 5) { if (aux != 0) props.push_back({ 110, 0, aux }); }
        else if (aux != 0) props.push_back({ elem == 0 ? 34 : 35, 0, aux });
        auto& actor = *world.actor;
        actor.character.header.status = std::uint8_t(exp ? (actor.character.header.status | 0x20) : (actor.character.header.status & ~0x20));
        actor.character.header.difficulty = { std::uint8_t(diff == 0 ? 0x80 : 0), std::uint8_t(diff == 1 ? 0x80 : 0), std::uint8_t(diff == 2 ? 0x80 : 0) };
        d2d::rules::Fighter defender;
        if (kind == 0) {
            actor.character.items = { worn(props) };
            actor.boost.until = 0xFFFFFFFFu;
            actor.boost.stats.clear();
            actor.curses[0] = {};
            if (curse_mode) { actor.curses[0].skill = v == -100 ? 66 : 87; actor.curses[0].level = 5; actor.curses[0].until = 0xFFFFFFFFu; }
            else if (v != 0) actor.boost.stats.push_back({ kStat[std::size_t(elem)], v });
            world.fight.update_fighters(now);
            defender = actor.player_combat;
        } else {
            actor.character.items.clear();
            actor.character.merc_items = { worn(props) };
            actor.curses[1] = {};
            if (curse_mode) { actor.curses[1].skill = v == -100 ? 66 : 87; actor.curses[1].level = 5; actor.curses[1].until = 0xFFFFFFFFu; }
            actor.merc_st = d2d::rules::MercStats{};
            actor.merc_st.resist = base;
            defender = world.fight.merc_fighter();
        }
        if (keep) std::fprintf(stderr, "#F %lld %d %d %d %d %d %d %d %d\n", case_id, defender.dr_pct, defender.dr_flat, defender.mdr,
                               defender.res[0], defender.res[1], defender.res[2], defender.res[3], defender.plr);
        std::string effres = "-";
        d2d::rules::MonStats attacker;
        attacker.to_hit = 1000000; attacker.level = 99;
        if (elem == 0) { effres = std::to_string(defender.dr_pct); attacker.a1_min = attacker.a1_max = roll; }
        else if (elem == 1) attacker.elements[0] = { 4, 100, roll, roll, 0, "A1" };
        else {
            effres = std::to_string(defender.res[std::size_t(elem - 2)]);
            if (elem == 5) attacker.elements[0] = { 3, 100, roll, roll, len, "A1" };
            else attacker.elements[0] = { elem - 2, 100, roll, roll, 0, "A1" };
        }
        d2d::rules::Taken taken;
        for (std::uint32_t seed = 1; seed < 5000 && !taken.hit; ++seed) {
            d2d::rules::Rng rng(seed);
            taken = d2d::rules::monster_blow(defender, 99, d2d::rules::Motion::standing, attacker, false, rng);
        }
        std::printf("%lld - %s ", case_id, effres.c_str());
        if (elem == 5) std::printf("0 %d %d\n", taken.poison, taken.poison_ticks);
        else std::printf("%d - -\n", taken.damage);
    }
    return 0;
} catch (const std::exception& error) {
    std::fprintf(stderr, "skdd_test: %s\n", error.what());
    return 1;
}
