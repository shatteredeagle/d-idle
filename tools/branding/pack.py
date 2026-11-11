#!/usr/bin/env python3
"""Packs the Aseprite PNG output (assets/branding/out/png) into the shipped formats and checks them. Stdlib only.

  pack.py OUT          write ico/icns/bmp/favicon and the per-consumer folders under OUT
  pack.py --check OUT  verify sizes, the 16-colour limit and that every file the consumers expect exists
  pack.py --preview OUT FILE.png   contact sheet of the small icons, upscaled, for looking at them
"""
import os, struct, sys, zlib

def read_png(path):
    d = open(path, "rb").read()
    assert d[:8] == b"\x89PNG\r\n\x1a\n", path
    pos, idat, w = 8, b"", None
    while pos < len(d):
        n, t = struct.unpack(">I4s", d[pos:pos + 8]); body = d[pos + 8:pos + 8 + n]; pos += 12 + n
        if t == b"IHDR": w, h, bd, ct, _, _, il = struct.unpack(">IIBBBBB", body)
        elif t == b"IDAT": idat += body
    assert bd == 8 and ct == 6 and il == 0, f"{path}: need 8-bit RGBA non-interlaced"
    bpp = 4
    raw = zlib.decompress(idat); stride = w * bpp; rows = []; prev = bytearray(stride); p = 0
    for _ in range(h):
        f = raw[p]; cur = bytearray(raw[p + 1:p + 1 + stride]); p += 1 + stride
        for i in range(stride):
            a = cur[i - bpp] if i >= bpp else 0; b = prev[i]; c = prev[i - bpp] if i >= bpp else 0
            if f == 1: cur[i] = (cur[i] + a) & 255
            elif f == 2: cur[i] = (cur[i] + b) & 255
            elif f == 3: cur[i] = (cur[i] + (a + b) // 2) & 255
            elif f == 4:
                pa, pb, pc = abs(b - c), abs(a - c), abs(a + b - 2 * c)
                cur[i] = (cur[i] + (a if pa <= pb and pa <= pc else b if pb <= pc else c)) & 255
        rows.append(bytes(cur)); prev = cur
    return w, h, rows

def write_png(path, w, h, rows):
    raw = b"".join(b"\0" + r for r in rows)
    def ch(t, b): return struct.pack(">I", len(b)) + t + b + struct.pack(">I", zlib.crc32(t + b))
    open(path, "wb").write(b"\x89PNG\r\n\x1a\n" + ch(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 6, 0, 0, 0)) + ch(b"IDAT", zlib.compress(raw, 9)) + ch(b"IEND", b""))

def colours(rows):
    s = set()
    for r in rows:
        for i in range(0, len(r), 4): s.add(r[i:i + 4])
    return s

def ico(entries):
    """entries: [(size, png bytes)]; PNG-compressed entries (Windows Vista and later)."""
    head = struct.pack("<HHH", 0, 1, len(entries)); off = 6 + 16 * len(entries); dirs = b""
    for s, png in entries:
        dirs += struct.pack("<BBBBHHII", s % 256, s % 256, 0, 0, 1, 32, len(png), off); off += len(png)
    return head + dirs + b"".join(p for _, p in entries)

def icns(entries):
    """entries: [(type, png bytes)]; PNG-compressed chunks."""
    body = b"".join(t.encode() + struct.pack(">I", 8 + len(p)) + p for t, p in entries)
    return b"icns" + struct.pack(">I", 8 + len(body)) + body

def bmp4(w, h, rows):
    """32-bit BITMAPV4HEADER with alpha, bottom-up BGRA: what d2d's SDL_LoadBMP expects (same as ImageMagick BMP4)."""
    px = b"".join(b"".join(bytes((r[i + 2], r[i + 1], r[i], r[i + 3])) for i in range(0, len(r), 4)) for r in reversed(rows))
    hdr = struct.pack("<IiiHHIIiiII", 108, w, h, 1, 32, 3, len(px), 2835, 2835, 0, 0)
    hdr += struct.pack("<IIII", 0x00FF0000, 0x0000FF00, 0x000000FF, 0xFF000000) + b"BGRs" + b"\0" * 36 + b"\0" * 12
    off = 14 + len(hdr)
    return b"BM" + struct.pack("<IHHI", off + len(px), 0, 0, off) + hdr + px

def png(out, n): return open(f"{out}/png/{n}", "rb").read()
def wr(out, rel, data):
    p = f"{out}/{rel}"; os.makedirs(os.path.dirname(p), exist_ok=True); open(p, "wb").write(data)

ICNS = [("icp4", 16), ("icp5", 32), ("ic11", 32), ("ic12", 64), ("ic07", 128), ("ic13", 256), ("ic08", 256), ("ic14", 512), ("ic09", 512), ("ic10", 1024)]
ICNS_ELECTRON = [("ic07", 128), ("ic08", 256), ("ic09", 512), ("ic10", 1024)]  # the layout apps/desktop-electron/test/platform.test.js pins
ICO_SIZES = [16, 32, 48, 64, 128, 256]

def pack(out):
    def icon(n):
        return png(out, f"icon-{n}.png")
    ico_win = ico([(n, icon(n)) for n in ICO_SIZES])
    icns_all = icns([(t, icon(n)) for t, n in ICNS])
    # shared
    wr(out, "favicon.ico", ico([(n, icon(n)) for n in (16, 32, 48)]))
    wr(out, "icon.ico", ico_win); wr(out, "icon.icns", icns_all)
    wr(out, "favicon-32.png", icon(32))
    # Electron: build/icon.png (512), icon.ico, icon.icns, tray.png (32), trayTemplate (22) and @2x (44)
    for rel, data in [("electron/icon.png", icon(512)), ("electron/icon.ico", ico_win), ("electron/icon.icns", icns([(t, icon(n)) for t, n in ICNS_ELECTRON])),
                      ("electron/tray.png", icon(32)), ("electron/trayTemplate.png", png(out, "template-22.png")),
                      ("electron/trayTemplate@2x.png", png(out, "template-44.png"))]:
        wr(out, rel, data)
    # d2d (assets/icon in the fork): d2d.png 256 launcher window icon, d2d.bmp 64 BMP4 window/dock icon, d2d.ico, d2d.icns, hero.png
    w, h, rows = read_png(f"{out}/png/icon-64.png")
    for rel, data in [("d2d/d2d.png", icon(256)), ("d2d/d2d.bmp", bmp4(w, h, rows)), ("d2d/d2d.ico", ico_win),
                      ("d2d/d2d.icns", icns_all), ("d2d/hero.png", png(out, "hero-1120x135.png"))]:
        wr(out, rel, data)
    wr(out, "banner-1280x320.png", png(out, "banner-1280x320.png")); wr(out, "card-1200x630.png", png(out, "card-1200x630.png"))
    print("packed", out)

def check(out):
    bad = []
    want = {"icon-16.png": 16, "icon-32.png": 32, "icon-48.png": 48, "icon-64.png": 64, "icon-128.png": 128, "icon-256.png": 256,
            "icon-256-native.png": 256, "icon-512.png": 512, "icon-1024.png": 1024, "template-16.png": 16, "template-22.png": 22, "template-32.png": 32, "template-44.png": 44}
    for n, s in want.items():
        w, h, rows = read_png(f"{out}/png/{n}")
        if (w, h) != (s, s): bad.append(f"{n}: {w}x{h}, want {s}")
        c = colours(rows)
        if len({x[:3] for x in c if x[3]}) > 15: bad.append(f"{n}: more than 15 opaque colours")
        if any(0 < x[3] < 255 for x in c): bad.append(f"{n}: partial alpha")
        if n.startswith("template"):
            if any(x[3] and x[:3] != b"\0\0\0" for x in c): bad.append(f"{n}: not black on transparent")
    for n, (w0, h0) in {"banner-1280x320.png": (1280, 320), "card-1200x630.png": (1200, 630), "hero-1120x135.png": (1120, 135)}.items():
        w, h, rows = read_png(f"{out}/png/{n}")
        if (w, h) != (w0, h0): bad.append(f"{n}: {w}x{h}")
        if len({x[:3] for x in colours(rows) if x[3]}) > 15: bad.append(f"{n}: more than 15 opaque colours")
    for rel in ["favicon.ico", "icon.ico", "icon.icns", "electron/icon.png", "electron/icon.ico", "electron/icon.icns", "electron/tray.png",
                "electron/trayTemplate.png", "electron/trayTemplate@2x.png", "d2d/d2d.png", "d2d/d2d.bmp", "d2d/d2d.ico", "d2d/d2d.icns", "d2d/hero.png"]:
        if not os.path.exists(f"{out}/{rel}") or os.path.getsize(f"{out}/{rel}") == 0: bad.append("missing " + rel)
    b = open(f"{out}/d2d/d2d.bmp", "rb").read()
    if struct.unpack("<ii", b[18:26]) != (64, 64) or struct.unpack("<H", b[28:30])[0] != 32: bad.append("d2d.bmp is not 64x64x32")
    if open(f"{out}/d2d/d2d.icns", "rb").read(4) != b"icns": bad.append("icns magic")
    if bad: sys.exit("branding check FAILED:\n  " + "\n  ".join(bad))
    print("branding check ok")

def preview(out, dest):
    sheet_h, parts, x = 0, [], 0
    for n, f in [("icon-16.png", 10), ("icon-32.png", 6), ("icon-48.png", 4), ("template-16.png", 10), ("template-22.png", 8)]:
        w, h, rows = read_png(f"{out}/png/{n}")
        if n.startswith("template"):
            rows = [b"".join(bytes((255,255,255,255)) if r[i+3] == 0 else bytes((0,0,0,255)) for i in range(0, len(r), 4)) for r in rows]
        big = [b"".join(r[i:i+4] * f for i in range(0, len(r), 4)) for r in rows for _ in range(f)]
        parts.append((x, big, w * f)); x += w * f + 16; sheet_h = max(sheet_h, h * f)
    rows = [bytearray(b"\x40\x40\x48\xff" * x) for _ in range(sheet_h)]
    for px, big, bw in parts:
        for y, r in enumerate(big): rows[y][px * 4:px * 4 + bw * 4] = r
    write_png(dest, x, sheet_h, [bytes(r) for r in rows])

if __name__ == "__main__":
    a = sys.argv[1:]
    if a[:1] == ["--check"]: check(a[1])
    elif a[:1] == ["--preview"]: preview(a[1], a[2])
    else: pack(a[0])
