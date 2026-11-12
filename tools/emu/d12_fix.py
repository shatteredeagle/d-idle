# decomp12 (worker host): apply the warp-arrival fix candidate to the scratch tree (travel.cpp / world.hpp).
import sys
T = sys.argv[1]
h = open(T + "/components/game/world.hpp").read()
anchor = "    void arrive(const Level* destination, float arrive_x, float arrive_y, const char* how);\n"
add = anchor + """    // Where a warp puts the player: FUN_00554ea0's spot round the other side's warp
    // unit (Level::warp_spot: FUN_0064e7b0 -> FUN_0064dea0, the subtile itself when
    // free, else the nearest by |dx| + |dy| in the first ring that has one, rings to
    // 49), as a cell position. Level::nearest_free scans each ring in another order
    // and gives up at another ring, so it lands on another subtile.
    [[nodiscard]] static std::pair<float, float> warp_arrival(const Level& level, float unit_x, float unit_y) {
        const auto spot = level.warp_spot(int(std::floor(unit_x * 5)), int(std::floor(unit_y * 5)));
        if (!spot) return level.nearest_free(unit_x, unit_y);
        return { (float(spot->first) + 0.5f) / 5, (float(spot->second) + 0.5f) / 5 };
    }
"""
assert h.count(anchor) == 1
open(T + "/components/game/world.hpp", "w").write(h.replace(anchor, add))
c = open(T + "/components/game/world/travel.cpp").read()
old = '        arrive(destination, arrive_x, arrive_y, "a warp");\n'
new = '        const auto [land_x, land_y] = warp_arrival(*destination, arrive_x, arrive_y);\n        arrive(destination, land_x, land_y, "a warp");\n'
assert c.count(old) == 1
open(T + "/components/game/world/travel.cpp", "w").write(c.replace(old, new))
