#!/usr/bin/env python3
"""Flip a dark knob on a sandbox, watch 2 h (user decision 2026-11-27: no 24 h waits), revert on a hard invariant breach (docs/process/dark-launch.md, card autorevert1).

  flip.py <instance> <knob>=<value> [<knob>=<value> ...] --why TEXT [--window 24h] [--dry-run]   flip a bundle, open ONE window
  flip.py bisect <instance> --without KNOB                                  the last bundle of the instance minus one knob (a re-flip line)
  flip.py <instance> <knob>=<value> ... --reflip                            also accept an `enabled` row (sandbox on, code default still 0)
  flip.py watch [--interval 60] [--once] [--daemon]                         poll every open window; revert or close (--daemon: never exits, for the unit)
  flip.py watch --install-unit [--write]                                    print (or write) a systemd user unit for the watcher; never enables it
  flip.py status                                                            list open windows

Stdlib only. Never prints tokens. No host is hard-coded: the transport is a command prefix from the environment or
local.env (via tools/env.sh):
  D2_FLIP_RUN      shell prefix that runs a command on the sandbox host (default: labrun $D2_LABRUN_HOST --). Sandbox containers live
                   in the docker of the worker host, so $D2_LABRUN_HOST must name that host (a 20 s `docker inspect` probe checks it)
  D2_FLIP_TUNING   tuning file inside the container (default /data/instances/main/state/tuning.json; {instance} is replaced)
  D2_FLIP_RESTART  container (default: docker restart of the whole container, always works) or bot (TERM to the bot
                   launcher only, the shim starts it again; U: not drilled on a sandbox, try it with a dry knob first)
  D2_FLIP_VERIFY_S seconds to wait for the flipped value to show in `ops status --why` (default 240)
  D2_FLIP_BIN      the bot binary inside the container (default /data/didle/current/bin/d2bot)
  D2_FLIP_STATE    where open windows are kept (default ~/.cache/d2flip); the ledger's runtime state is
                   <D2_FLIP_STATE>/ledger/ledger-state.json
Ledger (card led1): the design rows are read from data/darklaunch/<knob>.json of this checkout and never written; trial,
enabled and revert state go to the state file only. After a close or revert the tool prints the one-line commit for the
status word in git when that word must change.
The container is named like the instance. Refuses while D2BOT_RUN is set (never in a measured run) and when the
instance already has an open window.
"""
import argparse
import base64
import fcntl
import hashlib
import json
import os
import shlex
import subprocess
import sys
import time
from datetime import datetime, timedelta, timezone

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
sys.path.insert(0, os.path.join(ROOT, "tools", "darklaunch"))
import ledger as L  # noqa: E402

MAX_WRITE_B64 = 100000  # the text rides in a command line
PROBE_S = 20  # `docker inspect` before a flip: fail fast on a wrong transport
HEARTBEAT_STALE_S = 300  # a watcher older than this counts as dead
HARD_MISS_LIMIT = 10  # consecutive unreadable polls before the window counts as a breach


class FlipError(Exception):
    pass


def now_utc():
    return datetime.now(timezone.utc)


def iso(t):
    return t.strftime("%Y-%m-%dT%H:%M:%SZ")


def parse_iso(s):
    return datetime.strptime(s, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc)


def parse_window(s):
    n, unit = s[:-1], s[-1:]
    if not n.isdigit() or unit not in "smhd":
        raise FlipError(f"bad --window {s!r} (use 90s, 30m, 24h, 1d)")
    return int(n) * {"s": 1, "m": 60, "h": 3600, "d": 86400}[unit]


def load_env(root=None):
    """Host values come from the environment, then local.env through tools/env.sh (the public convention). env.sh finds
    local.env through git or the cwd, which is wrong under systemd (cwd is home), so it is pointed at this checkout."""
    root = root or ROOT
    env = dict(os.environ)
    env.setdefault("D2_BOT_DIR", root)
    try:
        out = subprocess.run(["sh", "-c", '. "$1/tools/env.sh" && env', "sh", root], capture_output=True, text=True, timeout=20, env=env, cwd=root).stdout
        for line in out.splitlines():
            k, _, v = line.partition("=")
            if k.startswith("D2_"):
                env.setdefault(k, v)
    except (OSError, subprocess.SubprocessError):
        pass
    return env


class Sandbox:
    """The four things the tool needs from a sandbox. Tests replace this class with a fake."""

    def __init__(self, instance, env):
        self.instance = instance
        run = env.get("D2_FLIP_RUN")
        if run is None:
            host = env.get("D2_LABRUN_HOST")
            if not host:
                raise FlipError("set D2_FLIP_RUN or D2_LABRUN_HOST in local.env")
            run = f"labrun {host} --"
        self.run = shlex.split(run)
        self.path = env.get("D2_FLIP_TUNING", "/data/instances/main/state/tuning.json").format(instance=instance)
        self.bin = env.get("D2_FLIP_BIN", "/data/didle/current/bin/d2bot")
        self.mode = env.get("D2_FLIP_RESTART", "container")
        if self.mode not in ("container", "bot"):
            raise FlipError("D2_FLIP_RESTART must be container or bot")

    def _x(self, args, stdin=None, timeout=120):
        try:
            r = subprocess.run(self.run + args, input=stdin, capture_output=True, text=True, timeout=timeout)
        except subprocess.TimeoutExpired:
            raise FlipError(f"{' '.join(args[:3])} timed out after {timeout}s: check D2_FLIP_RUN") from None
        if r.returncode != 0:
            raise FlipError(f"{' '.join(args[:3])} failed: {r.stderr.strip()[:200]}")
        return r.stdout

    def probe(self):
        """The container must be visible through the transport; names D2_FLIP_RUN when it is not (wrong host, no docker)."""
        try:
            self._x(["docker", "inspect", "--format", "{{.State.Running}}", self.instance], timeout=PROBE_S)
        except FlipError as e:
            raise FlipError(f"container {self.instance} not reachable through `{' '.join(self.run)}` ({e}); "
                            "set D2_FLIP_RUN (or D2_LABRUN_HOST) to the host whose local docker runs the sandboxes") from None

    def read_tuning(self):
        try:
            return self._x(["docker", "exec", self.instance, "cat", self.path])
        except FlipError:
            return None  # no file yet: the profile applies

    def write_tuning(self, text):
        """Atomic and checked: the text travels as a base64 ARGUMENT (labrun does not forward stdin; an empty file once
        reached a bot that way). It lands in a temp file, whose size and sha256 must match before the rename; any
        mismatch removes the temp file and raises, so the real file is never touched by a bad write."""
        raw = text.encode()
        b64 = base64.b64encode(raw).decode()
        if len(b64) > MAX_WRITE_B64:
            raise FlipError(f"tuning text is {len(b64)} bytes of base64 (limit {MAX_WRITE_B64}): use an ssh form of D2_FLIP_RUN")
        p, sha = self.path, hashlib.sha256(raw).hexdigest()
        script = (f'printf %s {b64} | base64 -d > "{p}.new" && [ "$(wc -c < "{p}.new")" -eq {len(raw)} ] '
                  f'&& [ "$(sha256sum < "{p}.new" | cut -d" " -f1)" = {sha} ] && mv "{p}.new" "{p}" || {{ rm -f "{p}.new"; exit 1; }}')
        self._x(["docker", "exec", self.instance, "sh", "-c", script])

    def remove_tuning(self):
        self._x(["docker", "exec", self.instance, "rm", "-f", self.path])

    def restart(self):
        if self.mode == "container":
            self._x(["docker", "restart", self.instance], timeout=300)
            return
        # Only the bot: TERM to the launcher (cmdline starts with "d2bot --backend"); the shim restarts it.
        self._x(["docker", "exec", self.instance, "sh", "-c",
                 'n=0; for p in /proc/[0-9]*; do tr "\\0" " " < $p/cmdline 2>/dev/null | grep -q "^d2bot --backend" && kill -TERM ${p#/proc/} && n=$((n+1)); done; [ $n -gt 0 ]'])

    def status(self):
        """The decision-ring view (`ops status --why`); its `invariants` list has the counters."""
        out = self._x(["docker", "exec", self.instance, self.bin, "ops", "status", "--why"], timeout=30)
        # `ops status --why` prints the JSON view first and may append human lines after it ("waiting for Start: ...",
        # "auto-start held back: ..."): read the first JSON document only.
        start = out.find("{")
        if start < 0:
            raise FlipError("ops status --why printed no JSON view")
        view, _ = json.JSONDecoder().raw_decode(out[start:])
        return view


def hard_counts(view):
    """-> (since, {name: count}) of the hard counters; FlipError when the view has none (the bot predates the surface)."""
    inv = view.get("invariants")
    if inv is None:
        raise FlipError("status has no invariants list (bot too old)")
    hard = {c["name"]: c.get("count", 0) for c in inv if c.get("hard")}
    since = next((c.get("since", "") for c in inv), "")
    if not hard:
        raise FlipError("no hard counters yet (bot not ticking)")
    return since, hard


def set_overrides(text, knobs):
    """All knobs of a bundle in ONE edit: (new text, {knob: old value or None})."""
    olds = {}
    for k, v in knobs.items():
        text, olds[k] = set_override(text, k, v)
    return text, olds


def set_override(text, knob, value):
    """The tuning file is the store envelope {"version": N, "data": {"profile", "overrides"}} (internal/store/file.go,
    internal/bot/tuner.go): the knob goes into data.overrides; every other key and the version stay as they were.
    Returns (new text, old value or None). A top-level "overrides" is never written: the bot would ignore it."""
    if text is None or not text.strip():
        env = {"version": 1, "data": {}}
    else:
        env = json.loads(text)
        if not isinstance(env, dict) or "version" not in env or not isinstance(env.get("data"), dict):
            raise FlipError("tuning file is not a store envelope {version, data}")
    data = env["data"]
    ov = data.setdefault("overrides", {})
    old = ov.get(knob)
    ov[knob] = value
    return json.dumps(env, indent=2) + "\n", old


class Store:
    """One JSON file per open window."""

    def __init__(self, d):
        self.d = d
        os.makedirs(d, exist_ok=True)

    def path(self, instance):
        return os.path.join(self.d, instance + ".json")

    def get(self, instance):
        try:
            with open(self.path(instance)) as f:
                return json.load(f)
        except FileNotFoundError:
            return None

    def put(self, instance, w):
        tmp = self.path(instance) + ".new"
        with open(tmp, "w") as f:
            json.dump(w, f, indent=1)
        os.replace(tmp, self.path(instance))

    def last(self, instance, w):
        """Keep the finished window (backup text left out) so `bisect` can print its bundle after a revert."""
        rec = {k: v for k, v in w.items() if k != "backup"}
        with open(os.path.join(self.d, instance + ".last"), "w") as f:
            json.dump(rec, f, indent=1)

    def get_last(self, instance):
        try:
            with open(os.path.join(self.d, instance + ".last")) as f:
                return json.load(f)
        except FileNotFoundError:
            return None

    def beat(self, t):
        with open(os.path.join(self.d, "watch.heartbeat"), "w") as f:
            f.write(iso(t) + "\n")

    def drop(self, instance):
        try:
            os.remove(self.path(instance))
        except FileNotFoundError:
            pass

    def all(self):
        return sorted(n[:-5] for n in os.listdir(self.d) if n.endswith(".json"))


def heartbeat_alive(d, now):
    try:
        with open(os.path.join(d, "watch.heartbeat")) as f:
            return (now - parse_iso(f.read().strip())).total_seconds() <= HEARTBEAT_STALE_S
    except (OSError, ValueError):
        return False


UNIT = """[Unit]
Description=d2lod-bot flip watcher (reverts a flipped knob on a hard invariant breach)

[Service]
Type=simple
WorkingDirectory={root}
ExecStart={py} {script} watch --daemon --interval {interval}
Restart=always
RestartSec=10

[Install]
WantedBy=default.target
"""


def unit_text(interval=60):
    return UNIT.format(root=ROOT, py=sys.executable or "/usr/bin/python3", script=os.path.abspath(__file__), interval=interval)


def unit_path():
    return os.path.expanduser("~/.config/systemd/user/d2flip-watch.service")


class Ledger:
    """The merged ledger view for the flip tool. Design rows come from `path` (data/darklaunch/ of this checkout, or an old
    one-file ledger) and are never written. Runtime state (trial: window open, enabled: closed clean, dark: reverted)
    goes to the state file, the only thing this class writes."""

    def __init__(self, path=None, state=None, env=None):
        self.path = path or L.default_source(ROOT)
        # An explicit ledger without a state path (tests, a scratch copy) keeps its state next to it, never in the real file.
        self.state = state or (os.path.join(self.path.rstrip(os.sep) + ".state", "ledger-state.json") if path else L.default_state(env))

    def items(self):
        return L.rows(self.path, self.state)

    def _load(self):
        return {"items": self.items()}

    def row(self, name):
        return next((it for it in self.items() if it["name"] == name), None)

    def design_status(self, name):
        return next((it.get("status") for it in L.load_design(self.path) if it["name"] == name), None)

    def update(self, name, **kv):
        """Never reads the checkout: a close or revert must land even when the knob's file is gone or mid-merge (flip()
        checked the row before the window opened; state of a name without a design row is simply not shown)."""
        if "status" in kv:
            kv["state"] = kv.pop("status")
        os.makedirs(os.path.dirname(self.state), exist_ok=True)
        with open(self.state + ".lock", "w") as lock:  # the watcher and a flip may write at the same moment
            fcntl.flock(lock, fcntl.LOCK_EX)
            st = L.load_state(self.state)
            st["rows"].setdefault(name, {}).update(kv)
            st["log"].append(dict(kv, name=name, at=iso(now_utc())))
            L.save_state(self.state, st)


class Flipper:
    def __init__(self, make_sandbox, store, ledger, clock=now_utc, sleep=time.sleep, out=print, env=None, dry_run=False, reflip=False):
        self.make, self.store, self.ledger, self.clock, self.sleep, self.out = make_sandbox, store, ledger, clock, sleep, out
        self.env = os.environ if env is None else env
        self.dry = dry_run
        self.reflip = reflip
        self.verify_s = int(self.env.get("D2_FLIP_VERIFY_S", 240))

    def verify(self, sb, knobs):
        """Wait for `ops status --why` to show the flipped value among the live knobs (the new child read the file).
        Returns the hard counters (since, counts) of the new child; FlipError when the value never shows."""
        waited, last = 0, "no answer"
        while True:
            # A flip always leaves a non-empty file; zero bytes (or none) after the restart is a hard failure, not a
            # wait: the bot quarantines an empty file and runs on defaults, so the live check below could mislead.
            txt = sb.read_tuning()
            if txt is None or not txt.strip():
                raise FlipError("verify failed: tuning file is " + ("missing" if txt is None else "zero bytes") + " after the restart")
            try:
                view = sb.status()
                live = view.get("knobs") or {}
                bad = [f"{k} is {live.get(k)}, expected {v}" for k, v in knobs.items() if live.get(k) is None or abs(live[k] - v) > 1e-9]
                if bad:
                    last = "; ".join(bad)
                else:
                    return hard_counts(view)
            except Exception as e:
                last = str(e)
            if waited >= self.verify_s:
                raise FlipError(f"verify failed: {last}")
            self.sleep(5)
            waited += 5

    # --- flip ---------------------------------------------------------------------------------------------
    def flip(self, instance, knobs, why, window_s):
        """knobs: {name: value}. One backup, one atomic write of all of them, one restart, every knob verified."""
        if self.env.get("D2BOT_RUN"):
            raise FlipError("D2BOT_RUN is set: never flip in a measured run")
        if not why.strip():
            raise FlipError("--why is required")
        if not knobs:
            raise FlipError("no knob given")
        if self.store.get(instance):
            raise FlipError(f"{instance} already has an open flip window (one per sandbox)")
        for knob in knobs:
            row = self.ledger.row(knob)
            ok = ("dark", "trial") + (("enabled",) if self.reflip else ())
            if row is None or row["status"] not in ok:
                hint = " (status enabled: pass --reflip to flip it again)" if row and row["status"] == "enabled" else ""
                raise FlipError(f"{knob}: needs a dark row in data/darklaunch/{hint}")
        sb = self.make(instance)
        if hasattr(sb, "probe"):
            sb.probe()
        if not self.dry and not heartbeat_alive(self.store.d, self.clock()):
            self.out("WARNING: no watcher is alive (no fresh heartbeat): this window will not be watched or reverted. "
                     "Start `flip.py watch --daemon` (see `flip.py watch --install-unit`)")
        before = sb.read_tuning()
        if before is not None and not before.strip():
            raise FlipError("tuning file is zero bytes before the flip: restore it from tuning.json.bak first (the bot would run on defaults)")
        new_text, olds = set_overrides(before, knobs)
        # Baseline: the counters of the running child before the flip. A flip during a breach is refused.
        since0, base0 = hard_counts(sb.status())
        ts = self.clock()
        tag = "bak-flip-" + ts.strftime("%Y%m%dT%H%M%SZ")
        names = ",".join(knobs)
        w = {"instance": instance, "knobs": {k: {"old": olds[k], "new": v} for k, v in knobs.items()}, "why": why,
             "start": iso(ts), "window_s": window_s, "end": iso(ts + timedelta(seconds=window_s)), "tag": tag, "backup": before,
             "pending": "", "misses": 0, "baseline": base0, "since": since0}
        if self.dry:
            self.out(f"DRY-RUN {instance}: baseline {since0 or '-'} {sorted(base0.items())}")
            self.out(f"DRY-RUN save backup {tag} ({'no file' if before is None else str(len(before)) + ' bytes'}), ONE write of data.overrides {knobs} (were {olds}), restart ({getattr(sb, 'mode', '?')}), verify every live knob, ledger {names} -> trial, window {window_s}s")
            self.out("DRY-RUN nothing written")
            return w
        self.store.put(instance, w)  # the backup is on disk before any knob moves
        try:
            sb.write_tuning(new_text)
            if sb.read_tuning() != new_text:  # the file must be exactly what was meant BEFORE anything restarts
                raise FlipError("written tuning differs from the intended text")
        except Exception as e:
            self.store.drop(instance)
            try:
                if sb.read_tuning() != before:
                    sb.write_tuning(before) if before is not None else sb.remove_tuning()
            except Exception:
                pass
            raise FlipError(f"write check failed, no restart, container untouched: {e}") from e
        try:
            sb.restart()
            w["since"], w["baseline"] = self.verify(sb, knobs)  # the restart reset the counters: new baseline
        except Exception as e:
            self.out(f"FLIP FAILED {instance} {names}: {e}; restoring the backup")
            self._restore(sb, w)  # best effort; on failure the window stays 'pending' and watch retries
            if w["pending"]:
                self.store.put(instance, w)
                raise FlipError(f"flip failed and the restore is pending ({w['pending']}): run flip.py watch --once") from e
            self.store.drop(instance)
            raise FlipError(f"flip failed, tuning restored: {e}") from e
        self.store.put(instance, w)
        for k, v in knobs.items():
            self.ledger.update(k, status="trial", trial_started=iso(ts), trial_instance=instance, trial_value=v,
                               trial_window=f"{window_s}s", trial_why=why, trial_bundle=names)
        self.out(f"FLIP {instance} {names} = {list(knobs.values())} (were {list(olds.values())}) window {window_s}s until {w['end']} backup {tag}")
        return w

    def _design_hint(self, word, knobs):
        """Runtime state is already in the state file. Print the one-line commit for the knobs whose status word in git
        differs from `word` (enabled after a clean close, dark after the revert of a re-flipped enabled row)."""
        try:
            lag = [k for k in knobs if self.ledger.design_status(k) != word]
        except Exception as e:  # never fails the watcher
            self.out(f"LEDGER: design rows unreadable ({e}); state file is written")
            return
        if lag:
            self.out(f"LEDGER git: {len(lag)} row(s) to {word}, commit on a branch: {L.status_line(word, lag)}")
        else:
            self.out(f"LEDGER git: nothing to commit (rows already {word})")

    # --- restore / revert ---------------------------------------------------------------------------------
    def _restore(self, sb, w):
        """Put the backup back and restart. Sets w['pending'] to the step that failed ('' when done)."""
        w["pending"] = ""
        if w["pending"] == "" and not w.get("restored"):
            try:
                if w["backup"] is None:
                    sb.remove_tuning()
                else:
                    sb.write_tuning(w["backup"])
                w["restored"] = True
            except Exception as e:
                w["pending"] = f"write: {e}"
                return
        try:
            sb.restart()
        except Exception as e:
            w["pending"] = f"restart: {e}"

    def revert(self, w, reason):
        sb = self.make(w["instance"])
        w["revert_reason"] = w.get("revert_reason") or reason
        self._restore(sb, w)
        if w["pending"]:
            self.store.put(w["instance"], w)
            self.out(f"REVERT FAILED {w['instance']} [{','.join(w['knobs'])}]: {w['pending']} (reason: {w['revert_reason']}); will retry")
            return False
        names = ",".join(w["knobs"])
        for k in w["knobs"]:
            self.ledger.update(k, status="dark", revert_reason=w["revert_reason"], reverted_at=iso(self.clock()),
                               revert_bundle=names, revert_breach=w.get("breach", ""))
        self.store.last(w["instance"], w)
        self.store.drop(w["instance"])
        self.out(f"REVERT {w['instance']} [{names}]: {w['revert_reason']}; tuning restored from {w['tag']}, child restarted, {len(w['knobs'])} row(s) dark")
        self._design_hint("dark", w["knobs"])
        return True

    def close(self, w):
        for k in w["knobs"]:
            self.ledger.update(k, status="enabled", enabled_at=iso(self.clock()))
        self.store.last(w["instance"], w)
        self.store.drop(w["instance"])  # the backup tag goes with the state file
        self.out(f"CLOSE {w['instance']} [{','.join(w['knobs'])}]: no hard breach for {w['window_s']}s; rows enabled, backup {w['tag']} removed")
        self._design_hint("enabled", w["knobs"])

    # --- watch --------------------------------------------------------------------------------------------
    def poll(self, instance):
        """One poll of one window. Returns 'open', 'closed' or 'reverted'."""
        w = self.store.get(instance)
        if w is None:
            return "closed"
        if w.get("revert_reason") and not w.get("done"):  # a revert that failed earlier
            return "reverted" if self.revert(w, w["revert_reason"]) else "open"
        sb = self.make(instance)
        breach = ""
        try:
            since, counts = hard_counts(sb.status())
            w["misses"] = 0
            if since != w["since"]:  # the child restarted: its counters begin at zero again
                w["since"], w["baseline"] = since, {}
            over = {n: c - w["baseline"].get(n, 0) for n, c in counts.items() if c > w["baseline"].get(n, 0)}
            if over:
                breach = "hard invariant breach: " + ",".join(f"{n}+{d}" for n, d in sorted(over.items()))
                w["breach"] = ",".join(sorted(over))
        except Exception as e:
            w["misses"] = w.get("misses", 0) + 1
            if w["misses"] >= HARD_MISS_LIMIT:
                breach = f"status unreadable for {w['misses']} polls: {e}"
        if breach:
            return "reverted" if self.revert(w, breach) else "open"
        if self.clock() >= parse_iso(w["end"]):
            self.close(w)
            return "closed"
        self.store.put(instance, w)
        return "open"

    def watch(self, interval=60, once=False, daemon=False):
        while True:
            self.store.beat(self.clock())
            names = self.store.all()
            states = [self.poll(n) for n in names]
            if once or (not daemon and "open" not in states):
                return states
            self.sleep(interval)


def bisect_line(w, without, window):
    """The re-flip command for the bundle of window w minus one knob (after a revert: which knob was it?)."""
    if without not in w["knobs"]:
        raise FlipError(f"{without} is not in the bundle {','.join(w['knobs'])}")
    rest = " ".join(f"{k}={v['new']:g}" for k, v in w["knobs"].items() if k != without)
    if not rest:
        raise FlipError("the bundle has only that knob")
    return f"python3 tools/flip/flip.py {w['instance']} {rest} --why {shlex.quote('bisect without ' + without + ': ' + w['why'])} --window {window}"


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("args", nargs="+", help="<instance> <knob>=<value> | watch | status")
    ap.add_argument("--why", default="")
    ap.add_argument("--window", default="2h", help="2h (default), 45m, 90s")
    ap.add_argument("--interval", type=int, default=60)
    ap.add_argument("--once", action="store_true")
    ap.add_argument("--without", default="", help="bisect: the knob to leave out")
    ap.add_argument("--daemon", action="store_true", help="watch: keep running with no open window (for the systemd unit)")
    ap.add_argument("--install-unit", action="store_true", help="watch: print the systemd user unit")
    ap.add_argument("--write", action="store_true", help="with --install-unit: write the unit file (never enables it)")
    ap.add_argument("--reflip", action="store_true", help="also accept ledger rows with status enabled")
    ap.add_argument("--commit-ledger", action="store_true", help=argparse.SUPPRESS)  # retired (led1): old command lines still parse
    ap.add_argument("--dry-run", action="store_true", help="print every step, write and restart nothing")
    a = ap.parse_args(argv)
    env = load_env()
    store = Store(env.get("D2_FLIP_STATE") or os.path.expanduser("~/.cache/d2flip"))
    ledger = Ledger(env=env)
    f = Flipper(lambda inst: Sandbox(inst, env), store, ledger, env=env, dry_run=a.dry_run, reflip=a.reflip)
    try:
        if a.args[0] == "watch":
            if a.install_unit:
                text = unit_text(a.interval)
                if a.write:
                    p = unit_path()
                    os.makedirs(os.path.dirname(p), exist_ok=True)
                    with open(p, "w") as fh:
                        fh.write(text)
                    print(f"wrote {p}; enable it yourself: systemctl --user daemon-reload && systemctl --user enable --now d2flip-watch")
                else:
                    print(text, end="")
            else:
                f.watch(a.interval, a.once, a.daemon)
        elif a.args[0] == "status":
            for n in store.all():
                w = store.get(n)
                print(f"{n} {' '.join(k + '=' + str(v['new']) for k, v in w['knobs'].items())} until {w['end']} pending={w['pending'] or '-'}")
        elif a.args[0] == "bisect":
            if len(a.args) != 2 or not a.without:
                raise FlipError("usage: flip.py bisect <instance> --without KNOB")
            w = store.get(a.args[1]) or store.get_last(a.args[1])
            if w is None:
                raise FlipError(f"{a.args[1]}: no open or finished window")
            print(bisect_line(w, a.without, a.window))
        else:
            if len(a.args) < 2 or any("=" not in x for x in a.args[1:]):
                raise FlipError("usage: flip.py <instance> <knob>=<value> [<knob>=<value> ...] --why TEXT [--window 24h]")
            knobs = {}
            for kv in a.args[1:]:
                knob, _, v = kv.partition("=")
                try:
                    knobs[knob] = float(v)
                except ValueError:
                    raise FlipError(f"value {v!r} of {knob} is not a number (tuning overrides are numbers)")
            f.flip(a.args[0], knobs, a.why, parse_window(a.window))
            if not a.dry_run:
                print(f"ledger: trial state written to {ledger.state}; nothing to commit")
    except FlipError as e:
        print(f"flip: {e}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
