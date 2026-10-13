"""Tell the user whether the free Npcap driver needed for GOOSE is installed."""
import os
from pathlib import Path

DOWNLOAD = "https://npcap.com/#download"


def npcap_status(system_root=None):
    root = Path(system_root or os.environ.get("SystemRoot", r"C:\Windows"))
    library = root / "System32" / "Npcap" / "wpcap.dll"
    installed = library.is_file()
    return {"installed": installed, "path": str(library) if installed else "", "download": DOWNLOAD,
            "message": "Npcap is installed, so Studio can send and receive GOOSE." if installed else
            "GOOSE needs the free Npcap driver. Install it from npcap.com, then restart Studio."}
