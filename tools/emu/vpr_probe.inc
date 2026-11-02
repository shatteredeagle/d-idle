    // Verifier VERIFYPRICES probe (local only, never committed to d2d): `--vpr CASES`.
    //   M id P npc diff qbits      synthetic item "vprm" of cost P: buy sell rep with npc (and "none" for no row)
    //   G id code dur maxdur qty ident eth indes chg npc diff     repair_cost of a real base
    //   N                          dump npc_prices
    if (argc > 2 && std::string_view(argv[1]) == "--vpr") {
        auto& rules = data->rules;
        const auto mark_repair = [](auto& type) { if constexpr (requires { type.repair; }) type.repair = true; };
        rules.item_base["vprm"] = {};
        rules.item_base["vprm"].durability = 40;
        rules.item_info["vprm"].type = "vprt";
        mark_repair(rules.types["vprt"]);
        std::ifstream cases(argv[2]);
        std::string line;
        while (std::getline(cases, line)) {
            std::istringstream fields(line);
            std::string kind, id;
            fields >> kind;
            if (kind == "N") {
                for (const auto& [name, p] : rules.npc_prices) {
                    std::printf("N %s %d %d %d", name.c_str(), p.buy, p.sell, p.rep);
                    for (int q = 0; q < 3; ++q) std::printf(" %d %d %d %d", p.qflag[std::size_t(q)], p.qbuy[std::size_t(q)], p.qsell[std::size_t(q)], p.qrep[std::size_t(q)]);
                    std::printf(" %d %d %d\n", p.max_buy[0], p.max_buy[1], p.max_buy[2]);
                }
                continue;
            }
            fields >> id;
            if (kind == "M") {
                long long cost = 0; std::string npc; int diff = 0, qbits = 0;
                fields >> cost >> npc >> diff >> qbits;
                if (!fields) continue;
                rules.item_base["vprm"].cost = int(cost);
                d2d::d2s::Item item; item.code = "vprm"; item.quality = 2; item.identified = true; item.quantity = 1;
                item.max_durability = 40; item.durability = 0;
                d2d::d2s::Header header; header.difficulty[std::size_t(diff)] |= 0x80;
                if (const auto row = rules.npc_prices.find(npc); row != rules.npc_prices.end())
                    for (int q = 0; q < 3; ++q)
                        if (qbits >> q & 1 && row->second.qflag[std::size_t(q)]) {
                            const int bit = row->second.qflag[std::size_t(q)] * 16;
                            header.quests[std::size_t(diff)][std::size_t(bit >> 3)] |= std::uint8_t(1 << (bit & 7));
                        }
                std::printf("%s %d %d %d\n", id.c_str(), d2d::rules::item_price(rules, item, npc, false, header), d2d::rules::item_price(rules, item, npc, true, header),
                            d2d::rules::repair_cost(rules, item, npc, header));
            } else if (kind == "G") {
                d2d::d2s::Item item; int ident = 0, eth = 0, indes = 0, chg = 0, diff = 0; std::string npc;
                fields >> item.code >> item.durability >> item.max_durability >> item.quantity >> ident >> eth >> indes >> chg >> npc >> diff;
                if (!fields) continue;
                item.quality = 2; item.identified = ident != 0; item.ethereal = eth != 0;
                if (indes) { d2d::d2s::ItemProp prop; prop.stat = d2d::d2s::kIndestructible; prop.value = 1; item.props.push_back(prop); }
                if (chg) { d2d::d2s::ItemProp prop; prop.stat = d2d::d2s::kChargedSkill; prop.param = 54 << 6 | 1; prop.value = 10 << 8 | (chg == 2 ? 3 : 10); item.props.push_back(prop); }
                d2d::d2s::Header header; header.difficulty[std::size_t(diff)] |= 0x80;
                std::printf("%s %d\n", id.c_str(), d2d::rules::repair_cost(rules, item, npc, header));
            }
        }
        return 0;
    }
