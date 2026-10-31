#!/usr/bin/env python3
"""Our own thin client + helpers for the d2d dev channel (run on the worker next to ~/d2d-lab).
Library use:  import d2p; d=d2p.D(); d.talk('Kashya'); ...
CLI use:      d2p.py "verb1" "verb2" ...   (also helper pseudo-verbs: talk NAME, pick TEXT, goto N)"""
import socket, sys, time, os, re
SOCK = os.path.expanduser("~/d2d-lab/d2d.sock")

class D:
    def __init__(s, timeout=20):
        s.s = socket.socket(socket.AF_UNIX); s.s.settimeout(timeout); s.s.connect(SOCK)
        s.f = s.s.makefile("rwb", buffering=0)
    def q(s, cmd):
        """send one line, return payload lines (a final 'ok ...' / 'err ...' line is kept)"""
        s.f.write((cmd + "\n").encode()); out = []
        while True:
            l = s.f.readline().decode().rstrip("\n")
            if l == "" : raise IOError("eof")
            if l == "ok": return out
            if l.startswith("ok ") or l.startswith("err"): return out + [l]
            out.append(l)
    def p(s, *cmds):
        for c in cmds:
            print("> " + c)
            if c.startswith("wait "): time.sleep(float(c[5:])); continue
            print("\n".join(s.q(c)))
    def state(s):
        r = s.q("state")[0]; d = {}
        for m in re.finditer(r"(\w+)=(\S*)", r): d[m.group(1)] = m.group(2)
        return d
    def gold(s): return int(s.state()["gold"])
    def life(s): a, b = s.state()["life"].split("/"); return int(a), int(b)
    def npcs(s):
        r = []
        for l in s.q("npcs"):
            f = l.split("\t")
            if len(f) >= 7: r.append(dict(name=f[0], sx=int(f[1]), sy=int(f[2]), menu=f[3], idx=int(f[4]), cx=float(f[5]), cy=float(f[6])))
        return r
    def menu(s):
        r = []
        for l in s.q("menu"):
            f = l.split("\t")
            if len(f) >= 3: r.append((f[0], int(f[1]), int(f[2])))
        return r
    def prime(s): s.q("click 400 500")        # quirk: moves are ignored after warp/load until one click
    def talk(s, name, tries=20, gap=2.5):
        """walk to NPC and open its menu; returns menu lines"""
        s.prime()
        for _ in range(tries):
            n = [x for x in s.npcs() if x["name"].lower().startswith(name.lower())]
            if not n: time.sleep(1); continue
            s.q("cmd interact %d" % n[0]["idx"]); time.sleep(gap)
            m = s.menu()
            if m: return m
        return []
    def pick(s, text):
        for (t, x, y) in s.menu():
            if text.lower() in t.lower():
                s.q("click %d %d" % (x, y)); return True
        return False
    def goto(s, lvl):
        r = s.q("debug goto %d" % lvl); time.sleep(3); s.prime(); return r
    def esc_all(s):
        for _ in range(3):
            if s.state()["gamemenu"].split(":")[0] == "0": break
            s.q("key Escape"); time.sleep(0.3)

if __name__ == "__main__":
    d = D()
    for c in sys.argv[1:]:
        if c.startswith("talk "): print("> " + c); print(d.talk(c[5:]))
        elif c.startswith("pick "): print("> " + c, d.pick(c[5:]))
        elif c.startswith("goto "): print("> " + c); print(d.goto(int(c[5:])))
        else: d.p(c)
