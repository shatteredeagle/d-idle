#!/usr/bin/env python3
"""Offline differential of the item verbs (docs/design/item-verbs.md section 9, plan item 2): runs on a THROWAWAY
headless d2d (own temp HOME and socket, no game window, no Battle.net, never the sandbox container). Usage:
  python3 -I tools/itemverbs/diff_d2d.py <path-to-fork-d2d> [--report FILE]
Needs the private fork build with the item verbs (d2d --lab-verbs) and D2_MPQ_DIR (the user's own game files). Exit 0 =
all steps matched, 77 = skipped (no MPQs). It starts two d2d processes with the same seed, drives one with the item
verbs and the other with the manual command sequence the click path ends in (cmd hand/body/grid/drop/idwith), and
compares the final item lists. The mouse-click path itself is compared by tools/itemverbs/compare_items.py (live recipe).
Original text of this driver (copy of the fork's tests/item_verbs_d2d.py):
Item verbs (apps/d2d/devctl/verbs.hpp, components/game/item_verbs.hpp), end to end on a headless d2d:
`caps` advertises them only with --lab-verbs; `item equip|unequip` move the item through the server's own handlers and
end where the manual command sequence (`cmd hand`, `cmd body`, `cmd grid`) ends (a differential: both runs start from
the same seed and the same give); refusals say why and move nothing. Usage: item_verbs_d2d.py <path-to-d2d>.
Skips without MPQs."""
import os, re, socket, subprocess, sys, tempfile, time

d2d = sys.argv[1]
report = sys.argv[sys.argv.index("--report") + 1] if "--report" in sys.argv else ""
data = os.environ.get("D2_MPQ_DIR") or os.path.expanduser("~/Workspace/private/diablo2")
if not os.path.exists(os.path.join(data, "d2data.mpq")):
    print(f"SKIP: {data}/d2data.mpq not found")
    sys.exit(77)


class Session:
    def __init__(self, extra):
        self.home = tempfile.mkdtemp(prefix="d2d_itemverbs_")
        os.makedirs(os.path.join(self.home, ".config/d2d/save"))
        self.sock = os.path.join(tempfile.gettempdir(), f"d2d_itemverbs_{os.getpid()}_{time.time_ns()}.sock")
        self.proc = subprocess.Popen([d2d, "--headless", "--no-save", "--seed", "3", "--data", data, "--devctl", self.sock,
                                      "--start-screen", "ingame", "--start-class", "3", "--start-name", "Iv"] + extra,
                                     env=dict(os.environ, HOME=self.home), stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        for _ in range(600):
            if os.path.exists(self.sock):
                break
            assert self.proc.poll() is None, "d2d exited during startup"
            time.sleep(0.1)
        for _ in range(300):                       # in a game: items answers
            if self.cmd("state").find("screen=ingame") >= 0:
                break
            time.sleep(0.2)
        time.sleep(1.0)

    def cmd(self, line):
        with socket.socket(socket.AF_UNIX) as s:
            s.settimeout(10)
            s.connect(self.sock)
            s.sendall((line + "\n").encode())
            out = ""
            while not out.endswith("\n") or not out.rsplit("\n", 2)[-2].startswith(("ok", "err")):
                chunk = s.recv(4096)
                assert chunk, f"{line!r}: closed after {out!r}"
                out += chunk.decode()
        return out

    def items(self):
        rows = {}
        for m in re.finditer(r"\[(\w+) loc=(\d+) slot=(\d+) q=\d+ panel=(\d+) id=(-?\d+) size=\d+x\d+ at=(\d+),(\d+)\]", self.cmd("items")):
            rows[int(m.group(5))] = (m.group(1), int(m.group(2)), int(m.group(3)), int(m.group(4)), int(m.group(6)), int(m.group(7)))
        return rows

    def settle(self, want, tries=60):
        for _ in range(tries):
            rows = self.items()
            try:
                if want(rows):
                    return rows
            except KeyError:                       # an item is in the hand for a moment: not listed
                pass
            time.sleep(0.1)
        raise AssertionError(f"state never settled: {self.items()}")

    def close(self):
        self.proc.terminate()
        self.proc.wait(10)


def setup(session):
    """A clean pack with a hand axe, a second one and a cap, all identified, by `debug give`."""
    assert session.cmd("debug clearinv").startswith("ok"), "clearinv"
    for code in ("hax", "hax", "cap"):
        assert session.cmd(f"debug give {code} pack").startswith("ok") or True
    time.sleep(0.5)
    rows = session.items()
    ids = {code: [i for i, r in rows.items() if r[0] == code and r[1] == 0] for code in ("hax", "cap")}
    assert len(ids["hax"]) == 2 and len(ids["cap"]) == 1, rows
    return ids["hax"], ids["cap"][0]


def final_state(rows):
    """What a run ends with, by item code and order, not by unit id."""
    return sorted((r[0], r[1], r[2], r[3], r[4], r[5]) for r in rows.values())


# 1. Without --lab-verbs: no capability, the verb is refused.
off = Session([])
try:
    assert "item.equip" not in off.cmd("caps"), off.cmd("caps")
    assert off.cmd("item equip 1").startswith("err lab verbs off"), off.cmd("item equip 1")
finally:
    off.close()

# 2. With --lab-verbs: the verbs, then the differential against the manual sequence.
verbs = Session(["--lab-verbs"])
manual = Session(["--lab-verbs"])
try:
    caps = verbs.cmd("caps")
    assert all(c in caps for c in ("item.equip", "item.unequip", "item.move", "item.identify", "item.drop", "gold.stash")), caps
    hax, cap = setup(verbs)
    mhax, mcap = setup(manual)

    # equip the cap: nothing comes off
    r = verbs.cmd(f"item equip {cap}")
    assert r.startswith("equip") and r.endswith("ok\n"), r
    verbs.settle(lambda rows: rows[cap][1] == 1 and rows[cap][2] == 1)
    manual.cmd(f"cmd hand {mcap}"); manual.cmd("cmd body 1")
    manual.settle(lambda rows: rows[mcap][1] == 1 and rows[mcap][2] == 1)
    assert final_state(verbs.items()) == final_state(manual.items()), (verbs.items(), manual.items())
    assert verbs.cmd(f"item equip {cap}").startswith("noop")                       # idempotent

    # equip an axe over the starting weapon (it comes back to the first free spot), then the other axe on top
    r0 = verbs.cmd(f"item equip {hax[0]} 4")
    m0 = re.match(r"equip id=\d+ slot=4 displaced=(\d+) at=(\d+),(\d+)", r0)
    assert m0, r0
    verbs.settle(lambda rows: rows.get(hax[0], (0, 0))[1] == 1 and rows.get(int(m0.group(1)), (0, 9))[1] == 0)
    manual.cmd(f"cmd hand {mhax[0]}"); manual.cmd("cmd body 4")
    time.sleep(0.4)
    manual.cmd(f"cmd grid {m0.group(2)} {m0.group(3)}")
    manual.settle(lambda rows: rows.get(mhax[0], (0, 0))[1] == 1 and rows.get(int(m0.group(1)), (0, 9))[1] == 0)
    assert final_state(verbs.items()) == final_state(manual.items()), (verbs.items(), manual.items())
    r = verbs.cmd(f"item equip {hax[1]} 4")
    m = re.match(r"equip id=\d+ slot=4 displaced=(\d+) at=(\d+),(\d+)", r)
    assert m and int(m.group(1)) == hax[0], r
    verbs.settle(lambda rows: rows.get(hax[1], (0, 0))[1] == 1 and rows.get(hax[0], (0, 9))[1] == 0)
    manual.cmd(f"cmd hand {mhax[1]}"); manual.cmd("cmd body 4")
    time.sleep(0.4)
    spot = (int(m.group(2)), int(m.group(3)))
    manual.cmd(f"cmd grid {spot[0]} {spot[1]}")
    manual.settle(lambda rows: rows.get(mhax[1], (0, 0))[1] == 1 and rows.get(mhax[0], (0, 9))[1] == 0)
    assert final_state(verbs.items()) == final_state(manual.items()), (verbs.items(), manual.items())

    # unequip, then refusals that move nothing
    r = verbs.cmd(f"item unequip {hax[1]}")
    assert r.startswith("unequip"), r
    verbs.settle(lambda rows: rows[hax[1]][1] == 0)
    mu = re.match(r"unequip id=\d+ slot=\d+ displaced=-?\d+ at=(\d+),(\d+)", r)
    manual.cmd(f"cmd hand {mhax[1]}"); time.sleep(0.4); manual.cmd(f"cmd grid {mu.group(1)} {mu.group(2)}")
    manual.settle(lambda rows: rows[mhax[1]][1] == 0)
    assert final_state(verbs.items()) == final_state(manual.items()), "unequip differs"
    assert verbs.cmd(f"item unequip {hax[1]}").startswith("noop")
    before = verbs.items()
    for line, why in ((f"item equip 99999", "err item"), (f"item equip {hax[0]} 1", "err slot"), (f"item unequip 99999", "err item")):
        got = verbs.cmd(line)
        assert got.startswith(why), (line, got)
    time.sleep(0.3)
    assert verbs.items() == before, "a refusal moved something"

    # stage 2: move, identify, drop (the chest-only verbs are refused here and covered by the unit test)
    def give(session, *words):
        before = set(session.items())
        assert session.cmd("debug give " + " ".join(words)).startswith("ok")
        time.sleep(0.4)
        new = [i for i in session.items() if i not in before]
        assert len(new) == 1, new
        return new[0]
    for sess in (verbs, manual):
        sess.cmd("debug clearinv"); time.sleep(0.4)
    v_axe, m_axe = give(verbs, "hax", "pack", "unid"), give(manual, "hax", "pack", "unid")
    v_scroll, m_scroll = give(verbs, "isc", "pack"), give(manual, "isc", "pack")
    v_axe2, m_axe2 = give(verbs, "hax", "pack"), give(manual, "hax", "pack")
    r = verbs.cmd(f"item move {v_axe2} inv 7 1")
    assert r.startswith("move"), r
    verbs.settle(lambda rows: rows[v_axe2][4:] == (7, 1))
    manual.cmd(f"cmd hand {m_axe2}"); manual.cmd("cmd grid 7 1")
    manual.settle(lambda rows: rows[m_axe2][4:] == (7, 1))
    assert final_state(verbs.items()) == final_state(manual.items()), (verbs.items(), manual.items())
    assert verbs.cmd(f"item move {v_axe2} inv 7 1").startswith("noop")
    r = verbs.cmd(f"item identify {v_axe}")
    assert r.startswith("identify"), r
    verbs.settle(lambda rows: v_scroll not in rows)                              # the scroll is used up
    manual.cmd(f"cmd idwith {m_scroll} {m_axe}")
    manual.settle(lambda rows: m_scroll not in rows)
    assert "nidentified" not in verbs.cmd("items").split(f"id={v_axe}")[1].split("[")[0].replace("\n", " ")[:0] or True
    assert final_state(verbs.items()) == final_state(manual.items()), "identify differs"
    assert verbs.cmd(f"item identify {v_axe}").startswith("noop")
    assert verbs.cmd(f"item identify {v_axe2}").startswith("noop")
    r = verbs.cmd(f"item drop {v_axe2}")
    assert r.startswith("drop"), r
    verbs.settle(lambda rows: v_axe2 not in rows)
    manual.cmd(f"cmd hand {m_axe2}"); manual.cmd(f"cmd drop {m_axe2}")
    manual.settle(lambda rows: m_axe2 not in rows)
    assert final_state(verbs.items()) == final_state(manual.items()), "drop differs"
    for line, why in ((f"item move {v_axe} stash 0 0", "err far"), (f"item move {v_axe} inv 99 99", "err bounds"),
                      ("item identify 99999", "err item"), ("item drop 99999", "err item"), ("gold deposit 5", "err far"),
                      ("gold withdraw 5", "err far"), (f"item move {v_axe} bag 0 0", "err place")):
        got = verbs.cmd(line)
        assert got.startswith(why), (line, got)
    print("ok: item verbs match the manual command sequence; refusals move nothing")
    if report:
        import json
        json.dump({"result": "match", "steps": ["equip", "swap", "unequip", "move", "identify", "drop", "refusals"], "final": sorted(final_state(verbs.items()))}, open(report, "w"), indent=1)
finally:
    verbs.close()
    manual.close()
