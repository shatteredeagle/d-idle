#!/usr/bin/env python3
"""Usage view: tokens per day, per agent name and per model, from the Claude Code
transcripts (read-only; stdlib only). See docs/usage.md.

  tools/usage/usage.py [--days N] [--root DIR] [--project SUBSTR] [--repo DIR]
                       [--top N] [--json]

Reads ~/.claude/projects/*/<session>/subagents/agent-*.jsonl (one per subagent, with its
.meta.json description) and the main session files *.jsonl (reported as coordinator
rows). Per assistant message the API `usage` is counted once (streamed chunks repeat
the same message id). Tokens: fresh = input + output + cache creation; cache reads are
shown apart (cheap per token, but large). No prices.
"""
import argparse, collections, datetime, glob, json, os, re, subprocess, sys

HOME = os.path.expanduser("~")


def fam(model):
    m = (model or "").lower()
    for k in ("opus", "sonnet", "haiku", "fable"):
        if k in m:
            return k
    return m or "unknown"


def agent_name(desc):
    w = re.split(r"[\s:(]", (desc or "").strip())[0]
    return w or "(none)"


def scan_file(path, cutoff):
    """Return {msgid: (day, model, in, out, cr, cc)} for one transcript."""
    out = {}
    with open(path, "rb") as f:
        for l in f:
            if b'"usage"' not in l:
                continue
            try:
                d = json.loads(l)
            except ValueError:
                continue
            m = d.get("message")
            if not isinstance(m, dict) or not isinstance(m.get("usage"), dict):
                continue
            u = m["usage"]
            day = (d.get("timestamp") or "")[:10]
            if day < cutoff:
                continue
            mid = m.get("id") or d.get("uuid")
            out[mid] = (day, fam(m.get("model")), u.get("input_tokens", 0) or 0,
                        u.get("output_tokens", 0) or 0, u.get("cache_read_input_tokens", 0) or 0,
                        u.get("cache_creation_input_tokens", 0) or 0)
    return out


def merges_per_day(repo, cutoff):
    try:
        o = subprocess.run(["git", "-C", repo, "log", "--first-parent", "main", "--format=%cs\t%s",
                            "--since=" + cutoff], capture_output=True, text=True, timeout=60).stdout
    except (OSError, subprocess.SubprocessError):
        return {}
    c = collections.Counter()
    for l in o.splitlines():
        d, _, s = l.partition("\t")
        if re.match(r"Merge (branch )?'?agent/", s):
            c[d] += 1
    return c


def fmt(n):
    return "{:,}".format(int(n))


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--days", type=int, default=7)
    ap.add_argument("--root", default=os.path.join(HOME, ".claude", "projects"))
    ap.add_argument("--project", default="", help="substring of the project dir name (default all)")
    ap.add_argument("--repo", default=os.path.join(HOME, "code", "d2lod-bot"))
    ap.add_argument("--top", type=int, default=10)
    ap.add_argument("--json", action="store_true")
    a = ap.parse_args()
    today = datetime.date.today()
    cutoff = (today - datetime.timedelta(days=a.days - 1)).isoformat()
    cut_ts = datetime.datetime.combine(today - datetime.timedelta(days=a.days + 1), datetime.time()).timestamp()

    agents = {}  # key -> dict
    for pdir in sorted(glob.glob(os.path.join(a.root, "*"))):
        proj = os.path.basename(pdir)
        if a.project not in proj:
            continue
        files = glob.glob(os.path.join(pdir, "*", "subagents", "agent-*.jsonl")) + \
            glob.glob(os.path.join(pdir, "*.jsonl"))
        for p in files:
            try:
                if os.path.getmtime(p) < cut_ts:
                    continue
            except OSError:
                continue
            sub = "/subagents/" in p
            msgs = scan_file(p, cutoff)
            if not msgs:
                continue
            if sub:
                desc, atype = "", ""
                try:
                    with open(p[:-6] + ".meta.json") as f:
                        meta = json.load(f)
                    desc, atype = meta.get("description", ""), meta.get("agentType", "")
                except (OSError, ValueError):
                    pass
                name = agent_name(desc)
            else:
                desc, atype, name = "", "coordinator", "(coordinator)"
            key = p
            r = agents.setdefault(key, {"project": proj, "name": name, "type": atype, "desc": desc,
                                        "sub": sub, "tok": collections.defaultdict(lambda: [0, 0, 0, 0]),
                                        "days": set()})
            for day, model, i, o, cr, cc in msgs.values():
                t = r["tok"][(day, model)]
                t[0] += i; t[1] += o; t[2] += cr; t[3] += cc
                r["days"].add(day)

    def fresh(t):
        return t[0] + t[1] + t[3]

    rows = []
    for r in agents.values():
        tot = [0, 0, 0, 0]
        models = collections.Counter()
        for (day, model), t in r["tok"].items():
            for k in range(4):
                tot[k] += t[k]
            models[model] += fresh(t)
        rows.append({"project": r["project"], "name": r["name"], "type": r["type"], "desc": r["desc"],
                     "sub": r["sub"], "first_day": min(r["days"]), "fresh": fresh(tot), "out": tot[1],
                     "cache_read": tot[2], "model": models.most_common(1)[0][0], "tok": r["tok"]})
    perday = collections.defaultdict(lambda: {"agents": set(), "mod": collections.Counter(), "cr": 0})
    for i, r in enumerate(rows):
        for (day, model), t in r["tok"].items():
            d = perday[day]
            d["agents"].add(i) if r["sub"] else None
            d["mod"][model] += fresh(t)
            d["cr"] += t[2]
    merges = merges_per_day(a.repo, cutoff)

    if a.json:
        print(json.dumps({"days": {k: {"agents": len(v["agents"]), "fresh_by_model": v["mod"], "cache_read": v["cr"],
                                       "merges": merges.get(k, 0)} for k, v in perday.items()},
                          "agents": [{k: v for k, v in r.items() if k != "tok"} for r in rows]}, default=list))
        return
    subs = [r for r in rows if r["sub"]]
    print("USAGE %s .. %s  (fresh = input+output+cache creation; cache reads apart; no prices)" % (cutoff, today))
    print("transcripts with usage: %d subagents + %d coordinator sessions, projects: %d" % (
        len(subs), len(rows) - len(subs), len({r["project"] for r in rows})))
    print("\nPer day: subagents, merges on main, fresh tokens by model, cache reads, fresh per merge")
    print("  %-10s %6s %6s %12s %12s %12s %12s %14s %10s" % ("day", "agents", "merges", "opus", "sonnet", "haiku", "fable", "cache read", "fresh/merge"))
    gt = collections.Counter()
    for day in sorted(perday):
        d = perday[day]
        tot = sum(d["mod"].values())
        for k, v in d["mod"].items():
            gt[k] += v
        gt["_cr"] += d["cr"]
        gt["_agents"] += len(d["agents"])
        gt["_merges"] += merges.get(day, 0)
        m = merges.get(day, 0)
        print("  %-10s %6d %6d %12s %12s %12s %12s %14s %10s" % (day, len(d["agents"]), m, fmt(d["mod"]["opus"]),
              fmt(d["mod"]["sonnet"]), fmt(d["mod"]["haiku"]), fmt(d["mod"]["fable"]), fmt(d["cr"]),
              fmt(tot / m) if m else "-"))
    ft = sum(v for k, v in gt.items() if not k.startswith("_"))
    print("  %-10s %6d %6d %12s %12s %12s %12s %14s %10s" % ("total", gt["_agents"], gt["_merges"], fmt(gt["opus"]),
          fmt(gt["sonnet"]), fmt(gt["haiku"]), fmt(gt["fable"]), fmt(gt["_cr"]),
          fmt(ft / gt["_merges"]) if gt["_merges"] else "-"))
    by = collections.defaultdict(lambda: [0, 0, 0, 0])
    for r in subs:
        g = by[r["name"]]
        g[0] += 1; g[1] += r["fresh"]; g[2] += r["cache_read"]; g[3] = max(g[3], r["fresh"])
    print("\nPer agent name (first word of the launch description): agents, fresh tokens, cache read, biggest single")
    for n, g in sorted(by.items(), key=lambda kv: -kv[1][1])[:max(a.top * 2, 20)]:
        print("  %-22s %4d %13s %14s %12s" % (n[:22], g[0], fmt(g[1]), fmt(g[2]), fmt(g[3])))
    bt = collections.defaultdict(lambda: [0, 0])
    for r in rows:
        k = r["type"] or "(unknown)"
        bt[k][0] += 1; bt[k][1] += r["fresh"]
    pj = collections.defaultdict(lambda: [0, 0, 0])
    for r in rows:
        g = pj[r["project"]]
        g[0] += 1 if r["sub"] else 0; g[1] += r["fresh"]; g[2] += r["cache_read"]
    print("\nPer project dir: subagents, fresh tokens (incl. coordinator), cache read")
    for k, g in sorted(pj.items(), key=lambda kv: -kv[1][1])[:12]:
        print("  %-50s %5d %13s %15s" % (k[-50:], g[0], fmt(g[1]), fmt(g[2])))
    print("\nPer agent type: count, fresh tokens")
    for k, g in sorted(bt.items(), key=lambda kv: -kv[1][1]):
        print("  %-18s %5d %14s" % (k, g[0], fmt(g[1])))
    print("\nTop %d most expensive agents (fresh tokens; one transcript = one agent)" % a.top)
    for r in sorted(subs, key=lambda r: -r["fresh"])[:a.top]:
        print("  %12s  %-6s %-14s %s  %s" % (fmt(r["fresh"]), r["model"], (r["type"] or "-")[:14], r["first_day"], r["desc"][:80]))
    top = sorted(rows, key=lambda r: -r["fresh"])[:3]
    print("\nCoordinator sessions (main conversation, not a subagent): top by fresh tokens")
    for r in [r for r in sorted(rows, key=lambda r: -r["fresh"]) if not r["sub"]][:3]:
        print("  %12s  %-6s %s  %s" % (fmt(r["fresh"]), r["model"], r["first_day"], r["project"][-40:]))
    print("\nNot visible: runners that leave no Claude Code transcript here (credit/API runs: see their own ledgers),")
    print("other machines, and transcripts deleted by cleanup. Merges = first-parent 'Merge agent/*' on main.")


if __name__ == "__main__":
    main()
