"""Run one golden scenario against one engine and write its transcript (JSON lines).

Step forms (golden/scenarios/<area>/<name>.json):
  {"req": "POST /api/v1/scl/import", "json": {...} | "multipart": {"file": "@demo"}, "form": {...},
   "query": {...}, "headers": {...}, "bind": {"ied": "ieds.0.id"}, "settle": 0.25}
  {"wait_event": {"kind": "runtime", "match": {"state": "running"}}, "timeout": 30}
`$name` in paths/bodies uses a value bound earlier (`$rev` = last `revision` seen). Uploads are
checked against the public manifest before they are sent; a refusal deletes partial output.
"""
import base64
import io
import json
from pathlib import Path
import shutil
import tempfile
import zipfile

from parity import launch
from parity.http_client import HttpClient
from parity.replay.manifest import Manifest, PrivacyError, sha256
from parity.replay.sse import SseTap

TOKEN = "golden-session-token"


def substitute(value, bound):
    if isinstance(value, str):
        if value.startswith("$") and value[1:] in bound:
            return bound[value[1:]]
        for name, item in sorted(bound.items(), key=lambda pair: -len(pair[0])):
            value = value.replace("{$" + name + "}", str(item))
        return value
    if isinstance(value, list):
        return [substitute(item, bound) for item in value]
    if isinstance(value, dict):
        return {key: substitute(item, bound) for key, item in value.items()}
    return value


def pick(tree, dotted):
    for part in dotted.split("."):
        tree = tree[int(part)] if isinstance(tree, list) else tree[part]
    return tree


def body_record(response):
    kind = response.mimetype
    disposition = response.headers.get("Content-Disposition", "")
    if kind == "application/json":
        return {"body_kind": "json", "body": response.data.decode("utf-8")}
    if kind.startswith("text/"):
        return {"body_kind": "text", "body": response.data.decode("utf-8", "replace")}
    if response.data[:2] == b"PK" or disposition.endswith(".scadabench"):
        with zipfile.ZipFile(io.BytesIO(response.data)) as archive:
            entries = [[item.filename, item.compress_type] for item in archive.infolist()]
            project = archive.read("project.json").decode("utf-8") if "project.json" in archive.namelist() else None
        return {"body_kind": "zip", "body": {"entries": entries, "project.json": project}}
    return {"body_kind": "bytes", "body": base64.b64encode(response.data).decode("ascii")}


class Recorder:
    def __init__(self, engine, manifest=None, native_exe=None, env=None):
        self.engine_name, self.manifest, self.env = engine, manifest or Manifest(), env
        self.native_exe = Path(native_exe or launch.STUDIO / "native/build/Release/studio_ied.exe")

    def _request(self, client, step, bound, allowed):
        method, path = step["req"].split(" ", 1)
        options = {"query_string": substitute(step.get("query"), bound),
                   "headers": substitute(step.get("headers"), bound)}
        if "json" in step:
            options["json"] = substitute(step["json"], bound)
        if "multipart" in step or "form" in step:
            data = {key: str(substitute(value, bound)) for key, value in (step.get("form") or {}).items()}
            for field, alias in (step.get("multipart") or {}).items():
                payload = self.manifest.load(alias[1:])
                data[field] = (io.BytesIO(payload), Path(self.manifest.inputs[alias[1:]]["path"] or "corpus.scd").name)
                allowed.update(self.manifest.inputs[alias[1:]]["ieds"])
            options["data"] = data
        if "raw" in step:
            options["data"] = step["raw"].encode("utf-8")
            options["content_type"] = step.get("content_type")
        return client.open(substitute(path, bound), method=method,
                           **{key: value for key, value in options.items() if value is not None})

    def run(self, scenario, output):
        scratch = Path(tempfile.mkdtemp(prefix="studio-golden-"))
        output = Path(output)
        engine = launch.EngineProcess(self.engine_name, scratch / "data", self.native_exe, scratch / "frontend",
                                      {"token": TOKEN, **(scenario.get("seams") or {})}, env=self.env)
        try:
            engine.start()
            records = self._steps(engine, scenario, scratch)
            orphans = engine.shutdown()
            if orphans:
                raise RuntimeError(f"orphan processes after scenario {scenario['name']}: {orphans}")
        except BaseException:
            output.unlink(missing_ok=True)
            raise
        finally:
            engine.kill()
            shutil.rmtree(scratch, ignore_errors=True)
        records = scrub(records, [(scratch / "data", "<data>"), (self.native_exe.parent, "<bin>"),
                                  (scratch, "<scratch>")])
        output.parent.mkdir(parents=True, exist_ok=True)
        output.write_text("".join(json.dumps(record, sort_keys=False) + "\n" for record in records), encoding="utf-8")
        return records

    def _steps(self, engine, scenario, scratch):
        observer = HttpClient(engine)
        if observer.post("/api/v1/session", json={"token": TOKEN}).status_code != 200:
            raise RuntimeError("golden observer could not sign in")
        cookie = "; ".join(f"{name}={value}" for name, value in observer.cookies.items())
        tap = SseTap(engine.port, cookie).start()
        header = {"scenario": scenario["name"], "engine": self.engine_name, "data": str(scratch / "data"),
                  "bin": str(self.native_exe.parent), "port": engine.port,
                  "sse": {"status": tap.status, "content_type": tap.headers.get("Content-Type"),
                          "cache_control": tap.headers.get("Cache-Control")}}
        tap.settle()
        records, client, bound = [header], HttpClient(engine), {}
        allowed = set(scenario.get("template_names", []))
        try:
            for index, step in enumerate(scenario["steps"]):
                records.append(self._step(index, step, client, tap, bound, allowed))
            names = [ied["name"] for ied in observer.get("/api/v1/project").json.get("ieds", [])]
            self.manifest.check_names(names, allowed)
            header["sse"]["head"] = tap.head().decode("utf-8", "replace")
        finally:
            tap.close()
        return records

    def _step(self, index, step, client, tap, bound, allowed):
        if "wait_event" in step:
            want = step["wait_event"]
            found = tap.wait_for(lambda frame: _matches(json.loads(frame["data"]), want), step.get("timeout", 30))
            return {"step": index, "wait_event": want, "found": found, "events": _events(tap.settle())}
        if step.get("needs") and step["needs"] not in bound:
            return {"step": index, "req": step["req"], "skipped": True, "events": _events(tap.settle(0.05))}
        response = self._request(client, step, bound, allowed)
        record = {"step": index, "req": step["req"], "status": response.status_code,
                  "headers": {name: response.headers.get(name) for name in
                              ("Content-Type", "Content-Disposition", "Cache-Control")},
                  "set_cookie": response.headers.get("Set-Cookie"),
                  "body_sha256": sha256(response.data), **body_record(response)}
        if record["body_kind"] == "json":
            parsed = json.loads(response.data)
            if isinstance(parsed, dict) and isinstance(parsed.get("revision"), int):
                bound["rev"] = parsed["revision"]
            for name, dotted in (step.get("bind") or {}).items():
                try:
                    bound[name] = pick(parsed, dotted)
                except (KeyError, IndexError, TypeError):
                    bound.pop(name, None)
        record["events"] = _events(tap.settle(step.get("settle", 0.25)))
        return record


def scrub(value, folders):
    """Local folders never reach a transcript: raw, JSON-escaped and slash forms become tokens."""
    if isinstance(value, str):
        for folder, token in folders:
            raw = str(folder)
            for form in (raw.replace("\\", "\\\\"), raw, raw.replace("\\", "/")):
                value = value.replace(form, token)
        return value
    if isinstance(value, list):
        return [scrub(item, folders) for item in value]
    if isinstance(value, dict):
        return {key: scrub(item, folders) for key, item in value.items()}
    return value


def _matches(event, want):
    return event.get("kind") == want.get("kind") and all(event.get(key) == value
                                                          for key, value in (want.get("match") or {}).items())


def _events(frames):
    return [{"id": frame["id"], "data": frame["data"]} for frame in frames]


__all__ = ["Recorder", "PrivacyError"]
