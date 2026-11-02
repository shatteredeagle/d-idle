// verifyskmelee3: d2d side. d2d's REAL rules::player_blow (combat.hpp of the tree on the include path) on the cases of
// vs3_game.py; the Fighter is filled as make_fighter fills it (low/high in 256ths, gear pct floored at -90).
// Build with -DVS3_HEAD for the candidate (Fighter::blunt / undead_pct, Target::undead exist there).
// Prints: id damage frac phys
#include "combat.hpp"
#include <algorithm>
#include <cstdio>
int main(int argc, char** argv) {
    FILE* in = std::fopen(argv[1], "r");
    const int res0 = argc > 2 ? std::atoi(argv[2]) : 0;
    long id; unsigned s0, s1; int armed, bm, bM, edg, dp, undead, blunt, s122, eskill, srcdam, str, dex, sb, db, eda, fine;
    while (std::fscanf(in, "%ld %u %u %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d", &id, &s0, &s1, &armed, &bm, &bM, &edg, &dp, &undead, &blunt,
                       &s122, &eskill, &srcdam, &str, &dex, &sb, &db, &eda, &fine) == 19) {
        d2d::rules::Fighter f{};
        std::int64_t lo = armed ? std::int64_t(bm) * (100 + edg) / 100 : 1, hi = armed ? std::int64_t(bM) * (100 + edg) / 100 : 2;
        lo <<= 8; if (lo < 256) lo = 256; hi = std::max<std::int64_t>(hi << 8, lo + 256);
        f.phys_lo = int(lo); f.phys_hi = int(hi);
        f.phys_pct = int(std::max<std::int64_t>(dp + (armed ? str * sb / 100 + dex * db / 100 : 0), -90));
        f.ar_base = 100000; f.attack_rating = 100000;
#ifdef VS3_HEAD
        f.blunt = blunt != 0; f.undead_pct = s122;
#endif
        d2d::rules::Target t{};
        t.res[0] = res0; t.level = 1; t.armor_class = 0;
#ifdef VS3_HEAD
        t.undead = undead != 0;
#endif
        d2d::rules::Swing sw{}; sw.ed_pct = eskill; sw.srcdam = srcdam;
        d2d::rules::Rng rng; rng.low = s0; rng.high = s1;
        const auto b = d2d::rules::player_blow(f, t, 1, rng, sw);
        std::printf("%ld %d %d %d %d\n", id, b.hit ? 1 : 0, b.damage, b.frac, b.phys);
    }
}
