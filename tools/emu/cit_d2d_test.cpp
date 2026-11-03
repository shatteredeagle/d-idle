// SPDX-License-Identifier: GPL-3.0-or-later
// covitemtypes: the d2d side of tools/emu/cit_oracle.py. Calls d2d's own rules::type_is, type_class and compcode::Types::isa
// on the real loaded tables (load_game_data). Built by hand (tools/emu/cit_build.sh), never part of upstream.
//   cit_d2d_test --isa      a b rules compcode   (row indexes a, b of ItemTypes; rules::type_is(code a, code b); Types::isa(row a, row b))
//   cit_d2d_test --item     code type type2 bits (one 0/1 per ItemTypes row: type_is(type, row) || type_is(type2, row), the cube.hpp rule)
//   cit_d2d_test --belt     code beltable_chain  (put_in_belt's beltable lambda on the item's type)
//   cit_d2d_test --flags    row code equiv1 equiv2 class_chain beltable throwable autostack store_page body1 body2 magic rare normal tc rarity
#include <gamedata.hpp>
#include <gamedata_load.hpp>
#include <rules.hpp>

#include <algorithm>
#include <cstdio>
#include <cstdlib>
#include <exception>
#include <filesystem>
#include <string>
#include <string_view>
#include <vector>

using namespace d2d::game;

int main(int argc, char** argv) try {
    if (argc < 2) return 2;
    const std::string_view mode = argv[1];
    const char* mpq_dir = std::getenv("D2_MPQ_DIR");
    const char* patch = std::getenv("D2_PATCH_INSTALLER");
    auto data = load_game_data(fs::path(mpq_dir), patch ? fs::path(patch) : fs::path{}, 0x1234);
    if (!data || !data->patched) { std::printf("FAIL: no patched data\n"); return 1; }
    namespace R = d2d::rules;
    const auto& tables = data->rules;
    const auto& types = *data->item_types;
    // rows in ItemTypes order: look up the code of every row index through Types::index (first row with that code)
    // ItemTypes.txt has 104 rows incl. the "Expansion" separator row, which game.exe's loader drops (103 records)
    std::vector<std::string> codes(104);
    for (const auto& [code, info] : tables.types) { (void)info; if (const int row = types.index(code); row >= 0 && std::size_t(row) < codes.size()) codes[std::size_t(row)] = code; }
    if (mode == "--isa") {
        for (std::size_t a = 0; a < codes.size(); ++a)
            for (std::size_t b = 0; b < codes.size(); ++b)
                if (!codes[a].empty() && !codes[b].empty())
                    std::printf("%s %s %d %d\n", codes[a].c_str(), codes[b].c_str(), int(R::type_is(tables, codes[a], codes[b])), int(types.isa(int(a), int(b))));
        return 0;
    }
    if (mode == "--item" || mode == "--belt") {
        for (const auto& [code, info] : tables.item_info) {
            const auto base = tables.item_base.find(code);
            const std::string type2 = base != tables.item_base.end() ? base->second.type2 : std::string{};
            if (mode == "--belt") {
                auto beltable = [&](auto&& self, const std::string& type, int depth) -> bool {
                    const auto found = tables.types.find(type);
                    if (found == tables.types.end() || depth > 8) return false;
                    return found->second.beltable || self(self, found->second.equiv[0], depth + 1) || self(self, found->second.equiv[1], depth + 1);
                };
                std::printf("%s %d\n", code.c_str(), int(beltable(beltable, info.type, 0)));
                continue;
            }
            std::printf("%s %s %s ", code.c_str(), info.type.c_str(), type2.c_str());
            for (std::size_t b = 0; b < codes.size(); ++b)
                std::printf("%c", codes[b].empty() ? '?' : (R::type_is(tables, info.type, codes[b]) || R::type_is(tables, type2, codes[b])) ? '1' : '0');
            std::printf("\n");
        }
        return 0;
    }
    if (mode == "--sock") {   // code sockets(ilvl 0..110): rules::most_sockets
        for (const auto& [code, info] : tables.item_info) {
            (void)info;
            std::printf("%s ", code.c_str());
            for (int ilvl = 0; ilvl <= 110; ++ilvl) {
                d2d::d2s::Item item;
                item.code = code;
                item.ilvl = ilvl;
                std::printf("%s%d", ilvl ? "," : "", R::most_sockets(tables, item));
            }
            std::printf("\n");
        }
        return 0;
    }
    if (mode == "--slot") {   // code bits(slot 0..12): rules::can_wear with a wearer of the item's own class and 999 str/dex/level
        for (const auto& [code, info] : tables.item_info) {
            const std::string cc = R::type_class(tables, info.type);
            R::Wearer wearer;
            wearer.cls = 1;
            for (int i = 0; i < 7; ++i) if (cc == R::kClassCode[std::size_t(i)]) wearer.cls = i;
            wearer.str = wearer.dex = wearer.lvl = 999;
            d2d::d2s::Item item;
            item.code = code;
            item.identified = true;
            std::printf("%s ", code.c_str());
            for (int slot = 0; slot <= 12; ++slot) std::printf("%c", R::can_wear(tables, item, slot, wearer) ? '1' : '0');
            std::printf("\n");
        }
        return 0;
    }
    if (mode == "--hands") {   // class codeA bits: rules::hands_ok(A, B, wearer of d2s class) over the items whose type's BodyLoc1 is the right arm (4)
        std::vector<std::string> order;
        for (const auto& [code, info] : tables.item_info) {
            const auto found = tables.types.find(info.type);
            if (found != tables.types.end() && found->second.body[0] == 4) order.push_back(code);
        }
        std::ranges::sort(order);
        for (const int cls : { 1, 4, 6 }) {
            R::Wearer wearer;
            wearer.cls = cls;
            for (const auto& a : order) {
                d2d::d2s::Item first;
                first.code = a;
                std::printf("%d %s ", cls, a.c_str());
                for (const auto& b : order) {
                    d2d::d2s::Item second;
                    second.code = b;
                    std::printf("%c", R::hands_ok(tables, first, second, wearer) ? '1' : '0');
                }
                std::printf("\n");
            }
        }
        return 0;
    }
    if (mode == "--item2") {   // code body1 body2 magic rare normal class_chain_nonempty (the item's own type row, d2d's view)
        for (const auto& [code, info] : tables.item_info) {
            const auto found = tables.types.find(info.type);
            if (found == tables.types.end()) { std::printf("%s ? ? ? ? ? ?\n", code.c_str()); continue; }
            const auto& t = found->second;
            std::printf("%s %d %d %d %d %d %d\n", code.c_str(), t.body[0], t.body[1], int(t.always_magic), int(t.can_rare), int(t.always_normal),
                        int(!R::type_class(tables, info.type).empty()));
        }
        return 0;
    }
    if (mode == "--flags") {
        for (std::size_t row = 0; row < codes.size(); ++row) {
            if (codes[row].empty()) continue;
            const auto& t = tables.types.at(codes[row]);
            std::printf("%zu %s %s %s %s %d %d %d %d %d %d %d %d %d %d %d\n", row, codes[row].c_str(), t.equiv[0].c_str(), t.equiv[1].c_str(),
                        R::type_class(tables, codes[row]).c_str(), int(t.beltable), int(t.throwable), int(t.autostack), t.store_page, t.body[0], t.body[1],
                        int(t.always_magic), int(t.can_rare), int(t.always_normal), int(t.treasure_class), t.rarity);
        }
        return 0;
    }
    return 2;
} catch (const std::exception& error) { std::printf("FAIL: %s\n", error.what()); return 1; }
