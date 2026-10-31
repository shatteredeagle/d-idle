#!/usr/bin/env python3
"""Offline mock of the instance-scoped server routes, for developing and screenshotting web/ without Go.

  go run ./cmd/d2dash -addr 127.0.0.1:8420 &      # simulated runner: serves the flat per-instance routes
  python3 web/mock/mock.py [--port 8430] [--backend http://127.0.0.1:8420] [--static]

Proxies /api/v1/instances/{id}/<flat route> to the backend (envelope-converting the log stream) and fakes instances, fleet,
health, deaths, knobs, sentinel and a viewport frame. With --static no backend is needed: canned JSON (used for screenshots
on a machine that only has python3 and the web/ directory).
"""
import argparse, json, os, re, struct, sys, threading, time, urllib.request, zlib
from http.server import ThreadingHTTPServer, BaseHTTPRequestHandler

HERE = os.path.dirname(os.path.abspath(__file__))
WEB = os.path.dirname(HERE)
NOW = lambda: time.strftime('%Y-%m-%dT%H:%M:%S%z') or time.strftime('%Y-%m-%dT%H:%M:%SZ')
IDS = [('i1', 'Navi', 'Sorceress', 24), ('i2', 'Combi', 'Paladin', 31), ('i3', 'Towni', 'Necromancer', 9)]
KNOBS = [
  {"key": "combat.chickenLifePct", "label": "Chicken below life", "unit": "%", "desc": "Leave the game under this life percentage.", "default": 30, "min": 5, "max": 60, "higherIsBetter": True},
  {"key": "combat.potionGapMs", "label": "Potion gap", "unit": "ms", "desc": "Minimum time between potions.", "default": 900, "min": 400, "max": 3000, "higherIsBetter": False},
  {"key": "combat.reactionMs", "label": "Reaction time", "unit": "ms", "desc": "Delay before reacting to danger.", "default": 120, "min": 40, "max": 700, "higherIsBetter": False},
  {"key": "nav.pathQuality", "label": "Path quality", "unit": "", "desc": "How tight routes are.", "default": 0.9, "min": 0.3, "max": 1, "higherIsBetter": True},
  {"key": "town.beltColumns", "label": "Belt columns", "unit": "count", "desc": "Potion columns to keep stocked.", "default": 4, "min": 1, "max": 4, "higherIsBetter": True},
  {"key": "loot.minValuePerCell", "label": "Minimum loot value per cell", "unit": "gold", "desc": "Skip items worth less.", "default": 500, "min": 0, "max": 20000, "higherIsBetter": True},
]
STATE = {"values": {k["key"]: k["default"] for k in KNOBS}, "profile": "optimal"}
UPGRADES = [
  {"id": "belt2", "track": "Survival", "tier": 1, "label": "Wider belt", "desc": "Carry more potions.", "cost": 5000, "requires": [], "effects": [{"knob": "town.beltColumns", "value": 4}], "state": "unlocked"},
  {"id": "react1", "track": "Survival", "tier": 2, "label": "Quick reflexes", "cost": 12000, "requires": ["belt2"], "effects": [{"knob": "combat.reactionMs", "value": 80}], "state": "available"},
  {"id": "react2", "track": "Survival", "tier": 3, "label": "Sharp reflexes", "cost": 40000, "requires": ["react1"], "effects": [{"knob": "combat.reactionMs", "value": 50}], "state": "locked"},
  {"id": "path1", "track": "Travel", "tier": 1, "label": "Better routes", "cost": {"gold": 8000}, "requires": [], "effects": [{"knob": "nav.pathQuality", "value": 1}], "state": "available"},
]
def knobs_body():
    return {"knobs": KNOBS, "values": STATE["values"], "profile": STATE["profile"],
            "profiles": [{"name": "optimal", "label": "Optimal", "desc": "Every knob at its best."}, {"name": "relaxed", "label": "Relaxed", "desc": "Slightly slower, safer for weak machines."}, {"name": "rookie", "label": "Rookie", "desc": "Plays like a beginner."}],
            "upgrades": UPGRADES, "telemetry": {"secPerRun": {"label": "Seconds per run", "value": 142.5, "unit": "s"}, "deathsPerHour": 0.8}}
DEATHS = {"total": 7, "lastHour": 2, "totalGold": 9400, "totalItems": 3, "pausedForDeaths": False,
  "byArea": [{"area": 3, "name": "Cold Plains", "deaths": 4}, {"area": 17, "name": "Burial Grounds", "deaths": 3}],
  "heat": [{"area": 3, "x": 100 + 8 * i, "y": 220 + 8 * (i % 3), "count": 1 + i % 4} for i in range(9)] + [{"area": 17, "x": 60, "y": 90, "count": 3}],
  "records": [{"time": "2026-10-28T10:%02d:00Z" % (10 + 5 * i), "area": 3, "areaName": "Cold Plains", "level": 20 + i, "difficulty": "normal", "killer": "Blood Raven" if i % 2 else "Fallen Shaman", "killerBoss": bool(i % 2),
     "corpseValue": 3200, "retrieved": i % 3 == 0, "abandoned": i % 3 == 1, "costGold": 800, "costItems": i % 2, "costSeconds": 95, "why": "Value above threshold and area survivable; retrying once.",
     "lifeCurve": [{"msBefore": 4000 - 400 * j, "pct": max(0, 100 - 10 * j * j // 3)} for j in range(10)]} for i in range(5)]}
SENTINEL = {"time": "2026-10-28T10:00:00Z", "mode": "act:all", "summary": "i1 OK, i2 ATTENTION, i3 OK", "recent": [
  {"time": "2026-10-28T09:58:00Z", "kind": "acted", "instance": "i2", "rule": "frozen-frame", "action": "dismiss-dialog", "evidence": "frame hash unchanged 62 s"},
  {"time": "2026-10-28T09:59:00Z", "kind": "would", "instance": "i2", "rule": "bot-stuck", "action": "restart-bot", "evidence": "no progress 5 min"}],
  "instances": [
   {"id": "i1", "character": "Navi", "agent": "NAV", "health": "ok", "line": "in game, life 100%", "obs": {"procAlive": True, "cpuPct": 82, "state": "ingame", "life": 310, "maxLife": 310}},
   {"id": "i2", "character": "Combi", "agent": "COMBAT", "health": "attention", "line": "frame frozen 62 s", "acting": "dismiss-dialog", "obs": {"procAlive": True, "cpuPct": 3, "state": "ingame", "hung": True, "dialogs": ["Diablo II Error"]}, "rules": [{"rule": "frozen-frame", "active": True, "forSec": 62, "attempts": 1, "step": 1}], "lastAction": {"time": "2026-10-28T09:58:00Z", "rule": "frozen-frame", "action": "dismiss-dialog"}},
   {"id": "i3", "character": "Towni", "agent": "TOWN", "health": "ok", "line": "in game, in town", "obs": {"procAlive": True, "cpuPct": 40, "state": "ingame", "life": 120, "maxLife": 120}}]}

def png(w=640, h=400):
    rows = b''.join(b'\x00' + b''.join(bytes((30 + x * 90 // w, 50 + y * 120 // h, 80, 255)) for x in range(w)) for y in range(h))
    def chunk(t, d): c = struct.pack('>I', len(d)) + t + d; return c + struct.pack('>I', zlib.crc32(t + d) & 0xffffffff)
    return b'\x89PNG\r\n\x1a\n' + chunk(b'IHDR', struct.pack('>IIBBBBB', w, h, 8, 6, 0, 0, 0)) + chunk(b'IDAT', zlib.compress(rows)) + chunk(b'IEND', b'')
FRAME = png()

# canned flat bodies for --static
def canned(path, iid):
    n = {'i1': 0, 'i2': 1, 'i3': 2}.get(iid, 0)
    goals = [
      {"id": "g1", "type": "farm_area", "title": "Farm Cold Plains", "params": {"area": "Cold Plains", "untilKills": 150}, "priority": 0, "state": "active", "endless": False,
       "progress": {"current": 62, "target": 150, "percent": 41.3, "kills": 62, "bossKills": 0, "drops": 3, "deaths": 1, "activeSeconds": 1260, "etaSeconds": 1800, "phase": "fight", "source": "events", "verified": False,
         "steps": [{"id": "a", "title": "Travel to Cold Plains", "state": "done"}, {"id": "b", "title": "Clear the area", "state": "active", "current": 62, "target": 150}, {"id": "c", "title": "Return to town", "state": "queued"}]}},
      {"id": "g2", "type": "kill_boss", "title": "Kill Mephisto x2", "params": {"boss": "Mephisto", "count": 2}, "priority": 1, "state": "queued", "endless": False, "progress": {"current": 0, "target": 2, "percent": 0, "kills": 0, "bossKills": 0, "drops": 0, "deaths": 0, "activeSeconds": 0},
       "schedule": [{"days": [], "start": "22:00", "end": "06:00"}]},
      {"id": "g3", "type": "reach_level", "title": "Reach level 30", "params": {"level": 30}, "priority": 0, "state": "paused", "endless": False, "note": "Held by you", "progress": {"current": 24 + n, "target": 30, "percent": 70, "kills": 5, "bossKills": 0, "drops": 0, "deaths": 0, "activeSeconds": 60}}]
    status = {"mode": "running" if n != 2 else "stopped", "activeGoalId": "g1", "activeGoal": goals[0], "xpPerHour": 52000 - 9000 * n,
      "runner": {"state": "running", "activity": "Fighting in Cold Plains", "character": {"name": IDS[n][1], "class": IDS[n][2], "level": IDS[n][3], "difficulty": "normal", "life": 230, "maxLife": 310, "mana": 90, "maxMana": 140, "xp": 4200, "xpNext": 9000, "area": "Cold Plains", "position": {"x": 412, "y": 221}}},
      "totals": {"kills": 812, "drops": 40, "deaths": 7, "xp": 120000}, "lastSeq": 5, "now": NOW(), "version": "mock"}
    pts = [{"t": int(time.time() - 3600 * (24 - i)), "xp": 1000 * i, "xpPerHour": 30000 + 9000 * ((i * 7) % 5), "kills": 10 * i, "drops": i, "deaths": 0, "potions": 3} for i in range(25)]
    return {'/status': status, '/goals': goals, '/events': [], '/settings': {"autoResume": False, "maxDeaths": 5, "chickenLifePct": 30, "potionLifePct": 50, "difficulty": "auto"},
      '/stats': {"totals": status["totals"], "xpPerHour": status["xpPerHour"], "qualities": {"unique": 1, "set": 2, "rare": 6, "magic": 20, "normal": 30}, "drops": [{"time": NOW(), "item": {"name": "Ist Rune", "type": "rune", "quality": "normal"}}], "areas": [{"area": "Cold Plains", "seconds": 1200}, {"area": "Rogue Encampment", "seconds": 300}]},
      '/stats/series': {"window": "24h", "step": 3600, "points": pts},
      '/catalog': {"runs": [{"key": "run:mephisto", "label": "Mephisto run"}], "areas": [{"name": "Cold Plains"}], "bosses": [{"name": "Mephisto"}], "itemTypes": ["rune"], "qualities": ["unique"], "difficulties": ["normal"]}}.get(path)

class H(BaseHTTPRequestHandler):
    backend = None; static = False
    def log_message(self, *a): pass
    def send(self, code, body, ctype='application/json'):
        if not isinstance(body, (bytes, bytearray)): body = json.dumps(body).encode()
        self.send_response(code); self.send_header('Content-Type', ctype); self.send_header('Content-Length', str(len(body))); self.send_header('Cache-Control', 'no-store'); self.end_headers(); self.wfile.write(body)
    def summaries(self):
        out = []
        for n, (i, t, c, lvl) in enumerate(IDS):
            c_ = canned('/status', i)
            out.append({"id": i, "title": t, "mode": c_["mode"], "runner": c_["runner"], "activeGoalId": "g1", "xpPerHour": c_["xpPerHour"], "totals": c_["totals"], "ok": i != 'i2', "ready": True, "lastSeq": 5, "live": True, "mounts": ["viewport"] if i == 'i1' else [], "now": NOW()})
        return out
    def do_GET(self):
        p = self.path.split('?')[0]
        if p == '/api/v1/instances': return self.send(200, self.summaries())
        if p == '/api/v1/fleet': return self.send(200, {"now": NOW(), "instances": self.summaries(), "sentinel": SENTINEL, "failsafe": True})
        if p == '/api/v1/fleet/events/stream': return self.sse(lambda: [('fleet', {"now": NOW(), "instances": self.summaries(), "sentinel": SENTINEL, "failsafe": True})], 2)
        if p == '/api/v1/sentinel': return self.send(200, SENTINEL)
        m = re.match(r'^/api/v1/instances/([a-z0-9-]+)/(.*)$', p)
        if m:
            iid, rest = m.groups()
            if rest == 'health': return self.send(200 if iid != 'i2' else 503, {"state": "Fighting", "status": "running", "activity": "Fighting", "sinceSeconds": 41, "lastProgressSeconds": 4 if iid != 'i2' else 330, "stuckReason": "" if iid != 'i2' else "no xp, position or kills for 5 minutes", "lastError": "", "ladderStep": 0 if iid != 'i2' else 2, "paused": False, "ok": iid != 'i2'})
            if rest == 'deaths': return self.send(200, DEATHS)
            if rest == 'knobs': return self.send(200, knobs_body())
            if rest == 'viewport/frame.jpg':
                return self.send(200, FRAME, 'image/png') if iid == 'i1' else self.send(404, {"error": "no viewport"})
            if rest == 'viewport/overlay': return self.send(200, {"width": 640, "height": 400, "shapes": [{"type": "rect", "x": 300, "y": 150, "w": 60, "h": 80, "text": "Blood Raven", "color": "#e8a33a"}, {"type": "circle", "x": 330, "y": 300, "r": 8}]})
            if rest == 'events/stream': return self.flat_stream(iid)
            if self.static:
                v = canned('/' + rest, iid)
                return self.send(200, v) if v is not None else self.send(404, {"error": "no route " + rest, "code": "not_found"})
            return self.proxy('/api/v1/' + rest)
        if p.startswith('/api/v1/'):
            if p in ('/api/v1/bot/health',): return self.send(404, {"error": "none"})
            if self.static:
                v = canned(p[7:], 'i1')
                return self.send(200, v) if v is not None else self.send(404, {"error": "no route", "code": "not_found"})
            return self.proxy(p)
        f = os.path.join(WEB, p.lstrip('/'))
        if p.endswith('/'): f += 'index.html'
        if os.path.isfile(f) and os.path.commonpath([f, WEB]) == WEB:
            ct = {'.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css'}.get(os.path.splitext(f)[1], 'application/octet-stream')
            return self.send(200, open(f, 'rb').read(), ct + '; charset=utf-8')
        self.send(404, {"error": "not found"})
    def do_PUT(self):
        n = int(self.headers.get('Content-Length') or 0); b = json.loads(self.rfile.read(n) or b'{}')
        if self.path.endswith('/knobs'):
            for k, v in (b.get('overrides') or {}).items(): STATE['values'][k] = v
            if b.get('profile'): STATE['profile'] = b['profile']
            return self.send(200, knobs_body())
        self.send(404, {"error": "no"})
    def do_POST(self):
        n = int(self.headers.get('Content-Length') or 0); self.rfile.read(n)
        m = re.search(r'/upgrades/([^/]+)/unlock$', self.path)
        if m:
            for u in UPGRADES:
                if u['id'] == m.group(1): u['state'] = 'unlocked'
            return self.send(200, knobs_body())
        self.send(200, {})
    def sse(self, fn, every):
        self.send_response(200); self.send_header('Content-Type', 'text/event-stream'); self.send_header('Cache-Control', 'no-store'); self.end_headers()
        try:
            while True:
                for name, data in fn(): self.wfile.write(('event: %s\ndata: %s\n\n' % (name, json.dumps(data))).encode())
                self.wfile.flush(); time.sleep(every)
        except Exception: pass
    def flat_stream(self, iid):
        if self.static:
            ev = [{"v": 1, "seq": 1, "time": NOW(), "instance": iid, "kind": "kill", "data": {"monster": "Fallen", "boss": False}}, {"v": 1, "seq": 2, "time": NOW(), "instance": iid, "kind": "death", "data": {"areaName": "Cold Plains", "killer": "Blood Raven"}}, {"v": 1, "seq": 3, "time": NOW(), "instance": iid, "kind": "drop", "data": {"item": {"name": "Ist Rune"}}}]
            sent = []
            def fn():
                out = [] if sent else [('log', e) for e in ev]
                sent.append(1)
                c = canned('/status', iid); return out + [('status', c), ('goals', canned('/goals', iid))]
            return self.sse(fn, 1)
        try:
            r = urllib.request.urlopen(self.backend + '/api/v1/events/stream', timeout=3600)
        except Exception as e:
            return self.send(502, {"error": str(e)})
        self.send_response(200); self.send_header('Content-Type', 'text/event-stream'); self.end_headers()
        name = None
        try:
            for raw in r:
                line = raw.decode().rstrip('\n')
                if line.startswith('event:'): name = line[6:].strip(); out = raw; continue
                if line.startswith('data:') and name == 'log':
                    e = json.loads(line[5:]); env = {"v": 1, "seq": e.get('seq'), "time": e.get('time'), "instance": iid, "kind": e['type'].replace('goal_', 'goal.'), "data": {k: v for k, v in e.items() if k not in ('seq', 'time', 'type', 'goalId')}}
                    self.wfile.write(('data: %s\n' % json.dumps(env)).encode()); continue
                if line.startswith('id:') or line.startswith('event:'): pass
                self.wfile.write(raw); self.wfile.flush()
        except Exception: pass
    def proxy(self, path):
        q = self.path.split('?', 1)[1] if '?' in self.path else ''
        try:
            r = urllib.request.urlopen(self.backend + path + ('?' + q if q else ''), timeout=10); self.send(r.status, r.read())
        except urllib.error.HTTPError as e: self.send(e.code, e.read())
        except Exception as e: self.send(502, {"error": str(e)})

ap = argparse.ArgumentParser(); ap.add_argument('--port', type=int, default=8430); ap.add_argument('--backend', default='http://127.0.0.1:8420'); ap.add_argument('--static', action='store_true')
a = ap.parse_args(); H.backend = a.backend; H.static = a.static
ThreadingHTTPServer.daemon_threads = True
print('mock on http://127.0.0.1:%d/' % a.port, flush=True)
ThreadingHTTPServer(('127.0.0.1', a.port), H).serve_forever()
