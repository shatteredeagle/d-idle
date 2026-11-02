# SPDX-License-Identifier: GPL-3.0-or-later
"""Scripted determinism scenarios on a headless d2d with the lab patch set (card dethash2).

    D2D_BIN=bin/d2d.NAME D2D_DATA=GAMEDIR D2D_SOCK=... D2D_SLOT_DIR=... \
        python3 det_scenarios_lab.py SCENARIO TAG [OUTDIR]

One fresh d2d process per run (seed 3, lockstep, no video): a scripted list of `step` blocks, a snapshot
after each block. Needs the lab verbs `debug spawn`, `debug merc`, `debug pets`, `debug players` and the
read-only `debug monstate` (hp, chill and stun deadlines, mode, position of every live monster).

A snapshot is five read-only replies: `monsters`, `debug monstate`, `debug players` (the player's cell),
`dump merc`, `debug pets`. Two hashes per run, md5 of the lines, first 12 hex:

    mon    the `monsters` replies only (the hash of the first det run, comparable with it)
    state  every snapshot line: monsters, chill/stun, the player's position, the merc, the pets

Scenarios (V = run, see docs/research/det-hashes-2026-11-01b.md):
    passive fight cold static cold2      the five of the first run (clvl 8 / 30, Ice Bolt 39, Static Field 42, Frozen Orb 64)
    hit_plain hit_champion hit_unique hit_boss hit_immune hit_minion
                                         Ice Bolt 39, Ice Blast 45, Frost Nova 44, Glacial Spike 55, one after the
                                         other on a freshly spawned target of the class, cast point blank; the log
                                         has the hp, chill and stun of the target per block
    hit_phys                             druid: Twister, Shock Wave, Tornado (physical missile skills, card playerroll),
                                         one after the other on a freshly spawned plain zombie, cast point blank
    merc                                 hired merc (type 0) and three zombies
    pet                                  druid with a Spirit Wolf and a Raven and two zombies
    follow                               druid with a wolf and a merc, the player walks about, zombies chase
"""

import hashlib
import os
import re
import socket
import subprocess
import sys
import time

SOCK = os.environ.get("D2D_SOCK", "d2d.sock")
OLD = {"passive", "fight", "cold", "static", "cold2"}
OLD_SKILL = {"cold": 39, "static": 42, "cold2": 64}  # Ice Bolt, Static Field, Frozen Orb
HIT_SKILLS = (39, 45, 44, 55)  # Ice Bolt, Ice Blast, Frost Nova, Glacial Spike
HIT_SKILLS_PHYS = (240, 243, 245)  # Twister, Shock Wave, Tornado: Skills.txt MinDam / MaxDam, no element
HIT_CLASS = {
    "plain": "zombie4 1",  # Drowned Carcass, a plain zombie with 147 life: one point of a cold skill does not kill it
    "champion": "zombie4 1 near champion",
    "unique": "zombie4 1 near unique",
    "boss": "bloodraven 1",  # a MonStats boss-column monster, not a unique
    "phys": "zombie4 1",  # the plain target again, hit by the druid's physical missiles
    "immune": "zombie1 1",  # in Hell (MonStats ColdResist(H) 100) the plain zombie is cold immune: a cold blow deals 0 damage
    "minion": None,  # Hell Blood Moor has a unique pack: its minions (boss column "minion") are the targets, no spawn verb makes one
}
HELL = {"immune", "minion"}
CLASS_OF = {"passive": "1", "fight": "1", "merc": "4", "pet": "5", "follow": "5"}


class Ctl:
    """The devctl socket: one command per line, a reply ends at ok / ok ... / err ... / a blank line."""

    def __init__(self, path=SOCK):
        self.sock = socket.socket(socket.AF_UNIX)
        self.sock.settimeout(900)
        self.sock.connect(path)
        self.file = self.sock.makefile("rwb", buffering=0)

    def cmd(self, text):
        self.file.write((text + "\n").encode())
        out = []
        while True:
            line = self.file.readline().decode().rstrip("\n")
            out.append(line)
            if line == "" or line == "ok" or line.startswith(("ok ", "err")):
                return out


def start(cls, logfile):
    """Launch d2d (headless, lockstep, seed 3) and wait for the devctl socket."""
    try:
        os.unlink(SOCK)
    except OSError:
        pass
    slot = os.environ.get("D2D_SLOT_DIR", ".")
    env = dict(os.environ, XDG_CONFIG_HOME=slot + "/config", XDG_DATA_HOME=slot + "/data")
    os.makedirs(slot, exist_ok=True)
    argv = ["nice", "-n10", os.environ["D2D_BIN"], "--data", os.environ["D2D_DATA"], "--headless", "--devctl", SOCK,
            "--no-video", "--toggle", "autoloot=off,autoparty=off", "--lockstep", "--seed", "3",
            "--start-screen", "ingame", "--start-class", cls, "--start-name", "Det"]
    with open(logfile, "w") as out:
        proc = subprocess.Popen(argv, stdout=out, stderr=subprocess.STDOUT, start_new_session=True, env=env)
    for _ in range(480):
        try:
            ctl = Ctl()
            if ctl.cmd("ping")[-1] == "ok":
                return ctl, proc
        except OSError:
            time.sleep(0.5)
    os.killpg(proc.pid, 15)
    raise SystemExit("d2d did not start")


class Run:
    def __init__(self, ctl):
        self.ctl = ctl
        self.log = []

    def do(self, text):
        reply = self.ctl.cmd(text)
        self.log.append("CMD " + text + " -> " + "|".join(reply))
        return reply

    def snap(self, label):
        """One snapshot: five read-only replies, each line tagged for the compare tool."""
        self.log.append("SNAP " + label)
        for tag, verb in (("M", "monsters"), ("S", "debug monstate"), ("P", "debug players"), ("G", "dump merc"), ("T", "debug pets")):
            reply = self.ctl.cmd(verb)
            if tag == "M":
                self.log.append("MON " + "|".join(reply))  # the first run's hash input, unchanged
            for line in reply:
                if line not in ("ok", ""):
                    self.log.append(tag + "|" + line)

    def rows(self, boss=None):
        """The live monsters as (id, x, y) from `monsters`, newest id last; only boss column `boss` when given."""
        out = []
        for line in self.ctl.cmd("monsters"):
            field = line.split("\t")
            if len(field) > 11 and field[6] not in ("DT", "DD") and boss in (None, field[7]):
                out.append((int(field[11][1:]), float(field[1]), float(field[2])))
        return sorted(out)


def setup(run, level, power=True, life=900):
    if power:
        for stat, val in ((7, life), (6, life), (9, 300), (8, 300)):  # life, max life, mana, max mana
            run.do(f"debug stat {stat} {val * 256}")
    run.do(f"debug stat 12 {level}")
    if life > 900:
        run.do("debug difficulty 2")  # Hell: a new level's monsters
    run.do("debug goto 2")  # Blood Moor
    time.sleep(3)
    run.do("step 10")


def old_scenario(run, mode):
    steps = 500 if mode == "passive" else 250
    setup(run, 8 if mode == "passive" else 30, power=mode in OLD_SKILL)
    if mode != "passive":
        run.do("debug warp 36.0 36.0")
    if mode in OLD_SKILL:
        skill = OLD_SKILL[mode]
        run.do(f"debug points {skill} 20")
        run.do("step 3")
        run.do(f"debug skill left {skill}")
    run.do("click 400 500")
    run.do("step 5")
    for block in range(8):
        if mode in OLD_SKILL:
            run.do(f"debug stat 8 {300 * 256}")
            run.do(f"debug stat 6 {900 * 256}")
            rows = run.ctl.cmd("monsters")
            first = [line.split("\t") for line in rows if "\t" in line]
            if first:
                unit = [tok for tok in first[0] if tok.startswith("#")]
                uid = int(unit[0][1:]) if unit else -1
                for _ in range(2):
                    run.do(f"cmd skill {OLD_SKILL[mode]} {first[0][1]} {first[0][2]} {uid} 1")
                    run.do("step 12")
        run.do(f"step {steps}")
        run.snap(f"block {block}")


def hit_scenario(run, cls):
    skills = HIT_SKILLS_PHYS if cls == "phys" else HIT_SKILLS
    setup(run, 30, life=9000 if cls in HELL else 900)
    if cls != "minion":
        run.do("debug kill")
    run.do("step 5")
    for skill in skills:
        run.do(f"debug points {skill} 1")
    run.do("step 3")
    used = set()
    for skill in skills:
        if cls != "minion":
            run.do("debug kill")
        run.do("step 3")
        run.do(f"debug stat 8 {300 * 256}")
        if cls == "minion":
            rows = [row for row in run.rows("minion") if row[0] not in used]
            used.update(row[0] for row in rows[-1:])
        else:
            run.do("debug warp 44.2 24.2")
            run.do("step 2")
            run.do("debug spawn " + HIT_CLASS[cls])
            run.do("step 1")
            rows = run.rows()
        if not rows:
            run.log.append(f"NOTARGET skill {skill}")
            continue
        uid, tx, ty = rows[-1]
        run.do(f"debug warp {tx:.2f} {ty + 1.6:.2f}")  # point blank: the target 1.6 cells north of the player
        run.do(f"debug skill left {skill}")
        run.do("step 2")
        run.snap(f"{cls} skill {skill} #{uid} before")
        run.do(f"cmd skill {skill} {tx:.2f} {ty:.2f} {uid} 1")
        for block in range(8):
            run.do("step 6")
            run.snap(f"{cls} skill {skill} #{uid} block {block}")


def party_scenario(run, mode):
    setup(run, 30)
    run.do("debug kill")
    run.do("step 5")
    if mode in ("pet", "follow"):
        for skill in (227, 221):  # Summon Spirit Wolf, Raven
            run.do(f"debug points {skill} 5")
        run.do("step 3")
        run.do("cmd skill 227 44.2 22.2 0 1")
        run.do("step 30")
        if mode == "pet":
            run.do("cmd skill 221 44.2 22.2 0 1")
            run.do("step 30")
    if mode in ("merc", "follow"):
        run.do("debug merc hire 0")
        run.do("step 20")
    run.do(f"debug spawn zombie1 {3 if mode == 'merc' else 2}")
    run.do("click 400 300")
    run.snap("start")
    goals = [(38, 30), (52, 22), (40, 18), (46, 30)]
    for block in range(8):
        if mode == "follow":
            run.do("cmd move {} {}".format(*goals[block % len(goals)]))
        run.do("step 100")
        run.snap(f"block {block}")


def hashes(log):
    def norm(line):  # wall-clock driven counters are not part of the state
        return re.sub(r"(frame|music|music_old|sfx|quake)=\S*", "", line)

    mon = [line for line in log if line.startswith("MON ")]
    full = [norm(line) for line in log if line[:2] in ("M|", "S|", "P|", "G|", "T|", "SN", "NO")]
    return (hashlib.md5("\n".join(mon).encode()).hexdigest()[:12], hashlib.md5("\n".join(full).encode()).hexdigest()[:12], len(mon))


def main(argv):
    mode, tag = argv[1], argv[2]
    outdir = argv[3] if len(argv) > 3 else "out"
    os.makedirs(outdir, exist_ok=True)
    if mode == "hit_phys":
        cls = "5"  # druid
    elif mode in OLD_SKILL or mode.startswith("hit_"):
        cls = "4"
    else:
        cls = CLASS_OF[mode]
    ctl, proc = start(cls, os.path.join(outdir, f"d2d_{mode}_{tag}.log"))
    run = Run(ctl)
    try:
        time.sleep(2)
        if mode in OLD:
            old_scenario(run, mode)
        elif mode.startswith("hit_"):
            hit_scenario(run, mode[4:])
        else:
            party_scenario(run, mode)
    finally:
        time.sleep(1)
        for sig in (15, 9):
            try:
                os.killpg(proc.pid, sig)
                proc.wait(30)
                break
            except (OSError, subprocess.TimeoutExpired):
                pass
    mon, state, count = hashes(run.log)
    with open(os.path.join(outdir, f"det8_{mode}_{tag}.txt"), "w") as out:
        out.write("\n".join(run.log) + "\n")
    print(f"{mode} {tag} mon={mon} state={state} monlines={count}")


if __name__ == "__main__":
    main(sys.argv)
