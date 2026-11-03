// SPDX-License-Identifier: GPL-3.0-or-later
// dmgroll: the d2d side of tools/emu/mdmg.py. Calls d2d's own rules::monster_blow (components/rules/combat.hpp of the tree it
// is built against, base or head) on the case file "A1min A1max seed_low seed_high" per line, the swing sure (no to-hit,
// block or dodge draws: the first draw is the physical roll), and prints "fine seed_low seed_high" per case: fine = the
// physical in 256ths (damage x 256 + frac where the tree has Taken::frac), seed = the attacker's seed after.
// Built by hand by tools/emu/mdmg_build.sh, never part of upstream.
#include <combat.hpp>

#include <cstdio>

namespace R = d2d::rules;

// Taken::frac where the tree has it (head), else 0 (base).
static long frac_of(const auto& taken) {
    if constexpr (requires { taken.frac; }) return taken.frac;
    else return 0;
}

int main(int argc, char** argv) {
    if (argc < 2) return 2;
    std::FILE* in = std::fopen(argv[1], "r");
    if (!in) return 2;
    unsigned lo = 0, hi = 0, sl = 0, sh = 0;
    while (std::fscanf(in, "%u %u %u %u", &lo, &hi, &sl, &sh) == 4) {
        R::MonStats stats;
        stats.a1_min = int(lo);
        stats.a1_max = int(hi);
        R::Rng seed;
        seed.low = sl;
        seed.high = sh;
        const R::Fighter defender;
        const auto taken = R::monster_blow(defender, 1, R::Motion::standing, stats, false, seed, false, true);
        const long fine = long(taken.damage) * 256 + frac_of(taken);
        std::printf("%ld %u %u\n", fine, seed.low, seed.high);
    }
    return 0;
}
