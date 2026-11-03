#!/usr/bin/env python3
"""verifydmg4: CI parity and suites for DMGROLL r4 9ea2afe9, read-only on the author's tree and build (w3_ci.py adapted).
1. g++ and clang++ -fsyntax-only with the CI flags plus -Wshadow -Wrange-loop-construct -Wconversion on the changed TUs,
   from the build's compile_commands.json; diagnostics on lines the commit added are counted (added lines from r4.diff);
2. the suites of that build, run in $D2_SCRATCH: rc, last line, and whether the binary is newer than every source of
   components/ and tests/ (the tree itself equals my git archive of 9ea2afe9: v4_build.sh).

    run-on-worker.sh --ttl 3000 v4_ci.py [syntax] [suites]
"""
import json
import os
import re
import subprocess
import sys

HOME = os.path.expanduser("~")
TREE = f"{HOME}/d2d-lab/wt-mdmg-r4"
BUILD = f"{HOME}/d2d-lab/build-mdr4"
S = os.environ["D2_SCRATCH"]
TUS = ["components/game/ai.cpp", "components/game/fight/pets.cpp", "components/game/fight/states.cpp", "components/game/fight/tick.cpp",
       "tests/game/test_game.cpp", "tests/test_combat.cpp", "components/game/fight/missiles.cpp", "components/game/fight/strike.cpp"]
FLAGS = ["-Wall", "-Wextra", "-Wpedantic", "-Wno-missing-field-initializers", "-Werror", "-Wshadow", "-Wrange-loop-construct",
         "-Wconversion", "-fsyntax-only"]
SUITES = ["test_rules", "test_game_rules", "test_monsters", "test_monster_thinks", "test_monster_ai", "test_combat", "test_game",
          "test_skills", "test_sk_ground", "test_monster_missile", "test_missile_blow_poison"]


def added_lines():
    out, f, ln = {}, None, 0
    for line in open(f"{S}/r4.diff"):
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
            args = [a for a in args[1:] if a != "-Werror" and not a.endswith("ccache")]
            drop = set()
            for i, a in enumerate(args):
                if a == "-o":
                    drop |= {i, i + 1}
                if a == "-c":
                    drop.add(i)
            args = [a for i, a in enumerate(args) if i not in drop]
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
    newest = max(os.stat(os.path.join(dp, f)).st_mtime for top in ("components", "tests")
                 for dp, _, fs in os.walk(f"{TREE}/{top}") for f in fs)
    print(f"newest source in components/ and tests/: {int(newest)}")
    env = dict(os.environ, D2_PATCH_INSTALLER=f"{HOME}/Downloads/LODPatch_114d.exe")
    for t in SUITES:
        b = f"{BUILD}/tests/{t}"
        st = os.stat(b)
        try:
            p = subprocess.run(["nice", "-n", "19", b], cwd=S, env=env, capture_output=True, text=True, timeout=1200)
            rc, last = p.returncode, (p.stdout.strip().splitlines() or [""])[-1][:160]
        except subprocess.TimeoutExpired:
            rc, last = "timeout", ""
        print(f"{t} rc={rc} built={int(st.st_mtime)} newer-than-sources={st.st_mtime > newest} | {last}")


if __name__ == "__main__":
    os.nice(19)
    what = sys.argv[1:] or ["syntax", "suites"]
    if "syntax" in what:
        syntax()
    if "suites" in what:
        suites()
