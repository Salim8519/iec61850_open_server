"""Initial values per basic type and Unicode text."""
from parity.corpus.base import Case, edit, float_value, status_value, template, utf8

FLOATS = ["1e3", "1E-7", "-0", "0.1", " 2.5 ", "2,5", "NaN", "INF", "-INF", "1e309", "3.4028236e38",
          "0.30000000000000004", "1234567.0", "1e16"]
INTEGERS = {
    "INT8": ["-129", "-128", "127", "128"], "INT8U": ["-1", "0", "255", "256"],
    "INT16": ["-32769", "32767"], "INT16U": ["65535", "65536"],
    "INT32": ["-2147483649", "-2147483648", "2147483647", "2147483648", "+5", "05", "0x10", "1.0", "1_000"],
    "INT32U": ["4294967295", "4294967296"], "INT64": ["9223372036854775807", "9223372036854775808"],
}
BOOLEANS = ["true", "false", "1", "0", "TRUE", "True", "yes", ""]
OTHERS = [("Timestamp", "2026-01-01T00:00:00Z"), ("Quality", "0000"), ("Dbpos", "on"),
          ("VisString64", "V" * 65), ("VisString255", "ok"), ("Unicode255", "\U0001F600 smile"),
          ("Octet64", "0a0b"), ("Octet64", "abc"), ("ObjRef", "CorpusCTRL/LLN0.Beh"), ("Check", "00"),
          ("EntryTime", "2026-01-01T00:00:00Z"), ("Tcmd", "higher"), ("PhyComAddr", "01-0C-CD-01-00-01")]
TEXTS = [("emoji", "\U0001F600 feeder"), ("arabic", "مغذي ١"),
         ("combining", "é ä"), ("zwj", "\U0001F469‍\U0001F527"), ("line_separator", "a b"),
         ("long_4k", "D" * 4096), ("control_ref", "a&#1;b")]


def _safe(value):
    return "".join(ch if ch.isalnum() else "_" for ch in value.strip())[:24] or "empty"


def cases():
    text = template()
    desc = 'desc="Simulation template; not a manufacturer relay model"'
    for index, value in enumerate(FLOATS):
        yield Case(f"float32_{index:02d}_{_safe(value).lower()}", f"FLOAT32 initial value {value!r}", "template",
                   utf8(float_value(text, value)))
    for kind, values in INTEGERS.items():
        for index, value in enumerate(values):
            yield Case(f"{kind.lower()}_{index:02d}_{_safe(value).lower()}", f"{kind} initial value {value!r}", "template",
                       utf8(status_value(text, kind, value)))
    for index, value in enumerate(BOOLEANS):
        yield Case(f"boolean_{index:02d}_{_safe(value).lower()}", f"BOOLEAN initial value {value!r}", "template",
                   utf8(status_value(text, "BOOLEAN", value)))
    for index, (kind, value) in enumerate(OTHERS):
        yield Case(f"other_{index:02d}_{kind.lower()}", f"{kind} initial value {value[:20]!r}", "template",
                   utf8(status_value(text, kind, value)))
    for name, value in TEXTS:
        yield Case(f"desc_{name}", f"desc text: {name}", "template", utf8(edit(text, desc, f'desc="{value}"')))
