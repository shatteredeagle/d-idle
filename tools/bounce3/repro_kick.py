#!/usr/bin/env python3
"""Warp arrival bounce, end to end on a THROWAWAY headless d2d (docs/bounce3.md). Never the sandbox container.
Usage: python3 -I tools/bounce3/repro_kick.py <socket> [--reach PX] [--safe]
Expects a d2d started with --headless --lab-verbs --devctl <socket> (tools/bounce3/repro.sh starts one in a throwaway
container of the play image). For each (from, to) pair it takes the warp with `debug enter`, then sends the bot's
priming click exactly as internal/d2dinput.kickToward places it (110 px from the feet toward a move target 4 cells
straight away from the return warp) and reports whether the level changed back. --safe places the click the way the
fixed bot does (kickSafe: outside warpClickR of every warp of the level). Exit 0 = no bounce, 1 = at least one bounce."""
import math, socket, sys, time

sock = sys.argv[1]
reach = float(sys.argv[sys.argv.index("--reach") + 1]) if "--reach" in sys.argv else 110.0
safe = "--safe" in sys.argv
PX, PY, SU, SV = 400.0, 284.0, 80.0, 40.0   # input.Calibrated per cell (16 and 8 px a subtile, 5 subtiles a cell)
R = 2.0                                     # d2d walk_to: a fresh move within 2 cells of a warp's centre takes it
MARGIN = 0.75


def cmd(line):
    with socket.socket(socket.AF_UNIX) as s:
        s.settimeout(10)
        s.connect(sock)
        s.sendall((line + "\n").encode())
        out = ""
        while not out.endswith("\n") or not out.rsplit("\n", 2)[-2].startswith(("ok", "err")):
            chunk = s.recv(65536)
            if not chunk:
                break
            out += chunk.decode()
    return out


def level():
    f = cmd("debug level").split()
    return int(f[1]), float(f[2]), float(f[3])


def warps():
    rows = []
    for ln in cmd("debug warps").splitlines():
        p = ln.split("\t")
        if len(p) == 4:
            rows.append((int(p[0]), float(p[1]), float(p[2]), int(p[3])))
    return rows


def to_screen(px, py, x, y):
    dx, dy = x - px, y - py
    return PX + SU * (dx - dy), PY + SV * (dx + dy)


def to_cell(px, py, sx, sy):
    u, v = (sx - PX) / SU, (sy - PY) / SV
    return px + (u + v) / 2, py + (v - u) / 2


def kick_pixel(px, py, tx, ty, ws):
    sx, sy = to_screen(px, py, tx, ty)
    dx, dy = sx - PX, sy - PY
    d = math.hypot(dx, dy)
    k = reach
    while True:
        x, y = PX + dx / d * k, PY + dy / d * k
        x, y = min(max(x, 5), 796), min(max(y, 5), 518)
        if not safe:
            return int(x), int(y)
        cx, cy = to_cell(px, py, int(x), int(y))
        if all(math.hypot(w[1] + .5 - cx, w[2] + .5 - cy) >= R + MARGIN for w in ws) or k > 600:
            return int(x), int(y)
        k += 20


def wait_level(want, secs=6):
    end = time.time() + secs
    while time.time() < end:
        lv = level()
        if lv[0] == want:
            return lv
        time.sleep(0.1)
    return level()


def trial(src, dst):
    cmd(f"debug goto {src}")
    time.sleep(0.5)
    w = [x for x in warps() if x[3] == dst]
    if not w:
        return None
    cmd(f"debug enter {w[0][0]}")
    lv = wait_level(dst)
    if lv[0] != dst:
        return None
    time.sleep(1.0)
    _, px, py = level()
    back = [x for x in warps() if x[3] == src]
    ws = warps()
    bx, by = (back[0][1] + .5, back[0][2] + .5) if back else (px, py - 1)
    ax, ay = px - bx, py - by
    d = math.hypot(ax, ay) or 1
    tx, ty = px + ax / d * 4, py + ay / d * 4
    x, y = kick_pixel(px, py, tx, ty, ws)
    cx, cy = to_cell(px, py, x, y)
    cmd(f"click {x} {y}")
    time.sleep(1.5)
    after = level()
    bounced = after[0] == src
    print(f"{src}->{dst} arrive=({px:.1f},{py:.1f}) warp=({bx:.1f},{by:.1f}) d={math.hypot(px-bx,py-by):.2f} "
          f"kick=({x},{y}) cell=({cx:.1f},{cy:.1f}) kickToWarp={math.hypot(cx-bx,cy-by):.2f} -> level {after[0]} {'BOUNCE' if bounced else 'stay'}")
    return bounced


pairs = [(3, 9), (9, 3), (2, 8), (8, 2), (1, 2), (2, 3)]
bounces = runs = 0
for src, dst in pairs:
    b = trial(src, dst)
    if b is None:
        print(f"{src}->{dst}: no warp/arrival, skipped")
        continue
    runs += 1
    bounces += b
print(f"RESULT runs={runs} bounces={bounces} reach={reach} safe={safe}")
sys.exit(1 if bounces else 0)
