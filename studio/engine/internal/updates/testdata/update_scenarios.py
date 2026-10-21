"""UpdateService runs for make_vectors.py: HTTP answers, requests sent and file bytes."""
import base64
import io
import json
import tempfile
from pathlib import Path
from urllib.error import HTTPError, URLError

from flask import Flask, jsonify
from license_signer import keys
from test_update_check import RECORD, answer
from studio_app.updates.service import UpdateService

APP = Flask("update-vectors")
T0, DAY = 1_000_000, 86400
GOOD = answer().decode()
IMPORTANT = answer({**RECORD, "min_version": "0.1.14"}).decode()


def ok(body=GOOD, etag='"e1"'):
    return {"body": body, "etag": etag}


SCENARIOS = [
    {"name": "newer release offered", "steps": [
        {"op": "due", "now": T0}, {"op": "check", "answers": [ok()]}, {"op": "due"},
        {"op": "manual", "now": T0 + 30}, {"op": "configure", "arg": {"dismissed": "0.1.14"}},
        {"op": "configure", "arg": {"auto": False}}, {"op": "due", "now": T0 + 2 * DAY},
        {"op": "status", "current": "0.1.14"}, {"op": "status", "current": "0.1.15+dev"}]},
    {"name": "refused answers", "steps": [
        {"op": "check", "now": T0, "answers": [ok(answer(domain=b"").decode())]},
        {"op": "check", "answers": [ok("[1]")]}, {"op": "check", "answers": [ok("{bad", None)]},
        {"op": "status"}]},
    {"name": "offline keeps the last answer", "steps": [
        {"op": "check", "now": T0, "answers": [ok()]}, {"op": "check", "now": 2 * T0, "answers": [{"urlerror": True}]},
        {"op": "check", "now": 3 * T0, "answers": [{"http": 304}]}, {"op": "check", "answers": [{"http": 404}]},
        {"op": "check", "answers": [{"http": 500}]}, {"op": "check", "answers": [ok(etag=None)]},
        {"op": "check", "answers": [{"http": 304}]}]},
    {"name": "important release", "steps": [
        {"op": "check", "now": T0, "answers": [ok(IMPORTANT)]},
        {"op": "configure", "arg": {"dismissed": "0.1.14"}}, {"op": "status"}]},
    {"name": "configure rules", "steps": [
        {"op": "configure", "now": T0, "arg": {"auto": "yes"}}, {"op": "configure", "arg": {"other": 1}},
        {"op": "configure", "arg": None}, {"op": "configure", "arg": [1]}, {"op": "configure", "arg": {}},
        {"op": "configure", "arg": {"dismissed": "x"}}, {"op": "configure", "arg": {"dismissed": 5}},
        {"op": "configure", "arg": {"dismissed": 0}}, {"op": "configure", "arg": {"dismissed": None, "auto": True}},
        {"op": "configure", "arg": {"auto": False, "dismissed": "bad"}}, {"op": "status"}]},
    {"name": "hand edited file", "steps": [
        {"op": "write", "data": '{"theme": 1, "checked_at": "x", "auto": 0, "release": {"version": "9.9.9"}}'},
        {"op": "status", "now": T0}, {"op": "due"}, {"op": "manual"}, {"op": "check", "answers": [ok()]},
        {"op": "write", "data": '{"release": {"date": "x"}}'}, {"op": "status"},
        {"op": "write", "data": "[1]"}, {"op": "status"}, {"op": "write", "data": "﻿{}"}, {"op": "status"},
        {"op": "write", "data": '{"release": {"version": "1.0.0", "min_version": 3}}'}, {"op": "status"},
        {"op": "write", "data": '{"release": {}, "dismissed": "1.0.0"}'}, {"op": "status"}]},
    {"name": "no server", "server_url": "ftp://x", "steps": [{"op": "check", "now": T0}, {"op": "status"}]},
    {"name": "server address unreadable", "server_url": None, "steps": [{"op": "check", "now": T0}]},
]


class Response(io.BytesIO):
    def __init__(self, body, etag):
        super().__init__(body)
        self.headers = {} if etag is None else {"ETag": etag}


def make_opener(queue, requests):
    def opener(request, timeout):
        requests.append({"url": request.full_url, "timeout": timeout,
                         "headers": {name.lower(): value for name, value in request.header_items()}})
        item = queue.pop(0)
        if item.get("urlerror"):
            raise URLError("no route")
        if "http" in item:
            raise HTTPError(request.full_url, item["http"], "x", {}, io.BytesIO(b""))
        return Response(item["body"].encode("utf-8"), item.get("etag"))
    return opener


def body(value):
    with APP.app_context():
        return jsonify(value).get_data(as_text=True)


def run_step(service, step):
    op = step["op"]
    if op == "due":
        return {"value": service.due()}
    if op == "status":
        return {"http": 200, "body": body(service.status())}
    if op in ("check", "manual"):
        return {"http": 200, "body": body(service.check(manual=op == "manual"))}
    try:
        return {"http": 200, "body": body(service.configure(step.get("arg")))}
    except ValueError as error:
        with APP.app_context():
            return {"http": 400, "body": jsonify(error=str(error), code="invalid").get_data(as_text=True)}


def run(scenario):
    with tempfile.TemporaryDirectory() as folder:
        clock, queue, requests, current = [0], [], [], ["0.1.13"]
        url = scenario.get("server_url", "https://license.scadabench.com")

        def server_url():
            if url is None:
                raise AttributeError("'list' object has no attribute 'get'")
            return url
        service = UpdateService(folder, server_url, current="0.1.13", keys=keys(),
                                opener=make_opener(queue, requests), clock=lambda: clock[0])
        path, results = Path(folder) / "update_check.json", []
        for step in scenario["steps"]:
            clock[0] = step.get("now", clock[0])
            service.current = step.get("current", current[0])
            if step["op"] == "write":
                path.write_text(step["data"], encoding="utf-8")
                results.append({})
                continue
            queue[:] = list(step.get("answers", []))
            requests.clear()
            try:
                result = run_step(service, step)
            except Exception as error:  # noqa: BLE001 - an uncaught exception is the vector
                result = {"crash": type(error).__name__}
            stored = base64.b64encode(path.read_bytes()).decode() if path.exists() else None
            results.append({**result, "requests": json.loads(json.dumps(requests)), "file": stored})
        return {**scenario, "results": results}


def run_scenarios():
    return [run(scenario) for scenario in SCENARIOS]
