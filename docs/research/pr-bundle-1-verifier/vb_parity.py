# SPDX-License-Identifier: GPL-3.0-or-later
"""verifybundle: CI-flag syntax pass over TUs on one build (worker host).
    python3 vb_parity.py BUILD_DIR SRC_DIR OUT_PREFIX TU...
For g++ and clang++: -fsyntax-only with -Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Werror
plus -Wshadow -Wrange-loop-construct -Wconversion (-Wno-error so all diagnostics show); writes OUT_PREFIX.<cc>.log."""
import json, os, re, shlex, subprocess, sys
build, src, out, tus = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4:]
db = json.loads(subprocess.run(["ninja", "-C", build, "-t", "compdb"], capture_output=True, text=True).stdout)
flags = ["-Wall", "-Wextra", "-Wpedantic", "-Wno-missing-field-initializers", "-Wshadow", "-Wrange-loop-construct", "-Wconversion", "-fsyntax-only"]
for cc in ("g++", "clang++"):
    with open(f"{out}.{cc}.log", "w") as log:
        for tu in tus:
            path = os.path.join(src, tu)
            ent = next((e for e in db if os.path.realpath(e["file"]) == os.path.realpath(path)), None)
            if not ent: log.write(f"NOCMD {tu}\n"); continue
            args = shlex.split(ent["command"])
            keep, skip = [], False
            for a in args[1:]:
                if skip: skip = False; continue
                if a in ("-o", "-MF", "-MT", "-MQ"): skip = True; continue
                if a in ("-c", "-MD", "-MMD") or a.startswith("-W") or a.startswith("-fdiagnostics"): continue
                keep.append(a)
            r = subprocess.run(["nice", "-n", "19", cc, *keep, *flags], cwd=ent["directory"], capture_output=True, text=True)
            log.write(f"=== {tu} rc {r.returncode}\n{r.stderr}")
