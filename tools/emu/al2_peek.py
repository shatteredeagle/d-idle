#!/usr/bin/env python3
"""alignment2 reading aid: print a part of an upstream emulator module (head N lines or grep PATTERN).
    al2_peek.py MODULE head N      al2_peek.py MODULE grep PATTERN [CONTEXT]"""
import importlib
import inspect
import re
import sys

mod = importlib.import_module(sys.argv[1])
lines = inspect.getsource(mod).splitlines()
if sys.argv[2] == "head":
    print("\n".join(lines[:int(sys.argv[3])]))
else:
    ctx = int(sys.argv[4]) if len(sys.argv) > 4 else 3
    for i, ln in enumerate(lines):
        if re.search(sys.argv[3], ln):
            print(f"--- {i + 1}")
            print("\n".join(lines[max(0, i - ctx):i + ctx + 1]))
