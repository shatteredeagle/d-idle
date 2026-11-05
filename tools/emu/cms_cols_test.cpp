// SPDX-License-Identifier: GPL-3.0-or-later
// COVMONSKILLS: the d2d side of the column differential (tools/emu/cms_cols.py). Hand-built against a scratch
// upstream build (tools/emu/cms_build.sh), never part of upstream.
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <skills.hpp>

#include <cstdio>
#include <cstdlib>
#include <exception>
#include <filesystem>

using namespace d2d::game;

int main() try {
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(fs::path(mpq_dir), patch ? fs::path(patch) : fs::path{}, 0x1234);
    if (!data || !data->patched) { std::printf("FAIL: no patched data\n"); return 1; }
    const auto emit = [&](int id) {
        const auto* s = data->skills.get(id);
        if (!s) { std::printf("%d missing\n", id); return; }
        std::printf("%d %d %d %d %d %d %d %d %d %d %d %d %d %d\n", id, int(s->srvstfunc), int(s->srvdofunc), int(s->prgfunc[0]), int(s->prgfunc[1]), int(s->prgfunc[2]),
                    s->mana, s->lvlmana, s->manashift, s->minmana, int(!s->srvmissile.empty()), int(!s->srvmissilea.empty()), int(!s->srvmissileb.empty()), int(!s->srvmissilec.empty()));
    };
    for (int id = 0; id <= 5; ++id) emit(id);
    for (int id = 156; id <= 220; ++id) emit(id);
    for (int id = 281; id <= 356; ++id) emit(id);
    return 0;
} catch (const std::exception& ex) { std::printf("FAIL: %s\n", ex.what()); return 1; }
