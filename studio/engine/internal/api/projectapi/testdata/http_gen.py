"""Record the frozen Flask app's answers for the projectapi routes (http_golden.json).

  python http_gen.py            writes generated/http_golden.json

Public inputs only (scd/model_substation.scd and generated payloads). Ids are made
deterministic: every uuid4() the app takes is the next counter value as 32 hex digits,
which the Go replay (http_test.go) reproduces with the same counter.
"""
import base64
import io
import json
import sys
import tempfile
from pathlib import Path

HERE = Path(__file__).resolve().parent
WORKTREE = HERE.parents[5]
sys.path.insert(0, str(WORKTREE / "studio" / "backend"))
MODEL = "scd/model_substation.scd"


class _Id:
    count = 0

    def __init__(self):
        _Id.count += 1
        self.hex = f"{_Id.count:032x}"


class _Off:
    enforce = False

    def start_background_verify(self):
        pass

    def start_background(self):
        pass

    def server_url(self):
        return ""


def _patch_ids():
    from studio_app.scl import importer
    from studio_app.projects import service, bulk_removal, startup_recovery, migration_backup
    for module in (importer, service, bulk_removal, startup_recovery, migration_backup):
        module.uuid4 = _Id


def _record_xsd(entries):
    """Record every verdict validate_schema gives, to seed the Go replay's validator."""
    import hashlib
    from studio_app.api import scl as scl_api
    from studio_app.scl import exporter, importer, schema_validation
    original = schema_validation.validate_schema

    def recording(source, root=None):
        result = original(source, root)
        data = source.encode("utf-8") if isinstance(source, str) else source
        if result.get("schema") is not None:
            entries[(hashlib.sha256(data).hexdigest(), result["schema"])] = {
                "sha256": hashlib.sha256(data).hexdigest(), "schema": result["schema"],
                "status": result["status"], "errors": result["errors"]}
        return result

    for module in (scl_api, importer, exporter):
        module.validate_schema = recording


def _headers(response):
    skip = {"content-length", "date", "set-cookie", "vary", "server"}
    return {key: value for key, value in response.headers.items() if key.lower() not in skip}


class Recorder:
    def __init__(self, client):
        self.client, self.cases = client, []

    def call(self, name, method, path, json_body=None, raw=None, content_type=None, form=None, file=None):
        request = {"name": name, "method": method, "path": path}
        kwargs = {}
        if json_body is not None:
            raw, content_type = json.dumps(json_body), "application/json"
        if raw is not None:
            request["body"], kwargs["data"] = raw, raw.encode("utf-8")
            if content_type:
                request["content_type"] = kwargs["content_type"] = content_type
        if form is not None or file is not None:
            data = dict(form or {})
            request["form"] = form or {}
            if file is not None:
                field, filename, payload = file
                request["file"] = {"field": field, "filename": filename,
                                   "b64": base64.b64encode(payload).decode("ascii")}
                data[field] = (io.BytesIO(payload), filename)
            kwargs["data"], kwargs["content_type"] = data, "multipart/form-data"
        response = self.client.open(path, method=method, **kwargs)
        binary = response.mimetype == "application/zip"
        self.cases.append({**request, "status": response.status_code, "headers": _headers(response),
                           "binary": binary, "response": "" if binary else response.get_data(as_text=True)})
        return response


def scenario(rec, app):
    model = (WORKTREE / MODEL).read_bytes()
    rec.call("project initial", "GET", "/api/v1/project")
    for label, body in [("no body", None), ("list keys", ["diagram", "revision", "tab_id"]),
                        ("string", "abc"), ("number", 5), ("empty", {}),
                        ("string revision", {"diagram": {}, "revision": "0", "tab_id": "t"}),
                        ("empty tab", {"diagram": {}, "revision": 0, "tab_id": ""}),
                        ("long tab", {"diagram": {}, "revision": 0, "tab_id": "x" * 65}),
                        ("stale", {"diagram": {"nodes": [], "edges": []}, "revision": 9, "tab_id": "t"}),
                        ("bool revision", {"diagram": {"nodes": [], "edges": []}, "revision": True, "tab_id": "t"}),
                        ("bad diagram", {"diagram": {"nodes": "x"}, "revision": 0, "tab_id": "t"}),
                        ("ok", {"diagram": {"nodes": [], "edges": []}, "revision": 0, "tab_id": "tab-1"})]:
        if body is None:
            rec.call("diagram " + label, "PUT", "/api/v1/project/diagram")
        else:
            rec.call("diagram " + label, "PUT", "/api/v1/project/diagram", json_body=body)
    rec.call("diagram bad json", "PUT", "/api/v1/project/diagram", raw="{", content_type="application/json")
    rec.call("template text", "POST", "/api/v1/ieds/template", raw="name", content_type="text/plain")
    for label, raw in [("bad json", '{"name": '), ("extra", '{} x'), ("bom", '﻿{}'),
                       ("control", '"a\x01"'), ("escape", '"\\x"'), ("key", '{1: 2}')]:
        rec.call("template " + label, "POST", "/api/v1/ieds/template", raw=raw, content_type="application/json")
    rec.call("template list", "POST", "/api/v1/ieds/template", json_body=[1])
    rec.call("template named", "POST", "/api/v1/ieds/template", json_body={"name": "Bay1"})
    rec.call("template default", "POST", "/api/v1/ieds/template", json_body={})
    rec.call("template bad name", "POST", "/api/v1/ieds/template", json_body={"name": "1 bad"})
    rec.call("validate no file", "POST", "/api/v1/scl/validate", form={})
    rec.call("validate model", "POST", "/api/v1/scl/validate", file=("file", "m.scd", model))
    rec.call("validate doctype", "POST", "/api/v1/scl/validate",
             file=("file", "d.scd", b'<!DOCTYPE SCL><SCL/>'))
    rec.call("preview garbage", "POST", "/api/v1/scl/preview", file=("file", "g.scd", b"<SCL"))
    rec.call("preview model", "POST", "/api/v1/scl/preview", file=("file", "m.scd", model))
    rec.call("import no selection", "POST", "/api/v1/scl/import", file=("file", "m.scd", model))
    for label, names in [("bad json", "["), ("empty", "[]"), ("object", "{}"), ("unknown", '["NOPE"]')]:
        rec.call("import " + label, "POST", "/api/v1/scl/import", form={"selected_names": names},
                 file=("file", "m.scd", model))
    rec.call("import two", "POST", "/api/v1/scl/import", form={"selected_names": '["FEED1", "BUS1"]'},
             file=("file", "", model))
    rec.call("preview again", "POST", "/api/v1/scl/preview", file=("file", "m.scd", model))
    ieds = app.extensions["projects"].project["ieds"]
    bay = next(ied for ied in ieds if ied["name"] == "Bay1")
    point = next(item for item in bay["signals"] if item["kind"] == "FLOAT32")
    rec.call("source", "GET", f"/api/v1/ieds/{bay['id']}/source")
    rec.call("source missing", "GET", "/api/v1/ieds/nope/source")
    rec.call("export ok", "POST", f"/api/v1/ieds/{bay['id']}/export-scl", json_body={"overrides": {point["ref"]: 37.25}})
    for label, body in [("empty", {}), ("list", {"overrides": []}), ("extra", {"overrides": {}, "x": 1}),
                        ("not dict", [1]), ("bad value", {"overrides": {point["ref"]: "abc"}})]:
        rec.call("export " + label, "POST", f"/api/v1/ieds/{bay['id']}/export-scl", json_body=body)
    rec.call("save", "POST", "/api/v1/project/save")
    rec.call("project after import", "GET", "/api/v1/project")
    archive = rec.call("download", "GET", "/api/v1/project/download").data
    rec.call("open no file", "POST", "/api/v1/project/open", form={})
    rec.call("open corrupt", "POST", "/api/v1/project/open", file=("file", "x.subsim", b"PK\x03\x04junk"))
    rec.call("open archive", "POST", "/api/v1/project/open", file=("file", "x.subsim", archive))
    rec.call("preferences", "GET", "/api/v1/preferences")
    rec.call("preferences bad", "PUT", "/api/v1/preferences", json_body={"nope": 1})
    rec.call("preferences none", "PUT", "/api/v1/preferences", raw="x", content_type="text/plain")
    rec.call("preferences ok", "PUT", "/api/v1/preferences", json_body={"tour_seen_version": 1, "sld_goose_layer": 1})
    rec.call("preferences after", "GET", "/api/v1/preferences")
    first = app.extensions["projects"].project["ieds"][0]["id"]
    rec.call("removal preview", "GET", f"/api/v1/ieds/{first}/removal-preview")
    rec.call("removal preview missing", "GET", "/api/v1/ieds/nope/removal-preview")
    rec.call("remove unconfirmed", "DELETE", f"/api/v1/ieds/{first}")
    rec.call("remove list body", "DELETE", f"/api/v1/ieds/{first}", json_body=[1])
    rec.call("bulk preview", "POST", "/api/v1/ieds/removal-preview", json_body={"ids": [first]})
    rec.call("bulk preview bad", "POST", "/api/v1/ieds/removal-preview", json_body={"ids": "x"})
    rec.call("bulk unconfirmed", "POST", "/api/v1/ieds/remove", json_body={"ids": [first]})
    rec.call("bulk stale", "POST", "/api/v1/ieds/remove",
             json_body={"ids": [first], "fingerprint": "0" * 64, "confirm": True})
    rec.call("remove one", "DELETE", f"/api/v1/ieds/{first}", json_body={"confirm": True})
    second = app.extensions["projects"].project["ieds"][0]["id"]
    preview = rec.call("bulk preview second", "POST", "/api/v1/ieds/removal-preview", json_body={"ids": [second]}).json
    rec.call("bulk remove", "POST", "/api/v1/ieds/remove",
             json_body={"ids": [second], "fingerprint": preview["fingerprint"], "confirm": True})
    rec.call("project final", "GET", "/api/v1/project")


def main():
    _patch_ids()
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
            scenario(rec, app)
        finally:
            app.extensions["runtime"].close()
    text = json.dumps({"ids_before_requests": 2, "cases": rec.cases,
                       "xsd": [xsd[key] for key in sorted(xsd)]}, indent=1, ensure_ascii=False)
    (HERE / "generated" / "http_golden.json").write_text(text + "\n", encoding="utf-8", newline="\n")
    print(len(rec.cases), "cases")


if __name__ == "__main__":
    main()
