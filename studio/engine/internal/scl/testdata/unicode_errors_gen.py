"""Write golden/utf8_errors.json: bytes.decode('utf-8-sig') outcome, as the importer stores `source`.

Run: python studio/engine/internal/scl/testdata/unicode_errors_gen.py
Each entry is {"input": base64, "result": "OK" | str(UnicodeDecodeError)}.
"""
import base64
import json
import pathsetup

BOM = b"\xef\xbb\xbf"
BAD = [
    b"\x80", b"\xbf", b"\xc0", b"\xc1", b"\xf5", b"\xf8", b"\xfc", b"\xfe", b"\xff",  # invalid start bytes
    b"\xc2", b"\xe2\x82", b"\xf0\x9f", b"\xf0\x9f\x98",  # truncated sequences
    b"\xc0\x80", b"\xc0\xaf", b"\xe0\x80\xaf", b"\xe0\x9f\xbf", b"\xf0\x80\x80\xaf", b"\xf0\x8f\xbf\xbf",  # overlongs
    b"\xed\xa0\x80", b"\xed\xbf\xbf", b"\xed\xa0\xbd\xed\xb8\x80",  # surrogates
    b"\xf4\x90\x80\x80", b"\xf7\xbf\xbf\xbf", b"\xf8\x88\x80\x80\x80",  # above U+10FFFF / 5-byte
    b"\xc2\x41", b"\xe2\x28\xa1", b"\xe2\x82\x28", b"\xf0\x28\x8c\xbc", b"\xf0\x90\x28\xbc", b"\xf0\x9f\x98\x28",
    b"\xe2\x82\xac\xff", b"\xfe\xff", b"\xff\xfe", b"\xef\xbb", b"\xef\xbf\xbe",
]
GOOD = [b"", b"A", BOM, BOM + BOM, b"\xc3\xa9", b"\xe2\x82\xac", b"\xf0\x9f\x98\x80", b"\xf4\x8f\xbf\xbf",
        b"\xef\xbf\xbf", b"\xe2\x80\xa8", b"\x00", b"\x7f", b"<SCL/>", "\u00c41 \ufdfd".encode("utf-8")]


def inputs():
    result = []
    for prefix in (b"", BOM):
        for item in GOOD:
            result.append(prefix + item)
        for item in BAD:
            for before, after in ((b"", b""), (b"ab", b""), (b"", b"cd"), (b"x\xc3\xa9", b"\n<SCL/>")):
                result.append(prefix + before + item + after)
    result.extend([b"\xef", b"\xef\xbb\xbf\xef", b"\xef\xbb\xbf\xbb", BOM[:2] + b"<SCL/>",
                   b"\xff\xfe<\x00S\x00", b"\xfe\xff\x00<", b"\xef\xbb\xbf" * 3 + b"\xff",
                   b"<SCL desc='" + b"x" * 5000 + b"\xe9'/>", b"\xe9" * 3, b"\xc3" * 3])
    return result


def outcome(data):
    try:
        data.decode("utf-8-sig")
    except UnicodeDecodeError as error:
        return str(error)
    return "OK"


def main():
    entries = [{"input": base64.b64encode(data).decode("ascii"), "result": outcome(data)} for data in inputs()]
    pathsetup.write_bytes(pathsetup.GOLDEN / "utf8_errors.json", json.dumps(entries).encode("ascii"))
    failures = sum(1 for entry in entries if entry["result"] != "OK")
    print(f"utf8_errors: {len(entries)} entries ({failures} decode errors)")


if __name__ == "__main__":
    main()
