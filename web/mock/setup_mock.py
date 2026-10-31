#!/usr/bin/env python3
"""Offline mock of the setup wizard backend (docs/setup.md) for developing and screenshotting web/setup.html. Stdlib only.

  python3 web/mock/setup_mock.py [--port 8440] [--speed 1.0] [--scenario happy|fail|gamedir]

Serves web/ at / and the /api/setup/ routes (state, checks, inputs, start, cancel, reset) from a scripted step progression.
POST /mock/scenario?name=happy|fail|gamedir|fresh resets the mock. Scenarios:
  happy    installer-dirs mode, installers wait for the user, everything succeeds, stops at the character screen.
  fail     a path containing "bad" fails the sources step; extract fails on its first attempt and works on retry.
  gamedir  existing install copied in (import-game), no installers.
"""
import argparse, json, os, re, threading, time
from http.server import ThreadingHTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse, parse_qs

WEB = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
TITLES = {"data-dir": "Choose data directory", "sources": "Locate your Diablo II files", "prefix": "Create base Wine prefix",
  "install-d2": "Install Diablo II", "install-lod": "Install Lord of Destruction", "import-game": "Import existing game folder",
  "patch": "Apply the 1.14d patch", "extract": "Extract data tables", "character": "Create the first throwaway test character"}
INTERACTIVE = {"install-d2", "install-lod", "patch"}
FIELDS = {"dataDir": str, "mode": str, "stubD2": str, "stubLoD": str, "installerD2Dir": str, "installerLoDDir": str,
  "gameDirSrc": str, "patchFile": str, "slots": int, "charName": str, "charClass": str}
LOGS = {
  "data-dir": ["using data directory {dataDir}", "free space 212 GB"],
  "sources": ["checking installer folders", "Diablo II: 1 file set found", "Lord of Destruction: 1 file set found", "piece hashes verified"],
  "prefix": ["creating Wine prefix", "wineboot --init", "installing fonts", "prefix ready"],
  "install-d2": ["starting the Diablo II installer in the virtual display", "installer window is open"],
  "install-lod": ["starting the Lord of Destruction installer", "installer window is open"],
  "import-game": ["copying game folder", "copied 1,204 files", "no symlinks found"],
  "patch": ["checking the patch file sha256", "patch accepted", "running the patcher", "patcher window is open"],
  "extract": ["reading Patch_D2.mpq", "extracting data tables", "tables written"],
  "character": ["starting the game", "creating {charName} ({charClass})", "saving and leaving the game"]}

class Mock:
    def __init__(self, scenario, speed):
        self.speed, self.lock = speed, threading.Lock()
        self.reset(scenario)
    def reset(self, scenario):
        with self.lock:
            self.scenario, self.gen = scenario, getattr(self, "gen", 0) + 1
            self.inputs = {"slots": 1}
            if scenario == "gamedir": self.inputs.update(mode="game-dir")
            self.status = {}; self.error = {}; self.note = {}; self.attempts = {}; self.started = {}; self.finished = {}
            self.log, self.running, self.cancel, self.fail_extract = [], False, False, scenario == "fail"
            self.checks_calls = 0
    def plan(self):
        gd = self.inputs.get("mode") == "game-dir"
        return ["data-dir", "sources", "prefix"] + ([] if gd else ["install-d2", "install-lod"]) + (["import-game"] if gd else []) + ["patch", "extract", "character"]
    def view(self):
        with self.lock:
            steps, cur = [], ""
            for sid in self.plan():
                st = self.status.get(sid, "pending")
                steps.append({"id": sid, "title": TITLES[sid], "interactive": sid in INTERACTIVE, "status": st, "error": self.error.get(sid, ""),
                  "note": self.note.get(sid, ""), "attempts": self.attempts.get(sid, 0), "started": self.started.get(sid, ""), "finished": self.finished.get(sid, "")})
                if not cur and st not in ("done", "skipped"): cur = sid
            return {"inputs": self.inputs, "steps": steps, "current": cur, "running": self.running, "done": cur == "",
              "display": ":99", "log": self.log[-200:], "updated": time.strftime("%Y-%m-%dT%H:%M:%SZ")}
    def say(self, line):
        with self.lock: self.log.append(time.strftime("%H:%M:%S ") + line)
    def sleep(self, s):
        end = time.time() + s / self.speed
        while time.time() < end:
            if self.cancel: return False
            time.sleep(0.05)
        return True
    def setstat(self, sid, st, **kw):
        with self.lock:
            self.status[sid] = st
            if st == "running": self.attempts[sid] = self.attempts.get(sid, 0) + 1; self.started[sid] = time.strftime("%Y-%m-%dT%H:%M:%SZ"); self.error[sid] = ""
            if st in ("done", "failed"): self.finished[sid] = time.strftime("%Y-%m-%dT%H:%M:%SZ")
            if "error" in kw: self.error[sid] = kw["error"]
            if "note" in kw: self.note[sid] = kw["note"]
    def start(self):
        with self.lock:
            if self.running: return False
            self.running, self.cancel, gen = True, False, self.gen
        threading.Thread(target=self.run, args=(gen,), daemon=True).start(); return True
    def run(self, gen):
        try:
            for sid in self.plan():
                if gen != self.gen or self.cancel: break
                if self.status.get(sid) in ("done", "skipped"): continue
                if sid == "character" and not self.inputs.get("charName"): break  # the page asks for a name first
                self.setstat(sid, "running", note="")
                for line in LOGS[sid]:
                    self.say(line.format(**{"dataDir": self.inputs.get("dataDir", ""), "charName": self.inputs.get("charName", ""), "charClass": self.inputs.get("charClass", "")}))
                    if not self.sleep(0.9): break
                    if sid in INTERACTIVE and line.endswith("is open"): break
                if self.cancel: self.setstat(sid, "pending", note="Cancelled."); break
                err = self.fails(sid)
                if err: self.setstat(sid, "failed", error=err); self.say("FAILED: " + err); break
                if sid in INTERACTIVE:
                    self.setstat(sid, "waiting-user", note={"install-d2": "Click through the Diablo II installer. Type your CD key in the installer's own key screen when it asks.",
                      "install-lod": "Click through the Lord of Destruction installer. Type your expansion CD key in the installer's own key screen.",
                      "patch": "Click Patch in the 1.14d patcher window and wait for it to say it is finished."}[sid])
                    if not self.sleep(5): self.setstat(sid, "pending"); break
                    self.say("installer finished, verifying")
                self.setstat(sid, "done", note="")
        finally:
            with self.lock: self.running = False
    def fails(self, sid):
        i = self.inputs
        if sid == "sources" and self.scenario != "gamedir":
            for k in ("installerD2Dir", "installerLoDDir", "stubD2", "stubLoD"):
                if "bad" in i.get(k, ""): return "folder not found or empty: " + i[k] + ". Check the path and that it holds the installer files."
        if sid == "sources" and "bad" in i.get("gameDirSrc", ""): return "no Game.exe in " + i["gameDirSrc"]
        if sid == "extract" and self.fail_extract:
            self.fail_extract = False; return "Patch_D2.mpq could not be read (file is too short). The patch step may not have finished."
        return ""

CHECKS = [
 ("host.container", "Running in a container", "pass", "Not in a container.", ""),
 ("host.runner", "Wine / Proton runner", "warn", "Plain Wine 9.0 found. GE-Proton11-11 is the tested runner.", "Install GE-Proton11-11 into ~/.local/share/d2lod/runners, or carry on with plain Wine at your own risk."),
 ("host.wine32", "32-bit Wine support", "fail", "wine32 libraries are missing.", "Install them with `sudo dpkg --add-architecture i386 && sudo apt update && sudo apt install wine32:i386`, then run the check again."),
 ("kernel.yama", "Memory-reading permission (Yama)", "fail", "ptrace_scope is 2. The bot cannot read the game.", "Run `sudo sysctl kernel.yama.ptrace_scope=0` for this boot. To keep it, put `kernel.yama.ptrace_scope = 0` in /etc/sysctl.d/10-ptrace.conf."),
 ("kernel.subreaper", "Process adoption probe", "pass", "Probe adopted an orphan and read its memory.", ""),
 ("display.xvfb", "Virtual display", "pass", "Xvfb found.", ""),
 ("resources.disk", "Free disk space", "pass", "212 GB free, 8.25 GB needed.", ""),
 ("resources.cpu", "CPU cores", "pass", "8 cores for 1 character.", ""),
 ("resources.ram", "Memory", "pass", "14.2 GB available.", ""),
 ("fs.prefix", "Disk type for the game files", "warn", "The data folder is on a network drive.", "Pick a folder on a local disk (ext4, xfs, btrfs or NTFS-free) if the game stutters."),
 ("net.ports", "Network ports", "pass", "Ports 8420, 47012 and 4000 are free.", ""),
 ("game.files", "Diablo II is installed", "skip", "Not installed yet. The wizard installs it next.", ""),
 ("game.version", "Game version 1.14d", "skip", "Skipped until the game is installed.", ""),
]
def checks_body(quick, n):
    res = []
    for id_, title, st, detail, fix in CHECKS:
        if n >= 1 and id_ in ("host.wine32", "kernel.yama"): st, detail, fix = "pass", "Fixed.", ""
        res.append({"id": id_, "title": title, "status": st, "detail": detail, "fix": fix, "durationMs": 12})
    if not quick: res.append({"id": "data.extract", "title": "Table extraction test", "status": "pass", "detail": "Extracted into a temp folder.", "slow": True, "durationMs": 2400})
    c = {k: sum(1 for r in res if r["status"] == k) for k in ("pass", "warn", "fail", "skip")}
    worst = "fail" if c["fail"] else "warn" if c["warn"] else "pass"
    return {"status": worst, "exit": {"pass": 0, "warn": 1, "fail": 2}[worst], "counts": c, "results": res}

M = None
class H(BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def send(self, code, body, ctype="application/json"):
        b = body if isinstance(body, bytes) else json.dumps(body).encode()
        self.send_response(code); self.send_header("Content-Type", ctype); self.send_header("Content-Length", str(len(b))); self.send_header("Cache-Control", "no-store"); self.end_headers(); self.wfile.write(b)
    def do_GET(self):
        u = urlparse(self.path); q = parse_qs(u.query)
        if u.path == "/api/setup/state": return self.send(200, M.view())
        if u.path == "/api/setup/checks":
            time.sleep(0.6 / M.speed); M.checks_calls += 1
            return self.send(200, checks_body(q.get("quick") == ["1"], M.checks_calls - 1 if M.scenario != "fresh" else 0))
        if u.path.startswith("/api/"): return self.send(404, {"error": "not found"})
        p = "index.html" if u.path == "/" else u.path.lstrip("/")
        f = os.path.normpath(os.path.join(WEB, p))
        if not f.startswith(WEB + os.sep) or not os.path.isfile(f): return self.send(404, b"not found", "text/plain")
        ct = {".html": "text/html; charset=utf-8", ".js": "text/javascript", ".css": "text/css"}.get(os.path.splitext(f)[1], "application/octet-stream")
        self.send(200, open(f, "rb").read(), ct)
    def do_PUT(self):
        if urlparse(self.path).path != "/api/setup/inputs": return self.send(404, {"error": "not found"})
        try:
            body = json.loads(self.rfile.read(int(self.headers.get("Content-Length", 0))) or b"{}")
            assert isinstance(body, dict) and all(k in FIELDS and isinstance(v, FIELDS[k]) for k, v in body.items())
        except Exception:
            return self.send(400, {"error": "bad inputs: unknown or malformed field (CD keys are never accepted here)"})
        if M.running: return self.send(409, {"error": "setup is running"})
        for k, v in body.items():
            if k.endswith(("Dir", "D2", "LoD", "Src", "File")) and v and not re.match(r"^(/|[A-Za-z]:[\\/]|\\\\)", v):
                return self.send(400, {"error": k + " must be an absolute path (starting with / or a drive letter)"})
        for k, v in body.items():
            if v not in ("", 0): M.inputs[k] = v
        self.send(200, M.view())
    def do_POST(self):
        u = urlparse(self.path); q = parse_qs(u.query)
        if u.path == "/mock/scenario": M.reset(q.get("name", ["happy"])[0]); return self.send(200, {"ok": True})
        if u.path == "/api/setup/start":
            if not M.start(): return self.send(409, {"error": "setup is already running"})
            return self.send(202, M.view())
        if u.path == "/api/setup/cancel": M.cancel = True; time.sleep(0.2); return self.send(202, M.view())
        if u.path == "/api/setup/reset":
            if M.running: return self.send(409, {"error": "setup is running"})
            s = q.get("step", [""])[0]
            for sid in ([s] if s else list(TITLES)):
                M.status.pop(sid, None); M.error.pop(sid, None); M.note.pop(sid, None)
            return self.send(200, M.view())
        self.send(404, {"error": "not found"})

if __name__ == "__main__":
    ap = argparse.ArgumentParser(); ap.add_argument("--port", type=int, default=8440); ap.add_argument("--speed", type=float, default=1.0); ap.add_argument("--scenario", default="happy")
    a = ap.parse_args(); M = Mock(a.scenario, a.speed)
    ThreadingHTTPServer(("127.0.0.1", a.port), H).serve_forever()
