"""Callback seams: an engine calls a fake that lives in the test process.

Only for things a test cannot control from outside (Windows network calls, outbound HTTP,
named fault points, duck-typed services). The protocol is plain JSON over loopback HTTP so
the Go parity build can call it too: POST /call {"id", "args", "kwargs"} answers
{"return": value} | {"raise": exception} | {"original": true}.
"""
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import secrets
import threading
import urllib.request

from parity import wire

_DIRECT = urllib.request.build_opener(urllib.request.ProxyHandler({}))


class CallbackServer:
    """Harness side: serves registered fakes to the engine process."""

    def __init__(self):
        self.key = secrets.token_hex(16)
        self._items = {}
        self._lock = threading.Lock()
        handler = type("Handler", (_Handler,), {"owner": self})
        self._server = ThreadingHTTPServer(("127.0.0.1", 0), handler)
        self._server.daemon_threads = True
        self.url = f"http://127.0.0.1:{self._server.server_address[1]}"
        threading.Thread(target=self._server.serve_forever, name="parity-callbacks", daemon=True).start()

    def register(self, function):
        with self._lock:
            key = f"cb{len(self._items) + 1}"
            self._items[key] = function
        return {"url": self.url, "key": self.key, "id": key}

    def invoke(self, key, args, kwargs):
        function = self._items[key]
        try:
            return {"return": wire.encode(function(*args, **kwargs))}
        except wire.DelegateToOriginal:
            return {"original": True}
        except Exception as error:  # the fake's exception is re-raised inside the engine
            return {"raise": wire.encode_exception(error)}

    def close(self):
        self._server.shutdown()
        self._server.server_close()


class _Handler(BaseHTTPRequestHandler):
    owner = None

    def do_POST(self):
        if self.headers.get("X-Parity-Key") != self.owner.key:
            self.send_error(403)
            return
        body = json.loads(self.rfile.read(int(self.headers.get("Content-Length", 0))))
        answer = self.owner.invoke(body["id"], wire.decode(body.get("args", [])), wire.decode(body.get("kwargs", {})))
        data = json.dumps(answer).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, *_args):
        pass


def remote_function(reference, original=None):
    """Engine side: a function that runs the registered fake in the test process."""

    def call(*args, **kwargs):
        payload = json.dumps({"id": reference["id"], "args": wire.encode(list(args)),
                              "kwargs": wire.encode(kwargs)}).encode()
        request = urllib.request.Request(reference["url"] + "/call", data=payload, method="POST",
                                         headers={"Content-Type": "application/json",
                                                  "X-Parity-Key": reference["key"]})
        with _DIRECT.open(request, timeout=120) as response:
            answer = json.loads(response.read())
        if answer.get("original"):
            if original is None:
                raise RuntimeError("parity fake asked for an original that does not exist")
            return original(*args, **kwargs)
        if "raise" in answer:
            raise wire.decode_exception(answer["raise"])
        return wire.decode(answer.get("return"))
    return call
