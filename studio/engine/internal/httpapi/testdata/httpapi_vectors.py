"""Python's answers for the Go httpapi helpers (one JSON object per line).

Usage: python -I httpapi_vectors.py --backend <studio/backend>
Kinds: cookie (Flask session signing at fixed times), netloc (urllib.parse.urlsplit),
query (urllib.parse.parse_qsl as Werkzeug reads request.args), pak (frontend.pak written
by launcher/frontend_assets.pack), disposition (Werkzeug send_file Content-Disposition).
"""
import argparse
import json
import sys
from urllib.parse import parse_qsl, urlsplit

ORIGINS = ["http://127.0.0.1:8765", "http://127.0.0.1:8765/", "null", "", "  http://a:1/x", "HTTP://x",
           "http://[::1]:5173", "http://[::1", "http://::1]", "http://[127.0.0.1]", "http://[zz]",
           "file:///c:/x", "x:y//z", "//host:1", "http://h\t:1\n2", "1http://h", "http://user@h:1?q#f",
           "\x01 http://h", "chrome-extension://abc"]
QUERIES = ["after=5", "after=%2B5", "after=+5+", "after", "after=", "a=1&after=7&after=8",
           "after=%zz", "after=%e9", "after=%C3%A9", "after=\xe9", "x=1;after=2", "&&after=3&", "after=%", "Aft=1&after=9"]
NAMES = ["index.html", "app-1a2b.js", "a b.csv", 'quote"d.csv', "back\\slash", "semi;colon", "x(1).txt", ""]


def cookies():
    from flask import Flask
    import itsdangerous.timed as timed
    app = Flask("vectors")
    app.secret_key = "studio-dev-secret"
    serializer = app.session_interface.get_signing_serializer(app)
    for stamp in (1760900000, 1, 4294967296, 1760947200):
        timed.TimestampSigner.get_timestamp = lambda self, stamp=stamp: stamp
        for session in ({"authenticated": True}, {"authenticated": True, "pad": "x" * 200}):
            yield {"kind": "cookie", "secret": "studio-dev-secret", "time": stamp,
                   "session": session, "cookie": serializer.dumps(session)}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--backend", required=True)
    args = parser.parse_args()
    sys.path.insert(0, args.backend)
    out = sys.stdout
    for line in cookies():
        out.write(json.dumps(line) + "\n")
    for origin in ORIGINS:
        try:
            line = {"kind": "netloc", "in": origin, "out": urlsplit(origin).netloc}
        except ValueError as error:
            line = {"kind": "netloc", "in": origin, "err": str(error)}
        out.write(json.dumps(line) + "\n")
    from werkzeug.wrappers import Request
    for query in QUERIES:
        environ = {"REQUEST_METHOD": "GET", "QUERY_STRING": query, "wsgi.url_scheme": "http",
                   "SERVER_NAME": "127.0.0.1", "SERVER_PORT": "8765", "PATH_INFO": "/"}
        try:
            line = {"after": Request(environ).args.get("after")}
        except Exception as error:  # noqa: BLE001 - the oracle records whatever Werkzeug does
            line = {"err": type(error).__name__}
        out.write(json.dumps({"kind": "query", "raw": query.encode("latin-1").hex(), **line}) + "\n")
    from studio_app.launcher.frontend_assets import pack
    files = {"index.html": b"<!doctype html>", "assets/app.js": b"console.log(1)\n" * 50, "assets/e.bin": b""}
    out.write(json.dumps({"kind": "pak", "hex": pack(files).hex(),
                          "files": {k: v.hex() for k, v in files.items()}}) + "\n")
    from werkzeug.http import dump_options_header
    for name in NAMES:
        out.write(json.dumps({"kind": "disposition", "in": name,
                              "out": dump_options_header("inline", {"filename": name})}) + "\n")


if __name__ == "__main__":
    main()
