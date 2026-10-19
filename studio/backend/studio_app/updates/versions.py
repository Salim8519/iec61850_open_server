"""Pure version and release-record rules for the update check (no I/O)."""
import re
from urllib.parse import urlsplit

_VERSION = re.compile(r"^(\d{1,4})\.(\d{1,4})\.(\d{1,4})(?:\+[0-9A-Za-z.-]+)?$")
_DATE = re.compile(r"^20\d\d-[01]\d-[0-3]\d$")
_SHA256 = re.compile(r"^[0-9a-f]{64}$")
# Same list as the license server: a release may only point at the product's own sites.
ALLOWED_HOSTS = {"scadabench.com", "www.scadabench.com", "download.scadabench.com"}
MAX_NOTES = 600


def parse(version):
    """(major, minor, patch) for "0.1.13" or a development "0.1.13+dev"; None when unreadable."""
    match = _VERSION.match(version or "")
    return tuple(int(part) for part in match.groups()) if match else None


def is_newer(candidate, current):
    """True only when both versions are readable and candidate is strictly newer."""
    new, old = parse(candidate), parse(current)
    return bool(new and old and new > old)


def _safe_url(value):
    if not isinstance(value, str):
        return False
    parts = urlsplit(value)
    return parts.scheme == "https" and parts.hostname in ALLOWED_HOSTS and not parts.username and not parts.port


def _download(value, required):
    if value is None and not required:
        return None
    if not isinstance(value, dict) or not _safe_url(value.get("url")) or \
            not isinstance(value.get("sha256"), str) or not _SHA256.match(value["sha256"]):
        raise ValueError("release download is not a checked scadabench.com link")
    return {"url": value["url"], "sha256": value["sha256"]}


def clean_record(payload):
    """Validate the signed release record again on this PC; return only known fields."""
    if not isinstance(payload, dict):
        raise ValueError("release record is not an object")
    version, minimum = payload.get("version"), payload.get("min_version") or ""
    notes = payload.get("notes", "")
    if not parse(version) or "+" in version or (minimum and not parse(minimum)):
        raise ValueError("release version is unreadable")
    if not isinstance(payload.get("date"), str) or not _DATE.match(payload["date"]):
        raise ValueError("release date is unreadable")
    if not isinstance(notes, str) or len(notes) > MAX_NOTES or not _safe_url(payload.get("page")):
        raise ValueError("release notes or page are not acceptable")
    return {"version": version, "date": payload["date"], "min_version": minimum, "notes": notes.strip(),
            "page": payload["page"], "installer": _download(payload.get("installer"), True),
            "portable": _download(payload.get("portable"), False)}
