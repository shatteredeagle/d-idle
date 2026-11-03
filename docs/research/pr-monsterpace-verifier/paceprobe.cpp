// paceprobe.cpp (verifymonsterpace): rules::mon_think for the seven pace thinks on random inputs (own seed), each
// result checked against a model written from my own reading of game.exe (FUN_005efe20, 005efb80, 005f00e0, 005f0b00,
// 005f5d50, 005f02c0, 005e6320): the pace FUN_005de190 gets on the branch taken, the Blood Hawks steps, Blood Ravens
// home rule. Built against base (no -DHEAD) and head (-DHEAD: ThinkIn::target_home_dist exists).
#include <monsters.hpp>
#include <rules.hpp>
#include <algorithm>
#include <cstdio>
#include <cstdlib>
#include <map>
#include <string>
using namespace d2d::rules;
static std::uint64_t s = 0;
static int R(int n) { s = s * 6364136223846793005ull + 1442695040888963407ull; return int((s >> 33) % std::uint64_t(n)); }
int main(int argc, char** argv) {
    const int n = argc > 1 ? std::atoi(argv[1]) : 100000;
    s = argc > 2 ? std::strtoull(argv[2], nullptr, 10) : 4711;
    const char* ais[] = { "Zombie", "Brute", "BloodHawk", "CorruptRogue", "CorruptLancer", "Fallen", "BloodRaven" };
    for (const char* ai : ais) {
        const std::string name = ai;
        std::map<std::string, int> bad, kinds; int miss = 0;
        for (int c = 0; c < n; ++c) {
            int state = R(3) == 0 ? 1 : 0, command = R(2), state2 = R(12), state3 = 0, pace = 0, steps = 0; bool rally = false;
            ThinkIn in;
            for (auto& a : in.aip) a = R(4) == 0 ? (R(2) ? 0 : 100) : R(101);
            if (name == "BloodHawk") in.aip[4] = R(201) - 100;
            in.in_melee = R(3) == 0; in.got_hit = R(4) == 0; in.dist = R(4) == 0 ? R(6) : R(70);
            in.difficulty = R(3); in.level = R(5) == 0 ? 17 : 2 + R(30); in.life_pct = R(5) == 0 ? (R(2) ? 40 : 100) : R(101);
            in.skill = { R(2) == 0, R(2) == 0, false };
            in.state = &state; in.command = &command; in.rally = &rally; in.state2 = &state2; in.state3 = &state3; in.pace = &pace; in.steps = &steps;
            in.dying = R(2) == 0; in.leader = R(4) == 0;
            int hd = R(4) == 0 ? 45 + R(12) : R(70), thd = R(4) == 0 ? 45 + R(12) : R(70);
            if (R(6) == 0) hd = R(2) ? 50 : 5;
            if (R(6) == 0) thd = R(2) ? 50 : 49;
            in.home_dist = hd;
#ifdef HEAD
            in.target_home_dist = thd;
#endif
            in.off_x = R(41) - 20; in.off_y = R(41) - 20;
            const int cmd0 = command, st0 = state;
            bool away_called = false; const bool away_ok = R(2) == 0;
            auto away = [&](int, bool) { away_called = true; return away_ok; };
            Rng rng{ std::uint32_t(R(1 << 30)) };
            const Think t = mon_think(name, in, rng, away);
            const int a = int(t.act);
            int want = 0; int want_steps = 0; bool check_steps = false; std::string why = "act" + std::to_string(a);
            if (name == "Zombie") want = (!in.in_melee && t.act == MonAct::run) ? 100 : 0;
            if (name == "Brute") want = in.in_melee ? 0 : 100 - std::clamp(in.life_pct, 40, 100);
            if (name == "BloodHawk") {
                if (st0 == 1 && in.in_melee) want = 0;
                else if (t.act == MonAct::walk) { want = in.aip[4]; check_steps = true; want_steps = in.dist; }
                else if (t.act == MonAct::wander) want = t.n == 4 ? -50 : 0;
                else want = away_called ? in.aip[3] : 0;
            }
            if (name == "CorruptRogue") want = t.act == MonAct::run ? in.aip[3] : 0;
            if (name == "CorruptLancer") want = t.act == MonAct::run ? 100 : 0;
            if (name == "Fallen") want = in.dying ? 50 : 0;
            if (name == "BloodRaven") {
                if (in.dist <= 45) {
                    const bool home = thd >= 50 || hd > 50 || (cmd0 && hd > 5);
                    if (home != (t.act == MonAct::home)) { ++bad["home-rule"]; ++miss; continue; }
                }
                if (t.act == MonAct::home || t.act == MonAct::around) want = 100;
                else if (t.act == MonAct::circle) want = 50;
                else want = away_called ? 100 : 0;
            }
            ++kinds[why];
            if (pace != want || (check_steps && steps != want_steps)) { ++bad[why]; ++miss; }
        }
        std::printf("%-14s cases %d mismatches %d |", ai, n, miss);
        for (auto& [k, v] : bad) std::printf(" %s:%d", k.c_str(), v);
        std::printf("\n");
    }
}
