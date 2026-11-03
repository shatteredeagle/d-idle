#!/usr/bin/env python3
"""verifydmg3: CI parity and suites for DMGROLL r3, read-only on the author's tree and build.
1. the author's tree (wt-mdmg-r3) equals this verifier's export of 2df7d1eb (t_r3, w3_git.sh) for components/ and tests/;
2. g++ and clang++ -fsyntax-only with the CI flags plus -Wshadow -Wrange-loop-construct -Wconversion on the changed TUs,
   from the build's compile_commands.json; diagnostics on lines the commit added are counted (added lines from r3.diff);
3. the suites of that build, run in $D2_SCRATCH, rc and last line.

    run-on-worker.sh --ttl 3000 w3_ci.py [syntax] [suites]
"""
import json
import os
import re
import subprocess
import sys

HOME = os.path.expanduser("~")
TREE = f"{HOME}/d2d-lab/wt-mdmg-r3"
BUILD = f"{HOME}/d2d-lab/build-mdr3"
S = os.environ["D2_SCRATCH"]
TUS = ["components/game/ai.cpp", "components/game/fight/missiles.cpp", "components/game/fight/pets.cpp", "components/game/fight/states.cpp", "components/game/fight/tick.cpp",
       "tests/game/test_game.cpp", "tests/test_combat.cpp", "components/game/fight/strike.cpp"]
FLAGS = ["-Wall", "-Wextra", "-Wpedantic", "-Wno-missing-field-initializers", "-Werror", "-Wshadow", "-Wrange-loop-construct",
         "-Wconversion", "-fsyntax-only"]
SUITES = ["test_rules", "test_game_rules", "test_monsters", "test_monster_thinks", "test_monster_ai", "test_combat", "test_game",
          "test_skills", "test_sk_ground", "test_monster_missile", "test_missile_blow_poison"]


def added_lines():
    out, f, ln = {}, None, 0
    for line in open(f"{S}/r3.diff"):
        if line.startswith("+++ b/"):
            f = line[6:].strip()
            out[f] = set()
        elif line.startswith("@@"):
            ln = int(re.search(r"\+(\d+)", line).group(1))
        elif f and line.startswith("+"):
            out[f].add(ln)
            ln += 1
        elif f and not line.startswith("-"):
            ln += 1
    return out


def same_tree():
    r = subprocess.run(["diff", "-rq", f"{TREE}/components", f"{S}/t_r3/components"], capture_output=True, text=True)
    t = subprocess.run(["diff", "-rq", f"{TREE}/tests", f"{S}/t_r3/tests"], capture_output=True, text=True)
    print(f"tree vs export of 2df7d1eb: components rc={r.returncode} lines={len(r.stdout.splitlines())}, tests rc={t.returncode} lines={len(t.stdout.splitlines())}")
    print((r.stdout + t.stdout)[:2000])


def syntax():
    cmds = {os.path.relpath(c["file"], TREE): c for c in json.load(open(f"{BUILD}/compile_commands.json"))}
    add = added_lines()
    for cxx in ("g++", "clang++"):
        total = ours = 0
        for tu in TUS:
            c = cmds.get(tu)
            if not c:
                print(f"{cxx} {tu}: no compile command")
                continue
            args = c["command"].split() if "command" in c else c["arguments"]
            args = [a for a in args[1:] if a not in ("-Werror",) and not a.endswith("ccache")]
            drop = set()
            for i, a in enumerate(args):
                if a == "-o":
                    drop |= {i, i + 1}
                if a in ("-c",):
                    drop.add(i)
            args = [a for i, a in enumerate(args) if i not in drop]
            if cxx == "clang++":
                args = [a for a in args if not a.startswith("-fconcepts-diagnostics") and a != "-fno-gnu-unique"]
            p = subprocess.run(["nice", "-n", "19", cxx] + args + FLAGS, cwd=c["directory"], capture_output=True, text=True)
            diags = [ln for ln in p.stderr.splitlines() if re.search(r": (warning|error):", ln)]
            mine = []
            for d in diags:
                m = re.match(r"(.*?):(\d+):\d+: (warning|error):", d)
                if m:
                    rel = os.path.relpath(os.path.realpath(os.path.join(c["directory"], m.group(1))), TREE)
                    if int(m.group(2)) in add.get(rel, set()):
                        mine.append(d)
            total += len(diags)
            ours += len(mine)
            print(f"{cxx} {tu}: rc={p.returncode} diagnostics={len(diags)} on added lines={len(mine)}")
            for d in mine[:10]:
                print("   ", d)
            if diags and not mine:
                print("    first:", diags[0][:300])
        print(f"== {cxx}: diagnostics {total}, on added lines {ours}")


def suites():
    env = dict(os.environ, D2_PATCH_INSTALLER=f"{HOME}/Downloads/LODPatch_114d.exe")
    for t in SUITES:
        b = f"{BUILD}/tests/{t}"
        st = os.stat(b)
        try:
            p = subprocess.run(["nice", "-n", "19", b], cwd=S, env=env, capture_output=True, text=True, timeout=1200)
            rc, last = p.returncode, (p.stdout.strip().splitlines() or [""])[-1][:160]
        except subprocess.TimeoutExpired:
            rc, last = "timeout", ""
        print(f"{t} rc={rc} built={int(st.st_mtime)} | {last}")


if __name__ == "__main__":
    os.nice(19)
    what = sys.argv[1:] or ["syntax", "suites"]
    same_tree()
    src_m = max(os.stat(os.path.join(dp, f)).st_mtime for dp, _, fs in os.walk(f"{TREE}/components/rules") for f in fs)
    print(f"newest components/rules source mtime {int(src_m)}")
    if "syntax" in what:
        syntax()
    if "suites" in what:
        suites()
