// SPDX-License-Identifier: GPL-3.0-or-later
// hirereader: the d2d side of tools/emu/hrd_pay.py. Calls d2d's own rules::hire and rules::resurrect_merc
// (components/rules/rules.hpp) on the same xorshift32 case stream. Built by hand against a scratch upstream build
// (tools/emu/hrd_build.sh), never part of upstream.
//   hrd_d2d_test hire N SEED     `g b p: ok g2 b2`
//   hrd_d2d_test revive N SEED   `g b L: ok g2 b2`

#include <merc.hpp>
#include <rules.hpp>

#include <algorithm>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <string_view>

namespace {

constexpr std::int64_t kGoldLimit = 1000000;

struct Stream {
    std::uint32_t x;
    std::uint32_t next() {
        x ^= x << 13;
        x ^= x >> 17;
        x ^= x << 5;
        return x;
    }
    std::int64_t pick(std::int64_t top) { return std::int64_t(next() % std::uint64_t(top + 1)); }
};

}  // namespace

int main(int argc, char** argv) {
    if (argc < 4) return 2;
    namespace R = d2d::rules;
    namespace S = d2d::d2s;
    const std::string_view mode = argv[1];
    const long n = std::atol(argv[2]);
    Stream r{ std::uint32_t(std::strtoul(argv[3], nullptr, 0)) };
    if (r.x == 0) r.x = 1;
    const bool revive = mode == "revive";
    constexpr std::int64_t scale[6] = { 3, 120, 1000, 20000, 100000, 2500000 };
    for (long i = 0; i < n; ++i) {
        const std::int64_t g = std::min(r.pick(scale[r.next() % 6]), kGoldLimit);
        const std::int64_t b = r.pick(scale[r.next() % 6]);
        std::int64_t x = 0;
        if (revive) {
            x = r.pick(255);
        } else {
            switch (r.next() % 4) {
                case 0: x = r.pick(120000); break;
                case 1: x = std::max<std::int64_t>(0, g + r.pick(4) - 2); break;
                case 2: x = std::max<std::int64_t>(0, g + b + r.pick(4) - 2); break;
                default: x = r.pick(40); break;
            }
        }
        S::Header header{};
        S::Stats stats{};
        stats.values[S::kGold] = g;
        stats.values[S::kGoldBank] = b;
        bool ok = false;
        if (revive) {
            header.merc_seed = 1;
            header.merc_dead = true;
            ok = R::resurrect_merc(int(x), header, stats);
        } else {
            R::MercOffer offer;
            offer.cost = int(x);
            ok = R::hire(offer, header, stats);
        }
        std::printf("%lld %lld %lld: %d %lld %lld\n", (long long)g, (long long)b, (long long)x, ok ? 1 : 0, (long long)stats.get(S::kGold),
                    (long long)stats.get(S::kGoldBank));
    }
    return 0;
}
