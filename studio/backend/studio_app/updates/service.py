"""Daily, quiet check for a newer ScadaBench (ADR-0071).

Asks the license server's GET /api/v1/release at most once a day, sends nothing about
this PC, the user or the license, verifies the Ed25519 signature (release domain prefix),
and remembers the answer in the data folder. It never downloads or installs anything:
the UI only shows a dismissible line and a link to the download page.
"""
import base64
import json
import logging
import os
import random
import threading
import time
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.request import Request
from studio_app.licensing import ed25519_verify, release_key
from studio_app.licensing.client import open_https
from studio_app.version import __version__
from . import versions

logger = logging.getLogger(__name__)
RELEASE_DOMAIN = b"scadabench-release-v1\n"
DAY = 24 * 3600
TIMEOUT_SECONDS = 6
MANUAL_GAP = 60          # "Check now" pressed again within a minute reuses the last answer.
START_DELAY = (20, 120)  # Seconds after start: never slows the launch; spreads server load.
MAX_BODY = 16 << 10


class UpdateService:
    def __init__(self, directory, server_url, current=__version__, keys=None, opener=open_https, clock=time.time):
        self.path = Path(directory) / "update_check.json"
        self.server_url, self.current, self.opener, self.clock = server_url, current, opener, clock
        self.keys = release_key.public_keys() if keys is None else keys
        self._lock = threading.Lock()
        self._checking = False

    # ----- stored state --------------------------------------------------
    def _read(self):
        try:
            stored = json.loads(self.path.read_text(encoding="utf-8"))
        except (OSError, ValueError):
            stored = {}
        return stored if isinstance(stored, dict) else {}

    def _write(self, state):
        self.path.parent.mkdir(parents=True, exist_ok=True)
        temporary = self.path.with_name(self.path.name + ".tmp")
        temporary.write_text(json.dumps(state, indent=2), encoding="utf-8")
        os.replace(temporary, self.path)

    def status(self):
        state = self._read()
        latest = state.get("release") if isinstance(state.get("release"), dict) else None
        available = bool(latest) and versions.is_newer(latest["version"], self.current)
        important = available and bool(latest.get("min_version")) and \
            versions.is_newer(latest["min_version"], self.current)
        return {"current": self.current, "auto": state.get("auto", True) is not False,
                "checked_at": state.get("checked_at"), "error": state.get("error"),
                "checking": self._checking, "available": available, "important": important,
                "dismissed": available and state.get("dismissed") == latest["version"] and not important,
                "latest": latest if available else None}

    def configure(self, changes):
        if not isinstance(changes, dict) or set(changes) - {"auto", "dismissed"}:
            raise ValueError("Only auto and dismissed can be changed")
        with self._lock:
            state = self._read()
            if "auto" in changes:
                if not isinstance(changes["auto"], bool):
                    raise ValueError("auto must be true or false")
                state["auto"] = changes["auto"]
            if "dismissed" in changes:
                if changes["dismissed"] is not None and not versions.parse(changes["dismissed"]):
                    raise ValueError("dismissed must be a version")
                state["dismissed"] = changes["dismissed"]
            self._write(state)
        return self.status()

    # ----- checking ------------------------------------------------------
    def _fetch(self, etag):
        url = (self.server_url() or "").rstrip("/")
        if not url.startswith(("https://", "http://127.0.0.1", "http://localhost")):
            raise ConnectionError("no_server")
        headers = {"User-Agent": "ScadaBench", "Accept": "application/json"}
        if etag:
            headers["If-None-Match"] = etag
        try:
            with self.opener(Request(url + "/api/v1/release", headers=headers), timeout=TIMEOUT_SECONDS) as response:
                return response.read(MAX_BODY + 1), response.headers.get("ETag")
        except HTTPError as error:
            if error.code == 304:
                return None, etag
            raise ConnectionError("no_release" if error.code == 404 else "server_error") from None
        except (URLError, OSError) as error:
            raise ConnectionError("offline") from error

    def _verify(self, raw):
        if len(raw) > MAX_BODY:
            raise ValueError("release answer too large")
        body = json.loads(raw)
        payload, signature = body.get("payload"), body.get("signature")
        if not isinstance(payload, str) or not isinstance(signature, str):
            raise ValueError("release answer is incomplete")
        signature_bytes = base64.urlsafe_b64decode(signature + "=" * (-len(signature) % 4))
        candidates = [self.keys[body["kid"]]] if body.get("kid") in self.keys else list(self.keys.values())
        message = RELEASE_DOMAIN + payload.encode("utf-8")
        if not any(ed25519_verify.verify(key, message, signature_bytes) for key in candidates):
            raise ValueError("release signature does not match")
        return versions.clean_record(json.loads(payload))

    def check(self, manual=False):
        """Ask the server now; a failure is stored quietly as an error code, never raised."""
        with self._lock:
            state = self._read()
            if manual and self.clock() - (state.get("checked_at") or 0) < MANUAL_GAP:
                return self.status()
            self._checking = True
        try:
            raw, etag = self._fetch(state.get("etag"))
            if raw is not None:
                state["release"], state["etag"] = self._verify(raw), etag
            state["error"] = None
        except ConnectionError as error:
            state["error"] = str(error)
        except (ValueError, KeyError, TypeError) as error:
            logger.warning("Update answer refused: %s", error)
            state["error"] = "bad_answer"
        finally:
            with self._lock:
                self._checking = False
                state = {**self._read(), **{key: state.get(key) for key in ("release", "etag", "error")},
                         "checked_at": int(self.clock())}
                self._write(state)
        return self.status()

    def due(self):
        state = self._read()
        return state.get("auto", True) is not False and self.clock() - (state.get("checked_at") or 0) >= DAY

    def start_background(self):
        """Once shortly after start, then once a day while the app stays open.
        The test suite turns it off (tests/conftest.py) so tests never reach the internet."""
        if os.environ.get("SCADABENCH_UPDATE_CHECK") == "off":
            return
        def loop():
            time.sleep(random.uniform(*START_DELAY))
            while True:
                try:
                    if self.due():
                        self.check()
                except Exception:
                    logger.exception("Update check failed")
                time.sleep(3600 + random.uniform(0, 600))
        threading.Thread(target=loop, name="update-check", daemon=True).start()
