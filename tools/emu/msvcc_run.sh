#!/bin/bash
# msvcc2026: split the big .inc literals, then compile a probe with g++ that includes the ORIGINAL and the SPLIT
# files and compares the compiled constants (length + bytes + a hash). Output under $D2_SCRATCH/msvcc.
set -u
S=${D2_SCRATCH:-/tmp}/msvcc
mkdir -p "$S/orig"
python3 tools/emu/msvcc_split.py 12000 || exit 1
T=$HOME/d2d-lab/fork/src/tests
cp "$T/pet_ais_cases.inc" "$T/shadow_master_cases.inc" "$T/shadow_init_cases.inc" "$S/orig/"
cat > "$S/probe.cpp" <<'EOF'
#include <array>
#include <cstdint>
#include <cstdio>
#include <string_view>
namespace o {
static const std::string_view pet =
#include "orig/pet_ais_cases.inc"
    ;
static const std::array<std::string_view, 3> master{
#include "orig/shadow_master_cases.inc"
};
}
namespace n {
static const std::string_view pet =
#include "new/pet_ais_cases.inc"
    ;
static const std::array<std::string_view, 3> master{
#include "new/shadow_master_cases.inc"
};
}
static std::uint64_t fnv(std::string_view s) { std::uint64_t h = 1469598103934665603ull; for (unsigned char c : s) { h ^= c; h *= 1099511628211ull; } return h; }
int main() {
    int bad = 0, cmp = 0;
    auto chk = [&](std::string_view a, std::string_view b, const char* what) {
        ++cmp; const bool same = a == b;
        std::printf("%s len %zu/%zu fnv %016llx/%016llx %s\n", what, a.size(), b.size(), (unsigned long long)fnv(a), (unsigned long long)fnv(b), same ? "same" : "DIFF");
        if (!same) ++bad;
    };
    chk(o::pet, n::pet, "pet");
    for (int i = 0; i < 3; ++i) chk(o::master[i], n::master[i], "master");
    std::printf("compared %d mismatches %d\n", cmp, bad);
    return bad;
}
EOF
g++ -std=c++20 -O0 -I"$S" -o "$S/probe" "$S/probe.cpp" 2>&1 | head -20
"$S/probe"; echo "probe exit $?"
