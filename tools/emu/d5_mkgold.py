import os,re
T=os.environ.get('D5_TREE',os.path.expanduser('~/d2d-lab/wt-decomp5'))
# setup.cpp fix
p=T+'/components/game/world/setup.cpp'
s=open(p).read()
old="        actor->character.stats.values[d2d::d2s::kStamina] = actor->character.stats.values[d2d::d2s::kMaxStamina];\n"
assert old in s
new=old+'''        // The same restore (FUN_0056af80, 0x56afb4..0x56b00f) puts gold and stash gold back through
        // FUN_0053ff00, which sets a stat to 0 when it would pass its cap: the purse's clvl x 10000
        // (FUN_00622e70), the stash's 2,500,000 (FUN_00623460).
        {
            using namespace d2d::d2s;
            auto& values = actor->character.stats.values;
            constexpr std::int64_t kStashGoldMax = 2500000;
            if (values[kGold] > values[kLevel] * 10000) values[kGold] = 0;
            if (values[kGoldBank] > kStashGoldMax) values[kGoldBank] = 0;
        }
'''
s=s.replace(old,new,1)
open(p,'w').write(s)
# unit test
p=T+'/tests/game/test_game.cpp'
s=open(p).read()
marker="    const auto poisoned = [&] { return std::ranges::any_of(hero.regen, &Fight::Regen::poison); };\n"
assert marker in s
add='''    // A saved purse or stash over its cap is 0 on entering a game, one under or at it is kept (FUN_0056af80
    // through FUN_0053ff00; tools/emu/fl_gold.py runs it against World::enter).
    {
        const auto before = hero.character;
        auto saved = before;
        using namespace d2d::d2s;
        saved.stats.values[kLevel] = 4; saved.stats.values[kGold] = 40000; saved.stats.values[kGoldBank] = 2500000;
        world.take_character(saved);
        assert(hero.character.stats.values[kGold] == 40000 && hero.character.stats.values[kGoldBank] == 2500000);
        saved.stats.values[kGold] = 40001; saved.stats.values[kGoldBank] = 2500001;
        world.take_character(saved);
        assert(hero.character.stats.values[kGold] == 0 && hero.character.stats.values[kGoldBank] == 0);
        hero.character = before;
        std::printf("OK: a save's gold over its cap is 0 on entering a game\\n");
    }
'''
s=s.replace(marker,add+marker,1)
open(p,'w').write(s)
