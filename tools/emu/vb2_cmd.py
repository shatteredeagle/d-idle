# SPDX-License-Identifier: GPL-3.0-or-later
"""verifyb2: print the include, define and std flags of FILE's entry in a compile_commands.json.
    python3 vb2_cmd.py compile_commands.json FILE"""
import json
import shlex
import sys

for e in json.load(open(sys.argv[1])):
    if e["file"] == sys.argv[2]:
        args = shlex.split(e["command"]) if "command" in e else e["arguments"]
        keep, it = [], iter(args[1:])
        for a in it:
            if a in ("-I", "-isystem", "-include", "-D"):
                keep += [a, next(it)]
            elif a.startswith(("-I", "-D", "-std=", "-isystem")):
                keep.append(a)
        print(" ".join(shlex.quote(k) for k in keep))
        sys.exit(0)
sys.exit(1)
