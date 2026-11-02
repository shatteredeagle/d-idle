// SPDX-License-Identifier: GPL-3.0-or-later
// test-links: rules
// `--melee CASES`: the player's plain Attack through d2d's REAL
// rules::make_fighter and rules::player_blow, for the cases
// tools/emu/sk_melee2.py writes (and runs game.exe's FUN_0057d9b0,
// FUN_0057b660 and FUN_0057b420 on). One line out per case line in:
//   H id s0 s1 ar pct mastery bonus def mdef alvl dlvl weapon
//     -> `H id hit`            (to-hit: the Rng seeded s0 / s1, the first draw)
//   M|F|U|O id d0 d1 weapon bm bM ed fmin fmax n111 dp str dex sb db mastery eskill off g0 g1 [undead blunt s122]
//     -> `T id hit phys damage frac`   (phys = Blow::phys, whole points before resistance; damage and frac =
//        the blow's whole points and 256ths against a target with no resistance: damage x 256 + frac is the
//        physical in 256ths where the build keeps them, else frac is 0)
// d0 / d1 seeds d2d's Rng; its first draw is the to-hit roll, its second
// the damage roll, which game.exe makes as its first draw from the state
// one step later (the oracle writes that state); `hit 0` lines (the
// to-hit roll % 100 >= 95, the cap) carry no damage.
// U cases carry the undead, Blunt and stat 122 terms: the build under test gets them through Target::undead,
// Fighter::blunt and Fighter::undead_pct when it has those members (the same source builds against a base
// without them, which then ignores the terms). O cases carry the off-weapon term only on the game side:
// `sum` 17 / 18 is read for kicks and Smite only.
// (Our copy; upstream path would be tests/game/test_melee.cpp.)

#include <combat.hpp>
#include <d2s_items.hpp>
#include <rules.hpp>

#include <cstdint>
#include <cstdio>
#include <exception>
#include <fstream>
#include <sstream>
#include <string>
#include <string_view>

using namespace d2d::rules;

int main(int argc, char** argv) try {
    if (argc < 3 || std::string_view(argv[1]) != "--melee") {
        std::printf("SKIP: an oracle mode, run `test_melee --melee CASES` (see tools/emu/sk_melee2.py)\n");
        return D2D_TEST_SKIPPED;
    }
    namespace s = d2d::d2s;
    Tables tables;
    d2d::d2s::Item weapon;
    weapon.code = "tst";
    std::ifstream cases(argv[2]);
    std::string line;
    while (std::getline(cases, line)) {
        std::istringstream in(line);
        std::string tag;
        long long id = 0;
        if (!(in >> tag >> id)) continue;
        StatSum sum{}, wsum{};
        d2d::d2s::Stats stats;
        Target target;
        Swing swing;
        if (tag == "H") {
            std::uint32_t s0 = 0, s1 = 0;
            long long ar = 0, pct = 0, mastery = 0, bonus = 0, def = 0, mdef = 0, alvl = 1, dlvl = 1, armed = 0;
            if (!(in >> s0 >> s1 >> ar >> pct >> mastery >> bonus >> def >> mdef >> alvl >> dlvl >> armed)) continue;
            tables.item_base["tst"] = { .mindam = 1, .maxdam = 2 };
            stats.values[s::kLevel] = alvl; stats.values[s::kDex] = 7; stats.values[s::kStr] = 10;
            sum[s::kToHit] = ar; sum[s::kToHitPercent] = pct;
            if (armed) sum[s::kPassiveMasteryMeleeToHit] = mastery;   // the caller adds a mastery only for its weapon type
            const auto fighter = make_fighter(tables, armed ? &weapon : nullptr, nullptr, sum, wsum, stats, {}, 0, {});
            target.armor_class = int(def + mdef); target.level = int(dlvl);
            swing.ar_pct = int(bonus);                                // stat 325 (FUN_0056f070 passes it as the bonus)
            Rng rng; rng.low = s0; rng.high = s1;
            std::printf("H %lld %d\n", id, player_blow(fighter, target, int(alvl), rng, swing).hit ? 1 : 0);
            continue;
        }
        std::uint32_t d0 = 0, d1 = 0;
        long long armed = 0, bm = 0, bM = 0, ed = 0, fmin = 0, fmax = 0, n111 = 0, dp = 0, str = 0, dex = 0, sb = 0, db = 0, mastery = 0, eskill = 0, off = 0;
        if (!(in >> d0 >> d1 >> armed >> bm >> bM >> ed >> fmin >> fmax >> n111 >> dp >> str >> dex >> sb >> db >> mastery >> eskill >> off)) continue;
        long long g0 = 0, g1 = 0, undead = 0, blunt = 0, s122 = 0;
        in >> g0 >> g1;
        if (tag == "U") in >> undead >> blunt >> s122;
        tables.item_base["tst"] = { .mindam = int(bm), .maxdam = int(bM), .str_bonus = int(sb), .dex_bonus = int(db) };
        stats.values[s::kLevel] = 1; stats.values[s::kStr] = str; stats.values[s::kDex] = dex;
        wsum[s::kMaxDamagePercent] = wsum[s::kMinDamagePercent] = ed;
        sum[s::kMaxDamagePercent] = sum[s::kMinDamagePercent] = ed + off;   // `sum` is everything worn, the weapon too
        sum[s::kMinDamage] = fmin; sum[s::kMaxDamage] = fmax; sum[s::kNormalDamage] = n111;
        sum[s::kDamagePercent] = dp; sum[s::kPassiveMasteryMeleeDamage] = mastery;
        sum[122] = s122;                                              // item_undeaddamage_percent (numeric: the base build has no name for it)
        tables.types["blun"] = {}; tables.types["mace"] = { .equiv = { "blun", "" } }; tables.types["swor"] = {};
        tables.item_info["tst"].type = blunt ? "mace" : "swor";      // the Blunt chain is ItemTypes' (mace -> blun)
        const auto fighter = make_fighter(tables, armed ? &weapon : nullptr, nullptr, sum, wsum, stats, {}, 0, {});
        [&](auto& t) { if constexpr (requires { t.undead; }) t.undead = undead != 0; }(target);   // a base without the member ignores the term
        target.armor_class = 0; target.level = 1;                    // 95 % to hit: the first draw decides
        target.hit_points = target.max_hp = 1 << 30;
        swing.ed_pct = int(eskill);
        Rng rng; rng.low = d0; rng.high = d1;
        const auto blow = player_blow(fighter, target, 1, rng, swing);
        std::printf("%s %lld %d %d %d %d\n", tag.c_str(), id, blow.hit ? 1 : 0, blow.phys, blow.damage, blow.frac);
    }
    return 0;
} catch (const std::exception& error) {
    std::fprintf(stderr, "test_melee: %s\n", error.what());
    return 1;
}
