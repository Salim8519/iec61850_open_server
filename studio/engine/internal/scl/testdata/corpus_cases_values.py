"""Corpus cases about initial values per basic type and Unicode text in descriptions."""
from corpus_base import IED_DESC, case, variant

BASE = "make_template(BayController)"
FLOAT = '<BDA name="f" bType="FLOAT32"><Val>0</Val></BDA>'
INT = '<DA name="stVal" bType="INT32" fc="ST" dchg="true"><Val>1</Val></DA>'
BOOL = '<DA name="stVal" bType="BOOLEAN" fc="ST" dchg="true"><Val>false</Val></DA>'
TIME = '<DA name="t" bType="Timestamp" fc="ST" />'
TEXT = '<DA name="vendor" bType="VisString255" fc="DC"><Val>ScadaBench</Val></DA>'
OCTET = '<BDA name="orIdent" bType="Octet64" />'
QUALITY = '<DA name="q" bType="Quality" fc="ST" qchg="true" />'
POS = '<DAI name="stVal"><Val>1</Val></DAI>'


def _float(value, kind="FLOAT32"):
    return variant((FLOAT, f'<BDA name="f" bType="{kind}"><Val>{value}</Val></BDA>'))


def _int(value, kind="INT32"):
    return variant((INT, f'<DA name="stVal" bType="{kind}" fc="ST" dchg="true"><Val>{value}</Val></DA>'))


def _bool(value):
    return variant((BOOL, f'<DA name="stVal" bType="BOOLEAN" fc="ST" dchg="true"><Val>{value}</Val></DA>'))


def _time(value):
    return variant((TIME, f'<DA name="t" bType="Timestamp" fc="ST"><Val>{value}</Val></DA>'))


def _text(value, kind="VisString255"):
    return variant((TEXT, f'<DA name="vendor" bType="{kind}" fc="DC"><Val>{value}</Val></DA>'))


def _desc(value):
    return variant((IED_DESC, f'desc="{value}"'))


FLOATS = [("1e3", "1e3"), ("1E-7", "1E-7"), ("neg_zero", "-0"), ("point_one", "0.1"),
          ("padded", " 2.5 "), ("comma", "2,5"), ("nan", "NaN"), ("inf", "INF"), ("neg_inf", "-INF"),
          ("1e309", "1e309"), ("float32_overflow", "3.4028236e38"), ("17_digits", "0.12345678901234567"),
          ("underscore", "1_000.5"), ("infinity_word", "Infinity"), ("hex", "0x1p3"), ("empty", ""),
          ("arabic_digits", chr(0x663) + "." + chr(0x665)), ("plus", "+7.25"), ("trailing_dot", "5.")]
INTS = [("int8_min_minus_1", "INT8", "-129"), ("int8_min", "INT8", "-128"), ("int8_max", "INT8", "127"),
        ("int8_max_plus_1", "INT8", "128"), ("plus_sign", "INT32", "+5"), ("leading_zero", "INT32", "05"),
        ("hex", "INT32", "0x10"), ("decimal_point", "INT32", "1.0"), ("int64_max", "INT64", "9223372036854775807"),
        ("int32u_max_plus_1", "INT32U", "4294967296"), ("underscore", "INT32", "1_000"),
        ("arabic_digit", "INT16", chr(0x663)), ("int128", "INT128", "-170141183460469231731687303715884105728"),
        ("int24u", "INT24U", "16777215"), ("padded", "INT32", "  42\t")]
BOOLS = [("upper", "TRUE"), ("title", "True"), ("one", "1"), ("zero", "0"), ("yes", "yes"), ("empty", ""),
         ("padded", " true ")]
TIMES = [("zulu", "2024-03-01T12:30:45.123Z"), ("offset", "2024-03-01T12:30:45+02:00"),
         ("no_zone", "2024-03-01T12:30:45"), ("date_only", "2024-03-01"), ("nanoseconds", "2024-03-01T12:30:45.123456789Z"),
         ("epoch_ms", "1700000000000"), ("invalid_month", "2024-13-01T00:00:00Z"), ("lower_t", "2024-03-01t12:30:45Z")]


def value_cases():
    result = []
    for name, value in FLOATS:
        result.append(case(f"val_float_{name}", f"FLOAT32 initial value {value!r}", BASE, _float(value)))
    result.append(case("val_float64_17_digits", "FLOAT64 with 17 significant digits", BASE,
                       _float("0.30000000000000004", "FLOAT64")))
    for name, kind, value in INTS:
        result.append(case(f"val_int_{name}", f"{kind} initial value {value!r}", BASE, _int(value, kind)))
    for name, value in BOOLS:
        result.append(case(f"val_bool_{name}", f"BOOLEAN initial value {value!r}", BASE, _bool(value)))
    for name, value in TIMES:
        result.append(case(f"val_time_{name}", f"Timestamp initial value {value!r}", BASE, _time(value)))
    result.extend([
        case("val_text_quote", "VisString with a double quote (cfg cannot encode)", BASE, _text('say "hi"')),
        case("val_text_newline", "VisString with a newline (cfg cannot encode)", BASE, _text("line1\nline2")),
        case("val_text_non_ascii", "VisString with non-ASCII text (JSON-escaped in cfg)", BASE,
             _text("Schaltfeld " + chr(0xDC) + "bersicht")),
        case("val_text_over_length", "VisString64 value longer than 64 bytes", BASE, _text("x" * 80, "VisString64")),
        case("val_text_backslash", "VisString with backslash and tab", BASE, _text("a\\b\tc")),
        case("val_unicode_emoji", "Unicode255 with an astral emoji", BASE,
             _text("ok " + chr(0x1F600), "Unicode255")),
        case("val_octet64", "Octet64 initial value in hex", BASE,
             variant((OCTET, '<BDA name="orIdent" bType="Octet64"><Val>0A0B0C</Val></BDA>'))),
        case("val_octet64_odd", "Octet64 initial value with odd hex length", BASE,
             variant((OCTET, '<BDA name="orIdent" bType="Octet64"><Val>ABC</Val></BDA>'))),
        case("val_objref", "ObjRef initial value", BASE, _text("BayControllerCTRL/XCBR1.Pos", "ObjRef")),
        case("val_currency", "Currency initial value", BASE, _text("EUR", "Currency")),
        case("val_quality_value", "Quality with an integer initial value", BASE,
             variant((QUALITY, '<DA name="q" bType="Quality" fc="ST" qchg="true"><Val>64</Val></DA>'))),
        case("val_dbpos_label", "Dbpos instance value given as a label", BASE,
             variant((POS, '<DAI name="stVal"><Val>on</Val></DAI>'))),
        case("val_dbpos_three", "Dbpos instance value 3 (bad-state)", BASE,
             variant((POS, '<DAI name="stVal"><Val>3</Val></DAI>'))),
        case("val_phycomaddr", "PhyComAddr attribute (vendor pattern)", BASE,
             variant((TIME, TIME + '<DA name="dstAddress" bType="PhyComAddr" fc="ST" />'))),
        case("val_entrytime", "EntryTime attribute with a value", BASE,
             variant((TIME, TIME + '<DA name="et" bType="EntryTime" fc="ST"><Val>5</Val></DA>'))),
    ])
    return result


def unicode_cases():
    return [
        case("uni_desc_emoji", "IED desc with emoji and a ZWJ family sequence", BASE,
             _desc("bay " + chr(0x1F600) + " " + chr(0x1F468) + chr(0x200D) + chr(0x1F469) + chr(0x200D) + chr(0x1F467))),
        case("uni_desc_arabic", "IED desc in Arabic (RTL)", BASE,
             _desc("".join(chr(c) for c in (0x645, 0x62D, 0x637, 0x629, 0x20, 0x631, 0x642, 0x645, 0x20, 0x661)))),
        case("uni_desc_combining", "IED desc with combining marks", BASE, _desc("e" + chr(0x301) + "a" + chr(0x308) + chr(0x20DD))),
        case("uni_desc_zwj", "IED desc with zero-width joiner and non-joiner", BASE,
             _desc("a" + chr(0x200D) + "b" + chr(0x200C) + "c" + chr(0xFEFF))),
        case("uni_desc_line_separator", "IED desc with U+2028 and U+2029", BASE,
             _desc("a" + chr(0x2028) + "b" + chr(0x2029) + "c")),
        case("uni_desc_4kb", "IED desc of 4096 characters", BASE, _desc(("Bay-" + chr(0xE9)) * 819 + "x")),
        case("uni_desc_control_charref", "IED desc with &#1; (invalid in XML 1.0)", BASE, _desc("a&#1;b")),
        case("uni_desc_noncharacter", "IED desc with U+FFFE as a character reference", BASE, _desc("a&#xFFFE;b")),
        case("uni_desc_surrogate_charref", "IED desc with a surrogate character reference", BASE, _desc("a&#xD800;b")),
        case("uni_desc_private_use", "IED desc with private-use and astral CJK characters", BASE,
             _desc(chr(0xE000) + chr(0x20000) + chr(0x10FFFD))),
    ]
