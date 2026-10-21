"""Golden cases for Go package signalindex: real HTTP answers of GET /api/v1/ieds/<id>/signals.

Each IED is recorded once ("ied <label>", args ["ied", label, ied]); each query records
["query", label, query string, Python exception class or None] and want "<status> <body>".
"args <n>" cases record how Werkzeug parses a raw query string (request.args).
Run from this folder: ..\\..\\..\\..\\..\\.venv\\Scripts\\python.exe gen_signals.py
"""
import copy
import json
from urllib.parse import quote

from common import app_with, close, record

from studio_app.api.signals import list_signals

TEMPLATES = [
    "", "q=XCBR", "q=xcbr", "q=Pos.stVal", "q=pos+stval", "q=%20", "q=+", "q=", "q", "q=LLN0&q=XCBR",
    "q=%E1%BA%9E", "q=%C3%9F", "q=SS", "q=%C4%B0", "q=i%CC%87", "q=stra%C3%9Fe", "q=%E2%82A", "q=%zz",
    "q=XCBR;fc=ST", "%71=XCBR", "q=nothing_matches_here", "q=%CE%A9", "q=%EF%AC%81", "q=DI%C5%9E",
    "fc=ST", "fc=", "fc=MX", "fc=st", "fc=CF&writable=true", "fc=ST&fc=MX",
    "kind=BOOLEAN", "kind=BOOLEAN,Dbpos", "kind=,BOOLEAN,,", "kind=,", "kind=", "kind=Quality&kind=BOOLEAN",
    "writable=true", "writable=True", "writable=1",
    "report={report}", "report=nope", "report=", "report={report}&in_dataset=true", "report=nope&offset=x",
    "report=nope&in_dataset=true", "report={report}&limit=x", "report=dup", "report=nodata",
    "ref={ref}", "ref=nope", "ref={ref}&q=zzz", "ref={ref}&fc=MX", "ref=", "ref={dupref}",
    "ld={ld}", "ld={ld_lower}", "ld=", "ld=FEED1LD2", "ld=NOSLASH.x", "ld=&q=noslash",
    "bad_quality=true", "bad_quality=TRUE", "bad_quality=true&q=beh", "bad_quality=true&ld={ld}",
    "in_dataset=true", "in_dataset=true&limit=1", "in_dataset=yes", "in_dataset=true&ld=FEED1LD2",
    "offset=-5", "offset=0", "offset=1", "offset=5&limit=3", "limit=0", "limit=1", "limit=250", "limit=251",
    "limit=%207%20", "offset=%2B5", "offset=1_0", "offset=x", "offset=", "limit=", "offset=%D9%A3",
    "offset=99999999999999999999999", "limit=-99999999999999999999", "limit=99999999999999999999",
    "offset=-99999999999999999999", "offset=1&offset=x", "offset=x&limit=y", "limit=x", "offset=" + "1" * 4301,
    "offset=" + "a" * 300, "offset=%27", "offset=%27%22", "offset=5%E2%82", "offset=%C2%A05", "offset=%1C5",
    "offset=1__0", "offset=0x10", "offset=1.5", "limit=%EF%BC%95", "offset=440&limit=250", "offset=%E2%80%A87",
    "q=xcbr&fc=ST&kind=Dbpos,BOOLEAN&writable=true&offset=0&limit=2", "q=.&ld={ld}&in_dataset=true&fc=ST",
]
FULL = {"", "q=XCBR", "bad_quality=true", "in_dataset=true", "limit=250", "ld=FEED1LD2"}
RAW_ARGS = ["", "a=%E2%82A&b=%zz&c&d=1;e=2&f=x+y&q=%C3%9F&&g==h&%71=k&r=%ff%E2%82%AC", "a=%", "a=%4", "a=%4g",
            "a+b=c+d", "x=%2B%26%3D", "=v", "&&&", "k=1&k=2&k=", "e=%ED%A0%80", "o=%F0%9F%98%80%F0%9F",
            "u=%e2%82%ac", "p=%C3%28", "m=%C0%AF", "n=%E0%80%AF%F4%90%80%80"]


def queries(ied):
    reports = ied["reports"]
    report = reports[0]["ref"] if reports else "D/LLN0.BR.none"
    ld = ied["signals"][0]["ref"].split("/", 1)[0]
    values = {"report": quote(report, safe=""), "ref": quote(ied["signals"][3]["ref"], safe=""), "ld": ld,
              "ld_lower": ld.lower(), "dupref": quote(ied["signals"][1]["ref"], safe="")}
    result = []
    for template in TEMPLATES:
        text = template.format(**values)
        if template not in FULL and "limit" not in template:  # keep bodies small; paging has own cases
            text = text + "&limit=6" if text else "limit=6"
        result.append(text)
    return result


def ied_arg(ied):
    result = copy.deepcopy(ied)
    result.pop("source", None)
    result.pop("cfg", None)
    return result


def ask(app, client, label, ied, text):
    path = f"/api/v1/ieds/{ied['id']}/signals" + (f"?{text}" if text else "")
    answer = client.get(path)
    failure = None
    with app.test_request_context(path):
        try:
            list_signals(ied["id"])
        except Exception as error:  # the route's error handler answers it; Go returns the same class
            failure = type(error).__name__
    return (f"query {label} {text[:80]}", ["query", label, text, failure],
            f"{answer.status_code} {answer.get_data(as_text=True)}")


def modify(ied):
    """FEED1-mod: quality values, unicode descriptions, extra devices, odd refs and reports."""
    signals = ied["signals"]
    quality_refs = [s["quality_ref"] for s in signals if s.get("quality_ref")]
    by_ref = {s["ref"]: s for s in signals}
    points = [by_ref[ref] for ref in dict.fromkeys(quality_refs) if ref in by_ref]
    for point, value in zip(points, [1, 0.0, False, None, True, "x", 2048, [], -1, 0, 0.5, "0"]):
        point["value"] = value
    points[12].pop("value", None)
    texts = ["Straße ẞ Größe", "İstanbul DIŞ", "ﬁle Ω ohm", "Pos stVal"]
    for index, text in enumerate(texts):
        signals[10 + index]["description"] = text
    signals[20]["quality_ref"] = None
    signals[21]["quality_ref"] = ""
    signals[22].pop("quality_ref", None)
    signals[23].pop("timestamp_ref", None)
    signals[24]["timestamp_ref"] = None
    signals[25]["writable"] = "yes"
    signals[26]["writable"] = 0
    signals[27]["writable"] = None
    signals[28]["kind"] = None
    extra = []
    for index in range(8):
        item = copy.deepcopy(signals[index])
        item["ref"] = "FEED1LD2/" + item["ref"].split("/", 1)[1]
        extra.append(item)
    extra.append({**copy.deepcopy(signals[1]), "value": 77, "description": "duplicate"})
    extra.append({**copy.deepcopy(signals[2]), "ref": "NOSLASH.x", "value": 3})
    extra.append({**copy.deepcopy(signals[2]), "ref": "", "value": 5, "quality_ref": "", "description": "empty"})
    extra.append({**copy.deepcopy(signals[2]), "ref": "Z/z", "fc": "CF", "writable": True})
    signals.extend(extra)
    ied["datasets"].append({"name": "LD2", "ref": "FEED1LD2/LLN0$LD2", "members": [], "unresolved": [],
                            "signal_refs": [extra[0]["ref"], extra[3]["ref"], "NOSLASH.x", "unknown"],
                            "fcda_refs": []})
    ied["reports"].extend([{"ref": "dup", "dataset_ref": "FEED1LD2/LLN0$LD2"},
                           {"ref": "dup", "dataset_ref": ied["datasets"][0]["ref"]},
                           {"ref": "nodata", "dataset_ref": "missing$DS"}])


def args_cases(app):
    cases = []
    for index, raw in enumerate(RAW_ARGS):
        with app.test_request_context("/x" + (f"?{raw}" if raw else "")):
            from flask import request
            pairs = sorted([key, request.args.getlist(key)] for key in set(request.args.keys()))
        cases.append((f"args {index}", ["args", raw], json.dumps(pairs)))
    return cases


def main():
    cases = []
    for name in ("model", "open"):
        app, client, folder = app_with(name)
        try:
            ieds = app.extensions["projects"].project["ieds"]
            for ied in list(ieds):
                cases.append((f"ied {ied['name']}", ["ied", ied["name"], ied_arg(ied)], ""))
                cases.extend(ask(app, client, ied["name"], ied, text) for text in queries(ied))
            if name == "model":
                feed = next(ied for ied in ieds if ied["name"] == "FEED1")
                modify(feed)
                cases.append(("ied FEED1-mod", ["ied", "FEED1-mod", ied_arg(feed)], ""))
                cases.extend(ask(app, client, "FEED1-mod", feed, text) for text in queries(feed))
                cases.extend(args_cases(app))
        finally:
            close(app, folder)
    record("signals_http.json", cases)


if __name__ == "__main__":
    main()
