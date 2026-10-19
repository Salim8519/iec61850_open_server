"""Write CPython reference vectors for CSV trace sources (validate_trace and trace_value).

Usage: studio/.venv/Scripts/python.exe csv_vectors.py OUT.json
Each row: [kind, spec, result-or-error, [[elapsed, value], ...]].
"""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[6]
sys.path.insert(0, str(ROOT / "studio" / "backend"))

from studio_app.domain.csv_trace import trace_value, validate_trace  # noqa: E402

TEXTS = [
    "t,v\n0,1\n1,2\n2,4\n", "﻿t,v\r\n0,1.5\r\n0.5,-2\r\n", "t,v\n0,1\n\n2,3\n", "t,v,w\n0,1,2\n1,2\n",
    "t,v\n", "", "t\n0\n", "t,t\n0,1\n", "t, \n0,1\n", "t,v\n1,2\n2,3\n", "t,v\n0,1\n0,2\n",
    "t,v\n0,abc\n", "t,v\n0,1e400\n", "t,v\n0,\" 7 \"\n1,1_000\n", "t,v\n0,\"1\"x\n", "t,v\n0,1\r2,3\n",
    "\"t\",\"v\"\n\"0\",\"5\"\n\"1\",\"6\"\n", "t,v\n0,nan\n", "t,v\n0,inf\n", "t,v\n0,3\n86401,4\n",
    "t,v\n0,2.5\n1,3\n", "t,v\n0,7\n", "a,b\n0,1\n", "t,v\n0,1\n1,\"multi\nline\"\n",
]


def main():
    rows = []
    for index, text in enumerate(TEXTS):
        for kind, interpolation, repeat in (("FLOAT32", "linear", False), ("INT32", "hold", True),
                                            ("FLOAT64", "hold", index % 2 == 0)):
            spec = {"mode": "csv", "csv_text": text, "time_column": "t", "value_column": "v",
                    "interpolation": interpolation, "repeat": repeat}
            try:
                result = validate_trace({"kind": kind, "fc": "MX"}, spec)
                samples = [[elapsed, trace_value(result, elapsed)] for elapsed in (0, 0.25, 0.5, 1, 1.75, 2, 3.5, 10)]
                rows.append([kind, spec, json.dumps(result), samples])
            except ValueError as error:
                rows.append([kind, spec, "ValueError: " + str(error), []])
    Path(sys.argv[1]).write_text(json.dumps(rows), encoding="utf-8")


if __name__ == "__main__":
    main()
