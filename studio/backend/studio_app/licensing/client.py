"""HTTPS client for the license server: proof of work, trial, activation, verify."""
import hashlib
import json
import ssl
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen
from studio_app.domain import trial_policy
from . import release_key

TIMEOUT_SECONDS = 20
# Server refusals and extra steps shown in the app's own plain words.
FRIENDLY_ERRORS = trial_policy.SERVER_ERRORS
MAX_DIFFICULTY = 26
_context = None


def https_context():
    """Verify the server with Windows' own certificate check, as browsers and curl do.

    Plain OpenSSL only sees roots already copied into the local store; Windows fetches
    newer roots on demand. Since 2026-10-16 the server chain ends at ISRG Root X2,
    which many PCs only hold as an expired cross-sign, so OpenSSL wrongly reported
    "certificate has expired" (ADR-0056). Expired, wrong-host or untrusted certificates
    are still refused.
    """
    global _context
    if _context is None:
        import truststore
        _context = truststore.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
    return _context


def open_https(request, timeout):
    return urlopen(request, timeout=timeout, context=https_context())


class LicenseServerError(RuntimeError):
    """A readable failure from (or while reaching) the license server."""

    def __init__(self, message, code="server_error"):
        super().__init__(message)
        self.code = code


def solve(nonce, difficulty):
    """Find a counter where SHA256(nonce + ":" + counter) starts with difficulty zero bits."""
    if not 0 < difficulty <= MAX_DIFFICULTY:
        raise LicenseServerError("The license server asked for an unexpected puzzle. Try again later.")
    base = hashlib.sha256(f"{nonce}:".encode("ascii"))
    shift = 32 - difficulty
    counter = 0
    while True:
        digest = base.copy()
        digest.update(str(counter).encode("ascii"))
        if int.from_bytes(digest.digest()[:4], "big") >> shift == 0:
            return counter
        counter += 1


def server_allowed(url, test_server=False):
    """The only address a packaged build talks to is the one baked into it (ADR-0097).

    test_server is the explicit test-only seam: https anywhere, or http on this PC. The
    release composition never sets it.
    """
    if test_server:
        return url.startswith(("https://", "http://127.0.0.1", "http://localhost"))
    return url == release_key.DEFAULT_SERVER_URL.rstrip("/")


class LicenseClient:
    def __init__(self, server_url, opener=open_https, test_server=False):
        self.server_url = (server_url or "").rstrip("/")
        self.opener = opener
        self.test_server = test_server

    def _call(self, method, path, body=None):
        if not server_allowed(self.server_url, self.test_server):
            raise LicenseServerError("No license server address is set. Open 'Server & offline options' "
                                     "and enter the address from your purchase email.", "no_server")
        data = json.dumps(body).encode("utf-8") if body is not None else None
        request = Request(self.server_url + path, data=data, method=method,
                          headers={"Content-Type": "application/json", "User-Agent": "ScadaBench"})
        try:
            with self.opener(request, timeout=TIMEOUT_SECONDS) as response:
                return json.loads(response.read(1 << 20))
        except HTTPError as error:
            try:
                detail = json.loads(error.read(1 << 16))
            except (ValueError, OSError):
                detail = {}
            code = detail.get("code", "server_error")
            raise LicenseServerError(FRIENDLY_ERRORS.get(code) or detail.get("error")
                                     or f"License server error {error.code}", code) from None
        except (URLError, OSError, ValueError) as error:
            raise LicenseServerError("Cannot reach the license server. Check the internet connection, "
                                     "or use offline activation.", "offline") from error

    def _proof(self):
        challenge = self._call("GET", "/api/v1/challenge")
        counter = solve(challenge["nonce"], int(challenge["difficulty"]))
        return {"pow_nonce": challenge["nonce"], "pow_ts": challenge["timestamp"],
                "pow_difficulty": challenge["difficulty"], "pow_mac": challenge["mac"],
                "pow_counter": counter}

    # Trial and activate return the whole answer: the certificate plus the
    # signed server time ("server_time") that trials anchor their clock to.
    def start_trial(self, hwid, parts=None, email=None, code=None):
        body = {"hwid": hwid, **self._proof()}
        if parts:
            body["parts"] = {kind: value for kind, value in parts.items() if value}
        if email:
            body.update(email=email, email_code=str(code or "").strip())
        return self._call("POST", "/api/v1/trial", body)

    def request_trial_code(self, hwid, email):
        return self._call("POST", "/api/v1/trial/email-code", {"hwid": hwid, "email": email, **self._proof()})

    def activate(self, license_key, hwid):
        body = {"license_key": license_key, "hwid": hwid, **self._proof()}
        return self._call("POST", "/api/v1/activate", body)

    def verify(self, license_key, hwid, nonce):
        """Ask whether the license is still valid; the answer must come back signed for this nonce."""
        return self._call("POST", "/api/v1/verify", {"license_key": license_key, "hwid": hwid, "nonce": nonce})
