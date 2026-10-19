"""Random CSV texts through CPython's csv module (reader strict/lax, writer)."""
import csv
import io
import os
import sys

sys.path.insert(0, os.environ["STUDIO_ORACLE_LIB"])
from oracle_lib import case, rng, seed  # noqa: E402

PIECES = ["a", "b", " ", ",", '"', '""', "\r", "\n", "\r\n", "x y", "\x00", "é", "\U0001F600", "1.5", ""]


def read(text, strict):
    return list(csv.reader(io.StringIO(text), strict=strict))


def write(rows):
    out = io.StringIO()
    writer = csv.writer(out, lineterminator="\r\n")
    for row in rows:
        writer.writerow(row)
    return out.getvalue()


random = rng(seed())
for number in range(int(sys.argv[2]) if len(sys.argv) > 2 else 3000):
    text = "".join(random.choice(PIECES) for _ in range(random.randint(0, 24)))
    strict = random.random() < 0.5
    case("read", [text, strict], lambda: read(text, strict))
    rows = [[random.choice(PIECES + [None, 7, 2.5, True]) for _ in range(random.randint(1, 4))]
            for _ in range(random.randint(1, 3))]
    case("write", [rows], lambda: write(rows))
