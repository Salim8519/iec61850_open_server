"""Every code point whose Python str.casefold() is not itself (golden for pyv.Casefold)."""
import json
import sys

from common import OUT

pairs = []
for code in range(sys.maxunicode + 1):
    if 0xD800 <= code <= 0xDFFF:
        continue
    char = chr(code)
    folded = char.casefold()
    if folded != char:
        pairs.append([code, [ord(item) for item in folded]])
OUT.mkdir(parents=True, exist_ok=True)
(OUT / "casefold.json").write_text(json.dumps({"unicode": __import__("unicodedata").unidata_version,
                                               "pairs": pairs}), encoding="utf-8")
print(f"casefold.json: {len(pairs)} code points")
