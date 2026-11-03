// verifydmg4: my own header glue of one tree's Fight::absorb after rules::monster_blow (the mirror; v4_probe.cpp calls
// the real one). -DV4_BASE (3e073888, the glue of verifydmg3's w3_d2d.cpp) or -DV4_R4 (9ea2afe9, read from its states.cpp).
// stdin: "min max seedlo seedhi pct flat pool256 exact espct esper16 mana256" (espct 0: no Energy Shield).
// stdout: "life256 pool256 seedlo seedhi mana256".
#include "combat.hpp"

#include <algorithm>
#include <array>
#include <cstdint>
#include <cstdio>

int main() {
    using namespace d2d::rules;
    int mn, mx, pct, flat, pool256, exact, espct, esper16;
    unsigned lo, hi;
    long long mana;
    while (std::scanf("%d %d %u %u %d %d %d %d %d %d %lld", &mn, &mx, &lo, &hi, &pct, &flat, &pool256, &exact, &espct, &esper16, &mana) == 11) {
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
        const int espc = std::clamp(espct, 0, 95), per16 = std::max(esper16, 1);
#if defined(V4_BASE)
        (void)exact;
        const Taken t = monster_blow(def, 1, Motion::standing, att, false, rng, false, true);
        int pool = pool256 >> 8;   // base: start_state keeps whole points
        std::array<StruckBlow, 1> blows{ StruckBlow{ t.damage, 0, false, t.physical, t.elemental, false } };
        int damage = t.damage;
        if (damage > 0 && pool > 0) damage -= absorb_blows(blows, pool, false);
        if (espct > 0 && damage > 0) {
            const int part = damage * espc / 100;
            const int can = int(std::min<std::int64_t>(part, (mana >> 8) * 16 / per16));
            mana -= (std::int64_t(can) * per16 / 16) << 8;
            damage -= can;
        }
        std::printf("%lld %d %u %u %lld\n", (long long)damage * 256, pool * 256, rng.low, rng.high, mana);
#else
        const Taken t = monster_blow(def, 1, Motion::standing, att, false, rng, false, true, exact != 0);
        int pool = pool256;
        std::array<StruckBlow, 1> blows{ StruckBlow{ t.damage, 0, false, t.physical, t.elemental, false, t.frac, t.raw, def.dr_pct, def.dr_flat } };
        const auto sum = [&] { long long s = 0; for (const auto& b : blows) s += ((long long)b.damage << 8) + b.frac; return s; };
        long long fine = ((long long)t.damage << 8) + t.frac;
        // r4: `if (fine <= 0 && foe.blows.empty()) return 0;` -- a landed hit always has its blow
        const long long before = sum();
        if (pool > 0) (void)absorb_blows(blows, pool, false);
        fine += sum() - before;
        if (espct > 0 && fine > 0) {   // r4: the pre-change shield on the reduced total, in 256ths
            const long long part = fine * espc / 100;
            const long long can = std::min<long long>(part, mana * 16 / per16);
            mana -= can * per16 / 16;
            fine -= can;
        }
        std::printf("%lld %d %u %u %lld\n", fine, pool, rng.low, rng.high, mana);
#endif
    }
}
