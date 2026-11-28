#!/usr/bin/env python3
"""Plan, run and follow up dark-knob flips on the sandboxes (docs/process/autoflip.md, card autoflip). Uses flip.py; never edits code.

  autoflip.py plan [--json]            bundles for every dark coordinator row, with the exact flip.py command
  autoflip.py run [--dry-run]          dry-run, then flip the first bundle of each instance that is free and healthy
  autoflip.py graduate-list            rows whose window closed clean (state note graduation-ready), checked against the
                                       per-knob ledger files, with the brief for a graduation card; exit 1 on a ledger problem

Instances are sandbox aliases (tools/flip/sandboxes.json) or container names; "both" in a flip_plan means every planned sandbox.

Read-only toward the sandbox except `run` (which calls Flipper.flip). Rows with by_design, an owner other than coordinator,
or a status other than dark are never planned.
"""
import argparse
import json
import os
import re
import subprocess
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import flip as F  # noqa: E402
from flip import L  # noqa: E402

HERE = os.path.dirname(os.path.abspath(__file__))
CONFIG = os.path.join(HERE, "autoflip.json")


def load_config(path=CONFIG):
    with open(path) as f:
        return json.load(f)


def row_instances(row, cfg):
    """-> (list of instances, bundle group, window, value) for a row; ([], ...) when no plan applies."""
    fp = row.get("flip_plan") or {}
    inst = fp.get("instance")
    everywhere = sorted({i for v in cfg["prefix_plans"].values() for i in v})
    if inst == "both":
        insts = everywhere
    elif inst:
        insts = [inst] if isinstance(inst, str) else list(inst)
    elif row["name"] in cfg.get("log_only_rows", []):
        insts = everywhere
    else:
        insts = next((v for p, v in cfg["prefix_plans"].items() if row["name"].startswith(p)), [])
    group = fp.get("bundle") or row["name"].split(".")[0]
    return insts, group, fp.get("window") or cfg["window"], float(fp.get("value", cfg["value"]))


def skip_reason(row):
    if row.get("by_design"):
        return "by_design"
    if row.get("owner") != "coordinator":
        return f"owner {row.get('owner')}"
    if row.get("status") != "dark":
        return f"status {row.get('status')}"
    return ""


def live_knobs(make_sandbox, instance, cache):
    """-> set of knob names the deployed build of the instance knows, or an error string."""
    if instance not in cache:
        try:
            cache[instance] = set((make_sandbox(instance).status().get("knobs")) or {})
        except Exception as e:
            cache[instance] = f"sandbox {instance} unreadable: {e}"
    return cache[instance]


def make_plan(items, cfg, make_sandbox):
    """-> (bundles, skipped). bundle: {instance, group, window, knobs{name: value}, order}; skipped: [(name, why)]."""
    skipped, groups, cache = [], {}, {}
    for row in items:
        why = skip_reason(row)
        if why:
            if row.get("status") == "dark":
                skipped.append((row["name"], why))
            continue
        insts, group, window, value = row_instances(row, cfg)
        if not insts:
            skipped.append((row["name"], "no default plan for this prefix"))
            continue
        for inst in insts:
            k = live_knobs(make_sandbox, inst, cache)
            if isinstance(k, str):
                skipped.append((row["name"], k))
            elif row["name"] not in k:
                skipped.append((row["name"], f"knob not in the deployed build of {inst}"))
            else:
                groups.setdefault((inst, group, window), {})[row["name"]] = value
    bundles, per_inst = [], {}
    for (inst, group, window), knobs in sorted(groups.items()):
        names = sorted(knobs)
        for i in range(0, len(names), cfg["max_bundle"]):
            n = per_inst.get(inst, 0)
            per_inst[inst] = n + 1
            bundles.append({"instance": inst, "group": group, "window": window, "order": n,
                            "knobs": {k: knobs[k] for k in names[i:i + cfg["max_bundle"]]}})
    return bundles, skipped


def command(b):
    kv = " ".join(f"{k}={v:g}" for k, v in b["knobs"].items())
    why = f"autoflip {b['group']} bundle"
    return f"python3 tools/flip/flip.py {b['instance']} {kv} --why '{why}' --window {b['window']}"


def runnable(bundles, store):
    """One window per instance: only the first bundle of an instance, and none when a window is open on it."""
    out = []
    for b in bundles:
        if b["order"] != 0:
            continue
        if store.get(b["instance"]):
            b = dict(b, blocked="window open")
        out.append(b)
    return out


def healthy(sb):
    _, hard = F.hard_counts(sb.status())
    bad = {k: v for k, v in hard.items() if v}
    if bad:
        raise F.FlipError(f"hard counters nonzero {bad}")


def cmd_plan(a, cfg, make_sandbox, items, store, out=print):
    bundles, skipped = make_plan(items, cfg, make_sandbox)
    if a.json:
        out(json.dumps({"bundles": bundles, "skipped": skipped}, indent=1))
        return 0
    for b in bundles:
        if b["order"] > 0:
            tag = f"QUEUED behind bundle {b['order']} of {b['instance']} (one window per instance)"
        else:
            tag = "WINDOW OPEN, wait" if store.get(b["instance"]) else "ready"
        out(f"# {b['instance']} {b['group']} ({len(b['knobs'])} knobs) {tag}")
        out(command(b))
    for n, why in skipped:
        out(f"# skipped {n}: {why}")
    return 0


def cmd_run(a, cfg, make_sandbox, items, store, ledger, env, out=print, sleep=None):
    bundles, _ = make_plan(items, cfg, make_sandbox)
    done = []
    for b in runnable(bundles, store):
        inst = b["instance"]
        if b.get("blocked"):
            out(f"skip {inst}: {b['blocked']}")
            continue
        try:
            sb = make_sandbox(inst)
            if hasattr(sb, "probe"):
                sb.probe()
            healthy(sb)
            kw = {"sleep": sleep} if sleep else {}
            secs = F.parse_window(b["window"])
            why = f"autoflip {b['group']} bundle"
            F.Flipper(make_sandbox, store, ledger, env=env, dry_run=True, out=out, **kw).flip(inst, b["knobs"], why, secs)
            if a.dry_run:
                continue
            F.Flipper(make_sandbox, store, ledger, env=env, out=out, **kw).flip(inst, b["knobs"], why, secs)
            done.append(f"{inst}:{','.join(b['knobs'])}")
        except F.FlipError as e:
            out(f"refuse {inst}: {e}")
    out(f"autoflip run: {len(done)} flipped" + (f" ({'; '.join(done)})" if done else "") + (" [dry-run]" if a.dry_run else ""))
    return 0


def grep(root, pattern, extra=()):
    r = subprocess.run(["grep", "-rnF", "--include=*.go", "--exclude=*_test.go", *extra, pattern, os.path.join(root, "internal"), os.path.join(root, "cmd")],
                       capture_output=True, text=True)
    return [l.replace(root + os.sep, "") for l in r.stdout.splitlines()]


def clean(r):
    """Closed and not reverted since: a revert before the last clean close (a re-flip that then held) does not count."""
    if not r.get("enabled_at"):
        return False
    if r.get("reverted_at"):
        return r["reverted_at"] < r["enabled_at"]
    return not r.get("revert_reason")


def _num(s):
    try:
        return float(s)
    except (TypeError, ValueError):
        return None


def graduates(items):
    return [r for r in items if r.get("status") == "enabled" and clean(r) and not r.get("by_design")]


def graduate_check(ledger, store=None, knobs=None):
    """Every row the runtime state or the merged view calls enabled, checked against the per-knob files in git.
    -> [{name, verdict, row, why[], notes[]}], verdict READY (brief printed), HOLD (clean so far, not yet a card) or
    PROBLEM (state file and git disagree). knobs: L.knob_rows() of docs/knobs.md, None to leave the code default out."""
    design = {d["name"]: d for d in L.load_design(ledger.path)}
    state = L.load_state(ledger.state)["rows"]
    merged = {r["name"]: r for r in ledger.items()}
    open_knobs = {k for n in store.all() for k in store.get(n)["knobs"]} if store else set()
    names = sorted({n for n, r in merged.items() if r.get("status") == "enabled"} | {n for n, r in state.items() if r.get("state") == "enabled"})
    out = []
    for n in names:
        d, r = design.get(n), merged.get(n) or dict(state[n], name=n)
        why, hold, notes = [], [], []
        if d is None:
            why.append(f"the state file says enabled but data/darklaunch/{n}.json does not exist (renamed or deleted knob: drop the state row)")
        elif d.get("by_design") or d.get("status") in ("default-on", "retired", "by_design"):
            continue  # graduated or never a candidate: the state row is history
        else:
            if not r.get("enabled_at"):
                why.append("enabled without enabled_at (no close on record)")
            elif not clean(r):
                why.append(f"reverted at {r.get('reverted_at') or '?'} after the close of {r['enabled_at']} ({r.get('revert_reason', '')}): the state should be dark")
            start, win = r.get("trial_started"), r.get("trial_window")
            if r.get("enabled_at") and start and win:
                if F.parse_iso(r["enabled_at"]) < F.parse_iso(start) + F.timedelta(seconds=F.parse_window(win)):
                    why.append(f"enabled_at {r['enabled_at']} is before the end of the {win} window that began {start}")
            elif r.get("criteria_met") == "yes":
                notes.append("no window on record (enabled by hand, criteria_met yes)")
            else:
                hold.append("no window on record (enabled by hand) and criteria_met is not yes")
            if knobs is not None:
                if n not in knobs:
                    why.append("the knob is not in docs/knobs.md (gate removed from the code?): set the row default-on or retired")
                elif _num(knobs[n][0]) is not None and _num(knobs[n][0]) == r.get("trial_value"):
                    why.append(f"the code default is already the trial value {knobs[n][0]}: set the row to default-on ({L.status_line('default-on', [n])})")
                elif L.MARKER not in knobs[n][1]:
                    notes.append(f"the knob has no {L.MARKER} tag in docs/knobs.md (default {knobs[n][0]}, not a default-0 gate): graduating means moving that default")
            if n in open_knobs:
                hold.append("a window with this knob is open")
            if r.get("criteria_met") == "no":
                hold.append("criteria_met no: the enable criteria failed or are not shown yet")
            elif r.get("criteria_met") != "yes":
                notes.append(f"criteria_met {r.get('criteria_met', 'U')}: confirm the enable criteria in the card")
            if d.get("owner") != "coordinator":
                hold.append(f"owner {d.get('owner')} decides")
            if d.get("status") == "dark":
                notes.append(f"git still says dark: {L.status_line('enabled', [n])}")
            if r.get("graduation_note") != F.GRADUATION_READY:
                notes.append("no graduation-ready note (closed before the watcher wrote it, or enabled by hand)")
        out.append({"name": n, "verdict": "PROBLEM" if why else "HOLD" if hold else "READY", "row": r, "why": why + hold, "notes": notes})
    return out


def cmd_graduate(ledger, root, store=None, out=print, knobs=None):
    checked = graduate_check(ledger, store, knobs)
    ready = [c for c in checked if c["verdict"] == "READY"]
    if not ready:
        out("no row has closed a clean window")
    for c in ready:
        r, k = c["row"], c["name"]
        defs = [l for l in grep(root, f'Key: "{k}"')]
        sites = [l for l in grep(root, k) if l not in defs]
        out(f"## graduate {k} ({r.get('graduation_note') or 'clean'}, enabled_at {F._est_or(r.get('enabled_at'))}, trial_value {r.get('trial_value', '-')}, criteria_met {r.get('criteria_met', 'U')})")
        for note in c["notes"]:
            out(f"- check: {note}")
        out(f"- default: {defs[0].split(':')[0] if defs else 'NOT FOUND'} (`Key: \"{k}\"`, change Default to the trial value, drop the [dark-launch] tag)")
        out("- old-path sites (grep of the knob name, review each):")
        for l in sites[:20] or ["(none found: look for an accessor of the knob in the file above)"]:
            out(f"    {l}")
        out(f"- ledger edit: data/darklaunch/{k}.json: status -> default-on (python3 -I tools/darklaunch/ledger.py status default-on {k}), note the merge commit; the file stays as history")
        out("- no automatic code edit: launch a worker with this brief")
    for c in checked:
        if c["verdict"] != "READY":
            out(f"# {c['verdict']} {c['name']}: {'; '.join(c['why'])}")
    bad = sum(c["verdict"] == "PROBLEM" for c in checked)
    out(f"# ledger check: {len(ready)} ready, {sum(c['verdict'] == 'HOLD' for c in checked)} hold, {bad} problem(s) against {ledger.path}")
    return 1 if bad else 0


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("cmd", choices=["plan", "run", "graduate-list"])
    ap.add_argument("--json", action="store_true")
    ap.add_argument("--dry-run", action="store_true")
    a = ap.parse_args(argv)
    cfg = load_config()
    env = F.load_env()
    ledger = F.Ledger(env=env)
    try:
        items = ledger.items()
        names = F.Names.load()
        store = F.Store(env.get("D2_FLIP_STATE") or os.path.expanduser("~/.cache/d2flip"), names.container)
        mk = lambda inst: F.Sandbox(inst, env, names)  # noqa: E731
        if a.cmd == "plan":
            return cmd_plan(a, cfg, mk, items, store)
        if a.cmd == "run":
            return cmd_run(a, cfg, mk, items, store, ledger, env)
        knobs_md = os.path.join(F.ROOT, "docs", "knobs.md")
        return cmd_graduate(ledger, F.ROOT, store, knobs=L.knob_rows(knobs_md) if os.path.isfile(knobs_md) else None)
    except (F.FlipError, ValueError, OSError) as e:
        print(f"autoflip: {e}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
