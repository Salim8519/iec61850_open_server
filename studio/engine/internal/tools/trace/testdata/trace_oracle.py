"""CSV traces, manual value rules and utf-8-sig decoding on random synthetic input."""
import os
import sys

sys.path.insert(0, os.environ["STUDIO_ORACLE_LIB"])
from oracle_lib import case, rng, seed  # noqa: E402
from studio_app.domain.csv_trace import read_rows, trace_value, validate_trace  # noqa: E402
from studio_app.domain.signal import validate_value  # noqa: E402

random = rng(seed())
KINDS = ["BOOLEAN", "FLOAT32", "FLOAT64", "INT8", "INT8U", "INT32", "INT64U", "Enum", "Dbpos", "Tcmd", "Quality",
         "Timestamp", "VisString64", "Unicode255", "ObjRef", "Currency", "Octet64", "INT", "VisString"]
CELLS = ["0", "0.5", "1", "1.0", "2", "-1", "1e3", "x", "", " 3 ", "nan", "inf", "86400", "86401", "1_0", "300",
         "4294967296", "1e40", "١", "3.5"]
VALUES = [True, False, 0, 1, -1, 3, 4, 255, 256, 8191, 1.5, float("nan"), 3.5e38, 2 ** 63, -(2 ** 63) - 1, "", "abc",
          "é" * 70, None, [1]]


def csv_text():
    if random.random() < 0.1:
        return random.choice(["", "\n", "a\n1", "a,a\n1,2", " ,b\n1,2", "﻿﻿t,v\n0,1", "t,v\n\"0,1\n",
                              "t,v\n0,1,2", "t,v\n\n\n0,1\n\n", "t,v\r\n0\r,1", "t," + ",".join("c%d" % n for n in range(260))])
    rows = ["t,v,w"] if random.random() < 0.9 else ["v,t"]
    at = 0.0
    for _ in range(random.randint(0, 6)):
        cell = random.choice(CELLS) if random.random() < 0.2 else repr(at)
        rows.append(f"{cell},{random.choice(CELLS)},{random.choice(CELLS)}")
        at += random.choice([0.5, 1, 0, 2])
    return random.choice(["\n", "\r\n"]).join(rows)


for number in range(int(sys.argv[2]) if len(sys.argv) > 2 else 2500):
    text = csv_text()
    case("read_rows", [text], lambda: read_rows(text))
    signal = {"kind": random.choice(KINDS)}
    spec = {"csv_text": text, "time_column": random.choice(["t", "v", "x", None]),
            "value_column": random.choice(["v", "w", "t"])}
    for key, choices in (("interpolation", ["hold", "linear", "cubic", 1]), ("repeat", [True, False, 1]),
                         ("filename", ["a.csv", "f" * 256, 3])):
        if random.random() < 0.6:
            spec[key] = random.choice(choices)
    case("validate_trace", [signal, spec], lambda: validate_trace(signal, spec))
    value = random.choice(VALUES)
    case("validate_value", [signal, value], lambda: validate_value(signal, value))
    raw = bytes(random.choice([0x41, 0x2c, 0x0a, 0xef, 0xbb, 0xbf, 0xc3, 0xa9, 0xe2, 0x82, 0xac, 0xed, 0xa0, 0xf0,
                               0x9f, 0x98, 0x80, 0xf4, 0x90, 0xff, 0xc0, 0x80]) for _ in range(random.randint(0, 8)))
    case("decode", [list(raw)], lambda: raw.decode("utf-8-sig"))
    points = [[0.0, 1.0], [2.0, 5.0], [3.5, -1.0]]
    for repeat, mode in ((False, "hold"), (True, "linear"), (False, "linear")):
        elapsed = random.choice([0, 0.5, 1.999, 2.0, 3.0, 3.5, 10.0, -0.25, 7.25])
        spec2 = {"points": points, "repeat": repeat, "duration": 3.5, "interpolation": mode}
        case("trace_value", [repeat, mode, elapsed], lambda: trace_value(spec2, elapsed))
