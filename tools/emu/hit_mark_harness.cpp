// SPDX-License-Identifier: GPL-3.0-or-later
// d2d's missile blow mark (`Blow::elemental`) on tools/emu/hit_mark.py's cases: the REAL d2d::rules::missile_blow
// from components/rules/combat.hpp (no copy of its lines), one MissileDamage per case, a plain target with the
// case's resistance for the element (physical 0), the case's pierce. Prints `id mark` (1: blow.elemental).
//
//   g++ -O2 -std=c++20 -I<d2d>/components/rules -I<d2d>/components -I<dir of d2s_items.hpp> \
//       -o hit_mark_harness hit_mark_harness.cpp && ./hit_mark_harness < cases.tsv > d2d.tsv
#include "combat.hpp"
#include <cstdio>

int main() {
    char line[512];
    while (std::fgets(line, sizeof line, stdin)) {
        long id, etype, plo, phi, elo, ehi, elen, res, pierce;
        unsigned long seed;
        if (line[0] == '#' || std::sscanf(line, "%ld %ld %ld %ld %ld %ld %ld %ld %ld %lu", &id, &etype, &plo, &phi, &elo, &ehi, &elen, &res, &pierce, &seed) != 10) continue;
        d2d::rules::MissileDamage damage{ .phys_lo = int(plo), .phys_hi = int(phi), .etype = int(etype), .elo = int(elo), .ehi = int(ehi), .elen = int(elen), .srcdam = 0 };
        d2d::rules::Target target;
        static constexpr int kRes[5] = { 2, 3, 4, 5, 1 };   // element -> Target::res index, as missile_blow reads it
        if (etype >= 0 && etype < 5) target.res[std::size_t(kRes[etype])] = int(res);
        std::array<int, 4> pierces{};
        if (etype >= 0 && etype < 4) pierces[std::size_t(etype)] = int(pierce);
        d2d::rules::Rng rng{ std::uint32_t(seed) };
        const d2d::rules::Blow blow = d2d::rules::missile_blow(damage, target, pierces, rng);
        std::printf("%ld %d\n", id, blow.elemental ? 1 : 0);
    }
}
