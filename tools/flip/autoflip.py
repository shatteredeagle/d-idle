#!/usr/bin/env python3
"""Plan, run and follow up dark-knob flips on the sandboxes (docs/process/autoflip.md, card autoflip). Uses flip.py; never edits code.

  autoflip.py plan [--json]            bundles for every dark coordinator row, with the exact flip.py command
  autoflip.py run [--dry-run]          dry-run, then flip the first bundle of each instance that is free and healthy
  autoflip.py graduate-list            rows whose window closed clean, with the brief for a graduation card

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

HERE = os.path.dirname(os.path.abspath(__file__))
CONFIG = os.path.join(HERE, "autoflip.json")


def load_config(path=CONFIG):
    with open(path) as f:
        return json.load(f)


def row_instances(row, cfg):
    """-> (list of instances, bundle group, window, value) for a row; ([], ...) when no plan applies."""
    fp = row.get("flip_plan") or {}
    inst = fp.get("instance")
    if inst:
        insts = [inst] if isinstance(inst, str) else list(inst)
    elif row["name"] in cfg.get("log_only_rows", []):
        insts = sorted({i for v in cfg["prefix_plans"].values() for i in v})
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


def graduates(items):
    return [r for r in items if r.get("status") == "enabled" and r.get("enabled_at") and not r.get("revert_reason") and not r.get("by_design")]


def cmd_graduate(items, root, out=print):
    rows = graduates(items)
    if not rows:
        out("no row has closed a clean window")
    for r in rows:
        k = r["name"]
        defs = [l for l in grep(root, f'Key: "{k}"')]
        sites = [l for l in grep(root, k) if l not in defs]
        out(f"## graduate {k} (enabled_at {r['enabled_at']}, trial_value {r.get('trial_value', '-')})")
        out(f"- default: {defs[0].split(':')[0] if defs else 'NOT FOUND'} (`Key: \"{k}\"`, change Default to the trial value, drop the [dark-launch] tag)")
        out("- old-path sites (grep of the knob name, review each):")
        for l in sites[:20] or ["(none found: look for an accessor of the knob in the file above)"]:
            out(f"    {l}")
        out(f"- ledger edit: data/darklaunch/{k}.json: status -> default-on (python3 -I tools/darklaunch/ledger.py status default-on {k}), note the merge commit; the file stays as history")
        out("- no automatic code edit: launch a worker with this brief")
    return 0


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("cmd", choices=["plan", "run", "graduate-list"])
    ap.add_argument("--json", action="store_true")
    ap.add_argument("--dry-run", action="store_true")
    a = ap.parse_args(argv)
    cfg = load_config()
    env = F.load_env()
    ledger = F.Ledger(env=env)
    items = ledger.items()
    store = F.Store(env.get("D2_FLIP_STATE") or os.path.expanduser("~/.cache/d2flip"))
    mk = lambda inst: F.Sandbox(inst, env)  # noqa: E731
    try:
        if a.cmd == "plan":
            return cmd_plan(a, cfg, mk, items, store)
        if a.cmd == "run":
            return cmd_run(a, cfg, mk, items, store, ledger, env)
        return cmd_graduate(items, F.ROOT)
    except F.FlipError as e:
        print(f"autoflip: {e}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
