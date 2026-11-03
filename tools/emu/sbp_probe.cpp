// SPDX-License-Identifier: GPL-3.0-or-later
// sbp_probe (card prbuildsmite): d2d's real `player_blow` (upstream components/rules/combat.hpp) for Smite against game.exe.
//   sbp_probe chances CASES SEED   -> per block chance 0 / 25 / 50 / 100: Smite hit, blocked, landed (damage > 0), damage sum; plain swing hit / blocked
//   sbp_probe cases < game.txt     -> one line per monster-defender case of the oracle's output (verify-smiteblock/vs_oracle.py):
//        `E id game_land game_blocked game_dmg d2d_hit d2d_blocked d2d_dmg bounds_ok plain_hit`
// The oracle's defender cell sets d2d's block chance: the game's own record flags bit 0x10 (it blocked) -> chance 100, else 0.
// The Fighter is built field by field; no table is read. Smite has no to-hit roll in d2d (the rule under test is the block).
#include <combat.hpp>
#include <rules.hpp>

#include <algorithm>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <iostream>
#include <map>
#include <sstream>
#include <string>
#include <vector>

using namespace d2d::rules;

static int chances(long long n, std::uint64_t seed) {
    auto next = [&] { seed = seed * 6364136223846793005ull + 1442695040888963407ull; return std::uint32_t(seed >> 32); };
    for (const int b : { 0, 25, 50, 100 }) {
        long long hit = 0, blocked = 0, landed = 0, damage = 0, plain_hit = 0, plain_blocked = 0;
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
                if (smite) { hit += blow.hit; blocked += blow.blocked; landed += blow.hit && blow.damage > 0; damage += blow.damage; }
                else { plain_hit += blow.hit; plain_blocked += blow.blocked; }
            }
        }
        std::printf("D block %d cases %lld smite_hit %lld smite_blocked %lld smite_landed %lld smite_damage %lld plain_hit %lld plain_blocked %lld\n",
                    b, n, hit, blocked, landed, damage, plain_hit, plain_blocked);
    }
    return 0;
}

static int cases() {
    std::string line;
    long long skipped = 0;
    while (std::getline(std::cin, line)) {
        if (line.compare(0, 2, "C ") != 0) continue;
        std::map<std::string, std::string> kv;
        std::istringstream in(line.substr(2));
        std::vector<std::string> tok;
        for (std::string t; in >> t;) tok.push_back(t);
        for (std::size_t i = 0; i + 1 < tok.size(); ++i) kv.emplace(tok[i], tok[i + 1]);
        const long id = std::stol(kv["C"] = tok[0]);
        if (kv["dt"] != "1") { ++skipped; continue; }            // a monster defender: what d2d's player_blow models
        const std::uint32_t flags_raw = kv.count("flags") && kv["flags"] != "-" ? std::uint32_t(std::stoul(kv["flags"], nullptr, 16)) : 0;
        const bool game_land = kv["consumer"] != "0" && kv["dmg"] != "-";
        const long long game_dmg = game_land ? std::stoll(kv["dmg"]) : 0;
        long long arg[6] = {};
        {
            std::istringstream a(kv["args"]);
            std::string t;
            for (int k = 0; k < 6 && std::getline(a, t, ','); ++k) arg[k] = std::stoll(t);
        }
        const long long ed = std::stoll(kv["ed"]), sb = std::stoll(kv["sb"]), str = std::stoll(kv["str"]);
        Fighter fighter;
        fighter.ar_base = 1000000;
        fighter.smite_lo = int(arg[1] >> 8); fighter.smite_hi = int(arg[2] >> 8);
        fighter.smite_pct = int(str * sb / 100);
        Target target;
        target.armor_class = 0; target.level = 1; target.hit_points = target.max_hp = 1 << 30;
        target.block = flags_raw & 0x10 ? 100 : 0;
        Rng rng; rng.low = std::uint32_t(id * 2654435761u + 1); rng.high = std::uint32_t(id + 666);
        Swing swing;
        swing.smite = true; swing.ed_pct = int(ed);
        swing.skill_lo = int(arg[1] & 255); swing.skill_hi = int(arg[2] & 255);
        const auto blow = player_blow(fighter, target, 30, rng, swing);
        // The game's range for this case (FUN_0057b420: roll in [min, max + 255] x (100 + ED + StrBonus) / 100), whole points, 1 of slack.
        const long long pct = 100 + ed + str * sb / 100;
        const long long lo = (arg[1] * pct / 100) >> 8, hi = ((arg[2] + 255) * pct / 100) >> 8;
        const bool bounds = blow.damage >= lo - 1 && blow.damage <= hi + 1;
        // A plain swing against the same defender (no forced hit): a block must stop it.
        Fighter plain_fighter = fighter;
        plain_fighter.phys_lo = plain_fighter.phys_hi = 10 << 8;
        Target plain_target = target;
        Swing plain;
        int plain_hit = 0;
        for (int k = 0; k < 8; ++k) plain_hit += player_blow(plain_fighter, plain_target, 99, rng, plain).hit;
        std::printf("E %ld %d %d %lld %d %d %d %d %d\n", id, game_land ? 1 : 0, flags_raw & 0x10 ? 1 : 0, game_dmg, blow.hit ? 1 : 0, blow.blocked ? 1 : 0, blow.damage,
                    bounds ? 1 : 0, target.block == 100 ? plain_hit : -1);
    }
    std::printf("S skipped_non_monster %lld\n", skipped);
    return 0;
}

int main(int argc, char** argv) {
    if (argc > 1 && !std::strcmp(argv[1], "chances")) return chances(argc > 2 ? std::atoll(argv[2]) : 2000, argc > 3 ? std::strtoull(argv[3], nullptr, 10) : 7);
    if (argc > 1 && !std::strcmp(argv[1], "cases")) return cases();
    std::fprintf(stderr, "usage: sbp_probe chances N SEED | sbp_probe cases < oracle.txt\n");
    return 2;
}
