// SPDX-License-Identifier: GPL-3.0-or-later
// test-links: game
// PALDUR: d2d's durability loss (Fight::wear_weapon / Fight::wear_armor / Fight::wear) on tools/emu/pal_wear.py's cases.
// Usage: pal_wear_test CASES.tsv > d2d.tsv. The REAL Fight on a Paladin; the player's seed is the case's (the World's rng).
// A case: id mode lo hi f70 ... (see pal_wear.py); f70 is game.exe's +0x70 and has no d2d counterpart (d2d never reads it).
// Output per case: `id lo hi` then the item states `dur/qty/flags`, `-` (none) or `gone` (out of ammo took it).

#include <character.hpp>
#include <character_store.hpp>
#include <d2s_items.hpp>
#include <fight.hpp>
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <rules.hpp>
#include <world.hpp>

#include <array>
#include <cassert>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <exception>
#include <filesystem>
#include <fstream>
#include <sstream>
#include <string>
#include <vector>

using namespace d2d::game;

int main(int argc, char** argv) try {
    if (argc < 2) { std::printf("SKIP: needs CASES.tsv (see tools/emu/pal_wear.py)\n"); return D2D_TEST_SKIPPED; }
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* home = std::getenv("HOME");
    const fs::path data_dir = mpq_dir ? fs::path(mpq_dir) : fs::path(home ? home : "") / "Workspace/private/diablo2";
    if (!fs::exists(data_dir / "d2data.mpq")) { std::printf("SKIP: no d2data.mpq in %s\n", data_dir.string().c_str()); return D2D_TEST_SKIPPED; }
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(data_dir, patch ? fs::path(patch) : fs::path{}, 0x1234);
    assert(data && data->town.id == 1 && !data->town.walk.empty());
    if (!data->patched) { std::printf("SKIP: needs the 1.14d data (D2_PATCH_INSTALLER)\n"); return D2D_TEST_SKIPPED; }
    d2d::rules::Rng made_rng(7);
    constexpr int kPaladin = 3;
    auto made = new_character(*data, kPaladin, "Headless", false, true, made_rng);
    Character character;
    character.character_class = kPaladin;
    character.name = "Headless";
    character.header = made.header;
    character.stats = made.stats;
    character.items = made.items;
    character.panel = panel_stats(*data, character.header, character.items, character.stats);
    World world(&*data, -1, -1);
    world.enter(character);

    using namespace d2d::d2s;
    const auto equipped = [](const char* code, int slot, int dur, int max, int qty, bool broken) {
        Item item;
        item.code = code;
        item.location = item_location::kEquipped;
        item.slot = slot;
        item.durability = dur;
        item.max_durability = max;
        item.quantity = qty;
        item.identified = true;
        item.flags = broken ? 0x100u : 0u;
        return item;
    };
    constexpr std::array<int, 7> kBody{ body_location::kHead, body_location::kTorso, body_location::kRightArm, body_location::kLeftArm,
                                        body_location::kBelt, body_location::kFeet, body_location::kGloves };
    constexpr std::array<const char*, 7> kCode{ "cap", "qui", "hax", "buc", "lbl", "lbt", "lgl" };   // right arm: a weapon
    for (const char* code : { "cap", "qui", "hax", "buc", "lbl", "lbt", "lgl", "jav" })
        if (!data->rules.item_info.contains(code)) { std::fprintf(stderr, "no item %s\n", code); return 1; }

    std::ifstream in(argv[1]);
    std::string line;
    while (std::getline(in, line)) {
        std::istringstream words(line);
        long id = 0, lo = 0, hi = 0, f70 = 0;
        std::string mode;
        if (!(words >> id >> mode >> lo >> hi >> f70)) continue;
        world.actor->character.items.clear();
        world.rng.low = std::uint32_t(lo);
        world.rng.high = std::uint32_t(hi);
        std::vector<std::string> shown;
        std::uint32_t now = 1000 * kTickMs;
        if (mode == "W") {
            long wtype, dur, max, qty, gate, brk;
            words >> wtype >> dur >> max >> qty >> gate >> brk;
            world.actor->character.items.push_back(equipped(wtype ? "jav" : "hax", body_location::kRightArm, int(dur), gate ? int(max) : 0, wtype ? int(qty) : -1, brk != 0));
            world.fight.wear_weapon(now);
            if (world.actor->character.items.empty()) shown.push_back("gone");
            else {
                const auto& item = world.actor->character.items[0];
                shown.push_back(std::to_string(item.durability) + "/" + std::to_string(wtype ? item.quantity : 1) + "/" + std::to_string(item.flags));
            }
        } else {
            std::array<int, 7> index{};
            for (std::size_t slot = 0; slot < 7; ++slot) {
                long present, dur, max, gate, brk;
                words >> present >> dur >> max >> gate >> brk;
                index[slot] = -1;
                if (!present) continue;
                index[slot] = int(world.actor->character.items.size());
                world.actor->character.items.push_back(equipped(kCode[slot], kBody[slot], int(dur), gate ? int(max) : 0, -1, brk != 0));
            }
            world.fight.wear_armor(now);
            for (std::size_t slot = 0; slot < 7; ++slot) {
                if (index[slot] < 0) { shown.push_back("-"); continue; }
                const auto& item = world.actor->character.items[std::size_t(index[slot])];
                shown.push_back(std::to_string(item.durability) + "/1/" + std::to_string(item.flags));
            }
        }
        std::printf("%ld\t%u\t%u", id, unsigned(world.rng.low), unsigned(world.rng.high));
        for (const auto& s : shown) std::printf("\t%s", s.c_str());
        std::printf("\n");
    }
    return 0;
} catch (const std::exception& error) {
    std::fprintf(stderr, "%s\n", error.what());
    return 1;
}
