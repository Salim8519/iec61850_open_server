"""CPython's answers for the Go pycompat differential test (one JSON object per line).

Usage: python -I pycompat_vectors.py --seed N --count N
Each line has "f" (the rule), the input and either "out" or "err" (exception class and
text). Full-table lines ("table") cover every code point for isspace, isprintable,
decimal digits and casefold.
"""
import argparse
import json
import random
import struct
import sys
from datetime import datetime, timedelta, timezone

SPACES = " \t\n\x0b\x0c\r\x1c\x1d\x1e\x1f\x85\xa0        　᠎​"
DIGITS = "0123456789٣१５\U0001d7d8০๓"
INT_POOL = DIGITS * 3 + SPACES + "+-__._eExXabc\x00\x7fé\U0001f600"
TEXT_POOL = SPACES + "abcXYZ'\"\\\x00\x07\x7f\x80\x9f\xad\xe9İẞßﬃͅςᎠꭰ\U0001f600\U000e0001 ͸"
ISO_POOL = "0123456789-:T .,+Z Wé"


def error(exc):
    return {"err": str(exc), "cls": type(exc).__name__}


def call(fn, *args):
    try:
        return {"out": fn(*args)}
    except (ValueError, OverflowError, OSError, TypeError) as exc:
        return error(exc)


def text(rng, pool, limit=12):
    return "".join(rng.choice(pool) for _ in range(rng.randint(0, limit)))


def int_case(rng):
    pick = rng.random()
    if pick < 0.02:
        s = rng.choice(("", "1" * 4300, "1" * 4301, "1" * 5000 + "x", "1" * 5000 + "é", "1_" * 4400 + "1"))
    elif pick < 0.4:
        s = rng.choice(("", " ", "+", "-")) + "".join(rng.choice(DIGITS + "_") for _ in range(rng.randint(1, 25))) + rng.choice(("", " ", "　", "x"))
    else:
        s = text(rng, INT_POOL)
    result = call(lambda v: str(int(v)), s)
    return {"f": "int", "in": s, **result}


def float_value(rng):
    pick = rng.random()
    if pick < 0.3:
        return struct.unpack("<d", rng.getrandbits(64).to_bytes(8, "little"))[0]
    if pick < 0.5:
        return rng.randint(-10 ** 6, 10 ** 6) / rng.choice((2, 4, 8, 16, 1000, 200))
    if pick < 0.6:
        return rng.choice((0.5, 1.5, 2.5, -0.5, -2.5, 2.675, 0.125, 0.375, 1e300, 1.7976931348623157e308, 5e-324, -0.0, 0.0, float("nan"), float("inf"), float("-inf"), 1250.0, 1350.0, 1249.5, 6000.0, 5000.0, 5000.5))
    return rng.uniform(-1, 1) * 10 ** rng.randint(-20, 25)


def round_case(rng):
    x = float_value(rng)
    if rng.random() < 0.3:
        result = call(lambda v: str(round(v)), x)
        return {"f": "round", "x": x, "n": None, **result}
    n = rng.choice((0, 1, 2, 3, 6, 10, 17, 30, -1, -2, -3, -5, -10, -100, -308, -309, 323, 324, -400))
    result = call(lambda v: repr(round(v, n)), x)
    return {"f": "round", "x": x, "n": n, **result}


def iso_case(rng):
    pick = rng.random()
    if pick < 0.5:
        year = rng.choice((1, 2, 999, 1970, 2024, 2025, 9999, rng.randint(1, 9999)))
        s = f"{year:04d}" + rng.choice(("-01-02", "0102", "-02-29", "-13-01", "-W01", "W01", "-W53-7", "W537", "-W00", "-W01-8", "-1-2"))
        if rng.random() < 0.8:
            s += rng.choice(("T", " ", "x", "é", "-", "1"))
            s += rng.choice(("12", "12:30", "1230", "12:30:45", "123045", "12:30:45.5", "12:30:45,123456789", "24:00", "12:60", "12:30:45.", "1230451234", "12:30:4"))
            s += rng.choice(("", "", "Z", "+05:30", "-0800", "+01", "+05:30:15", "+05:30:15.000001", "+00:00:00.5", "+24:00", "-23:59:59.999999", "Zx", "+5", "+05:3", "+05:99"))
    else:
        s = text(rng, ISO_POOL, 26)
    result = call(lambda v: list_dt(datetime.fromisoformat(v)), s)
    return {"f": "fromisoformat", "in": s, **result}


def list_dt(value):
    offset = value.utcoffset()
    micro = None if offset is None else (offset // timedelta(microseconds=1))
    return [value.year, value.month, value.day, value.hour, value.minute, value.second, value.microsecond, micro]


def isoformat_case(rng):
    micro = rng.choice((0, 1, 999, 1000, 123456, 999999))
    offset = rng.choice((None, 0, 19800, -28800, 3600 * 23 + 59 * 60 + 59, -1, 1))
    offset_micro = rng.choice((0, 0, 0, 1, 500000)) if offset not in (None, 0) else 0
    tz = None if offset is None else timezone(timedelta(seconds=offset, microseconds=offset_micro))
    value = datetime(rng.randint(1, 9999), rng.randint(1, 12), rng.randint(1, 28), rng.randint(0, 23),
                     rng.randint(0, 59), rng.randint(0, 59), micro, tz)
    spec = rng.choice(("auto", "hours", "minutes", "seconds", "milliseconds", "microseconds", "bogus"))
    result = call(lambda: value.isoformat(timespec=spec))
    return {"f": "isoformat", "dt": list_dt(value), "spec": spec, **result}


def timestamp_case(rng):
    seconds = rng.choice((rng.uniform(-50000, 4e10), rng.uniform(0, 2e9), float_value(rng), rng.randint(0, 2 * 10 ** 9) + rng.choice((0.5, 0.0000005, 0.0000015, 0.0000025))))
    result = call(lambda v: datetime.fromtimestamp(v, timezone.utc).isoformat(), seconds)
    return {"f": "fromtimestamp", "x": seconds, **result}


def bytes_case(rng):
    pool = (b"a", b"\xef\xbb\xbf", b"\xc3\xa9", b"\xe2\x82\xac", b"\xf0\x9f\x98\x80", b"\xff", b"\x80", b"\xc0\xaf",
            b"\xe0\x80\x80", b"\xed\xa0\x80", b"\xf4\x90\x80\x80", b"\xe2\x82", b"\xf0\x9f", b"\xc3", b"\xf5")
    data = b"".join(rng.choice(pool) for _ in range(rng.randint(0, 6)))
    return {"f": "utf8", "hex": data.hex(), "sig": call(lambda: data.decode("utf-8-sig")),
            "replace": data.decode("utf-8", "replace")}


def text_case(rng):
    s = text(rng, TEXT_POOL)
    return {"f": "text", "in": s, "repr": repr(s), "strip": s.strip(), "lstrip": s.lstrip(),
            "rstrip": s.rstrip(), "split": s.split(), "casefold": s.casefold()}


def tables():
    chars = [c for c in range(0x110000) if not 0xD800 <= c <= 0xDFFF]
    yield {"f": "table", "name": "isspace", "codes": [c for c in chars if chr(c).isspace()]}
    yield {"f": "table", "name": "printable", "codes": ranges(c for c in chars if chr(c).isprintable())}
    yield {"f": "table", "name": "decimal", "codes": [[c, int(chr(c))] for c in chars if chr(c).isdecimal()]}
    yield {"f": "table", "name": "casefold", "codes": [[c, chr(c).casefold()] for c in chars if chr(c).casefold() != chr(c)]}


def ranges(codes):
    result = []
    for code in codes:
        if result and result[-1][1] == code - 1:
            result[-1][1] = code
        else:
            result.append([code, code])
    return result


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--seed", type=int, default=1)
    parser.add_argument("--count", type=int, default=1000)
    args = parser.parse_args()
    rng = random.Random(args.seed)
    out = sys.stdout
    for line in tables():
        out.write(json.dumps(line) + "\n")
    makers = (int_case, round_case, iso_case, isoformat_case, timestamp_case, bytes_case, text_case)
    for index in range(args.count):
        out.write(json.dumps(makers[index % len(makers)](rng)) + "\n")


if __name__ == "__main__":
    main()
