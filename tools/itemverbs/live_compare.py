#!/usr/bin/env python3
"""Live click-vs-verb comparison of the item actions (docs/design/item-verbs.md 9a, docs/probes/itemverbs-live-*.md).

THROWAWAY ONLY: runs on the worker host with its own temp data dir, ports and a fresh character; it never touches the
sandbox container or its volume. For each mode (click = knob invdesk.verbs 0, verb = 1) and class it
  A. creates the hero in a harness d2d (lab verbs), sets the scenario up with declared SETUP cheats (`debug give`,
     `debug stat`), saves and quits;
  B. starts the real d2bot on that hero with an idle goal, runs the player actions through the watch API
     (/api/inventory/*, the same route the page uses), and dumps the inventory view after every step;
  C. compares the two modes step by step (item code, container, cell/slot, gold, scroll count) and records how=verb|click.
Usage: live_compare.py --bot BIN --d2d BIN --game DIR --out DIR [--classes 3,0] [--only STEP,...]
Needs: D2_* none. Exit 0 = all steps equal (or equal-by-design), 1 = a difference."""
import argparse, json, os, re, shutil, socket, subprocess, sys, tempfile, time, urllib.request, urllib.error

ap = argparse.ArgumentParser()
ap.add_argument("--bot", required=True); ap.add_argument("--d2d", required=True); ap.add_argument("--game", required=True)
ap.add_argument("--out", required=True); ap.add_argument("--classes", default="2,1"); ap.add_argument("--seed", default="3")
ap.add_argument("--port", type=int, default=18420); ap.add_argument("--probe", action="store_true")
args = ap.parse_args()
os.makedirs(args.out, exist_ok=True)
TOKEN = "ivtoken-0123456789abcdef"


class Dev:
    """One devctl line at a time (single client)."""
    def __init__(self, sock): self.sock = sock
    def cmd(self, line, tries=1):
        with socket.socket(socket.AF_UNIX) as s:
            s.settimeout(15); s.connect(self.sock); s.sendall((line + "\n").encode())
            out = ""
            while not out.endswith("\n") or not out.rsplit("\n", 2)[-2].startswith(("ok", "err")):
                c = s.recv(4096)
                if not c: break
                out += c.decode()
        return out


def stage_a(work, cls, name):
    """Create the hero and set the scenario up; save; quit. Returns the `state` line."""
    home = os.path.join(work, "instances", "iv", "d2d")
    os.makedirs(os.path.join(home, "config"), exist_ok=True); os.makedirs(os.path.join(home, "data"), exist_ok=True)
    sock = os.path.join(work, "a.sock")
    env = dict(os.environ, XDG_CONFIG_HOME=home + "/config", XDG_DATA_HOME=home + "/data", SDL_VIDEODRIVER="dummy", SDL_AUDIODRIVER="dummy")
    p = subprocess.Popen([args.d2d, "--data", args.game, "--headless", "--devctl", sock, "--no-video", "--seed", args.seed, "--lab-verbs",
                          "--toggle", "autoloot=off,autoparty=off", "--start-screen", "ingame", "--start-class", str(cls), "--start-name", name],
                         env=env, stdout=open(work + "/a.out", "w"), stderr=subprocess.STDOUT)
    try:
        for _ in range(300):
            if os.path.exists(sock): break
            time.sleep(0.2)
        d = Dev(sock)
        for _ in range(100):
            if "screen=ingame" in d.cmd("state"): break
            time.sleep(0.3)
        time.sleep(1.5)
        state = d.cmd("state")
        d.cmd("debug clearinv"); time.sleep(0.3)
        gives = ["hax pack", "hax pack", "cap pack", "plt pack", "hax pack unid", "hax pack unid", "hax pack unid", "isc pack", "isc pack", "sst pack", "cap pack q4 l60"]
        for g in gives:
            r = d.cmd("debug give " + g)
            if not r.startswith("ok"): print("setup give failed:", g, r.strip())
            time.sleep(0.15)
        d.cmd("debug stat 14 5000"); d.cmd("debug stat 15 2000"); d.cmd("debug stat 12 1")
        time.sleep(0.5)
        items = d.cmd("items")
        open(work + "/stageA.items", "w").write(items)
        print("stage A", name, "class", cls, re.findall(r"class=(\w+)", state), "saved:", d.cmd("save").strip())
        time.sleep(1.0)
        d.cmd("quit")
        p.wait(20)
    finally:
        if p.poll() is None: p.terminate(); p.wait(10)
    return state


def api(method, path, body=None):
    port = args.port + (2 if method == "POST" else 1)
    req = urllib.request.Request(f"http://127.0.0.1:{port}{path}", method=method, data=json.dumps(body).encode() if body is not None else None,
                                 headers={"Authorization": "Bearer " + TOKEN + "w", "Content-Type": "application/json", "Origin": f"http://127.0.0.1:{args.port+1}"})
    try:
        with urllib.request.urlopen(req, timeout=15) as r:
            return r.status, json.loads(r.read() or b"null")
    except urllib.error.HTTPError as e:
        raw = e.read()
        try: return e.code, json.loads(raw)
        except Exception: return e.code, raw.decode(errors="replace")
    except (urllib.error.URLError, ConnectionError, OSError) as e:
        return 0, str(e)


def start_bot(work, name, verbs):
    env = dict(os.environ, D2BOT_INSTANCE="iv", D2D_RUN="on", D2D_BIN=args.d2d, D2D_DATA=args.game)
    cfg = os.path.join(work, "config.json")
    json.dump({"tuning": {"invdesk.verbs": 1 if verbs else 0, "town.sellJunk": 0, "town.stashGoldBase": 1000000, "town.stashGoldPerLvl": 0}}, open(cfg, "w"))
    cmd = [args.bot, "--backend", "d2d", "--managed", "--d2d-class", "-1", "--char", name, "--seed", args.seed, "--toggle", "autoloot=off,autoparty=off",
           "--d2d-args", "--item-verbs", "--d2d-linkwait", "60s", "--gamedir", args.game, "--d2d-data", args.game, "--data", work,
           "--d2d-sock", work + "/d2d.sock", "--addr", f"127.0.0.1:{args.port}", "--token", "api" + TOKEN, "--config", cfg, "--run-for", "1500s", "--stop-game",
           "--ladder", "--watch", "--watch-addr", f"127.0.0.1:{args.port+1}", "--watch-answer-addr", f"127.0.0.1:{args.port+2}",
           "--watch-origin", f"http://127.0.0.1:{args.port+1}", "--watch-token", TOKEN + "w"]
    p = subprocess.Popen(cmd, env=env, stdout=open(work + "/bot.log", "w"), stderr=subprocess.STDOUT)
    for _ in range(300):
        st, body = api("GET", "/api/inventory")
        if _ % 10 == 0: print("api", st, str(body)[:160], flush=True)
        if st == 200: return p
        if p.poll() is not None: raise SystemExit("bot exited early: see " + work + "/bot.log")
        time.sleep(1)
    p.terminate(); p.wait(30)
    raise SystemExit("the inventory API never answered: " + work + "/bot.log")



def view():
    for _ in range(8):
        st, v = api("GET", "/api/inventory")
        if st == 200 and isinstance(v, dict):
            return v
        time.sleep(1)
    return {}


def norm(v):
    """The state a step leaves, independent of unit ids."""
    pack = lambda lst: sorted((i["name"], i.get("x", -1), i.get("y", -1), bool(i.get("identified"))) for i in lst or [])
    eq = {k: (i["name"], bool(i.get("identified"))) for k, i in (v.get("equipment") or {}).items()}
    return {"equipment": eq, "backpack": pack(v.get("backpack")), "stash": pack(v.get("stash")), "belt": len(v.get("belt") or []),
            "gold": v.get("gold"), "stashGold": v.get("stashGold")}


def find(v, name, where="backpack", ident=None, nth=0):
    hits = [i for i in (v.get(where) or []) if i["name"] == name and (ident is None or bool(i.get("identified")) == ident)]
    return hits[nth] if len(hits) > nth else None


def wait_idle(timeout=120):
    t0 = time.time()
    while time.time() - t0 < timeout:
        v = view()
        if v and not v.get("queue"):
            time.sleep(1.0)
            return v
        time.sleep(0.5)
    return view()


def step(name, path, body):
    t0 = time.time()
    st, rep = api("POST", path, body)
    rec = {"step": name, "http": st}
    if st != 200:
        rec["refused"] = rep.get("code") if isinstance(rep, dict) else str(rep)[:80]
        rec["state"] = norm(view())
        return rec
    v = wait_idle()
    rec["secs"] = round(time.time() - t0, 1)
    rec["state"] = norm(v)
    last = (v.get("recent") or [{}])[0]
    rec["result"] = last.get("state")
    rec["reason"] = last.get("reason", "")
    rec["how"] = last.get("how", "")
    return rec


def scenario():
    """The same steps in both modes. A step whose item is missing is recorded as skipped (and differs between modes then)."""
    out = []
    def do(name, path, pick, extra=None):
        v = view()
        it = pick(v)
        if it is None:
            out.append({"step": name, "skipped": "no such item"}); return
        body = dict(extra or {}); body["id"] = it["id"]
        out.append(step(name, path, body))
    out.append({"step": "start", "state": norm(view())})
    do("equip cap -> head (empty slot)", "/api/inventory/equip", lambda v: find(v, "Cap", ident=True, nth=0), {"slot": "head"})
    do("equip full plate -> torso (strength too low)", "/api/inventory/equip", lambda v: find(v, "Plate Mail"), {"slot": "torso"})
    do("equip hand axe -> rhand (swap)", "/api/inventory/equip", lambda v: find(v, "Hand Axe", ident=True, nth=0), {"slot": "rhand"})
    do("equip short staff -> rhand (two-hander)", "/api/inventory/equip", lambda v: find(v, "Short Staff"), {"slot": "rhand"})
    do("identify hand axe 1 (scroll)", "/api/inventory/identify", lambda v: find(v, "Hand Axe", ident=False, nth=0))
    do("identify hand axe 2 (scroll)", "/api/inventory/identify", lambda v: find(v, "Hand Axe", ident=False, nth=0))
    do("identify hand axe 3 (no scroll left)", "/api/inventory/identify", lambda v: find(v, "Hand Axe", ident=False, nth=0))
    do("drop hand axe (trash)", "/api/inventory/trash", lambda v: find(v, "Hand Axe", ident=True, nth=0), {"confirm": True})
    do("stash cap (pack -> stash)", "/api/inventory/stash", lambda v: find(v, "Cap", ident=True, nth=0))
    do("take cap (stash -> pack)", "/api/inventory/take", lambda v: find(v, "Cap", where="stash"))
    out.append(step("gold deposit 300", "/api/inventory/gold", {"op": "deposit", "count": 300}))
    out.append(step("gold withdraw 100", "/api/inventory/gold", {"op": "withdraw", "count": 100}))
    return out


def ctl(method, path, body=None):
    req = urllib.request.Request(f"http://127.0.0.1:{args.port}{path}", method=method, data=json.dumps(body).encode() if body is not None else None,
                                 headers={"Authorization": "Bearer api" + TOKEN, "Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=15) as r: return r.status, r.read()[:200]
    except urllib.error.HTTPError as e: return e.code, e.read()[:200]
    except OSError as e: return 0, str(e)


def run_mode(root, cls, mode):
    work = tempfile.mkdtemp(prefix=f"iv_{cls}_{mode}_", dir=root)
    stage_a(work, cls, "Ivp")
    p = start_bot(work, "Ivp", mode == "verb")
    try:
        print("goal", ctl("POST", "/api/v1/goals", {"type": "farm_area", "params": {"area": "Blood Moor"}}), ctl("POST", "/api/v1/control/start"), flush=True)
        time.sleep(3)
        res = scenario()
    finally:
        p.terminate()
        try: p.wait(40)
        except Exception: p.kill()
    json.dump(res, open(f"{root}/trace_{cls}_{mode}.json", "w"), indent=1)
    return res


if __name__ == "__main__":
    root = os.path.abspath(args.out)
    diffs = 0
    table = []
    for cls in [int(c) for c in args.classes.split(",")]:
        traces = {m: run_mode(root, cls, m) for m in ("click", "verb")}
        for a, b in zip(traces["click"], traces["verb"]):
            same = {k: a.get(k) for k in ("step", "http", "refused", "state", "result", "skipped")} == {k: b.get(k) for k in ("step", "http", "refused", "state", "result", "skipped")}
            diffs += 0 if same else 1
            table.append((cls, a["step"], "same" if same else "DIFF", a.get("result") or a.get("refused") or a.get("skipped") or "", a.get("how", ""), b.get("how", ""), a.get("secs", ""), b.get("secs", ""), b.get("reason", "") if not same else ""))
    print("class\tstep\tverdict\toutcome\thow(click run)\thow(verb run)\tsecs click\tsecs verb")
    for r in table: print("\t".join(str(x) for x in r))
    print("differences:", diffs)
    sys.exit(1 if diffs else 0)
