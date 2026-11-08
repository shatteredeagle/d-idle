import os
T=os.environ.get('D5_TREE','/home/jesse/d2d-lab/wt-decomp5')
p=T+'/tests/game/test_game.cpp'
s=open(p).read()
marker='    // `--waypoint-refuse CASES`'
add='''    // `--load-gold CASES`: the purse and stash gold a character enters a game with (World::enter),
    // as tools/emu/d5_gold_load.py prints game.exe's loader FUN_0056b180 -> FUN_0056af80 (0x56afb4..0x56b00f).
    // A case is `id level gold bank`; per case `id gold bank`.
    if (argc > 2 && std::string_view(argv[1]) == "--load-gold") {
        std::ifstream cases(argv[2]);
        d2d::rules::Rng made_rng(7);
        auto made = new_character(*data, 4, "Headless", false, true, made_rng);
        std::string id;
        std::int64_t level = 0, gold = 0, bank = 0;
        while (cases >> id >> level >> gold >> bank) {
            using namespace d2d::d2s;
            Character saved;
            saved.character_class = 4;
            saved.name = "Headless";
            saved.header = made.header;
            saved.stats = made.stats;
            saved.items = made.items;
            saved.panel = panel_stats(*data, saved.header, saved.items, saved.stats);
            saved.stats.values[kLevel] = level; saved.stats.values[kGold] = gold; saved.stats.values[kGoldBank] = bank;
            World world(&*data, -1, -1);
            world.enter(saved);
            const auto& got = world.actor->character.stats.values;
            std::printf("%s %lld %lld\\n", id.c_str(), static_cast<long long>(got[kGold]), static_cast<long long>(got[kGoldBank]));
        }
        return 0;
    }
'''
assert marker in s
s=s.replace(marker,add+marker,1)
open(p,'w').write(s)
