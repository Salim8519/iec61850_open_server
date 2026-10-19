"""Corpus cases about raw bytes: BOMs, encodings, line endings and non-XML payloads."""
import io
import zipfile
from corpus_base import IED_DESC, case, pretty, sub, template_text

DECL = "<?xml version='1.0' encoding='utf-8'?>"
BASE = "make_template(BayController)"
E_ACUTE = chr(0xE9)
EURO = chr(0x20AC)


def _declared(encoding, text=None):
    return sub(text or template_text(), DECL, f'<?xml version="1.0" encoding="{encoding}"?>')


def _desc(text, value):
    return sub(text, IED_DESC, f'desc="{value}"')


def _raw_desc(text, raw):
    """Template bytes whose IED desc holds the given raw (possibly invalid) bytes."""
    return _desc(text, "PLACEHOLDER").encode("utf-8").replace(b"PLACEHOLDER", raw)


def _zip_bytes():
    buffer = io.BytesIO()
    with zipfile.ZipFile(buffer, "w", zipfile.ZIP_DEFLATED) as archive:
        info = zipfile.ZipInfo("station.scd", date_time=(2020, 1, 1, 0, 0, 0))
        archive.writestr(info, template_text())
    return buffer.getvalue()


def cases():
    text = template_text()
    lines = pretty(text)
    bom = b"\xef\xbb\xbf"
    latin = _desc(_declared("ISO-8859-1"), f"Caf{E_ACUTE} bay")
    cp1252 = _desc(_declared("windows-1252"), f"Price {EURO} caf{E_ACUTE}")
    return [
        case("enc_utf8_bom", "UTF-8 BOM before the declaration; source keeps no BOM", BASE, bom + text.encode()),
        case("enc_utf8_double_bom", "two UTF-8 BOMs: the second is content before the root", BASE,
             bom + bom + text.encode()),
        case("enc_no_declaration", "UTF-8 without an XML declaration", BASE, text.split("\n", 1)[1]),
        case("enc_declaration_utf8_upper", "declaration in double quotes, UTF-8 upper case", BASE,
             _declared("UTF-8")),
        case("enc_standalone_yes", "declaration with standalone='yes'", BASE,
             sub(text, DECL, '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>')),
        case("enc_xml_version_11", "declaration with version 1.1", BASE,
             sub(text, DECL, '<?xml version="1.1" encoding="UTF-8"?>')),
        case("enc_utf16le_bom", "UTF-16 LE with BOM: parses, then utf-8-sig decode fails", BASE,
             b"\xff\xfe" + _declared("UTF-16").encode("utf-16-le")),
        case("enc_utf16be_bom", "UTF-16 BE with BOM: parses, then utf-8-sig decode fails", BASE,
             b"\xfe\xff" + _declared("UTF-16").encode("utf-16-be")),
        case("enc_utf16le_no_bom", "UTF-16 LE declared, no BOM", BASE, _declared("UTF-16").encode("utf-16-le")),
        case("enc_utf16_bom_declares_utf8", "UTF-16 LE BOM while the declaration says UTF-8", BASE,
             b"\xff\xfe" + text.encode("utf-16-le")),
        case("enc_latin1_desc", "ISO-8859-1 declared, e-acute in desc as one byte", BASE, latin.encode("latin-1")),
        case("enc_latin1_ascii_only", "ISO-8859-1 declared, ASCII-only content", BASE,
             _declared("ISO-8859-1").encode("latin-1")),
        case("enc_windows1252_desc", "windows-1252 declared with euro sign and e-acute", BASE,
             cp1252.encode("cp1252")),
        case("enc_windows1252_undefined", "windows-1252 declared with undefined byte 0x81", BASE,
             _raw_desc(_declared("windows-1252"), b"x \x81 y")),
        case("enc_unknown_encoding", "declaration names an encoding expat does not know", BASE,
             _declared("x-unknown-42")),
        case("enc_utf8_invalid_byte", "declared UTF-8 with invalid bytes 0xFF 0xFE in desc", BASE,
             _raw_desc(text, b"bad \xff\xfe byte")),
        case("enc_utf8_overlong", "declared UTF-8 with an overlong slash C0 AF", BASE,
             _raw_desc(text, b"a \xc0\xaf b")),
        case("enc_usascii_charref", "US-ASCII declared with &#233; character reference", BASE,
             _desc(_declared("US-ASCII"), "Caf&#233; &#x1F600;")),
        case("enc_usascii_high_byte", "US-ASCII declared with a raw 0xE9 byte", BASE,
             _raw_desc(_declared("US-ASCII"), b"Caf\xe9")),
        case("enc_lf_pretty", "one element per line, LF endings", BASE, lines),
        case("enc_crlf_pretty", "one element per line, CRLF endings", BASE, lines.replace("\n", "\r\n")),
        case("enc_cr_pretty", "one element per line, bare CR endings", BASE, lines.replace("\n", "\r")),
        case("enc_crlf_in_val", "CRLF inside a VisString Val text (cfg rejects newline)", BASE,
             sub(text, "<Val>ScadaBench</Val>", "<Val>Scada\r\nBench</Val>")),
        case("enc_tab_newline_in_attr", "literal tab, LF and CRLF inside attribute values", BASE,
             _desc(text, "a\tb\nc\r\nd")),
        case("enc_charref_newline_in_attr", "&#9; &#10; &#13; inside attribute values", BASE,
             _desc(text, "a&#9;b&#10;c&#13;d")),
        case("enc_trailing_whitespace", "whitespace after the root element", BASE, text + "\n\n \t\r\n"),
        case("enc_trailing_garbage", "text after the root element", BASE, text + "garbage"),
        case("enc_trailing_element", "second element after the root", BASE, text + "<SCL/>"),
        case("enc_trailing_comment_pi", "comment and PI after the root", BASE, text + "<!-- end --><?done?>"),
        case("enc_nul_byte", "NUL byte inside a desc attribute", BASE, _raw_desc(text, b"a \x00 b")),
        case("enc_empty", "zero bytes", "none", b""),
        case("enc_whitespace_only", "only whitespace", "none", b" \r\n\t\n"),
        case("enc_bom_only", "only a UTF-8 BOM", "none", bom),
        case("enc_html_page", "an HTML page uploaded by mistake", "none",
             "<!DOCTYPE html>\n<html><head><title>Login</title></head><body><p>Hi</p></body></html>\n"),
        case("enc_xhtml_root", "well-formed XML with a non-SCL root", "none",
             '<?xml version="1.0"?><html xmlns="http://www.w3.org/1999/xhtml"><body/></html>'),
        case("enc_zip_bytes", "a ZIP archive renamed to .scd", BASE, _zip_bytes()),
        case("enc_json_text", "a JSON document renamed to .scd", "none", '{"SCL": []}\n'),
    ]
