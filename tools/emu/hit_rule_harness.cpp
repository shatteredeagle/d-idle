// SPDX-License-Identifier: GPL-3.0-or-later
// d2d's get-hit rule BEFORE the fixes, on tools/emu/hit_rule.py's cases (d2lod-bot, card hitrec).
// The AFTER side is upstream's own `test_combat --get-hit`.
//
// Default: upstream main 32768ddb (the frame's summed damage, whole points):
//   player  fight.cpp:4903  big = foe.damage > 0 && int64(foe.damage) * 12 >= stats.fixed(kMaxLife); a stunned
//                           player's landed hit is a sure GH (:4904); `foe.damage` is the frame's sum (Foe::take)
//   merc    fight.cpp:4800  side[1].damage * 12 >= merc_max_life()  (same rule as the player's: not run here)
//   monster ai.cpp:2853     damage * 8 >= monster.stats.hit_points, one call per blow (land / hurt); no stun, frozen or GH-mode test
// --sum: PR #134's first commit d55973e1 (rules::get_hit on the frame's summed damage, the last blow's hit class,
//   any blow's element mark, d2d's rng; the monster per blow on its seed), with the stun / frozen / poison-only /
//   GH-mode gates modelled as the after side models them, so its count isolates the per-blow question.
// A record that is all poison never reaches either test in d2d (poison runs over ticks): no GH, no damage summed.
//
//   g++ -O2 -std=c++20 -o hit_rule_harness hit_rule_harness.cpp && ./hit_rule_harness [--sum] < cases.tsv > before.tsv
#include <cstdint>
#include <cstdio>
#include <cstring>

struct Rng {   // d2d rules::Rng = FUN_00472210
    std::uint32_t low = 0, high = 666;
    int operator()(int bound) {
        const std::uint64_t product = std::uint64_t(low) * 0x6AC690C5u + high;
        low = std::uint32_t(product); high = std::uint32_t(product >> 32);
        return (bound & (bound - 1)) == 0 ? int(low & std::uint32_t(bound - 1)) : int(low % std::uint32_t(bound));
    }
};
static int divisor(int hit_class) {
    switch (hit_class) { case 2: case 6: case 10: case 11: return 8; case 4: case 8: return 32; case 5: return 64; default: return 16; }
}
static bool get_hit(std::int64_t max_life, std::int64_t damage, int hit_class, bool marked, Rng& rng) {   // a copy of rules::get_hit (d55973e1)
    if (damage < 256) return false;
    const int d = marked ? 16 : divisor(hit_class);
    if (damage < max_life / d) return false;
    if (damage < max_life / (d / 2) && rng(2) == 0) return false;
    if (damage < max_life / (d / 4) && rng(4) == 0) return false;
    return true;
}

int main(int argc, char** argv) {
    const bool sum_model = argc > 1 && std::strcmp(argv[1], "--sum") == 0;
    long id, kind, maxlife, damage, hc, rec64, fire, ltng, cold, poison, f2000, mgh, stun, frozen, next;
    unsigned long lo, hi;
    char line[512];
    bool carry = false, frame_gh = false, frame_marked = false;
    long frame_id = 0, frame_sum = 0, frame_class = 0;
    Rng seed;
    while (std::fgets(line, sizeof line, stdin)) {
        if (line[0] == '#' || std::sscanf(line, "%ld %ld %ld %ld %ld %ld %ld %ld %ld %ld %ld %ld %ld %ld %lu %lu %ld", &id, &kind, &maxlife, &damage, &hc, &rec64,
                                           &fire, &ltng, &cold, &poison, &f2000, &mgh, &stun, &frozen, &lo, &hi, &next) != 17) continue;
        if (!carry) { frame_id = id; frame_gh = frame_marked = false; frame_sum = frame_class = 0; seed = Rng{}; seed.low = std::uint32_t(lo); seed.high = std::uint32_t(hi); }
        const bool poison_only = poison != 0 && poison == damage;
        // The main model works in d2d's whole points; the --sum model takes the record's 256ths (as the
        // after side does), so its count shows the per-blow question alone, not the rounding limit.
        const long whole = poison_only ? 0 : damage >> 8, max_whole = maxlife >> 8, fine = poison_only ? 0 : damage;
        const bool marked = rec64 == 0 && (fire > 0 || ltng > 0 || cold > 0 || poison > 0 || f2000 != 0);
        // --sum keeps the stun, frozen, poison-only and GH-mode tests around the function as the after side does.
        const bool gate = sum_model ? (frozen == 0 && (kind == 0 || mgh != 0)) : true;
        if (kind == 1) {                                                   // the monster: one hurt() per blow in both builds
            if (sum_model) frame_gh = frame_gh || stun != 0 || (gate && fine > 0 && get_hit(maxlife, fine, int(hc), marked, seed));
            else frame_gh = frame_gh || (whole > 0 && whole * 8 >= max_whole);
        } else {                                                           // the player: Foe::take sums the frame
            frame_sum += sum_model ? fine : whole; frame_class = hc; frame_marked = frame_marked || (marked && fine > 0);
            if (stun != 0) frame_gh = true;
        }
        if (next != 0) { carry = true; continue; }
        carry = false;
        if (kind == 0 && frame_sum > 0 && !frame_gh)
            frame_gh = sum_model ? gate && get_hit(maxlife, frame_sum, int(frame_class), frame_marked, seed) : frame_sum * 12 >= max_whole;
        std::printf("%ld %d -1\n", frame_id, frame_gh ? 1 : 0);
    }
}
