// SPDX-License-Identifier: GPL-3.0-or-later
// verifycovmissiles: the REAL d2d rules::missile_velocity / rules::missile_range (components/rules/missiles.hpp) on `vel vlev level
// rng lrng` lines read from stdin; prints `velocity range`, one line per input line.
#include <missiles.hpp>

#include <cstdio>

int main() {
    int vel, vlev, level, rng, lrng;
    while (std::scanf("%d %d %d %d %d", &vel, &vlev, &level, &rng, &lrng) == 5)
        std::printf("%d %d\n", d2d::rules::missile_velocity(vel, vlev, level), d2d::rules::missile_range(rng, lrng, level));
    return 0;
}
