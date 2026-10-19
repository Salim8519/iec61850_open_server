"""WSGI wrapper: /api/v1/_parity/* goes to the inspect operations, everything else to the
unmodified product app. Parity headers are removed before the product sees a request.

The X-Parity-Remote-Addr header (valid only with the run's parity key) stands in for Flask's
`environ_base={"REMOTE_ADDR": ...}`: a test cannot connect from a LAN address over loopback.
"""
import json
from urllib.parse import parse_qs

from parity import wire
from parity.inspect_ops import OPERATIONS

PREFIX = "/api/v1/_parity/"


class InspectApp:
    def __init__(self, app, key, stopping):
        self.app, self.key, self.stopping = app, key, stopping

    def __call__(self, environ, start_response):
        path = environ.get("PATH_INFO", "")
        trusted = environ.pop("HTTP_X_PARITY_KEY", None) == self.key
        remote = environ.pop("HTTP_X_PARITY_REMOTE_ADDR", None)
        if path.startswith(PREFIX):
            if not trusted:
                return _reply(start_response, 403, {"error": "parity key required"})
            return self.inspect(environ, start_response, path[len(PREFIX):])
        if remote and trusted:
            environ["REMOTE_ADDR"] = remote
        return self.app(environ, start_response)

    def inspect(self, environ, start_response, operation):
        method = environ.get("REQUEST_METHOD", "GET")
        handler = OPERATIONS.get((method, operation))
        if handler is None:
            return _reply(start_response, 404, {"error": f"unknown parity operation {method} {operation}"})
        size = int(environ.get("CONTENT_LENGTH") or 0)
        raw = environ["wsgi.input"].read(size) if size else b""
        body = wire.loads(raw) if raw else {}
        query = {key: values[-1] for key, values in parse_qs(environ.get("QUERY_STRING", "")).items()}
        context = {"extensions": self.app.extensions, "stopping": self.stopping}
        try:
            with self.app.app_context():
                value = handler(context, body, query)
        except Exception as error:  # re-raised with the same class in the test process
            return _reply(start_response, 500, {"raise": wire.encode_exception(error)})
        return _reply(start_response, 200, {"value": wire.encode(value)})


def _reply(start_response, status, payload):
    data = json.dumps(payload).encode()
    reason = {200: "OK", 403: "FORBIDDEN", 404: "NOT FOUND", 500: "INTERNAL SERVER ERROR"}[status]
    start_response(f"{status} {reason}", [("Content-Type", "application/json"), ("Content-Length", str(len(data)))])
    return [data]
