# Disassemble game.exe at ADDR [COUNT] (verifyresistq). Wrapper over upstream disasm.py.
# tools/emu/run-on-worker.sh vq_dis.py 5a1370 400 > $D2_SCRATCH/x.txt
import os, runpy, sys
src = os.path.expanduser("~/d2d-lab/src/tools/emu")
sys.path.insert(0, src)
sys.argv = ["disasm.py", "--exe", "/tmp/casttrace/game.exe"] + sys.argv[1:]
runpy.run_path(os.path.join(src, "disasm.py"), run_name="__main__")
