"""HTTPS client for the license server: proof of work, trial, activation, verify."""
import hashlib
import json
import ssl
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

TIMEOUT_SECONDS = 20
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


class LicenseClient:
    def __init__(self, server_url, opener=open_https):
        self.server_url = (server_url or "").rstrip("/")
        self.opener = opener

    def _call(self, method, path, body=None):
        if not self.server_url.startswith(("https://", "http://127.0.0.1", "http://localhost")):
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
            raise LicenseServerError(detail.get("error") or f"License server error {error.code}",
                                     detail.get("code", "server_error")) from None
        except (URLError, OSError, ValueError) as error:
            raise LicenseServerError("Cannot reach the license server. Check the internet connection, "
                                     "or use offline activation.", "offline") from error

    def _proof(self):
        challenge = self._call("GET", "/api/v1/challenge")
        counter = solve(challenge["nonce"], int(challenge["difficulty"]))
        return {"pow_nonce": challenge["nonce"], "pow_ts": challenge["timestamp"],
                "pow_difficulty": challenge["difficulty"], "pow_mac": challenge["mac"],
                "pow_counter": counter}

    def start_trial(self, hwid):
        return self._call("POST", "/api/v1/trial", {"hwid": hwid, **self._proof()})["certificate"]

    def activate(self, license_key, hwid):
        body = {"license_key": license_key, "hwid": hwid, **self._proof()}
        return self._call("POST", "/api/v1/activate", body)["certificate"]

    def transfer(self, license_key, hwid):
        body = {"license_key": license_key, "hwid": hwid, **self._proof()}
        return self._call("POST", "/api/v1/transfer", body)["certificate"]

    def verify(self, license_key, hwid):
        return self._call("POST", "/api/v1/verify", {"license_key": license_key, "hwid": hwid})
