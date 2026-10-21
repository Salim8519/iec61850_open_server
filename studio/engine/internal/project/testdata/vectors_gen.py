"""Generate the golden vectors of the Go project package from the frozen Python backend.

Usage (deterministic; rewrites testdata/generated/fixtures.json and vectors_*.json):
    studio/.venv/Scripts/python.exe -I studio/engine/internal/project/testdata/vectors_gen.py

Each vector holds a function name, its JSON arguments, the fixture IEDs it runs against
and either "out" (json.dumps of the result) or "err" (class, message, code and details;
pydantic errors as errors(include_context=False, include_input=False)). IED fixtures are
built with make_template/import_scl from the BayController and Bay2 templates and FEED1 of
scd/model_substation.scd; "@source:NAME" placeholders in arguments stand for their SCL.
"""
import copy
import hashlib
import json
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
OUT = HERE / "generated"  # generated outputs: excluded from the 299-line rule by folder name
sys.path[:0] = [str(HERE), str(HERE.parents[3] / "backend")]

import cases_contract  # noqa: E402
import cases_diagram  # noqa: E402
import cases_gateway  # noqa: E402
import cases_migration  # noqa: E402
import cases_network  # noqa: E402
import cases_removal  # noqa: E402
import cases_restore  # noqa: E402
import cases_scenarios  # noqa: E402
import cases_sources  # noqa: E402
from vectors_common import build_fixtures, capture, fixture_sources, roundtrip  # noqa: E402
from vectors_run import RUNNERS  # noqa: E402

AREAS = {"contract": cases_contract, "diagram": cases_diagram, "sources": cases_sources,
         "scenarios": cases_scenarios, "migration": cases_migration, "removal": cases_removal,
         "network": cases_network, "gateway": cases_gateway, "restore": cases_restore}


def substitute(value, sources):
    """Replace "@source:NAME" strings with the fixture SCL text and {"@repeat": v, "count": n}
    with a list of n copies of v (keeps large limit vectors small)."""
    if isinstance(value, str) and value.startswith("@source:"):
        return sources[value[len("@source:"):]][0]
    if isinstance(value, list):
        return [substitute(item, sources) for item in value]
    if isinstance(value, dict) and set(value) == {"@repeat", "count"}:
        return [substitute(value["@repeat"], sources) for _ in range(value["count"])]
    if isinstance(value, dict):
        return {key: substitute(item, sources) for key, item in value.items()}
    return value


def run_area(module, fixtures, sources):
    vectors = []
    for case in module.cases(copy.deepcopy(fixtures)):
        stored = roundtrip(case["args"])
        args = substitute(roundtrip(stored), sources)
        ieds = [roundtrip(fixtures[name]) for name in case["ieds"]]
        result = capture(lambda: RUNNERS[case["fn"]](args, ieds))
        if len(result.get("out", "")) > 20000:
            out = result.pop("out")
            result.update(out_sha256=hashlib.sha256(out.encode("ascii")).hexdigest(), out_head=out[:300])
        vectors.append({"name": case["name"], "fn": case["fn"], "args": stored, "ieds": case["ieds"], **result})
    return vectors


def main():
    fixtures = build_fixtures()
    sources = fixture_sources()
    OUT.mkdir(exist_ok=True)
    templates = {name: text for name, (text, _) in sources.items() if name != "FEED1"}
    (OUT / "fixtures.json").write_text(json.dumps({"models": fixtures, "sources": templates}), encoding="ascii")
    total = 0
    for area, module in AREAS.items():
        vectors = run_area(module, fixtures, sources)
        total += len(vectors)
        errors = sum("err" in vector for vector in vectors)
        text = json.dumps(vectors, indent=1)
        (OUT / f"vectors_{area}.json").write_text(text + "\n", encoding="ascii")
        print(f"{area}: {len(vectors)} vectors ({errors} errors), {len(text) // 1024} KiB")
    print(f"total: {total}")


if __name__ == "__main__":
    main()
