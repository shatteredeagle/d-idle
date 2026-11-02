// SPDX-License-Identifier: GPL-3.0-or-later
// test-links: game
// verifyfreezeend2 d2d probe (own; modelled on the verifier probe of the freeze resume).
// `--vfe2 CASES`: replays tools/emu/vfe2_run.py gen cases (`id diff row T`) on a real d2d build: a zombie1 body
// (its think swings when its foe stands in melee) carrying the case row's loaded aidel, a think due at frame T
// (next_act), a get-hit through the REAL `hurt` at frame 0, monster_update every tick from frame -2.
// Prints `id swing gh_end`: the first frame the monster swings (mode A1 / A2) and the frame its mode leaves GH
// (`-` none within the horizon). Frame 0's hit comes before the monster's update (game.exe's missile / player pass
// runs before the monster group, FUN_005414d0).

#include <ai.hpp>
#include <character.hpp>
#include <character_store.hpp>
#include <fight.hpp>
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <level_ids.hpp>
#include <rules.hpp>
#include <world.hpp>

#include <array>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <exception>
#include <filesystem>
#include <fstream>
#include <sstream>
#include <string>
#include <string_view>
#include <vector>

using namespace d2d::game;

int main(int argc, char** argv) try {
    if (argc < 3 || std::string_view(argv[1]) != "--vfe2") {
        std::printf("SKIP: probe, run `test_vfe2_gh --vfe2 CASES`\n");
        return D2D_TEST_SKIPPED;
    }
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    if (!mpq_dir) { std::printf("SKIP: D2_MPQ_DIR\n"); return D2D_TEST_SKIPPED; }
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(fs::path(mpq_dir), patch ? fs::path(patch) : fs::path{}, 0x1234);
    if (!data || !data->patched) { std::printf("SKIP: needs the 1.14d patch\n"); return D2D_TEST_SKIPPED; }

    d2d::rules::Rng rng(11);
    auto made = new_character(*data, 1, "Probe", false, true, rng);
    Character character;
    character.character_class = 1;
    character.name = "Probe";
    character.header = made.header;
    character.stats = made.stats;
    character.items = made.items;
    character.panel = panel_stats(*data, character.header, character.items, character.stats);
    World world(&*data, -1, -1);
    world.enter(character);

    const auto body_it = data->monsters.by_id.find("zombie1");
    if (body_it == data->monsters.by_id.end()) { std::printf("SKIP: rows\n"); return D2D_TEST_SKIPPED; }
    auto& body = const_cast<d2d::rules::MonType&>(data->monsters.types[body_it->second]);
    const auto saved_diff = body.diff;
    Level here{};
    here.id = d2d::rules::level_ids::kBloodMoor;
    const Crowd crowd;
    std::vector<Missile> shots;
    constexpr int kBase = 3000, kHorizon = 200;

    std::ifstream cases(argv[2]);
    std::string line;
    while (std::getline(cases, line)) {
        if (line.empty() || line[0] == '#') continue;
        std::istringstream in(line);
        int id, diff, think_at;
        std::string row;
        if (!(in >> id >> diff >> row >> think_at)) continue;
        const auto src = data->monsters.by_id.find(row);
        if (src == data->monsters.by_id.end()) { std::printf("%d NOROW -\n", id); continue; }
        body.diff[std::size_t(diff)].aidel = data->monsters.types[src->second].diff[std::size_t(diff)].aidel;
        d2d::rules::Rng spawn{ 0x51e00000u ^ std::uint32_t(id) };
        Monster monster = make_monster(*data, int(body_it->second), 10.f, 10.f, spawn, diff);
        monster.boss = d2d::rules::Boss::none;
        monster.stats.hit_points = 1000000;
        monster.hit_points = 10000000;
        monster.life_frac = 0;
        monster.aware = true;
        monster.next_act = std::uint32_t(kBase + think_at) * kTickMs;
        world.fight.game_difficulty = diff;
        world.fight.monsters.clear();
        world.fight.add_monster(monster);
        auto& m = world.fight.monsters[0];
        std::array<Foe, 1> foes{};
        foes[0].x = 10.4f;
        foes[0].y = 10.f;
        d2d::rules::Rng think_rng{ 0x7a110000u ^ std::uint32_t(id) };
        int swing = 0, gh_end = 0;
        bool have_swing = false, have_end = false, was_gh = false;
        for (int frame = -2; frame <= kHorizon && !(have_swing && have_end); ++frame) {
            const std::uint32_t now = std::uint32_t(kBase + frame) * kTickMs;
            if (frame == 0) { hurt(*data, m, 200000, now); was_gh = m.mode == "GH"; }
            const std::string_view before_mode = m.mode;
            foes[0].alive = true;
            monster_update(*data, here, m, foes, think_rng, now, 0.04f, crowd, shots, world.fight.monsters);
            if (!have_swing && m.mode != before_mode && (m.mode == "A1" || m.mode == "A2")) { swing = frame; have_swing = true; }
            if (!have_end && was_gh && m.mode != "GH") { gh_end = frame; have_end = true; }
        }
        const auto out = [](bool have, int v) { return have ? std::to_string(v) : std::string("-"); };
        std::printf("%d %s %s %s\n", id, out(have_swing, swing).c_str(), out(have_end, gh_end).c_str(), was_gh ? "gh" : "nogh");
        body.diff = saved_diff;
    }
    return 0;
} catch (const std::exception& error) {
    std::fprintf(stderr, "%s\n", error.what());
    return 1;
}
