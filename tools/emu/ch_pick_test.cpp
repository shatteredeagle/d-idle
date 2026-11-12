// SPDX-License-Identifier: GPL-3.0-or-later
// decomp11 (fid-chain-lightning D4): the d2d side of tools/emu/ch_pick.py: d2d's real rules::ring_pick (components/rules/rules.hpp),
// the pick Chain Lightning's hop makes among the units it found. Input lines `no last n id1..idn`, output `no id|-`.
// Built by hand against a scratch checkout, never part of upstream (tools/emu/ch_pick_run.sh).
#include <rules.hpp>

#include <cstdint>
#include <cstdio>
#include <vector>

int main() {
    unsigned long long no = 0, last = 0, n = 0;
    while (std::scanf("%llu %llu %llu", &no, &last, &n) == 3) {
        std::vector<int> ids(n);
        for (auto& id : ids) { unsigned long long v = 0; std::scanf("%llu", &v); id = int(std::uint32_t(v)); }
        const int pick = d2d::rules::ring_pick(ids, int(std::uint32_t(last)));
        if (pick == -1 && n == 0) std::printf("%llu -\n", no);
        else if (pick == -1) std::printf("%llu -\n", no);
        else std::printf("%llu %u\n", no, std::uint32_t(pick));
    }
    return 0;
}
