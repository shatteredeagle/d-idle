// verifydmg3 (third independent check of DMGROLL r3): d2d's own rules::monster_blow, absorb_blows (and r3's
// shield_blows) of one tree, compiled header-only (w3_build.sh). -DW3_BASE (3e073888), -DW3_R2 (b9ecb415), else r3 (2df7d1eb).
// stdin: "min max seedlo seedhi pct flat pool256 exact espct esper16 mana256" (espct 0: no Energy Shield).
// stdout: "life256 pool256 seedlo seedhi mana256".
// The glue after monster_blow mirrors each tree's Fight::absorb (fight/states.cpp): Bone Armor, then Energy Shield,
// one blow a frame. exact = 1: the fixed-record callers (Corpse Explosion, Diablo's cold touch) as each tree's ai.cpp
// calls them (r3 passes exact = true; base and r2 have no such parameter).
#include "combat.hpp"

#include <algorithm>
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
#if defined(W3_BASE)
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
#elif defined(W3_R2)
        (void)exact;
        const Taken t = monster_blow(def, 1, Motion::standing, att, false, rng, false, true);
        int pool = pool256;
        std::array<StruckBlow, 1> blows{ StruckBlow{ t.damage, 0, false, t.physical, t.elemental, false, t.frac } };
        long long fine = ((long long)t.damage << 8) + t.frac;
        if (fine > 0 && pool > 0) fine -= absorb_blows(blows, pool, false);
        if (espct > 0 && fine > 0) {
            const long long part = fine * espc / 100;
            const long long can = std::min<long long>(part, mana * 16 / per16);
            mana -= can * per16 / 16;
            fine -= can;
        }
        std::printf("%lld %d %u %u %lld\n", fine, pool, rng.low, rng.high, mana);
#else
        const Taken t = monster_blow(def, 1, Motion::standing, att, false, rng, false, true, exact != 0);
        int pool = pool256;
        std::array<StruckBlow, 1> blows{ StruckBlow{ t.damage, 0, false, t.physical, t.elemental, false, t.frac, t.raw, def.dr_pct, def.dr_flat } };
        const auto sum = [&] { long long s = 0; for (const auto& b : blows) s += ((long long)b.damage << 8) + b.frac; return s; };
        long long fine = ((long long)t.damage << 8) + t.frac;
#ifdef W3_NOGUARD
        if (true) {   // diagnosis only: Fight::absorb without its `fine <= 0 -> return 0` guard
#else
        if (fine > 0) {   // Fight::absorb: `if (fine <= 0) return 0;` on the REDUCED hit, before any state runs
#endif
            const long long before = sum();
            if (pool > 0) (void)absorb_blows(blows, pool, false);
            if (espct > 0) { std::int64_t m = mana; (void)shield_blows(blows, espc, per16, m); mana = m; }
            fine += sum() - before;
        }
        std::printf("%lld %d %u %u %lld\n", fine, pool, rng.low, rng.high, mana);
#endif
    }
}
