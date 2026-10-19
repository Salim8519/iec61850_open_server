"""Write CPython reference vectors for the Go energisation and switching hints.

Usage: studio/.venv/Scripts/python.exe energize_vectors.py OUT.json
Uses the synthetic test project's diagram plus a second drawing with VT, CT and both infeed
states; every combination of switch positions (open, closed, unknown, bool, missing) gives
json.dumps of {"nodes", "edges", "hints"} exactly as the energization event carries it.
"""
import itertools
import json
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[5]
sys.path.insert(0, str(ROOT / "studio" / "backend"))

from studio_app.domain.energization import Sections, describe  # noqa: E402
from studio_app.domain.interlock_hints import switching_hints  # noqa: E402


def node(key, kind, label=None, **data):
    return {"id": key, "data": {"label": label or key, "kind": kind, **data}}


def wire(key, source, target, sh=None, th=None):
    return {"id": key, "source": source, "target": target, "sourceHandle": sh, "targetHandle": th}


def second():
    nodes = [node("g1", "INFEED"), node("g2", "INFEED", infeed_state="dead"), node("q1", "CB", "Q1"),
             node("d1", "DS", "D1"), node("d2", "DS", "D2"), node("e1", "ES", "E1"), node("ct", "CT"),
             node("vt", "VT"), node("j", "JUNCTION"), node("relay", "RELAY")]
    edges = [wire("a", "g1", "d1", "south", "north"), wire("b", "d1", "ct", "south", None),
             wire("c", "ct", "q1", None, "north"), wire("d", "q1", "j", "south"), wire("e", "j", "vt"),
             wire("f", "j", "e1", None, "west"), wire("g", "g2", "d2"), wire("h", "d2", "j", "east", None),
             wire("i", "relay", "j"), wire("k", "missing", "j")]
    return {"nodes": nodes, "edges": edges}


def main():
    project = json.loads((HERE.parents[1] / "testdata" / "project.json").read_text(encoding="utf-8"))
    choices = [1, 2, 0, None, True, 2.0, "x"]
    rows = []
    diagrams = [project["diagram"], second()]
    for number, diagram in enumerate(diagrams):
        switches = [item["id"] for item in diagram["nodes"] if item["data"]["kind"] in {"CB", "DS", "ES"}]
        for index, combo in enumerate(itertools.product(choices, repeat=len(switches))):
            if index % 7 and len(rows) > 50:  # keep the file small: every 7th combination after the first 50
                continue
            positions = {key: value for key, value in zip(switches, combo) if value != "x"}
            sections = Sections(diagram, positions)
            result = {**describe(sections), "hints": switching_hints(sections)}
            rows.append([number, positions, json.dumps(result)])
    Path(sys.argv[1]).write_text(json.dumps({"diagrams": diagrams, "rows": rows}), encoding="utf-8")


if __name__ == "__main__":
    main()
