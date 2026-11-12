# prwarp scratch: the claim commit's edits on the fork worktree (arg 1 = its root).
import os
import sys

root = sys.argv[1]


def edit(rel, old, new):
    p = os.path.join(root, rel)
    s = open(p, encoding="utf-8").read()
    assert s.count(old) == 1, (rel, old[:60])
    open(p, "w", encoding="utf-8").write(s.replace(old, new))


edit("components/game/world.hpp", '''    void arrive(const Level* destination, float arrive_x, float arrive_y, const char* how);
''', '''    void arrive(const Level* destination, float arrive_x, float arrive_y, const char* how);
    // Where a warp puts the player: the free spot nearest the other side's warp unit, as
    // FUN_005550b0 -> FUN_0064e7b0 -> FUN_0064dea0 find it (0x555144 to 0x555157; FUN_00554ea0
    // searches again from that spot and finds it free): Level::warp_spot, the subtile itself
    // when free, else the least |dx| + |dy| in the first ring that has one, rings 1 to 49.
    // Level::nearest_free scans each ring in another order and goes out to ring 199, so it lands
    // on another subtile. No spot: nullopt, the warp is not taken (FUN_00554ea0 returns 0).
    [[nodiscard]] static std::optional<std::pair<float, float>> warp_arrival(const Level& level, float unit_x, float unit_y) {
        const auto spot = level.warp_spot(int(std::floor(unit_x * 5)), int(std::floor(unit_y * 5)));
        if (!spot) return std::nullopt;
        return std::pair{ (float(spot->first) + 0.5f) / 5, (float(spot->second) + 0.5f) / 5 };
    }
''')

edit("components/game/world/travel.cpp", '''        arrive(destination, arrive_x, arrive_y, "a warp");
''', '''        const auto landing = warp_arrival(*destination, arrive_x, arrive_y);
        if (!landing) {
            d2d::log::info("no free spot within 49 subtiles of the warp unit in level {}: the warp is not taken", destination->id);
            return;
        }
        arrive(destination, landing->first, landing->second, "a warp");
''')

edit("docs/research/re/drlg.md", "wired up; warp arrival traced, not diffed |", "wired up; warp arrival diffed (`tools/emu/sk_tele_spot.py warp`) |")

edit("docs/research/re/drlg.md", '''linked class, with that record. The player is put at the free spot
nearest that unit (FUN_0064e7b0, collision 0x1c09, FUN_00554ea0), then
told to walk (FUN_005809d0(0, 2, ..)) to that spot + the record's
ExitWalkX / ExitWalkY''', '''linked class, with that record. The free spot is searched twice. First
FUN_005550b0 calls FUN_0064e7b0(room, &xy, size 2, 0x1c09, flag 1)
(0x555144 to 0x555157): with the flag set, no spot returns the unit's own
subtile; this is the spot the player lands on. Then FUN_00554ea0 searches
from that spot (0x5551bc, flag 0), finds it free, and places the player;
when it finds none (flag 0 returns 0) the warp is not taken (0x5551c3).
FUN_0064dea0 does both: the start subtile when it is in a room and free,
else rings 1 to 49 (it goes on while ring + 1 < 50), in a ring the free
subtile of least |dx| + |dy|, a tie to the first in scan order (columns
x - r and x + r row by row, left first, corners included; then rows
y - r and y + r column by column, top first; a later one only when
strictly nearer). The player is then
told to walk (FUN_005809d0(0, 2, ..)) to that spot + the record's
ExitWalkX / ExitWalkY''')

edit("docs/research/re/drlg.md", '''wall orientations 10 / 11, FUN_0066e160); act 1's are all 'b'. d2d:
`Level::Warp::unit_x/unit_y`, `World::use_warp`.''', '''wall orientations 10 / 11, FUN_0066e160); act 1's are all 'b'. d2d:
`Level::Warp::unit_x/unit_y`, `World::use_warp`, `World::warp_arrival`
(`Level::warp_spot`; none, the warp is not taken). Differential:
`tools/emu/sk_tele_spot.py warp` (README), `tests/game/test_warp_arrival.cpp`.''')
