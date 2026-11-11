"""decomp9 (worker host): find the srvdofunc table of game.exe 1.14d by the known entry do 58 = FUN_005c56c0 and print entries around the ids given.
    python d9_dotab.py ID [ID..]       (needs D2_MPQ_DIR with bin/game.exe)"""
import os, struct, sys, pefile
pe = pefile.PE(os.path.join(os.environ["D2_MPQ_DIR"], "bin", "game.exe"), fast_load=True)
img = pe.get_memory_mapped_image(); base = pe.OPTIONAL_HEADER.ImageBase
needle = struct.pack("<I", 0x5c56c0)
pos = [i for i in range(0, len(img) - 4) if img[i:i + 4] == needle]
print("hits of 0x5c56c0 in image:", [hex(base + p) for p in pos])
for p in pos:
    start = p - 58 * 4
    for want in map(int, sys.argv[1:]):
        v = struct.unpack_from("<I", img, start + want * 4)[0]
        print("table at %#x: do %d -> %#x" % (base + start, want, v))
