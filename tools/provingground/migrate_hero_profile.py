#!/usr/bin/env python3
"""Write the explicit pre.hero_profile into every scenario file (fix6, review 2026-11-28 fix 2).

A scenario without the key gets "banked", the effective default until now; one that names a profile is left alone.
The edit is textual (the key is inserted right after `"pre": {`), so the diff is the 200-odd scenarios whose setup the key
decides and nothing else. Idempotent. Usage: migrate_hero_profile.py [scenario dir]
"""
import json
import pathlib
import re
import sys

PRE = re.compile(r'"pre":\s*\{')


def migrate(path):
    text = path.read_text()
    doc = json.loads(text)
    scs = doc.get("scenarios") if isinstance(doc, dict) else None
    if not isinstance(scs, list):
        return 0
    todo = []
    for s in scs:
        if "pre" not in s:
            sys.exit(f"{path}: scenario {s.get('id')} has no pre block: add one by hand")
        todo.append("hero_profile" not in s["pre"])
    lines, out, k = text.split("\n"), [], 0
    for i, line in enumerate(lines):
        m = PRE.search(line)
        if not m:
            out.append(line)
            continue
        if k >= len(todo):
            sys.exit(f"{path}:{i + 1}: more pre blocks than scenarios")
        need, k = todo[k], k + 1
        head, rest = line[: m.end()], line[m.end() :]
        if not need:
            out.append(line)
        elif rest.strip() == "":  # the block opens here, its keys follow one per line
            nxt = lines[i + 1]
            pad = nxt[: len(nxt) - len(nxt.lstrip())]
            out.append(line)
            out.append(f'{pad}"hero_profile": "banked",')
        elif rest.lstrip().startswith("}"):  # "pre": {}
            out.append(f'{head}"hero_profile": "banked"{rest.lstrip()}')
        else:  # "pre": {"level": 18},
            out.append(f'{head}"hero_profile": "banked", {rest}')
    if k != len(todo):
        sys.exit(f"{path}: {len(todo)} scenarios, {k} pre blocks found (a scenario's pre block was not found)")
    new = "\n".join(out)
    want = json.loads(text)
    for s in want["scenarios"]:
        s["pre"].setdefault("hero_profile", "banked")
    if json.loads(new) != want:
        sys.exit(f"{path}: the edit changed more than the profile; nothing written")
    if new != text:
        path.write_text(new)
    return sum(todo)


def main():
    d = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else "tools/provingground/scenarios")
    total = 0
    for p in sorted(d.glob("*.json")):
        n = migrate(p)
        if n:
            print(f"{p.name}: {n}")
        total += n
    print(f"hero_profile written into {total} scenarios")


if __name__ == "__main__":
    main()
