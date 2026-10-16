"""Record and compose the tutorial videos:  python marketing/videos/make.py [slug ...]

Needs Microsoft Edge, ffmpeg on PATH, Playwright for Python and the built app
(studio/frontend/dist and the native workers). Output goes to marketing/videos/out/.
"""
import sys

from compose import compose
from director import record
from lessons_basics import cid_scd, getting_started, simulation, sld
from lessons_more import activation, activation_prepare, goose, real_devices

LESSONS = [
    ("getting-started", "IEC 61850 and ScadaBench in one minute", getting_started, None),
    ("cid-scd", "CID and SCD files", cid_scd, None),
    ("sld", "The single-line diagram (SLD)", sld, None),
    ("simulation", "Simulate IEDs and change values", simulation, None),
    ("goose", "GOOSE and the lab network", goose, None),
    ("real-devices", "Connect to real devices", real_devices, None),
    ("activation", "Activate, move or go offline", activation, activation_prepare),
]

if __name__ == "__main__":
    wanted = set(sys.argv[1:])
    for number, (slug, title, scenario, prepare) in enumerate(LESSONS, 1):
        if wanted and slug not in wanted:
            continue
        print(f"Recording {slug}...", flush=True)
        record(slug, scenario, prepare)
        print(f"Composing {slug}...", flush=True)
        print("  ->", compose(slug, number, title), flush=True)
