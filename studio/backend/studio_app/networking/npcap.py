"""Tell the user whether the free Npcap driver is installed.

GOOSE between simulated IEDs needs no Npcap (ADR-0073); sharing GOOSE with the network,
hearing real devices and receiving Sampled Values do."""
import os
from pathlib import Path

DOWNLOAD = "https://npcap.com/#download"


def npcap_status(system_root=None):
    root = Path(system_root or os.environ.get("SystemRoot", r"C:\Windows"))
    library = root / "System32" / "Npcap" / "wpcap.dll"
    installed = library.is_file()
    return {"installed": installed, "path": str(library) if installed else "", "download": DOWNLOAD,
            "message": "Npcap is installed, so ScadaBench can share GOOSE with the network and hear real devices." if installed
            else "Sharing GOOSE with the network needs the free Npcap driver. Install it from npcap.com, "
            "then restart ScadaBench. GOOSE between your simulated IEDs works without it."}
