"""Write golden/xml_errors.json: parse_document() outcome for malformed and mutated inputs.

Run: python studio/engine/internal/scl/testdata/xml_errors_gen.py
Each entry is {"input": base64, "result": "OK" | str(StudioError)}. A non-StudioError
exception (none is expected) is written as "<ExceptionType>: <message>".
"""
import base64
import json
import random
import pathsetup

pathsetup.install()

from studio_app.contracts.errors import StudioError  # noqa: E402
from studio_app.scl.document import parse_document  # noqa: E402
from xml_error_snippets import snippets  # noqa: E402

SEED = 1234
MUTATIONS = 3000
VALID = ('<?xml version="1.0" encoding="UTF-8"?>\n'
         '<SCL xmlns="http://www.iec.ch/61850/2003/SCL" version="2007" revision="B">\n'
         '<Header id="h"/>\n'
         '<IED name="A1" desc="caf\u00e9 &amp; &#65; x"><!-- c --><AccessPoint name="P"><Server/></AccessPoint></IED>\n'
         "<IED name='B_2'><Private type=\"t\"><![CDATA[a<b]]><?pi d?></Private></IED>\n"
         "</SCL>\n").encode("utf-8")


def mutate(rng, data):
    """One deterministic single-step mutation: insert, delete, replace, swap or truncate."""
    data = bytearray(data)
    operation = rng.randrange(5)
    position = rng.randrange(len(data))
    if operation == 0:
        data.insert(position, rng.randrange(256))
    elif operation == 1:
        del data[position]
    elif operation == 2:
        data[position] = rng.randrange(256)
    elif operation == 3:
        other = rng.randrange(len(data))
        data[position], data[other] = data[other], data[position]
    else:
        del data[position:]
    return bytes(data)


def outcome(data):
    try:
        parse_document(data)
    except StudioError as error:
        return str(error)
    except Exception as error:  # recorded so the Go side can see it; not expected from parse_document
        return f"{type(error).__name__}: {error}"
    return "OK"


def inputs():
    rng = random.Random(SEED)
    return snippets() + [VALID] + [mutate(rng, VALID) for _ in range(MUTATIONS)]


def main():
    entries = [{"input": base64.b64encode(data).decode("ascii"), "result": outcome(data)} for data in inputs()]
    payload = json.dumps(entries).encode("ascii")
    pathsetup.write_bytes(pathsetup.GOLDEN / "xml_errors.json", payload)
    kinds = sorted({entry["result"].split(":")[0] if entry["result"].startswith("Invalid or unsafe XML")
                    else entry["result"][:40] for entry in entries})
    print(f"xml_errors: {len(entries)} entries, {len(payload)} bytes, {len(kinds)} distinct result kinds")


if __name__ == "__main__":
    main()
