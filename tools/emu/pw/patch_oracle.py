# prwarp scratch: patch tools/emu/sk_tele_spot.py of the fork worktree (arg 1 = that file).
import sys
p = sys.argv[1]
s = open(p, encoding="utf-8").read()

def rep(old, new):
    global s
    assert s.count(old) == 1, old
    s = s.replace(old, new)

rep('''    uv run python sk_tele_spot.py probe          # where the snap lands round a blocked target
''', '''    uv run python sk_tele_spot.py probe          # where the snap lands round a blocked target

The same script is the oracle of a level warp's arrival spot (FUN_005550b0):

    uv run python sk_tele_spot.py gen --warp 20000 1 > cases.tsv
    uv run python sk_tele_spot.py warp cases.tsv > game.tsv
    ../../build/tests/test_warp_arrival --warp-arrival cases.tsv | grep -E '^[0-9]+ ' > d2d.tsv    # tests/game/test_warp_arrival.cpp
    uv run python sk_tele_spot.py diff game.tsv d2d.tsv cases.tsv

A warp case is `id 1 x y x y n {x y bits}...`: the other side's warp unit's
subtile (twice, so the lines parse as above) and the collision words round
it. `warp` runs FUN_0064e7b0 with FUN_005550b0's arguments (0x555144 to
0x555157: size 2, mask 0x1c09, flag 1: no spot returns the unit's own
subtile) and then FUN_00554ea0 from that subtile (0x5551bc: flag 0, a unit
that is typed a monster so it takes the short tail), as the warp does; the
answer is `id 1 x y` where FUN_00650be0 put the unit, `id 0 - -` when
FUN_00554ea0 finds no spot (the warp is not taken). The world is one room
for the whole 200 x 200 window (U: a ring that crosses room edges is not
modelled). `gen --warp` blocks a patch round the unit; a few cases fill every ring to 44 to 50 so that some
are refused.
''')

rep('''        self.moved = None
        self.line = None
        e.hook(0x619790''', '''        self.moved = None
        self.line = None
        self.xy_buf = e.alloc(8)
        e.hook(0x619790''')

rep('''def make_case(rng, units):''', '''class WarpOracle(Oracle):
    """One room for the whole 200 x 200 window, as d2d's collision is one array: the ring search never crosses a room
    edge. (FUN_0064dea0 looks each subtile's room up from the room it last found, so on a level of several rooms a
    ring can lose a neighbour; the real-level runs cover that, this world does not model it: U.)"""

    def __init__(self):
        super().__init__()
        e = self.e
        room, coll = e.alloc(0x100), e.alloc(0x40)
        rect = e.view("room", room)
        rect.x, rect.y, rect.w, rect.h = 0, 0, SIDE, SIDE
        grid = e.view("grid", coll)
        grid.x, grid.y, grid.w, grid.h = 0, 0, SIDE, SIDE
        grid.x_copy, grid.y_copy, grid.w_copy, grid.h_copy = 0, 0, SIDE, SIDE
        grid.cells = e.alloc(SIDE * SIDE * 2)
        array = e.alloc(4)
        e.w32(array, room)
        self.rooms, self.coll, self.near = {(0, 0): room}, {room: (0, 0, coll)}, {room: (array, 1)}
        self.here = room

    def load(self, words):
        buf = bytearray(SIDE * SIDE * 2)
        for (x, y), value in words.items():
            at = (y * SIDE + x) * 2
            buf[at:at + 2] = value.to_bytes(2, "little")
        self.e.mu.mem_write(self.e.view("grid", self.coll[self.here][2]).cells, bytes(buf))

    def warp(self, ux, uy, words):
        """FUN_005550b0's two spot searches for a unit on subtile (ux, uy): (did, (x, y) or None)."""
        e = self.e
        self.load(words)
        self.moved = None
        e.w32(self.xy_buf, ux)
        e.w32(self.xy_buf + 4, uy)
        room = e.call(0x64e7b0, self.here, self.xy_buf, 2, MASK, 1) & 0xFFFFFFFF          # 0x555144..0x555157
        if not room:
            return 0, None
        x, y = e.s32(self.xy_buf), e.s32(self.xy_buf + 4)
        ret = e.call(0x554ea0, room, x, y, 0, 0, ecx=0x2000, edx=self.unit) & 0xFFFFFFFF     # 0x5551bc
        return ret, self.moved


def make_warp_case(rng):
    """One warp case: a warp unit in the middle of the window with a patch of walls round it, or (5 percent) every ring to 44..50 filled."""
    ux, uy = rng.randint(70, 130), rng.randint(70, 130)
    words = {}
    if rng.random() < 0.05:
        radius = rng.choice((44, 45, 46, 47, 48, 49, 50))
        for y in range(uy - radius, uy + radius + 1):
            for x in range(ux - radius, ux + radius + 1):
                words[(x, y)] = 1
    else:
        radius = rng.choice((1, 2, 3, 4, 6, 9, 14))
        density = rng.choice((0.3, 0.5, 0.7, 0.85, 0.95, 1.0))
        for y in range(uy - radius, uy + radius + 1):
            for x in range(ux - radius, ux + radius + 1):
                if rng.random() < density:
                    words[(x, y)] = rng.choice((0x1, 0x1, 0x9, 0x400, 0x800, 0x4))
        for _ in range(rng.randint(0, 12)):
            at = (ux + rng.randint(-radius - 4, radius + 4), uy + rng.randint(-radius - 4, radius + 4))
            words[at] = words.get(at, 0) | rng.choice((0x1, 0x8, 0x400))
        if rng.random() < 0.15:
            words.pop((ux, uy), None)
    return ux, uy, {at: value for at, value in words.items() if 0 <= at[0] < SIDE and 0 <= at[1] < SIDE}


def gen_warp(count, seed):
    rng = random.Random(seed)
    lines = []
    for cid in range(count):
        ux, uy, words = make_warp_case(rng)
        flat = " ".join(f"{x} {y} {value}" for (x, y), value in sorted(words.items()))
        lines.append(f"{cid} 1 {ux} {uy} {ux} {uy} {len(words)} {flat}".rstrip())
    return lines


def make_case(rng, units):''')

rep('''    if cmd == "gen":
        units = "--no-units" not in sys.argv
        args = [arg for arg in sys.argv[2:] if arg != "--no-units"]
        print("\\n".join(gen(int(args[0]), int(args[1]), units)))
''', '''    if cmd == "gen" and "--warp" in sys.argv:
        args = [arg for arg in sys.argv[2:] if arg != "--warp"]
        print("\\n".join(gen_warp(int(args[0]), int(args[1]))))
    elif cmd == "gen":
        units = "--no-units" not in sys.argv
        args = [arg for arg in sys.argv[2:] if arg != "--no-units"]
        print("\\n".join(gen(int(args[0]), int(args[1]), units)))
    elif cmd == "warp":
        oracle = WarpOracle()
        for line in read(sys.argv[2]):
            cid, _, ux, uy, _, _, words = parse(line)
            ret, pos = oracle.warp(ux, uy, words)
            print(show(cid, ret, pos))
''')
open(p, "w", encoding="utf-8").write(s)
