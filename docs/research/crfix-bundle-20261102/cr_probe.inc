    // crfixbundle probe (local only, never committed to d2d): `--crr CASES`, the d2d side of cr_repair_game.py.
    //   R id code dur maxdur indes base npc diff    repair_cost minus charges_repair_cost of a real base (cost forced to
    //   `base`, armor at its max defense, normal quality, one charged skill with charges missing)
    if (argc > 2 && std::string_view(argv[1]) == "--crr") {
        auto& rules = data->rules;
        std::ifstream cases(argv[2]);
        std::string line;
        while (std::getline(cases, line)) {
            std::istringstream fields(line);
            std::string kind, id, npc;
            d2d::d2s::Item item; int indes = 0, base = 0, diff = 0;
            fields >> kind >> id >> item.code >> item.durability >> item.max_durability >> indes >> base >> npc >> diff;
            if (!fields) continue;
            auto& entry = rules.item_base[item.code];
            entry.cost = base;
            item.defense = entry.maxac;
            item.quality = 2; item.identified = true; item.quantity = 1;
            if (indes) { d2d::d2s::ItemProp prop; prop.stat = d2d::d2s::kIndestructible; prop.value = 1; item.props.push_back(prop); }
            d2d::d2s::ItemProp charged; charged.stat = d2d::d2s::kChargedSkill; charged.param = 54 << 6 | 1; charged.value = 10 << 8 | 3; item.props.push_back(charged);
            d2d::d2s::Header header; header.difficulty[std::size_t(diff)] |= 0x80;
            std::printf("%s %d\n", id.c_str(), d2d::rules::repair_cost(rules, item, npc, header) - d2d::rules::charges_repair_cost(rules, item));
        }
        return 0;
    }
