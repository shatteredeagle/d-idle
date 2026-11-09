// decomp7: d2d's real rules::kill_exp over the xp_award.py grid (exp x clvl 1..99 x mlvl 1..110).
// Build: g++ -std=c++23 -O1 -I<d2d>/components/rules -I<d2d>/components d7_kill_exp.cpp -o d7_kill_exp
#include <cstdio>
#include "rules/combat.hpp"
int main(int argc, char**) {
    if (argc > 1) {  // stdin mode: "exp clvl mlvl [award]" per line
        long long e, c, m, a;
        while (std::scanf("%lld %lld %lld %lld", &e, &c, &m, &a) == 4)
            std::printf("%lld %lld %lld %lld\n", e, c, m, (long long)d2d::rules::kill_exp(int(e), int(c), int(m)));
        return 0;
    }
    const int exps[] = {1, 7, 18, 21, 33, 42, 100, 165, 1000, 12345, 0x7fffff};
    for (int e : exps)
        for (int c = 1; c <= 99; ++c)
            for (int m = 1; m <= 110; ++m)
                std::printf("%d %d %d %lld\n", e, c, m, (long long)d2d::rules::kill_exp(e, c, m));
}
