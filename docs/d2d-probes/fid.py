#!/usr/bin/env python3
"""D2DFIDELITY helper library: start/stop one niced lockstep d2d (private fork), talk to it, safety checks.
Runs on the worker. Needs the private fork at ~/d2d-lab/fork (see docs/d2d-fork.md); data dir ~/d2d-lab/fidelity (not in repo)."""
import os, re, socket, subprocess, sys, time, json
LAB = os.path.expanduser("~/d2d-lab")
FID = LAB + "/fidelity"
BIN = LAB + "/fork/src/build/apps/d2d/d2d"
SOCK = FID + "/d2d.sock"
HOME = FID + "/home"

def safety():
    """pause (no d2d running) while the 1-minute load average is above 45; stop for good if / has < 15 GB free"""
    for _ in range(60):
        la = float(open("/proc/loadavg").read().split()[0])
        st = os.statvfs("/"); free = st.f_bavail * st.f_frsize / 2**30
        if free < 15: raise SystemExit("safety stop: free %.1f GB" % free)
        if la <= 45: return la, free
        stop(); time.sleep(30)
    raise SystemExit("safety stop: load stayed above 45 for 30 min")

class D:
    def __init__(s, timeout=1800):
        s.s = socket.socket(socket.AF_UNIX); s.s.settimeout(timeout); s.s.connect(SOCK)
        s.f = s.s.makefile("rwb", buffering=0)
    def q(s, cmd):
        s.f.write((cmd + "\n").encode()); out = []
        while True:
            l = s.f.readline().decode().rstrip("\n")
            if l == "": raise IOError("eof")
            if l == "ok" or l.startswith("ok ") or l.startswith("err"):
                return out + ([l] if l != "ok" else [])
            out.append(l)
    realtime = False
    def step(s, n):
        if s.realtime: time.sleep(n * 0.04); return
        while n > 0:                      # chunks keep one reply short
            k = min(n, 5000); s.q("step %d" % k); n -= k
    def state(s):
        r = s.q("state")[0]; return dict(re.findall(r"(\w+)=(\S*)", r))
    def pair(s, key):
        a, b = s.state()[key].split("/"); return float(a), float(b)
    def monsters(s):
        r = []
        for l in s.q("monsters"):
            f = l.split("\t")
            if len(f) >= 12:
                r.append(dict(id=f[0], x=float(f[1]), y=float(f[2]), hp=f[5], mode=f[6], boss=f[7], mods=f[8], lvl=int(f[9][3:]), name=f[10], unit=int(f[11][1:])))
        return r
    def player_xy(s):
        r = s.q("debug level")[-1].split(); return float(r[2]), float(r[3])

def start(name, cls, seed=3, extra=(), lockstep=True):
    """fresh character via --start-class; lockstep on. Returns D."""
    safety()
    os.makedirs(HOME, exist_ok=True)
    stop()
    try: os.unlink(SOCK)
    except OSError: pass
    env = dict(os.environ, XDG_CONFIG_HOME=HOME + "/config", XDG_DATA_HOME=HOME + "/data", SDL_VIDEODRIVER="dummy")
    args = ["nice", "-n", "10", BIN, "--data", LAB + "/data", "--headless", "--devctl", SOCK, "--no-video", *(["--lockstep"] if lockstep else []),
            "--seed", str(seed), "--start-screen", "ingame", "--start-class", str(cls), "--start-name", name, *extra]
    subprocess.Popen(args, env=env, stdout=open(FID + "/d2d.log", "a"), stderr=subprocess.STDOUT, start_new_session=True)
    for _ in range(120):
        try:
            d = D(); d.q("ping"); d.realtime = not lockstep; time.sleep(1.5); return d
        except Exception: time.sleep(0.5)
    raise SystemExit("d2d did not start")

def stop():
    subprocess.run(["pkill", "-f", "^" + BIN], check=False); time.sleep(1.0)

def prime(d):
    d.q("click 400 500"); d.step(2)

def nums(line):
    return [float(x) for x in re.findall(r"-?\d+\.?\d*", line)]
