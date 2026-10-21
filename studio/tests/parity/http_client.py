"""HTTP client with Flask's test-client call signature, so the test files stay unchanged.

Encoding follows werkzeug's EnvironBuilder: `json=` through Flask's JSON provider
(sort_keys, ensure_ascii), `data=` dicts with (stream, filename) tuples as multipart, plain
strings as form fields, `query_string=` dicts URL-encoded. A cookie jar keeps the
`studio_session` cookie; no Origin header unless the test sets one, like the test client.
"""
import http.client
from http.cookies import SimpleCookie
import json
import mimetypes
import secrets
from urllib.parse import urlencode

from parity.response import Response


class HttpClient:
    def __init__(self, engine):
        self.engine = engine
        self.cookies = {}

    def get(self, path, **options):
        return self.open(path, method="GET", **options)

    def post(self, path, **options):
        return self.open(path, method="POST", **options)

    def put(self, path, **options):
        return self.open(path, method="PUT", **options)

    def patch(self, path, **options):
        return self.open(path, method="PATCH", **options)

    def delete(self, path, **options):
        return self.open(path, method="DELETE", **options)

    def open(self, path, method="GET", json=None, data=None, headers=None, query_string=None,
             content_type=None, environ_base=None, **unsupported):
        if unsupported:
            raise TypeError(f"parity HttpClient does not support {sorted(unsupported)}")
        body, kind = _body(json, data, content_type)
        target = path + (("&" if "?" in path else "?") + _query(query_string) if query_string else "")
        sent = {"Host": f"127.0.0.1:{self.engine.port}"}
        if kind:
            sent["Content-Type"] = kind
        if body is not None or method in ("POST", "PUT", "PATCH"):
            sent["Content-Length"] = str(len(body or b""))
        if self.cookies:
            sent["Cookie"] = "; ".join(f"{name}={value}" for name, value in self.cookies.items())
        for name, value in dict(headers or {}).items():
            sent[name] = value
        remote = (environ_base or {}).get("REMOTE_ADDR")
        if remote:
            sent["X-Parity-Key"] = self.engine.parity_key
            sent["X-Parity-Remote-Addr"] = remote
        connection = http.client.HTTPConnection("127.0.0.1", self.engine.port, timeout=600)
        try:
            connection.putrequest(method, target, skip_host=True, skip_accept_encoding=True)
            for name, value in sent.items():
                connection.putheader(name, value)
            connection.endheaders(body)
            reply = connection.getresponse()
            response = Response(reply.status, reply.getheaders(), reply.read())
        finally:
            connection.close()
        self._remember(response)
        return response

    def _remember(self, response):
        for header in response.headers.getlist("Set-Cookie"):
            cookie = SimpleCookie()
            cookie.load(header)
            for name, morsel in cookie.items():
                if morsel["max-age"] == "0" or (morsel["expires"] and "1970" in morsel["expires"]):
                    self.cookies.pop(name, None)
                else:
                    self.cookies[name] = morsel.coded_value


def _query(values):
    if isinstance(values, (str, bytes)):
        return values.decode() if isinstance(values, bytes) else values
    return urlencode(values, doseq=True)


def _body(payload, data, content_type):
    if payload is not None:
        text = json.dumps(payload, ensure_ascii=True, sort_keys=True)
        return text.encode("utf-8"), content_type or "application/json"
    if data is None:
        return None, content_type
    if isinstance(data, str):
        return data.encode("utf-8"), content_type
    if isinstance(data, bytes):
        return data, content_type
    if any(isinstance(value, tuple) or hasattr(value, "read") for value in data.values()):
        return _multipart(data)
    return urlencode(data, doseq=True).encode("ascii"), content_type or "application/x-www-form-urlencoded"


def _multipart(fields):
    boundary = "----parity" + secrets.token_hex(12)
    chunks = []
    for name, value in fields.items():
        head = f'--{boundary}\r\nContent-Disposition: form-data; name="{name}"'
        if isinstance(value, tuple) or hasattr(value, "read"):
            stream, filename = value[:2] if isinstance(value, tuple) else (value, getattr(value, "name", ""))
            kind = mimetypes.guess_type(filename)[0] or "application/octet-stream"
            content = stream.read()
            content = content.encode("utf-8") if isinstance(content, str) else content
            chunks.append(f'{head}; filename="{filename}"\r\nContent-Type: {kind}\r\n\r\n'.encode() + content + b"\r\n")
        else:
            chunks.append(f"{head}\r\n\r\n{value}\r\n".encode("utf-8"))
    chunks.append(f"--{boundary}--\r\n".encode())
    return b"".join(chunks), f"multipart/form-data; boundary={boundary}"
