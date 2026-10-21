"""Random JSON documents with CPython's and Flask's exact output, for the Go pyjson test.

Usage: python -I pyjson_vectors.py --seed N --count N
Writes three ASCII lines per document: json.dumps(separators=(",", ":")) (the input the Go
side decodes), the Flask jsonify body without its final newline, and json.dumps(doc).
"""
import argparse
import json
import random
import struct
import sys

from flask import Flask

POOLS = (
    "abcxyzABC019 _-./:",
    '"\\',
    "".join(chr(code) for code in range(0x20)) + "\x7f",
    "éüßñ€£°µ×÷",
    "中文日本語한국어",
    "😀🚀𝄞𐍈",
    "<>&'",
)
SPECIAL_FLOATS = (0.0, -0.0, 1.0, -1.0, 0.1, 0.1 + 0.2, 1e15, 1e16, 1e17, 9999999999999998.0,
                  1e-4, 1e-5, 1.25e-7, 5e-324, 2.2250738585072014e-308, 1.7976931348623157e308,
                  123456.789, float("nan"), float("inf"), float("-inf"), 100.0, 0.5, 2.5)


def text(rng):
    return "".join(rng.choice(rng.choice(POOLS)) for _ in range(rng.randint(0, 12)))


def number(rng):
    pick = rng.random()
    if pick < 0.2:
        return rng.randint(-1000, 1000)
    if pick < 0.3:
        return rng.choice((-1, 1)) * rng.getrandbits(rng.randint(60, 200))
    if pick < 0.5:
        return struct.unpack("<d", rng.getrandbits(64).to_bytes(8, "little"))[0]
    if pick < 0.65:
        return rng.choice(SPECIAL_FLOATS)
    if pick < 0.8:
        return round(rng.uniform(-1e6, 1e6), rng.randint(0, 6))
    return rng.uniform(-1, 1) * 10 ** rng.randint(-30, 30)


def scalar(rng):
    pick = rng.random()
    if pick < 0.35:
        return number(rng)
    if pick < 0.75:
        return text(rng)
    return rng.choice((None, True, False))


def value(rng, depth=0):
    if depth >= 4 or rng.random() < 0.45:
        return scalar(rng)
    if rng.random() < 0.5:
        return [value(rng, depth + 1) for _ in range(rng.randint(0, 6))]
    return {text(rng): value(rng, depth + 1) for _ in range(rng.randint(0, 6))}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--seed", type=int, default=1)
    parser.add_argument("--count", type=int, default=1000)
    args = parser.parse_args()
    rng = random.Random(args.seed)
    app = Flask("pyjson_vectors")
    out = sys.stdout
    with app.app_context():
        for _ in range(args.count):
            doc = value(rng)
            body = app.json.response(doc).get_data(as_text=True)
            out.write(json.dumps(doc, separators=(",", ":")) + "\n")
            out.write(body[:-1] + "\n")
            out.write(json.dumps(doc) + "\n")


if __name__ == "__main__":
    main()
