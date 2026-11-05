// SPDX-License-Identifier: GPL-3.0-or-later
// dmgroll r2: the d2d side of tools/emu/mdmg2.py. Calls d2d's own rules::monster_blow (the reduction included) and
// rules::absorb_blows (Bone Armor on the physical) from the headers of the tree it is built against, on a frame of
// up to three sure blows from one attacker seed, like Fight::world / Fight::absorb: each blow is rolled and reduced
// by monster_blow, then the pool takes from the blows in landing order. Case line (one frame):
//   id pool_fine pct flat n seed_low seed_high min1 max1 min2 max2 min3 max3
// Output per frame: "id total_fine pool_fine seed_low seed_high" (total = what the player's life loses, 256ths).
// -DMDMG_BASE builds against the base tree (whole points: no Taken::frac, no StruckBlow::frac, pool in whole points);
// -DMDMG_R3 against the third round (raw physical kept, absorb first, then the reduction); -DMDMG_R1 against the first candidate (6f9ded7c): the fraction exists but the pool is whole points and a partial absorb drops it.
// Built by hand by tools/emu/mdmg_build.sh-style commands (mdmg2_build.sh), never part of upstream.
#include <combat.hpp>

#include <cstdio>
#include <vector>

namespace R = d2d::rules;

int main(int argc, char** argv) {
    if (argc < 2) return 2;
    std::FILE* in = std::fopen(argv[1], "r");
    if (!in) return 2;
    long id = 0, pool = 0, pct = 0, flat = 0, n = 0, mm[6] = {};
    unsigned sl = 0, sh = 0;
    while (std::fscanf(in, "%ld %ld %ld %ld %ld %u %u %ld %ld %ld %ld %ld %ld", &id, &pool, &pct, &flat, &n, &sl, &sh, &mm[0], &mm[1], &mm[2], &mm[3], &mm[4], &mm[5]) == 13) {
        R::Rng seed;
        seed.low = sl;
        seed.high = sh;
        R::Fighter defender;
        defender.dr_pct = int(pct);
        defender.dr_flat = int(flat);
        std::vector<R::StruckBlow> blows;
        for (long i = 0; i < n; ++i) {
            R::MonStats stats;
            stats.a1_min = int(mm[2 * i]);
            stats.a1_max = int(mm[2 * i + 1]);
            const auto taken = R::monster_blow(defender, 1, R::Motion::standing, stats, false, seed, false, true);
#if defined(MDMG_BASE)
            blows.push_back({ taken.damage, 2, false, taken.physical, 0, false });
#elif defined(MDMG_R3)
            blows.push_back({ taken.damage, 2, false, taken.physical, 0, false, taken.frac, taken.raw, int(pct), int(flat) });   // Foe::take
#else
            blows.push_back({ taken.damage, 2, false, taken.physical, 0, false, taken.frac });
#endif
        }
#if defined(MDMG_R1)
        int p = int(pool >> 8);
        long before = 0, fracs = 0, after = 0;
        for (const auto& blow : blows) { before += blow.damage; fracs += blow.frac; }
        if (p > 0) (void)R::absorb_blows(blows, p, false);
        for (const auto& blow : blows) after += blow.damage;
        const long total = after * 256 + (after == before ? fracs : 0);   // tick.cpp: the fraction is dropped when a state absorbed
        pool = long(p) * 256;
#elif defined(MDMG_BASE)
        int p = int(pool >> 8);
        if (p > 0) (void)R::absorb_blows(blows, p, false);
        long total = 0;
        for (const auto& blow : blows) total += long(blow.damage) * 256;
        pool = long(p) * 256;
#else   // r2 and r3: the pool in 256ths; r3's blows carry raw / pct / flat, so absorb_blows takes before the reduction
        int p = int(pool);
        if (p > 0) (void)R::absorb_blows(blows, p, false);
        long total = 0;
        for (const auto& blow : blows) total += long(blow.damage) * 256 + blow.frac;
        pool = p;
#endif
        std::printf("%ld %ld %ld %u %u\n", id, total, pool, seed.low, seed.high);
    }
    return 0;
}
