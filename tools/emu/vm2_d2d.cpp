// SPDX-License-Identifier: GPL-3.0-or-later
// verifyskmelee2: d2d side of the independent melee check. Reads the case file tools/emu/vm2_game.py
// wrote (H and B lines; U lines are skipped) and runs d2d's REAL rules::make_fighter and
// rules::player_blow on each. One line out per case:
//   H id hit
//   B id hit phys frac
// H: player (barehanded, so no mastery) at a monster, melee. d2d's first draw is the to-hit roll.
// B: the to-hit draw is under the 95 cap by construction; the damage is the second draw.
#include <combat.hpp>
#include <d2s_items.hpp>
#include <rules.hpp>

#include <cstdint>
#include <cstdio>
#include <exception>
#include <fstream>
#include <sstream>
#include <string>

using namespace d2d::rules;

int main(int argc, char** argv) try {
    if (argc < 2) { std::printf("usage: vm2_d2d CASEFILE\n"); return 2; }
    namespace s = d2d::d2s;
    std::ifstream cases(argv[1]);
    std::string line;
    d2d::d2s::Item weapon;
    weapon.code = "tst";
    while (std::getline(cases, line)) {
        std::istringstream in(line);
        std::string tag;
        long long id = 0;
        if (!(in >> tag >> id)) continue;
        Tables tables;
        StatSum sum{}, wsum{};
        d2d::d2s::Stats stats;
        Target target;
        Swing swing;
        if (tag == "H") {
            std::uint32_t s0, s1;
            long long dex, clvl, gear, togain, pct, bonus, dfn, dlvl, armed;
            if (!(in >> s0 >> s1 >> dex >> clvl >> gear >> togain >> pct >> bonus >> dfn >> dlvl >> armed)) continue;
            tables.item_base["tst"] = { .mindam = 1, .maxdam = 2 };
            stats.values[s::kLevel] = clvl; stats.values[s::kDex] = dex; stats.values[s::kStr] = 10;
            sum[s::kToHit] = gear; sum[s::kToHitPercent] = pct;
            ClassGains gains; gains.to_hit = int(togain);
            const auto fighter = make_fighter(tables, armed ? &weapon : nullptr, nullptr, sum, wsum, stats, gains, 0, {});
            target.armor_class = int(dfn); target.level = int(dlvl);
            swing.ar_pct = int(bonus);
            Rng rng; rng.low = s0; rng.high = s1;
            std::printf("H %lld %d\n", id, player_blow(fighter, target, int(clvl), rng, swing).hit ? 1 : 0);
        } else if (tag == "B" || tag == "C") {
            std::uint32_t s0, s1;
            long long armed, bm, bM, sb, db, ed, fmin, fmax, n111, dp, strn, dex, mastery, eskill, srcdam;
            if (!(in >> s0 >> s1 >> armed >> bm >> bM >> sb >> db >> ed >> fmin >> fmax >> n111 >> dp >> strn >> dex >> mastery >> eskill >> srcdam)) continue;
            tables.item_base["tst"] = { .mindam = int(bm), .maxdam = int(bM), .str_bonus = int(sb), .dex_bonus = int(db) };
            stats.values[s::kLevel] = 1; stats.values[s::kStr] = strn; stats.values[s::kDex] = dex;
            wsum[s::kMaxDamagePercent] = wsum[s::kMinDamagePercent] = ed;
            sum[s::kMinDamage] = fmin; sum[s::kMaxDamage] = fmax; sum[s::kNormalDamage] = n111;
            sum[s::kDamagePercent] = dp; sum[s::kPassiveMasteryMeleeDamage] = mastery;
            const auto fighter = make_fighter(tables, armed ? &weapon : nullptr, nullptr, sum, wsum, stats, {}, 0, {});
            target.armor_class = 0; target.level = 1;
            target.hit_points = target.max_hp = 1 << 30;
            swing.ed_pct = int(eskill); swing.srcdam = int(srcdam);
            Rng rng; rng.low = s0; rng.high = s1;
            const auto blow = player_blow(fighter, target, 1, rng, swing);
            std::printf("%s %lld %d %d %d\n", tag.c_str(), id, blow.hit ? 1 : 0, blow.phys, blow.frac);
        }
    }
    return 0;
} catch (const std::exception& error) {
    std::fprintf(stderr, "vm2_d2d: %s\n", error.what());
    return 1;
}
