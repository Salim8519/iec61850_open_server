"""Write generated/vectors.json: CPython's answers for the pytext rules (whitespace, int(), JSON).

Usage (from the worktree root):
  studio/.venv/Scripts/python.exe -I -B studio/engine/internal/licensing/pytext/testdata/make_vectors.py
The Go tests read vectors.json only; Python is not needed to run them.
"""
import base64
import json
import random
import sys
import tempfile
import unicodedata
from pathlib import Path

HERE = Path(__file__).resolve().parent


def b64(data):
    return base64.b64encode(data).decode("ascii")


def int_cases():
    texts = ["0", "7", "-7", "+7", " 12 ", "\t-3\n", "007", "1_000", "1__0", "_1", "1_", "+_1",
             "- 1", "", " ", "1.0", "0x10", "٣١", "１２", "\U0001d7d8", "1e3",
             " 12　", "\x1c5\x1f", "9" * 30, "-" + "9" * 25, "1 2", "²", "Ⅷ",
             "12\x00", "--1", "+", "-", "๓", "၀၁", "1" * 4300, "1" * 4301]
    cases = []
    for text in texts:
        try:
            cases.append({"text": text, "value": str(int(text))})
        except ValueError:
            cases.append({"text": text, "value": None})
    return cases


def random_value(rng, depth=0):
    pick = rng.random()
    if depth < 3 and pick < 0.2:
        return [random_value(rng, depth + 1) for _ in range(rng.randint(0, 3))]
    if depth < 3 and pick < 0.4:
        return {rng.choice(["a", "b", "z", "é", "key", "\n"]) + str(index): random_value(rng, depth + 1)
                for index in range(rng.randint(0, 3))}
    return rng.choice([None, True, False, 0, -5, 2**70, 1.5, 1e-7, 1e16, float("nan"), "",
                       "text", "é😀\n\"\\", 1760000000])


def indent_cases():
    rng = random.Random(61850)
    values = [{}, [], {"a": {}}, {"a": []}, [[]], {"payload": "{\"a\":1}", "signature": "x-_"},
              {"server_url": "https://license.example"}, None, 1, "s"]
    values += [random_value(rng) for _ in range(300)]
    return [{"input": json.dumps(value, separators=(",", ":")), "output": json.dumps(value, indent=2)}
            for value in values]


def loads_cases():
    raw = [b'{"a":1}', b'\xef\xbb\xbf{"a":1}', b'{"a":"\xff"}', b' [1, 2] ', b'nope', b'{"a":NaN}',
           '{"é":"€"}'.encode("utf-8"), b'\xef\xbb\xbf', b'{"a":1}\n']
    cases = []
    for data in raw:
        entry = {"data": b64(data)}
        try:
            entry["bytes"] = json.dumps(json.loads(data), separators=(",", ":"))
        except ValueError:
            entry["bytes"] = None
        try:
            entry["str"] = json.dumps(json.loads(data.decode("utf-8")), separators=(",", ":"))
        except ValueError:
            entry["str"] = None
        cases.append(entry)
    return cases


def write_cases():
    texts = ["{}", "{\n  \"a\": 1\n}", "1760000000", "a\nb\n", "\n\n", "x\r\ny"]
    cases = []
    with tempfile.TemporaryDirectory() as folder:
        target = Path(folder) / "file.json"
        for text in texts:
            target.write_text(text, encoding="utf-8")
            cases.append({"text": text, "disk": b64(target.read_bytes())})
    return cases


def main():
    space = [code for code in range(sys.maxunicode + 1) if chr(code).isspace()]
    digits = [[code, unicodedata.decimal(chr(code))] for code in range(sys.maxunicode + 1)
              if unicodedata.category(chr(code)) == "Nd"]
    vectors = {"python": sys.version.split()[0], "unicode": unicodedata.unidata_version,
               "space": space, "digits": digits, "ints": int_cases(), "indent": indent_cases(),
               "loads": loads_cases(), "write_text": write_cases()}
    (HERE / "generated").mkdir(exist_ok=True)
    (HERE / "generated" / "vectors.json").write_text(json.dumps(vectors, indent=1) + "\n", encoding="utf-8", newline="\n")


if __name__ == "__main__":
    main()
