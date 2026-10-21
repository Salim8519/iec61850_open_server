"""datetime.fromisoformat and TimeAnchor vectors (time_anchor.py) for make_vectors.py."""
import random
import tempfile
from datetime import datetime
from pathlib import Path

from vector_common import b64
from studio_app.licensing.time_anchor import TimeAnchor

BASES = ["2026-10-14T09:54:34.341681+00:00", "2026-10-14T09:54:34+00:00", "2026-01-01T00:00:00+00:00",
         "2026-10-14 09:54:34.5-03:30", "20261014T095434Z", "2026-W42-3T10:00Z", "2026W423T10Z",
         "2026-W53-1", "2026W01", "2026-10-14T10:00X+00:00", "2026-10-14é09:54+01:00",
         "2026-10-14T10:00:00,123456789+05:45:30.25", "0001-01-01T00:00:00+01:00",
         "9999-12-31T23:59:59.999999-01:00", "2026-02-29T00:00Z", "2024-02-29T23:59:59.999999+00:00",
         "2026-10-14T24:00Z", "2026-10-14T10:00:00+24:00", "2026-10-14T10:00:00+23:59:59.999999",
         "2026-10-14T1000+0100", "2026-10-14T10:00:00:5+00:00", "2026-10-14T10:00:00.+00:00",
         "2026-10-14T10Z", "2026-10-14T10:0Z", "2026-10-14T", "2026-10-14T10:00Zx", "1969-12-31T23:59:59.5+00:00",
         "2026-10-14T10:00　+00:00", "2026-10-14\U0001F600" "10:00Z", "2026-W01-8", "2026-W00", "2020-W53"]
ALPHABET = "0123456789-:T+Z.,W éX　"


def classify(text):
    try:
        value = datetime.fromisoformat(text)
    except ValueError:
        return {"text": text, "error": True}
    if value.tzinfo is None:
        return {"text": text, "naive": [value.year, value.month, value.day, value.hour, value.minute,
                                        value.second, value.microsecond]}
    return {"text": text, "ts": int(value.timestamp())}


def mutate(rng, text):
    chars = list(text)
    for _ in range(rng.randint(1, 3)):
        kind = rng.random()
        position = rng.randrange(len(chars) + 1)
        if kind < 0.4 or not chars:
            chars.insert(position, rng.choice(ALPHABET))
        elif kind < 0.7:
            chars[min(position, len(chars) - 1)] = rng.choice(ALPHABET)
        else:
            del chars[min(position, len(chars) - 1)]
    return "".join(chars)


def iso_cases():
    rng = random.Random(8601)
    texts = list(BASES)
    texts += [mutate(rng, rng.choice(BASES)) for _ in range(3000)]
    texts += [base[:cut] for base in BASES[:6] for cut in range(len(base))]
    return [classify(text) for text in dict.fromkeys(texts)]


def runs_texts():
    line = '{"kind":"run_started","run_id":"%s","at":"%s","application":"0.1.0"}\n'
    big = "".join(line % ("r%04d" % index, "2026-10-%02dT10:00:00+00:00" % (1 + index % 28)) for index in range(200))
    return [b"", (line % ("a", "2026-01-01T00:00:00+00:00")).encode(),
            (line % ("a", "2026-10-14T09:54:34.341681+00:00") + line % ("b", "2026-03-01T00:00:00+04:00")).encode(),
            ('{"at":"2099-01-01T00:00:00+00:00"}\n' + big).encode(), big.encode(),
            b'{"at":"2026-10-14\xff10:00+00:00"} {"at":"2026-10-14\xff\xfe10:00+00:00"}',
            b'{"at":"bad"}{"at":"2026-10-15T00:00:00Z', b'"at":"1969-12-31T00:00:00+00:00"',
            b'"at":"2026-10-14T10:00:00+00:00" "at":"2027-10-14T10:00:00-12:00"']


def anchor_cases():
    anchors = [None, b"", b"1760000000", b" 1760000000\r\n", b"17_60", b"abc", b"-5", b"\xd9\xa3", b"9" * 40]
    cases = []
    with tempfile.TemporaryDirectory() as folder:
        anchor_file, runs_file = Path(folder) / ".time_anchor", Path(folder) / "runs.jsonl"
        for runs in runs_texts():
            for anchor in anchors:
                for path, data in ((runs_file, runs), (anchor_file, anchor)):
                    path.unlink(missing_ok=True)
                    if data is not None:
                        path.write_bytes(data)
                case = {"runs": b64(runs), "anchor": None if anchor is None else b64(anchor),
                        "max_seen": TimeAnchor(anchor_file, runs_file).max_seen(),
                        "anchor_only": TimeAnchor(anchor_file).max_seen()}
                advanced = {}
                for now in (0, 1760000000, 1760000001, 1900000000):
                    if anchor is None:
                        anchor_file.unlink(missing_ok=True)
                    else:
                        anchor_file.write_bytes(anchor)
                    TimeAnchor(anchor_file).advance(now)
                    advanced[str(now)] = b64(anchor_file.read_bytes()) if anchor_file.exists() else None
                case["advance"] = advanced
                cases.append(case)
    return cases
