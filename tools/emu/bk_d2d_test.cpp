// SPDX-License-Identifier: GPL-3.0-or-later
// decomp11 (fid-monster-block): the d2d side of tools/emu/bk_mon.py. For every loaded monster type: its can_block (MonStats2 mBL) and
// ToBlock of the three difficulties, as d2d's real loader reads them. d2d's block chance of a monster defender is
// `can_block ? to_block : 0` (ai.hpp:373), no cap. Built by hand against a scratch build (tools/emu/bk_run.sh), never part of upstream.
#include <ai.hpp>
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <montypes.hpp>

#include <cstdio>
#include <cstdlib>
#include <filesystem>

using namespace d2d::game;

int main() try {
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(fs::path(mpq_dir), patch ? fs::path(patch) : fs::path{}, 0x1234);
    if (!data || !data->patched) { std::printf("FAIL: no patched data\n"); return 1; }
    for (const auto& [name, index] : data->monsters.by_id) {
        const auto& t = data->monsters.types[index];
        std::printf("%s %d %d %d %d\n", name.c_str(), t.can_block ? 1 : 0, t.diff[0].to_block, t.diff[1].to_block, t.diff[2].to_block);
    }
    return 0;
} catch (const std::exception& ex) { std::printf("FAIL %s\n", ex.what()); return 1; }
