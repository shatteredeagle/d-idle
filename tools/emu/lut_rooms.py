#!/usr/bin/env python3
"""Room records of the player's level in a live game.exe 1.14d, over whisper's peek verb.

Usage: lut_rooms.py HOST[:PORT] TOKEN_FILE [rooms|town|state]

Lab helper for the LUTGHOLEIN investigation (docs/research/lutgholein-trace.md).
Offsets as whisper.py's `town` / upstream tools/emu/drlg.py use them (S: read there;
V: the same reads gave the matching room lists live):
player unit 0x7a6a70, +0x2c path, path +0x1c room1, room1 +0x10 room2, room2 +0x58 level;
level +0x10 first room2, +0x1c/+0x20 origin, +0x1b4 act (+0x458 map seed), +0x1d0 id;
room2 +0x24 next, +0x34/+0x38 x/y, +0x3c/+0x40 w/h, +0x30 room1 (room1 +0x6c seed),
+0x50 dt1 mask, +0x54 tiles, +0x4 /+0x8 seed words (U: which words FUN_0066ee40 resets),
+0x14 preset-roll seed (S drlg.md), +0x48 kind, +0x60 flags (S drlg.md: populated bit).
The token file holds the whisper token (never printed)."""
import socket
import sys


class Whisper:
    def __init__(self, address, token):
        host, _, port = address.partition(":")
        self.conn = socket.create_connection((host, int(port or 4100)), timeout=10)
        self.reader = self.conn.makefile("rb")
        self.conn.sendall(f"{token}\n".encode())
        if self.reader.readline() != b"ok whisper\n":
            sys.exit("refused (token?)")

    def ask(self, line):
        self.conn.sendall(line.encode() + b"\n")
        lines = []
        while text := self.reader.readline().decode():
            lines.append(text.rstrip("\n"))
            if text.startswith(("ok", "err")):
                break
        return lines


class Memory:
    def __init__(self, whisper):
        self.whisper, self.blocks = whisper, {}

    def r32(self, address):
        base = address & ~0xFF
        if base not in self.blocks:
            lines = self.whisper.ask(f"peek {base:x} 64")
            self.blocks[base] = [0 if line.split()[1] == "?" else int(line.split()[1], 16) for line in lines[:-1]]
        return self.blocks[base][(address & 0xFF) >> 2]

    def s32(self, address):
        value = self.r32(address)
        return value - (1 << 32) if value & 0x80000000 else value


def rooms(whisper):
    mem = Memory(whisper)
    player = mem.r32(0x7A6A70)
    room1 = mem.r32(mem.r32(player + 0x2C) + 0x1C) if player else 0
    if not room1:
        return ["no player room"]
    my_room2 = mem.r32(room1 + 0x10)
    level = mem.r32(my_room2 + 0x58)
    x0, y0 = mem.s32(level + 0x1C), mem.s32(level + 0x20)
    out = [f"level {mem.r32(level + 0x1d0)} seed {mem.r32(mem.r32(level + 0x1b4) + 0x458)} at {x0},{y0} size {mem.s32(level + 0x24)}x{mem.s32(level + 0x28)}"]
    path = mem.r32(player + 0x2C)
    out.append(f"player subtile {mem.r32(path + 2) & 0xffff},{mem.r32(path + 6) & 0xffff} room2 {mem.s32(my_room2 + 0x34) - x0},{mem.s32(my_room2 + 0x38) - y0}")
    room, index = mem.r32(level + 0x10), 0
    while room:
        r1 = mem.r32(room + 0x30)
        out.append(
            f"list{index} room {mem.s32(room + 0x34) - x0},{mem.s32(room + 0x38) - y0} {mem.s32(room + 0x3c)}x{mem.s32(room + 0x40)}"
            f" kind {mem.r32(room + 0x48)} seed {mem.r32(room + 4):08x}/{mem.r32(room + 8):08x} s14 {mem.r32(room + 0x14):08x}/{mem.r32(room + 0x18):08x}"
            f" mask {mem.r32(room + 0x50):x} tiles {'yes' if mem.r32(room + 0x54) else 'no'} flags {mem.r32(room + 0x60):x}"
            f" room1 {'%08x' % r1 if r1 else '-'} r1seed {('%08x' % mem.r32(r1 + 0x6c)) if r1 else '-'}"
        )
        room, index = mem.r32(room + 0x24), index + 1
    return out


def main():
    if len(sys.argv) < 3:
        sys.exit(__doc__)
    token = open(sys.argv[2]).read().split()[0]
    whisper = Whisper(sys.argv[1], token)
    verb = sys.argv[3] if len(sys.argv) > 3 else "rooms"
    if verb == "rooms":
        print("\n".join(rooms(whisper)))
    else:
        print("\n".join(whisper.ask(verb)))


if __name__ == "__main__":
    main()
