// dmgroll3 (re-run of the second verifier's harness on DMGROLL r3): d2d's own rules::monster_blow and rules::absorb_blows of one tree,
// compiled header-only (v3_build.sh). -DV2_BASE for base 3e073888 (whole points, no Taken::frac, pool in whole points); -DV3_R3 for r3 (Taken::raw, exact flag).
// stdin lines: "min max seedlo seedhi pct flat pool256 exact"; stdout: "life256 pool256 seedlo seedhi".
// exact = 1: the fixed-record callers (Corpse Explosion, Diablo's cold touch) as d2d's ai.cpp calls them (missile false, sure true; r3 also passes exact).
// The glue after monster_blow mirrors Fight::absorb's Bone Armor branch of each tree (fight/states.cpp), one blow a frame.
#include "combat.hpp"

#include <cstdio>

int main() {
    using namespace d2d::rules;
    int mn, mx, pct, flat, pool256, exact;
    unsigned lo, hi;
    while (std::scanf("%d %d %u %u %d %d %d %d", &mn, &mx, &lo, &hi, &pct, &flat, &pool256, &exact) == 8) {
        Fighter def{};
        def.dr_pct = pct;
        def.dr_flat = flat;
        MonStats att{};
        att.a1_min = mn;
        att.a1_max = mx;
        att.level = 1;
        for (auto& e : att.elements) e.type = -1;
        Rng rng;
        rng.low = lo;
        rng.high = hi;
        #ifdef V3_R3
        const Taken t = monster_blow(def, 1, Motion::standing, att, false, rng, false, true, exact != 0);
#else
        const Taken t = monster_blow(def, 1, Motion::standing, att, false, rng, false, true);
#endif
#ifdef V2_BASE
        int pool = pool256 >> 8;   // base keeps the pool in whole points (start_state: calc >> 8)
        StruckBlow b{ t.damage, 0, false, t.physical, t.elemental, false };
        std::array<StruckBlow, 1> blows{ b };
        int damage = t.damage;
        if (pool > 0) damage -= absorb_blows(blows, pool, false);
        std::printf("%lld %d %u %u\n", (long long)damage * 256, pool * 256, rng.low, rng.high);
#elif defined(V3_R3)
        int pool = pool256;   // Foe::take builds the blow with raw and the defender's reduction; Fight::absorb sums the blows before and after
        StruckBlow b{ t.damage, 0, false, t.physical, t.elemental, false, t.frac, t.raw, pct, flat };
        std::array<StruckBlow, 1> blows{ b };
        long long fine = ((long long)t.damage << 8) + t.frac;
        const long long before = fine;
        if (pool > 0) (void)absorb_blows(blows, pool, false);
        fine += ((long long)blows[0].damage << 8) + blows[0].frac - before;
        std::printf("%lld %d %u %u\n", fine, pool, rng.low, rng.high);
#else
        int pool = pool256;
        StruckBlow b{ t.damage, 0, false, t.physical, t.elemental, false, t.frac };
        std::array<StruckBlow, 1> blows{ b };
        long long fine = ((long long)t.damage << 8) + t.frac;
        if (pool > 0) fine -= absorb_blows(blows, pool, false);
        std::printf("%lld %d %u %u\n", fine, pool, rng.low, rng.high);
#endif
    }
}
