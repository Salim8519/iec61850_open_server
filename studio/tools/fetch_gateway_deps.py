"""Fetch the pinned protocol libraries of the gateway simulator (ADR-0057).

- lib60870-C v2.3.6 (IEC 60870-5-104 slave), MZ Automation, GPLv3 or commercial;
- opendnp3 3.1.2 (DNP3 outstation), Apache-2.0, plus the three header-only
  dependencies its CMake normally downloads (asio, exe4cpp, ser4cpp).

Everything lands under .local/native and is verified against pinned commits and
hashes, so a build never downloads at configure time.
Usage: python studio/tools/fetch_gateway_deps.py
"""
import hashlib
import io
import ssl
import subprocess
import sys
import urllib.request
import zipfile
from pathlib import Path

NATIVE = Path(__file__).resolve().parents[2] / ".local" / "native"
REPOSITORIES = {
    "lib60870": ("https://github.com/mz-automation/lib60870.git", "v2.3.6", "63028a9287da3b075d4225effcf1dfaf5ea47ed1"),
    "opendnp3": ("https://github.com/dnp3/opendnp3.git", "3.1.2", "26b4c01e4839bbbda8866655e086471c4917ee53"),
}
ARCHIVES = {
    "asio": ("https://github.com/chriskohlhoff/asio/archive/asio-1-16-0.zip",
             "6bdd33522d5b95b36445abb2072a481f7ce15402"),
    "exe4cpp": ("https://github.com/automatak/exe4cpp/archive/fb878a4de598ba9d6e4338afebf83f96e03af1b8.zip",
                "18b141e8cf09dc8d28cc62dd5fa2920670d501bd"),
    "ser4cpp": ("https://github.com/automatak/ser4cpp/archive/3c449734dc530a8f465eb0982de29165cc4e23d5.zip",
                "937b759b7cc80180da26b47037e796b59798a672"),
}


def repository(name, url, tag, commit):
    target = NATIVE / name
    if not target.exists():
        subprocess.run(["git", "clone", "--depth", "1", "--branch", tag, url, str(target)], check=True)
    head = subprocess.run(["git", "-C", str(target), "rev-parse", "HEAD"], check=True,
                          capture_output=True, text=True).stdout.strip()
    if head != commit:
        raise SystemExit(f"{name}: commit {head} is not the pinned {commit}; remove {target} and run again")
    print(f"{name} {tag} ok ({commit[:12]})")


def archive(name, url, sha1):
    target = NATIVE / "opendnp3-deps" / name
    if target.exists():
        print(f"{name} ok")
        return
    with urllib.request.urlopen(url, context=ssl.create_default_context(), timeout=120) as response:
        payload = response.read()
    digest = hashlib.sha1(payload).hexdigest()
    if digest != sha1:
        raise SystemExit(f"Refusing {name}: SHA-1 {digest} does not match the pinned {sha1}")
    with zipfile.ZipFile(io.BytesIO(payload)) as content:
        root = content.namelist()[0].split("/")[0]
        staging = target.parent / f".{name}"
        for member in content.infolist():
            destination = (staging / member.filename).resolve()
            if staging.resolve() not in destination.parents and destination != staging.resolve():
                raise SystemExit(f"Refusing unsafe archive path {member.filename}")
        content.extractall(staging)
        (staging / root).rename(target)
        staging.rmdir()
    print(f"{name} unpacked")


def main():
    for name, (url, tag, commit) in REPOSITORIES.items():
        repository(name, url, tag, commit)
    (NATIVE / "opendnp3-deps").mkdir(parents=True, exist_ok=True)
    for name, (url, sha1) in ARCHIVES.items():
        archive(name, url, sha1)
    return 0


if __name__ == "__main__":
    sys.exit(main())
