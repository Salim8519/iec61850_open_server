"""Write the synthetic SCL corpus: corpus/<case>.scd (raw bytes) and corpus/manifest.json.

Run: python studio/engine/internal/scl/testdata/corpus_gen.py
`expect` is measured with the frozen Python importer (all IEDs): reject when import_scl
raises, warn when any imported IED carries warnings, accept otherwise.
"""
import hashlib
import json
import sys
import pathsetup

pathsetup.install()

import corpus_cases_blocks as blocks  # noqa: E402
import corpus_cases_bytes as raw  # noqa: E402
import corpus_cases_comm as comm  # noqa: E402
import corpus_cases_samples as samples  # noqa: E402
import corpus_cases_types as types  # noqa: E402
import corpus_cases_values as values  # noqa: E402
import corpus_cases_xml as xml  # noqa: E402
from studio_app.scl.importer import import_scl  # noqa: E402


def all_cases():
    groups = [raw.cases(), xml.xml_cases(), xml.edition_cases(), types.name_cases(), types.type_cases(),
              types.enum_cases(), values.value_cases(), values.unicode_cases(), comm.comm_cases(),
              comm.sv_cases(), blocks.report_cases(), blocks.extref_cases(), blocks.server_cases(),
              blocks.substation_cases(), samples.sample_cases()]
    cases = [item for group in groups for item in group]
    names = [item["case"] for item in cases]
    duplicates = sorted({name for name in names if names.count(name) > 1})
    if duplicates:
        raise SystemExit(f"duplicate corpus case names: {duplicates}")
    return sorted(cases, key=lambda item: item["case"])


def expectation(data):
    try:
        ieds = import_scl(data, "case.scd")
    except Exception as error:  # the corpus records every failure class, not only StudioError
        return "reject", f"{type(error).__name__}: {error}"
    return ("warn" if any(ied["warnings"] for ied in ieds) else "accept"), ""


def main():
    manifest = []
    target = pathsetup.CORPUS
    if target.exists():
        for stale in target.glob("*.scd"):
            stale.unlink()
    for item in all_cases():
        pathsetup.write_bytes(target / f"{item['case']}.scd", item["data"])
        expect, reason = expectation(item["data"])
        manifest.append({"case": item["case"], "purpose": item["purpose"], "base": item["base"],
                         "expect": expect, "sha256": hashlib.sha256(item["data"]).hexdigest()})
        if "--verbose" in sys.argv:
            print(ascii(f"{item['case']}: {expect} {reason}")[1:240])
    pathsetup.write_bytes(target / "manifest.json", json.dumps(manifest).encode("ascii"))
    counts = {key: sum(1 for entry in manifest if entry["expect"] == key) for key in ("accept", "warn", "reject")}
    print(f"corpus: {len(manifest)} cases {counts}")


if __name__ == "__main__":
    main()
