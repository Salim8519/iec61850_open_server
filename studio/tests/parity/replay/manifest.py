"""Privacy guard: only public, hash-listed inputs may be uploaded while recording.

golden/generated/public_manifest.json lists every allowed input (alias, repo path, sha256, IED names).
`python -m parity.replay.manifest --write` regenerates it from PUBLIC_INPUTS; customer files
(test_cids/, anything not listed) can never be recorded.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re

REPO = Path(__file__).resolve().parents[4]
GOLDEN = Path(__file__).resolve().parents[1] / "golden"
MANIFEST = GOLDEN / "generated" / "public_manifest.json"
CORPUS = Path(__file__).resolve().parents[1] / "corpus" / "fixtures"
PUBLIC_INPUTS = {
    "demo": "marketing/videos/demo/scadabench_demo.scd",
    "model_substation": "scd/model_substation.scd",
    "open_substation": "scd/open_substation.scd",
    "protection_relay": "scd/protection_relay.scd",
}
IED_NAME = re.compile(rb"<(?:\w+:)?IED\b[^>]*?\bname=\"([^\"]*)\"")


class PrivacyError(RuntimeError):
    """An input outside the public manifest reached the recorder."""


def sha256(data):
    return hashlib.sha256(data).hexdigest()


def ied_names(data):
    return sorted({match.decode("utf-8", "replace") for match in IED_NAME.findall(data)})


def build():
    entries = []
    for alias, relative in sorted(PUBLIC_INPUTS.items()):
        data = (REPO / relative).read_bytes()
        entries.append({"alias": alias, "path": relative, "sha256": sha256(data), "ieds": ied_names(data)})
    corpus = CORPUS / "manifest.json"
    if corpus.is_file():
        from parity.corpus.scl_corpus import bytes_for
        for case in json.loads(corpus.read_text(encoding="utf-8"))["cases"]:
            data = bytes_for(case["case"])
            path = None if case["generated"] else (CORPUS / case["file"]).relative_to(REPO).as_posix()
            entries.append({"alias": "corpus:" + case["case"], "path": path, "sha256": sha256(data),
                            "ieds": ied_names(data)})
    return {"version": 1, "inputs": entries}


class Manifest:
    def __init__(self, document=None):
        document = document or json.loads(MANIFEST.read_text(encoding="utf-8"))
        self.inputs = {entry["alias"]: entry for entry in document["inputs"]}
        self.hashes = {entry["sha256"] for entry in document["inputs"]}
        self.names = {name for entry in document["inputs"] for name in entry["ieds"]}

    def load(self, alias):
        """Bytes of a listed input; the file must still match its recorded hash."""
        if alias not in self.inputs:
            raise PrivacyError(f"input {alias!r} is not in the public manifest")
        entry = self.inputs[alias]
        if entry["path"] is None:
            from parity.corpus.scl_corpus import bytes_for
            data = bytes_for(alias.split(":", 1)[1])
        else:
            data = (REPO / entry["path"]).read_bytes()
        self.check_upload(data, alias)
        return data

    def check_upload(self, data, label):
        if sha256(data) not in self.hashes:
            raise PrivacyError(f"refusing to record {label}: sha256 {sha256(data)} is not a public input")

    def check_names(self, names, allowed_extra=()):
        unknown = sorted(set(names) - self.names - set(allowed_extra))
        if unknown:
            raise PrivacyError(f"refusing to record: project shows IEDs outside the manifest ({len(unknown)})")


def main(argv=None):
    parser = argparse.ArgumentParser(description="Write golden/public_manifest.json")
    parser.add_argument("--write", action="store_true")
    args = parser.parse_args(argv)
    document = build()
    if args.write:
        MANIFEST.parent.mkdir(parents=True, exist_ok=True)
        MANIFEST.write_text(json.dumps(document, indent=2) + "\n", encoding="utf-8")
    print(f"{len(document['inputs'])} public inputs")


if __name__ == "__main__":
    main()
