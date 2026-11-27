#!/usr/bin/env python3
"""Flip a dark knob on a sandbox, watch 2 h (user decision 2026-11-27: no 24 h waits), revert on a hard invariant breach (docs/process/dark-launch.md, card autorevert1).

  flip.py <instance> <knob>=<value> [<knob>=<value> ...] --why TEXT [--window 24h] [--dry-run]   flip a bundle, open ONE window
  flip.py bisect <instance> --without KNOB                                  the last bundle of the instance minus one knob (a re-flip line)
  flip.py watch [--interval 60] [--once]                                    poll every open window; revert or close
  flip.py status                                                            list open windows

Stdlib only. Never prints tokens. No host is hard-coded: the transport is a command prefix from the environment or
local.env (via tools/env.sh):
  D2_FLIP_RUN      shell prefix that runs a command on the sandbox host (default: labrun $D2_LABRUN_HOST --)
  D2_FLIP_TUNING   tuning file inside the container (default /data/instances/main/state/tuning.json; {instance} is replaced)
  D2_FLIP_RESTART  container (default: docker restart of the whole container, always works) or bot (TERM to the bot
                   launcher only, the shim starts it again; U: not drilled on a sandbox, try it with a dry knob first)
  D2_FLIP_VERIFY_S seconds to wait for the flipped value to show in `ops status --why` (default 240)
  D2_FLIP_BIN      the bot binary inside the container (default /data/didle/current/bin/d2bot)
  D2_FLIP_STATE    where open windows are kept (default ~/.cache/d2flip)
The container is named like the instance. Refuses while D2BOT_RUN is set (never in a measured run) and when the
instance already has an open window.
"""
import argparse
import base64
import hashlib
import json
import os
import shlex
import subprocess
import sys
import time
from datetime import datetime, timedelta, timezone

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
LEDGER = os.path.join(ROOT, "data", "darklaunch.json")
MAX_WRITE_B64 = 100000  # the text rides in a command line
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


def load_env():
    """Host values come from the environment, then local.env through tools/env.sh (the public convention)."""
    env = dict(os.environ)
    try:
        out = subprocess.run(["sh", "-c", '. "$1/tools/env.sh" && env', "sh", ROOT], capture_output=True, text=True, timeout=20, env=env).stdout
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
        r = subprocess.run(self.run + args, input=stdin, capture_output=True, text=True, timeout=timeout)
        if r.returncode != 0:
            raise FlipError(f"{' '.join(args[:3])} failed: {r.stderr.strip()[:200]}")
        return r.stdout

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
        return json.loads(self._x(["docker", "exec", self.instance, self.bin, "ops", "status", "--why"], timeout=30))


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

    def drop(self, instance):
        try:
            os.remove(self.path(instance))
        except FileNotFoundError:
            pass

    def all(self):
        return sorted(n[:-5] for n in os.listdir(self.d) if n.endswith(".json"))


class Ledger:
    """data/darklaunch.json rows. Statuses used here: trial (window open), dark (reverted), enabled (window closed clean)."""

    def __init__(self, path=LEDGER):
        self.path = path

    def _load(self):
        with open(self.path) as f:
            return json.load(f)

    def row(self, name):
        for it in self._load()["items"]:
            if it["name"] == name:
                return dict(it)
        return None

    def update(self, name, **kv):
        d = self._load()
        for it in d["items"]:
            if it["name"] == name:
                it.update(kv)
                break
        else:
            raise FlipError(f"{name}: no row in {os.path.basename(self.path)}")
        tmp = self.path + ".new"
        with open(tmp, "w") as f:
            json.dump(d, f, indent=1, ensure_ascii=False)
            f.write("\n")
        os.replace(tmp, self.path)


class Flipper:
    def __init__(self, make_sandbox, store, ledger, clock=now_utc, sleep=time.sleep, out=print, env=None, dry_run=False):
        self.make, self.store, self.ledger, self.clock, self.sleep, self.out = make_sandbox, store, ledger, clock, sleep, out
        self.env = os.environ if env is None else env
        self.dry = dry_run
        self.verify_s = int(self.env.get("D2_FLIP_VERIFY_S", 240))

    def verify(self, sb, knobs):
        """Wait for `ops status --why` to show the flipped value among the live knobs (the new child read the file).
        Returns the hard counters (since, counts) of the new child; FlipError when the value never shows."""
        waited, last = 0, "no answer"
        while True:
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
            if row is None or row["status"] not in ("dark", "trial"):
                raise FlipError(f"{knob}: needs a dark row in data/darklaunch.json")
        sb = self.make(instance)
        before = sb.read_tuning()
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
        return True

    def close(self, w):
        for k in w["knobs"]:
            self.ledger.update(k, status="enabled", enabled_at=iso(self.clock()))
        self.store.last(w["instance"], w)
        self.store.drop(w["instance"])  # the backup tag goes with the state file
        self.out(f"CLOSE {w['instance']} [{','.join(w['knobs'])}]: no hard breach for {w['window_s']}s; rows enabled, backup {w['tag']} removed")

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

    def watch(self, interval=60, once=False):
        while True:
            names = self.store.all()
            states = [self.poll(n) for n in names]
            if once or "open" not in states:
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
    ap.add_argument("--dry-run", action="store_true", help="print every step, write and restart nothing")
    a = ap.parse_args(argv)
    env = load_env()
    store = Store(env.get("D2_FLIP_STATE") or os.path.expanduser("~/.cache/d2flip"))
    f = Flipper(lambda inst: Sandbox(inst, env), store, Ledger(), env=env, dry_run=a.dry_run)
    try:
        if a.args[0] == "watch":
            f.watch(a.interval, a.once)
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
    except FlipError as e:
        print(f"flip: {e}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
