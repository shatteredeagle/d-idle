import re,sys
import os
T=os.environ.get('D5_TREE','/home/jesse/d2d-lab/wt-decomp5')
p=T+'/tests/game/test_game.cpp'
if not os.path.exists(p): p=T+'/tests/test_game.cpp'
s=open(p).read()
marker='    // `--waypoint-refuse CASES`'
m2='    // `--skill-tips'
add='''    // `--load-d2s FILE`: a real save through the join (World::enter) and then
    // one Fight::update_fighters, as the game does at its first frame: the
    // stats before and after (256ths) and what the worn items add to each
    // attribute (the gear share op 8 / 9 of FUN_00626200 reads).
    if (argc > 2 && std::string_view(argv[1]) == "--load-d2s") {
        std::ifstream file(argv[2], std::ios::binary);
        std::vector<char> raw((std::istreambuf_iterator<char>(file)), std::istreambuf_iterator<char>());
        const auto bytes = std::as_bytes(std::span(raw));
        using namespace d2d::d2s;
        Character saved;
        saved.header = parse_header(bytes);
        saved.character_class = saved.header.cls;
        saved.name = saved.header.name;
        saved.stats = parse_stats(bytes, *data->item_tables);
        saved.items = parse_items(bytes, *data->item_tables);
        saved.panel = panel_stats(*data, saved.header, saved.items, saved.stats);
        World world(&*data, -1, -1);
        world.enter(saved);
        const auto show = [&](const char* when) {
            const auto& v = world.actor->character.stats.values;
            std::printf("%s life %lld/%lld mana %lld/%lld stamina %lld/%lld\\n", when, static_cast<long long>(v[kLife]), static_cast<long long>(v[kMaxLife]),
                        static_cast<long long>(v[kMana]), static_cast<long long>(v[kMaxMana]), static_cast<long long>(v[kStamina]), static_cast<long long>(v[kMaxStamina]));
        };
        show("enter ");
        world.fight.update_fighters(1000);
        show("update");
        std::printf("panel.bonus maxmana %lld maxlife %lld ene %lld vit %lld; items %zu\\n", static_cast<long long>(world.actor->character.panel.bonus[kMaxMana]),
                    static_cast<long long>(world.actor->character.panel.bonus[kMaxLife]), static_cast<long long>(world.actor->character.panel.bonus[kEne]),
                    static_cast<long long>(world.actor->character.panel.bonus[kVit]), saved.items.size());
        for (const auto& item : saved.items)
            for (const auto& prop : item.props)
                if (prop.stat == kEne || prop.stat == kVit || prop.stat == kMaxMana || prop.stat == kMaxLife || prop.stat == kMaxManaPercent || prop.stat == kMaxLifePercent)
                    std::printf("item %s loc %d slot %d prop %d value %lld\\n", item.code.c_str(), item.location, item.slot, prop.stat, static_cast<long long>(prop.value));
        return 0;
    }
'''
if marker not in s: marker=m2
assert marker in s
s=s.replace(marker,add+marker,1)
open(p,'w').write(s)
