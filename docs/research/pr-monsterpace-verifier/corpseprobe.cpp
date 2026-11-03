// corpseprobe.cpp (verifymonsterpace): a unique Fallen Shaman's corpse test, head's rules::shaman_corpse_reach and base's
// expression from ai.cpp, against my model of FUN_005f1380 (class 0x13 or 0x3a, monster flags & 0xc == 0, dx*dx + dy*dy <= aip4^2
// from FUN_005b0bd0 and the limit set at 0x5f14c8). Boss kind -> flags: champion 4, unique 8 (superunique: assumed 8 too).
#include <monsters.hpp>
#include <cstdio>
#include <cstdlib>
using namespace d2d::rules;
static unsigned long long s = 1;
static int R(int n) { s = s * 6364136223846793005ull + 1442695040888963407ull; return int((s >> 33) % (unsigned long long)n); }
int main(int argc, char** argv) {
    const int n = std::atoi(argv[1]); s = std::strtoull(argv[2], nullptr, 10);
    const int bases[] = { 19, 58, 19, 58, 20, 57, 59, 0, 18, 64 };
    int head_bad = 0, base_bad = 0, edge = 0;
    for (int c = 0; c < n; ++c) {
        const int base = bases[R(10)], kind = R(5);   // 0 none, 1 champion, 2 unique, 3 superunique, 4 minion
        const int aip4 = R(4) == 0 ? R(3) : R(60);
        int dx = R(161) - 80, dy = R(161) - 80;
        if (R(3) == 0) { dx = aip4 * (R(2) ? 1 : -1); dy = R(3) - 1; ++edge; }   // on the boundary
        const bool champion = kind == 1, unique = kind == 2 || kind == 3;
        const int flags = (champion ? 4 : 0) | (unique ? 8 : 0);
        const bool game = (base == 0x13 || base == 0x3a) && (flags & 0xc) == 0 && dx * dx + dy * dy <= aip4 * aip4;
        const bool head = shaman_corpse_reach(base, champion, unique, dx, dy, aip4);
        const bool old = (base == 19 || base == 58) && !unique && ai_distance(dx, dy) < 80;
        head_bad += head != game; base_bad += old != game;
    }
    std::printf("cases %d (boundary %d): head differs %d, base differs %d\n", n, edge, head_bad, base_bad);
}
