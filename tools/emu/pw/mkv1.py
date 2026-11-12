# prwarp scratch: make the oracle commit's test (before World::warp_arrival exists) from the final one.
import os
d = os.path.dirname(os.path.abspath(__file__))
s = open(os.path.join(d, "test_warp_arrival.cpp")).read()
s = s.replace("#include <string_view>\n#include <utility>", "#include <string_view>\n#include <tuple>\n#include <utility>")
s = s.replace("return std::tuple{ world.level == destination, subtile(world.actor->player.x) - subtile(back->unit_x), subtile(world.actor->player.y) - subtile(back->unit_y), destination, source };", "return std::tuple{ world.level == destination, subtile(world.actor->player.x) - subtile(back->unit_x), subtile(world.actor->player.y) - subtile(back->unit_y) };")
s = s.replace("const auto [reached, dx, dy, destination, source] = take(from, to);", "const auto [reached, dx, dy] = take(from, to);")
s = s.replace("        (void)destination; (void)source;\n", "")
open(os.path.join(d, "test_warp_arrival.cpp"), "w").write(s)
v = s
v = v.replace('''    const auto landing = World::warp_arrival(level, centre(x), centre(y));
    if (!landing) return std::nullopt;
    return std::pair{ subtile(landing->first), subtile(landing->second) };''', '''    const auto landing = level.nearest_free(centre(x), centre(y));      // World::arrive's search
    return std::pair{ subtile(landing.first), subtile(landing.second) };''')
a = v.index("    // Hand cases;")
v = v[:a] + '''    assert(arrival(window({}), 100, 100) == Spot(std::pair(100, 100)));   // a free unit subtile: the unit's own
    std::printf("OK: --warp-arrival CASES prints where d2d puts a warp's arrival (the oracle: tools/emu/sk_tele_spot.py warp)\\n");
    return 0;
} catch (const std::exception& error) {
    std::fprintf(stderr, "test_warp_arrival: %s\\n", error.what());
    return 1;
}
'''
for inc in ["#include <character.hpp>\n", "#include <character_store.hpp>\n", "#include <gamedata_load.hpp>\n", "#include <rules.hpp>\n", "#include <cstdlib>\n", "#include <filesystem>\n", "#include <tuple>\n"]:
    v = v.replace(inc, "")
v = v.replace("namespace fs = std::filesystem;\n", "")
v = v.replace("//   no argument: hand cases (answers from the same oracle) and, with the game data, the\n//   whole use_warp on seed 5 (SKIPs without D2_MPQ_DIR and the 1.14d patch installer).\n", "//   no argument: a free unit subtile is the arrival.\n")
v = v.replace("Words square(int x, int y, int half) {\n    Words words;\n    for (int row = y - half; row <= y + half; ++row)\n        for (int column = x - half; column <= x + half; ++column) words.push_back({ column, row, 1 });\n    return words;\n}\n\n", "")
open(os.path.join(d, "test_warp_arrival.v1.cpp"), "w").write(v)
