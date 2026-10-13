"""Download and unpack the pinned Npcap SDK used to compile GOOSE support.

The SDK is a build input only. End users install the Npcap driver themselves; the
SDK is never shipped. Usage: python studio/tools/fetch_npcap_sdk.py
"""
import hashlib
import io
import ssl
import sys
import urllib.request
import zipfile
from pathlib import Path

VERSION = "1.15"
URL = f"https://npcap.com/dist/npcap-sdk-{VERSION}.zip"
SHA256 = "52c7b9fb4abee3ad9fe739bb545c3efe77b731c8e127122bdf328eafdae3ed4f"
TARGET = Path(__file__).resolve().parents[2] / ".local" / "native" / "npcap-sdk"


def main():
    if (TARGET / "Lib" / "x64" / "wpcap.lib").exists():
        print(f"Npcap SDK {VERSION} already present at {TARGET}")
        return 0
    with urllib.request.urlopen(URL, context=ssl.create_default_context(), timeout=60) as response:
        payload = response.read()
    digest = hashlib.sha256(payload).hexdigest()
    if digest != SHA256:
        print(f"Refusing the download: SHA-256 {digest} does not match the pinned {SHA256}", file=sys.stderr)
        return 1
    TARGET.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(io.BytesIO(payload)) as archive:
        for member in archive.infolist():
            destination = (TARGET / member.filename).resolve()
            if TARGET.resolve() not in destination.parents and destination != TARGET.resolve():
                print(f"Refusing unsafe archive path {member.filename}", file=sys.stderr)
                return 1
        archive.extractall(TARGET)
    print(f"Npcap SDK {VERSION} unpacked to {TARGET}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
