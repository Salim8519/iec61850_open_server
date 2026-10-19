"""Record the frozen Flask app's answers for the design routes (http_routes_golden.json):
PUT /ieds/<key>/settings, PUT /scenarios, GET /project/substation-sources,
POST /project/diagram/from-scl/preview and the /signal-flow routes.

  python http_gen_routes.py            writes generated/http_routes_golden.json

Public inputs only (the demo station and scd/open_substation.scd). Same recorder and
deterministic ids as http_gen.py; the replay is TestFlaskParityDesignRoutes.
"""
import json
import tempfile
from pathlib import Path

from http_gen import HERE, WORKTREE, Recorder, _Id, _Off, _patch_ids, _record_xsd

DEMO = "marketing/videos/demo/scadabench_demo.scd"
OPEN = "scd/open_substation.scd"


def previews(rec, label, source):
    for mode in ("replace", "beside", "bogus"):
        rec.call(f"preview {label} {mode}", "POST", "/api/v1/project/diagram/from-scl/preview",
                 json_body={"source_id": source, "mode": mode})


def design(rec, app):
    rec.call("flow sources empty", "GET", "/api/v1/signal-flow/sources")
    rec.call("flow graph empty", "GET", "/api/v1/signal-flow")
    rec.call("flow csv empty", "GET", "/api/v1/signal-flow/edges/x.csv")
    rec.call("substation sources empty", "GET", "/api/v1/project/substation-sources")
    for label, body in [("no body", None), ("empty", {}), ("one key", {"source_id": "x"}),
                        ("number", {"source_id": 1, "mode": "replace"}), ("list", []),
                        ("extra", {"source_id": "x", "mode": "replace", "extra": 1})]:
        if body is None:
            rec.call("preview " + label, "POST", "/api/v1/project/diagram/from-scl/preview")
        else:
            rec.call("preview " + label, "POST", "/api/v1/project/diagram/from-scl/preview", json_body=body)
    rec.call("preview unknown source", "POST", "/api/v1/project/diagram/from-scl/preview",
             json_body={"source_id": "nope", "mode": "replace"})
    for path, names in ((DEMO, ["FEED1", "FEED2", "BUS1", "BUS2", "TR1", "TR2"]), (OPEN, ["IED1_XCBR", "IED2_PTOC"])):
        rec.call("import " + path, "POST", "/api/v1/scl/import", form={"selected_names": json.dumps(names)},
                 file=("file", Path(path).name, (WORKTREE / path).read_bytes()))
    sources = rec.call("substation sources", "GET", "/api/v1/project/substation-sources").json["sources"]
    replaced = None
    for index, entry in enumerate(sources):
        previews(rec, f"source {index}", entry["id"])
        answer = json.loads(rec.cases[-3]["response"])
        replaced = replaced or ("diagram" in answer and answer)
    preview = replaced
    rec.call("apply replace", "PUT", "/api/v1/project/diagram",
             json_body={"diagram": preview["diagram"], "revision": preview["revision"], "tab_id": "tab-1"})
    previews(rec, "after apply", sources[0]["id"])
    flow = rec.call("flow sources", "GET", "/api/v1/signal-flow/sources").json["sources"]
    graph = rec.call("flow graph default", "GET", "/api/v1/signal-flow").json
    for entry in flow:
        for ied in ["", entry["ieds"][0], "NOPE", "FEED"]:
            rec.call(f"flow graph {entry['filename']} {ied}", "GET",
                     f"/api/v1/signal-flow?source={entry['id']}&ied={ied}")
    rec.call("flow graph unknown source", "GET", "/api/v1/signal-flow?source=nope")
    for edge in graph["edges"][:4]:
        rec.call("flow csv " + edge["type"], "GET", f"/api/v1/signal-flow/edges/{edge['id']}.csv")
    rec.call("flow csv unknown", "GET", "/api/v1/signal-flow/edges/nope.csv")
    rec.call("flow csv other source", "GET", f"/api/v1/signal-flow/edges/x.csv?source={flow[-1]['id']}")
    ieds = app.extensions["projects"].project["ieds"]
    first, second = ieds[0]["id"], ieds[1]["id"]
    for label, body in [("no body", None), ("empty", {}), ("unknown key", {"bogus": 1}), ("port", {"port": 1103}),
                        ("port text", {"port": "x"}), ("port bool", {"port": True}), ("port range", {"port": 70000}),
                        ("role real", {"role": "real", "live": {"ip": "10.20.30.40"}}),
                        ("live merge", {"live": {"port": 103}}), ("role bad", {"role": "x"}),
                        ("lab ip", {"lab_ip": "10.1.1.9"}), ("lab ip bad", {"lab_ip": "999.1.1.1"}),
                        ("goose", {"goose_publish": False}), ("list", [1])]:
        if body is None:
            rec.call("settings " + label, "PUT", f"/api/v1/ieds/{first}/settings")
        else:
            rec.call("settings " + label, "PUT", f"/api/v1/ieds/{first}/settings", json_body=body)
    rec.call("settings conflict", "PUT", f"/api/v1/ieds/{second}/settings",
             json_body={"role": "real", "live": {"ip": "10.20.30.40", "port": 103}})
    rec.call("settings unknown ied", "PUT", "/api/v1/ieds/nope/settings", json_body={"port": 102})
    rec.call("scenarios no body", "PUT", "/api/v1/scenarios")
    rec.call("scenarios bad json", "PUT", "/api/v1/scenarios", raw="[", content_type="application/json")
    for label, body in [("empty", []), ("dict", {}), ("bad item", [{"x": 1}]), ("bad name", [{"id": "a", "name": ""}])]:
        rec.call("scenarios " + label, "PUT", "/api/v1/scenarios", json_body=body)
    rec.call("project final", "GET", "/api/v1/project")


def main():
    _patch_ids()
    _Id.count = 0
    xsd = {}
    _record_xsd(xsd)
    from studio_app.app_factory import create_app
    from studio_app.settings import Settings
    with tempfile.TemporaryDirectory() as folder:
        root = Path(folder)
        app = create_app(Settings(root, root / "unused.exe", root), "http-golden",
                         license_service=_Off(), update_service=_Off())
        client = app.test_client()
        client.post("/api/v1/session", json={"token": "http-golden"})
        app.extensions["startup"].wait()
        rec = Recorder(client)
        try:
            design(rec, app)
        finally:
            app.extensions["runtime"].close()
    text = json.dumps({"ids_before_requests": 2, "cases": rec.cases,
                       "xsd": [xsd[key] for key in sorted(xsd)]}, indent=1, ensure_ascii=False)
    (HERE / "generated" / "http_routes_golden.json").write_text(text + "\n", encoding="utf-8", newline="\n")
    print(len(rec.cases), "cases")


if __name__ == "__main__":
    main()
