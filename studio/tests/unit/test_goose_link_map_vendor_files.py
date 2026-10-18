"""Real vendor files (local only, never committed): the link map reads them without inventing data.

The owner's private test files live in `test_cids/` at the repository root of the main
checkout. The test is skipped when they are absent and asserts only structural facts,
so no customer name appears in the repository.
"""
from pathlib import Path

import pytest

from studio_app.contracts.errors import StudioError
from studio_app.domain.goose_inputs import bind_inputs
from studio_app.domain.goose_link_map import build_link_map
from studio_app.scl.goose_sources import read_goose_sources
from studio_app.scl.importer import import_scl

ROOTS = [Path(__file__).resolve().parents[3], *Path(__file__).resolve().parents[3].parents]
FOLDER = next((root / "test_cids" for root in ROOTS if (root / "test_cids").is_dir()), None)
FILES = sorted(path for path in FOLDER.iterdir() if path.suffix.lower() == ".cid") if FOLDER else []
pytestmark = pytest.mark.skipif(not FILES, reason="private vendor files are not on this machine")


def _main_ied(path):
    import re
    return re.search(rb'<IED name="([^"]+)"', path.read_bytes()).group(1).decode()


def test_every_file_reads_and_every_link_has_a_status():
    documents = [(path.read_text(encoding="utf-8-sig"), [_main_ied(path)]) for path in FILES]
    declarations = read_goose_sources(documents)
    names = [name for _, loaded in documents for name in loaded]
    result = build_link_map(declarations, [{"id": name, "name": name} for name in names])
    statuses = {link["status"] for link in result["links"]}
    assert statuses <= {"declared", "remote", "disabled_in_file", "no_dataset", "no_address", "not_sendable"}
    assert all(link["inputs"] for link in result["links"])
    # Vendor-disabled blocks exist in these files and are never presented as sendable.
    vendor = [block for block in declarations["blocks"].values() if block["vendor"]]
    assert vendor and all(block["status"] != "sendable" for block in vendor
                          if any(v.get("Function") == "Disabled" for v in block["vendor"].values()))
    assert all(item["hint"] for item in result["diagnostics"])


def test_bound_inputs_of_simulated_bay_units_resolve_to_their_own_data():
    checked = 0
    for path in FILES:
        try:
            ied = import_scl(path.read_bytes(), path.name, [_main_ied(path)])[0]
        except StudioError:  # For example an IED with several access points: not simulated here.
            continue
        for subscription in ied["goose"]["subscribes"]:
            bindings, problems = bind_inputs(ied, subscription)
            if subscription["inputs"] and all("." in item["int_addr"] and "|" not in item["int_addr"]
                                              for item in subscription["inputs"]):
                assert len(bindings) == len(subscription["inputs"]), problems
                checked += len(bindings)
    assert checked
