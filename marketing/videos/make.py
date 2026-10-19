"""Record and compose the tutorial videos:  python marketing/videos/make.py [slug ...]

Needs Microsoft Edge, ffmpeg on PATH, Playwright for Python and the built app
(studio/frontend/dist and the native workers). Output goes to marketing/videos/out/.
"""
import sys
import traceback

from compose import compose
from director import record
from lessons_basics import cid_scd, getting_started, sld
from lessons_license import activation, activation_prepare
from lessons_more import goose, real_devices, simulation
from lessons_tools import ied_explorer, scada_gateway

# Slugs are the public video addresses (/videos/<slug>.mp4); keep them stable.
LESSONS = [
    ("getting-started", "IEC 61850 and ScadaBench in one minute", getting_started, None),
    ("cid-scd", "CID and SCD files", cid_scd, None),
    ("sld", "The single-line diagram (SLD)", sld, None),
    ("simulation", "Simulate IEDs and change values", simulation, None),
    ("goose", "GOOSE and the lab network", goose, None),
    ("real-devices", "Connect to real devices", real_devices, None),
    ("ied-explorer", "IED Explorer and Find IEDs", ied_explorer, None),
    ("scada-gateway", "Be the SCADA gateway", scada_gateway, None),
    ("activation", "Activate, move or go offline", activation, activation_prepare),
]

if __name__ == "__main__":
    wanted = set(sys.argv[1:])
    record_only = "--record-only" in wanted
    wanted.discard("--record-only")
    for number, (slug, title, scenario, prepare) in enumerate(LESSONS, 1):
        if wanted and slug not in wanted:
            continue
        try:
            print(f"Recording {slug}...", flush=True)
            record(slug, scenario, prepare)
            if record_only:
                continue
            print(f"Composing {slug}...", flush=True)
            print("  ->", compose(slug, number, title), flush=True)
        except Exception:  # keep going so one broken lesson does not stop the others
            print(f"FAILED {slug}", flush=True)
            traceback.print_exc()
