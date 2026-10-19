"""validate_sources vectors: generator parameters and CSV traces (Python csv module rules)."""

B = "1" * 32
P = "BayControllerCTRL/"
ANIN, POS = P + "GGIO1.AnIn1.mag.f", P + "XCBR1.Pos.stVal"
ALL = ["BayController", "Bay2", "FEED1"]
CSV_TEXTS = [
    "t,v\n0,1\n1,2\n", "t,v\r\n0,1\r\n1,2\r\n", "t,v\r0,1\r", "t,v\n0,1\r1,2\n", "﻿﻿t,v\n0,1\n",
    "t,v\n\n0,1\n\n2,3", "t,v\n0,1,2\n", "t,v\n0\n", "t\n0\n", "", "\n", "t,v\n", "t,t\n0,1\n", "t, \n0,1\n",
    '"t","v"\n"0","1"\n', '"t,x",v\n0,1\n', '"a""b",v\n0,1\n', '"t"x,v\n0,1\n', '"t,v\n0,1\n',
    't,"v\nw"\n0,1\n', "t,v\n0,abc\n", "t,v\nabc,1\n", "t,v\n0,inf\n", "t,v\n0,nan\n", "t,v\n0,1e39\n",
    "t,v\n0,1_0\n", "t,v\n 0 , 1 \n", "t,v\n-1,1\n", "t,v\n0,1\n0,2\n", "t,v\n1,1\n2,2\n", "t,v\n0,1\n86401,2\n",
    "t,v\n0,1\n86400,2\n", "t,v\n0,1\ninf,2\n", "t,v,w\n0,1,2\n1,2,3\n", "a,t,v\nx,0,5\ny,0.5,6\n",
    "t,v\n0," + "9" * 400 + "\n", "t,v\n-0.0,1\n", "t,v\n0,١\n", "t,v\r\n0,1\r\n\r\n", "t,v\n0,1\n\"", "t,v\n0,\"1\"x\n",
    "t,v\n" + "0," + "x" * 131073 + "\n", "t,v\n0," + "1" * 131072 + "\n",
]


def source(spec, ref=ANIN):
    return {"ied_id": B, "ref": ref, "spec": spec}


def cases(fixtures):
    out = []

    def add(name, sources, ieds=ALL):
        out.append({"name": name, "fn": "validate_sources", "args": {"sources": sources}, "ieds": ieds})

    for index, value in enumerate([None, {}, "x", [5], [None], [{}], [[1]]]):
        add(f"shape {index}", value)
    add("ok constant", [source({"mode": "constant"})])
    add("ok all params", [source({"mode": "sine", "offset": 1, "amplitude": "2.5", "period": True, "seed": "7"})])
    add("unknown signal", [source({"mode": "ramp"}, ref="zz")])
    add("duplicate", [source({"mode": "ramp"}), source({"mode": "sine"})])
    add("list ied", [{"ied_id": [B], "ref": ANIN, "spec": {"mode": "ramp"}}])
    add("dict ref", [{"ied_id": B, "ref": {"a": 1}, "spec": {"mode": "ramp"}}])
    add("int ied", [{"ied_id": 5, "ref": ANIN, "spec": {"mode": "ramp"}}])
    add("non numeric", [source({"mode": "ramp"}, ref=POS)])
    add("ST int", [source({"mode": "ramp"}, ref=P + "LLN0.Health.stVal")])
    for index, spec in enumerate([None, [], {}, {"mode": "x"}, {"mode": ["x"]}, {"mode": None}, {"mode": "manual"},
                                  {"mode": "step", "offset": "abc"}, {"mode": "step", "offset": None},
                                  {"mode": "step", "offset": [1]}, {"mode": "step", "amplitude": {}},
                                  {"mode": "step", "offset": 10 ** 400}, {"mode": "step", "offset": "inf"},
                                  {"mode": "step", "amplitude": float("nan")}, {"mode": "step", "period": 0.09},
                                  {"mode": "step", "period": 0.1}, {"mode": "step", "period": "1_0"},
                                  {"mode": "step", "seed": 1.9}, {"mode": "step", "seed": -1.9},
                                  {"mode": "step", "seed": float("inf")}, {"mode": "step", "seed": float("nan")},
                                  {"mode": "step", "seed": "1.5"}, {"mode": "step", "seed": " 12 "},
                                  {"mode": "step", "seed": None}, {"mode": "step", "seed": 10 ** 30},
                                  {"mode": "step", "seed": 1e20}, {"mode": "step", "seed": True},
                                  {"mode": "noise", "offset": 1, "amplitude": 2, "period": 3, "seed": 4, "x": 5}]):
        add(f"spec {index}", [source(spec)])
    for index, text in enumerate(CSV_TEXTS):
        add(f"csv {index}", [source({"mode": "csv", "csv_text": text, "time_column": "t", "value_column": "v"})])
    base = {"mode": "csv", "csv_text": "t,v\n0,1\n2,3\n", "time_column": "t", "value_column": "v"}
    variants = [{"csv_text": None}, {"csv_text": 5}, {"time_column": "v"}, {"time_column": "x"}, {"value_column": 5},
                {"time_column": None}, {"interpolation": "linear"}, {"interpolation": "cubic"},
                {"interpolation": ["hold"]}, {"repeat": True}, {"repeat": 1}, {"filename": "a.csv"},
                {"filename": "x" * 255}, {"filename": "x" * 256}, {"filename": 5},
                {"csv_text": "t,v\n0,1\n", "repeat": True}, {"csv_text": "t,v\n0,1\n", "repeat": False}]
    for index, change in enumerate(variants):
        add(f"csv variant {index}", [source({**base, **change})])
    feed = fixtures["FEED1"]
    for signal in [s for s in feed["signals"] if s["fc"] == "MX"][:6]:
        spec = {**base, "csv_text": "t,v\n0,1\n1,2.5\n2,-3\n3,1e10\n"}
        add(f"FEED1 csv {signal['kind']} {signal['ref']}", [{"ied_id": "3" * 32, "ref": signal["ref"], "spec": spec}])
        add(f"FEED1 linear {signal['ref']}", [{"ied_id": "3" * 32, "ref": signal["ref"],
                                               "spec": {**spec, "interpolation": "linear"}}])
    int_ref = "FEED1LD1/TCTR1.AmpSv.instMag.i"
    float_ref = next(s["ref"] for s in feed["signals"] if s["fc"] == "MX" and s["kind"] == "FLOAT32")
    for index, text in enumerate(["t,v\n0,1\n1,2.0\n2,-5\n3,1_0\n", "t,v\n0,2147483647\n1,-2147483648\n",
                                  "t,v\n0,2147483648\n", "t,v\n0,1e20\n", "t,v\n0,inf\n", "t,v\n0,nan\n",
                                  "t,v\n0,-0.0\n1,3e9\n", "t,v\n0, 7 \n"]):
        for ref in (int_ref, float_ref):
            add(f"FEED1 numbers {index} {ref}", [{"ied_id": "3" * 32, "ref": ref,
                                                   "spec": {**base, "csv_text": text, "repeat": index == 0}}])
    add("FEED1 int generator", [{"ied_id": "3" * 32, "ref": int_ref, "spec": {"mode": "ramp", "seed": 3}}])
    int_spec = {**base, "csv_text": "t,v\n0,1\n1,2.0\n2,2.5\n"}
    add("bay2 analog csv", [{"ied_id": "2" * 32, "ref": "Bay2CTRL/GGIO1.AnIn2.mag.f", "spec": int_spec}])
    for spec in ({"mode": "ramp", "offset": 1.0}, {**base, "points": [[0, 1]], "row_count": 1}, {}):
        out.append({"name": f"summary {len(out)}", "fn": "source_summary", "args": {"spec": spec}, "ieds": []})
    return out
