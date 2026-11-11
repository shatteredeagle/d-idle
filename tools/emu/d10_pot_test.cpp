// SPDX-License-Identifier: GPL-3.0-or-later
// decomp10 (potions): the d2d side of tools/emu/d10_pot.py: d2d's own rules::potion_amount (components/rules/rules.hpp, the class bonus,
// the vitality / energy roll and the doubling) with the same seed state, plus the draw count read back from the Rng.
//   d10_pot_test < cases     class life calc stat low high ::: amount low' high'
#include <rules.hpp>

#include <cstdio>

int main() {
    namespace R = d2d::rules;
    long long cls, life, calc, stat;
    unsigned long long low, high;
    while (std::scanf("%lld %lld %lld %lld %llu %llu ::: %*d %*u %*u", &cls, &life, &calc, &stat, &low, &high) == 6) {
        R::Rng rng;
        rng.low = std::uint32_t(low);
        rng.high = std::uint32_t(high);
        const int amount = R::potion_amount(int(calc), int(cls), life != 0, int(stat), rng);
        std::printf("%lld %lld %lld %lld %llu %llu ::: %d %u %u\n", cls, life, calc, stat, low, high, amount, rng.low, rng.high);
    }
    return 0;
}
