#!/usr/bin/env python3
"""Build an Archify architecture JSON-IR from `go list` (no LLM, no network).

usage: gen.py <repo-dir> <overview|packages> <out.json> [--html-name NAME]

The picture is derived from the Go import graph of <repo-dir>; the grouping
below is data (edit GROUPS to retitle or move a package). Packages that match
no group are put in "Other" so a new package never disappears.
"""
import json
import re
import subprocess
import sys
from collections import defaultdict

# group id -> (tier column, row slot, node type, title, subtitle)
# tier left to right follows the dependency direction (callers on the left).
GROUPS = {
    "cmd":    (0, 0, "external", "Commands",          "cmd/* mains"),
    "screen": (1, 0, "frontend", "Player screen",     "watch, api, status"),
    "ops":    (1, 1, "security", "Ops and lab",       "fleet, setup, drills"),
    "brain":  (2, 0, "backend",  "Brain",             "bot, goals, task"),
    "behave": (3, 0, "cloud",    "Behaviours",        "combat, nav, town"),
    "d2d":    (4, 0, "messagebus", "d2d backend",     "d2dctl, d2d* glue"),
    "game":   (4, 1, "database", "Game state",        "mem, d2, d2map"),
    "base":   (5, 0, "database", "Foundation",        "store, event, env"),
}
SEG = {
    "screen": "watch api server status invdesk access unlock overlay art viewport skilltree telemetry stepgroup",
    "ops": "fleet launch setup selfupdate sentinel drill smoke fidprobe canary cheat saves d2dsoakstat d2dmesh d2dreplay "
           "boss replay",
    "brain": "bot goals task campaign quest questmem decide ladder readiness plandata goldfarm party pact",
    "behave": "combat nav town loot leveler merc resource repair talk cubejobs gear currency senses",
    "d2d": "d2dctl d2dcombat d2denv d2dinput d2dloot d2dmerc d2dsrc d2dtown d2dworld",
    "game": "mem memfd d2 d2map data input",
    "base": "store event env paths logx metrics gclock buildinfo",
}
SEG2GROUP = {s: g for g, ss in SEG.items() for s in ss.split()}
LABEL = {}

CMD_PLAYER = "d2bot d2watch d2view d2dash didleboot d2multi d2fleet d2game d2input oog d2doctor d2dsetup d2restore"
CMD_KIND = {"player": CMD_PLAYER.split()}


def go_list(repo):
    out = subprocess.run(
        ["nice", "go", "list", "-json=ImportPath,Imports,Doc", "./..."],
        cwd=repo, check=True, capture_output=True, text=True).stdout
    mod = subprocess.run(["go", "list", "-m"], cwd=repo, check=True, capture_output=True, text=True).stdout.strip()
    pk = [json.loads(x) for x in re.split(r"(?<=\})\n(?=\{)", out.strip())]
    return mod, pk


def ident(s):
    return re.sub(r"[^a-z0-9]+", "-", s.lower()).strip("-")


def classify(path, mod):
    """Return (node id, group, label) for an import path, or None for non-module paths."""
    if not path.startswith(mod + "/"):
        return None
    rel = path[len(mod) + 1:]
    if rel.startswith("cmd/"):
        name = rel[4:]
        kind = "player" if name in CMD_KIND["player"] else ("d2d" if name.startswith("d2d") else "probe")
        return "cmd-" + kind, "cmd", {"player": "player binaries", "d2d": "d2d lab tools", "probe": "probes, drills"}[kind]
    if rel.startswith("internal/"):
        top = rel.split("/")[1]
        g = SEG2GROUP.get(top)
        if g is None:
            if top.endswith("sim") or top.endswith("test"):
                g = "ops"
            else:
                g = "behave" if False else "ops"
        if "/" in rel[9:] and rel.split("/")[-1] in ("sim", "fake", "geartest", "setuptest"):
            g = "ops"  # test doubles live in the ops tier
        node = ident(rel[9:])
        return node, g, LABEL.get(rel[9:], rel[9:])
    return None


def build(repo, mode, title, out):
    mod, pkgs = go_list(repo)
    nodes = {}  # id -> (group, label, doc, count)
    edges = defaultdict(int)
    doc = {}
    for p in pkgs:
        c = classify(p["ImportPath"], mod)
        if not c:
            continue
        nid, g, label = c
        n = nodes.setdefault(nid, [g, label, 0])
        n[2] += 1
        doc.setdefault(nid, p.get("Doc", ""))
    for p in pkgs:
        a = classify(p["ImportPath"], mod)
        if not a:
            continue
        for imp in p.get("Imports", []) or []:
            b = classify(imp, mod)
            if b and b[0] != a[0]:
                edges[(a[0], b[0])] += 1

    comps, conns, bounds = [], [], []
    extra = ""
    if mode == "overview":
        gcount = defaultdict(int)
        for nid, (g, label, k) in nodes.items():
            gcount[g] += k
        for g, (col, row, typ, name, sub) in GROUPS.items():
            comps.append({"id": g, "type": typ, "label": name,
                          "sublabel": "%d pkgs: %s" % (gcount[g], sub), "size": [200, 64],
                          "row": row, "col": col})
        agg = defaultdict(int)
        for (a, b), k in edges.items():
            ga, gb = nodes[a][0], nodes[b][0]
            if ga != gb and gb != "base":
                agg[(ga, gb)] += 1
        omitted = 0
        for (ga, gb), k in sorted(agg.items()):
            if ga == "cmd" or abs(GROUPS[ga][0] - GROUPS[gb][0]) > 1:
                omitted += k  # long jumps would cross other boxes; counted in the subtitle
                continue
            conns.append({"id": "%s-%s" % (ga, gb), "from": ga, "to": gb, "width": round(1 + k / 6.0, 1)})
        # everything depends on Foundation: draw only the strongest feeders
        feed = defaultdict(int)
        for (a, b), k in edges.items():
            if nodes[b][0] == "base" and nodes[a][0] != "base":
                feed[nodes[a][0]] += 1
        extra = " Line thickness grows with the number of package imports between two groups. Left out: imports from Commands, imports that jump over a tier (%d) and imports of Foundation, which everything uses." % omitted
        layout = {"mode": "grid", "cols": 6, "gapX": 90, "gapY": 80, "cellW": 170, "cellH": 70}
    else:
        # packages: one boundary per group, packages in two columns inside
        rowcur = defaultdict(int)  # tier -> next free row
        order = sorted(GROUPS, key=lambda g: (GROUPS[g][0], GROUPS[g][1]))
        for g in order:
            tier, slot, typ, name, sub = GROUPS[g]
            members = sorted(n for n, v in nodes.items() if v[0] == g)
            r0 = rowcur[tier]
            for i, nid in enumerate(members):
                comps.append({"id": nid, "type": typ, "label": nodes[nid][1],
                              "row": r0 + i // 2, "col": tier * 2 + i % 2, "size": [150, 34]})
            if members:
                bounds.append({"kind": "region", "label": name, "wraps": members})
                rowcur[tier] = r0 + (len(members) + 1) // 2 + 1
        # edges: only cross-group, and only toward non-foundation groups
        extra = " Boxes only: 280 cross-package edges would be a hairball, see the overview for the flow."
        layout = {"mode": "grid", "cols": 12, "gapX": 40, "gapY": 22, "cellW": 150, "cellH": 40}

    legend = {"mode": "visible", "entries": {
        "external": {"label": "Commands"}, "frontend": {"label": "Player screen"},
        "security": {"label": "Ops and lab"}, "backend": {"label": "Brain"},
        "cloud": {"label": "Behaviours"}, "messagebus": {"label": "d2d backend"},
        "database": {"label": "Game state, foundation"}}}
    doc_ = {
        "schema_version": 1, "diagram_type": "architecture",
        "meta": {"title": title, "subtitle": "Go import graph, generated by tools/arch (derived, not hand drawn)." + extra,
                 "output": out, "quality_profile": "standard", "visual_preset": "blueprint"},
        "layout": layout, "components": comps,
    }
    if conns:
        doc_["connections"] = conns
    if bounds:
        doc_["boundaries"] = bounds
    return doc_


def main():
    a = sys.argv[1:]
    if len(a) < 4:
        sys.exit(__doc__)
    repo, mode, out_json, html = a[0], a[1], a[2], a[3]
    title = {"overview": "d.idle: overview", "packages": "d.idle: every package"}[mode]
    d = build(repo, mode, title, html)
    json.dump(d, open(out_json, "w"), indent=1)
    print("%s: %d nodes, %d edges" % (mode, len(d["components"]), len(d.get("connections", []))))


if __name__ == "__main__":
    main()
