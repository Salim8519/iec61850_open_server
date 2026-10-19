"""Write CPython reference vectors for the Go source generators (gate G1 "Generators").

Usage: studio/.venv/Scripts/python.exe generator_vectors.py OUT.json
Each row: [mode, seed, elapsed, start_elapsed, offset, amplitude, period, value_at(...)].
The noise rows cover CPython's MT19937 seeding (negative, zero, 32- and 64-bit and larger seeds).
"""
import json
import random
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[6]
sys.path.insert(0, str(ROOT / "studio" / "backend"))

from studio_app.simulation.generators import value_at  # noqa: E402


def main():
    rng = random.Random(20261020)
    rows = []
    seeds = [0, 1, -1, 7, 2**31 - 1, 2**32, 2**40 + 5, -(2**63), 2**64 + 3, 2**100 + 17]
    modes = ["noise"] * 6 + ["sine", "ramp", "step", "constant"]
    for index in range(10000):
        mode = modes[index % len(modes)]
        seed = seeds[index % len(seeds)] if index % 3 else rng.randint(-10**6, 10**6)
        elapsed = round(rng.uniform(0, 5000), rng.choice([1, 2, 3, 6, 12]))
        start = rng.choice([0, 0, 0.5, 12.3, rng.uniform(0, 100)])
        spec = {"mode": mode, "seed": seed, "start_elapsed": start, "offset": rng.uniform(-100, 100),
                "amplitude": rng.uniform(0, 1000), "period": rng.choice([0.1, 0.5, 1, 2, 10, 60, rng.uniform(0.1, 300)])}
        rows.append([mode, seed, elapsed, start, spec["offset"], spec["amplitude"], spec["period"],
                     value_at(spec, elapsed)])
    Path(sys.argv[1]).write_text(json.dumps(rows, separators=(",", ":")), encoding="utf-8")


if __name__ == "__main__":
    main()
