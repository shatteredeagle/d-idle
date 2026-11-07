#!/usr/bin/env python3
"""Currency slice 1 live acceptance (docs/probes/currency1.md): a fresh headless d2d with the didle-v1 overlay and
--lab-verbs spawns and kills Blood Raven N times in the Burial Grounds, then counts `mrk` items on the ground.
The character is throwaway (--no-save, private HOME); no Wine, no display, no Orion game instance.
Usage: currency1-drill.py <d2d> <overlay-dir> [N=300] [batch=15]   env D2_MPQ_DIR = the install (default /tmp/d2mpq)
Prints one JSON line (counts, expected, interval). Exit 0 when the count is inside the 99.9 percent interval."""
import json, math, os, shutil, socket, subprocess, sys, tempfile, time

d2d, overlay = sys.argv[1], sys.argv[2]
n_total = int(sys.argv[3]) if len(sys.argv) > 3 else 300
batch = int(sys.argv[4]) if len(sys.argv) > 4 else 15
data = os.environ.get("D2_MPQ_DIR", "/tmp/d2mpq")
PER_KILL = 1 - (1 - 1 / 183) ** 5          # Blood Raven: 5 picks, 1/61 for the DI Mark A entry, then 1/3 (docs/research/currency-rows.md)

home = tempfile.mkdtemp(prefix="d2d_cur1_")
os.makedirs(os.path.join(home, ".config/d2d/save"))
sock_path = os.path.join(tempfile.gettempdir(), f"d2d_cur1_{os.getpid()}.sock")
proc = subprocess.Popen([d2d, "--headless", "--no-save", "--seed", "7", "--data", data, "--devctl", sock_path, "--lab-verbs",
                         "--overlay-dir", overlay, "--start-screen", "charcreate", "--start-class", "1", "--start-name", "Marky", "--scale", "2"],
                        env=dict(os.environ, HOME=home))

def cmd(line, tries=1):
    with socket.socket(socket.AF_UNIX) as s:
        s.settimeout(30)
        s.connect(sock_path)
        s.sendall((line + "\n").encode())
        out = ""
        while True:
            out += s.recv(65536).decode()
            last = out.rstrip("\n").rsplit("\n", 1)[-1]
            if last.startswith(("ok", "err")):
                return out

def frames(n):
    cmd(f"step {n}")

try:
    for _ in range(600):
        if os.path.exists(sock_path):
            break
        assert proc.poll() is None, "d2d exited during startup"
        time.sleep(0.1)
    info = cmd("info")
    assert "overlay=" in info and "overlay=off" not in info, info
    frames(1)
    cmd("click 326 532"); frames(2)       # classic character (as tests/smoke_d2d.py)
    cmd("click 690 555"); frames(10)
    assert "ingame" in cmd("state"), cmd("state")
    r = cmd("debug goto 17")              # Burial Grounds (Levels.txt id 17: U, the first run confirms)
    assert r.startswith("ok"), r
    drops, kills, log = 0, 0, []
    for b in range(0, n_total, batch):
        k = min(batch, n_total - b)
        r = cmd(f"debug spawn bloodraven {k}")
        assert r.startswith("ok"), r
        frames(5)
        cmd("debug kill")
        frames(60)
        mons = [l for l in cmd("monsters").splitlines() if "bloodraven" in l.lower() or "Blood Raven" in l]
        kills += k
        ground = cmd("ground").splitlines()
        marks = sum(1 for l in ground if l.split("\t")[0] == "mrk")
        log.append({"kills": kills, "marks_on_ground": marks, "raven_left": len(mons)})
    ground = cmd("ground").splitlines()
    marks = [l for l in ground if l.split("\t")[0] == "mrk"]
    labels = sorted({l.split("\t")[1] for l in marks if len(l.split("\t")) > 1})
    mean = kills * PER_KILL
    sd = math.sqrt(kills * PER_KILL * (1 - PER_KILL))
    lo, hi = max(0, mean - 3.3 * sd), mean + 3.3 * sd
    res = {"kills": kills, "mrk_items": len(marks), "labels": labels, "expected": round(mean, 2), "sd": round(sd, 2),
           "interval99.9": [round(lo, 1), round(hi, 1)], "batches": log, "info": info.strip()}
    print(json.dumps(res))
    sys.exit(0 if lo <= len(marks) <= hi else 1)
finally:
    if proc.poll() is None:
        proc.terminate()
        try:
            proc.wait(10)
        except subprocess.TimeoutExpired:
            proc.kill()
    shutil.rmtree(home, ignore_errors=True)
    if os.path.exists(sock_path):
        os.remove(sock_path)
