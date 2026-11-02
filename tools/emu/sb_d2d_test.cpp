// SPDX-License-Identifier: GPL-3.0-or-later
// sb_d2d_test (card smiteblock): d2d's real `player_blow` (upstream components/rules/combat.hpp) with swing.smite against a
// target with block chance b: how often does the Smite blow land (blow.hit) and how often is it blocked (blow.blocked)?
//   sb_d2d_test CASES SEED   -> one line per block chance b in (0, 25, 50, 100): `D b cases hit blocked plain_hit plain_blocked`
// The Fighter is built field by field (shield damage 3 .. 5, attack rating 5000, target armor 0): no table is read.
#include <combat.hpp>
#include <rules.hpp>

#include <cstdint>
#include <cstdio>
#include <cstdlib>

using namespace d2d::rules;

int main(int argc, char** argv) {
    const long long n = argc > 1 ? std::atoll(argv[1]) : 2000;
    std::uint64_t seed = argc > 2 ? std::strtoull(argv[2], nullptr, 10) : 7;
    auto next = [&] { seed = seed * 6364136223846793005ull + 1442695040888963407ull; return std::uint32_t(seed >> 32); };
    for (const int b : { 0, 25, 50, 100 }) {
        long long hit = 0, blocked = 0, plain_hit = 0, plain_blocked = 0;
        for (long long i = 0; i < n; ++i) {
            Fighter fighter;
            fighter.ar_base = 5000;
            fighter.smite_lo = 3; fighter.smite_hi = 5;
            Target target;
            target.armor_class = 0; target.level = 1; target.block = b;
            target.hit_points = target.max_hp = 1 << 30;
            const std::uint32_t s0 = next(), s1 = next();
            for (int smite = 1; smite >= 0; --smite) {
                Swing swing;
                swing.smite = smite != 0;
                Rng rng; rng.low = s0; rng.high = s1;
                const auto blow = player_blow(fighter, target, 30, rng, swing);
                (smite ? hit : plain_hit) += blow.hit ? 1 : 0;
                (smite ? blocked : plain_blocked) += blow.blocked ? 1 : 0;
            }
        }
        std::printf("D %d %lld %lld %lld %lld %lld\n", b, n, hit, blocked, plain_hit, plain_blocked);
    }
    return 0;
}
