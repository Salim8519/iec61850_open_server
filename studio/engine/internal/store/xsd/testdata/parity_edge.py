"""Malformed, encoding and size edge inputs for the studio_xsd parity test (README plan 3-4).

lxml_verdict() is the lxml part of scl/schema_validation.py's validate_schema (the part the
helper reproduces); the engine picks the schema and refuses the DOCTYPE itself.
"""
from lxml import etree

NS = b'xmlns="http://www.iec.ch/61850/2003/SCL"'
ROOT4 = b'<SCL ' + NS + b' version="2007" revision="B" release="4">'
# Declared encodings: the README list, the earlier probe list, and names that tell
# GNU libiconv 1.17 (no extra encodings, DOS code pages on) from other iconv builds.
ENCODINGS = [
    "UTF-8", "utf8", "UTF-16", "UTF-16LE", "UTF-16BE", "UTF-32", "UCS-2", "UCS-4", "ISO-8859-1",
    "latin1", "US-ASCII", "ASCII", "ANSI_X3.4-1968", "ISO-8859-2", "ISO-8859-7", "ISO-8859-15",
    "ISO-8859-16", "windows-1252", "cp1252", "CP1256", "windows-1256", "Shift_JIS", "SJIS",
    "WINDOWS-31J", "CP932", "EUC-JP", "ISO-2022-JP", "GB2312", "GBK", "CP936", "GB18030",
    "GB18030:2005", "GB18030:2022", "BIG5", "BIG5-HKSCS", "BIG5-HKSCS:2008", "CP950", "EUC-KR",
    "CP949", "JOHAB", "EUC-TW", "ISO-IR-165", "HZ", "KOI8-R", "KOI8-U", "KOI8-T", "CP437",
    "CP850", "CP866", "CP1125", "CP1131", "IBM037", "IBM-1047", "EBCDIC-US", "macintosh",
    "MacRoman", "GEORGIAN-PS", "GEORGIAN-ACADEMY", "TIS-620", "CP874", "UTF-7", "CP65001",
    "TCVN", "VISCII", "ARMSCII-8", "JAVA", "C99", "CP1133", "MULELAO-1", "PT154", "RK1048",
    "TDS565", "ATARIST", "RISCOS-LATIN1", "EUC-JISX0213", "BIG5-2003", "HP-ROMAN8", "NEXTSTEP",
    "WCHAR_T", "char", "", "bogus-enc"]
SAMPLE = "café ÄÖÜß € Ωβ Жя 中文 日本 한국 ქ"
HIGH = bytes(range(0xA0, 0x100))


def lxml_verdict(source, version):
    from studio_app.scl.schema_validation import load_schema
    parser = etree.XMLParser(resolve_entities=False, no_network=True, load_dtd=False, huge_tree=False)
    try:
        document = etree.fromstring(source, parser)
        if document.getroottree().docinfo.doctype:
            return {"doctype": True}
        schema = load_schema(version)
        valid = schema.validate(document)
        errors = [{"line": item.line, "message": item.message} for item in list(schema.error_log)[:100]]
        return {"status": "valid" if valid else "invalid", "errors": errors}
    except etree.XMLSyntaxError as error:
        return {"status": "invalid", "errors": [{"line": error.lineno, "message": str(error)}]}


def doc(body=b'<Header id="h"/>', declaration=b'<?xml version="1.0" encoding="UTF-8"?>\n'):
    return declaration + ROOT4 + b"\n" + body + b"\n</SCL>\n"


def malformed():
    yield "valid", doc()
    yield "empty", b""
    yield "blank", b"   \n\n"
    yield "junk-before", b"junk" + doc()
    yield "decl-not-first", b"\n" + doc()
    yield "unclosed", doc()[:-8]
    yield "mismatched", doc(b'<Header id="h"></Head>')
    yield "duplicate-attribute", doc(b'<Header id="h" id="i"/>')
    yield "after-root", doc() + b"<more/>"
    yield "undefined-entity", doc(b'<Header id="&foo;"/>')
    yield "undefined-entity-text", doc(b'<Header id="h"><Text>&foo; &bar;</Text></Header>')
    yield "undefined-prefix", doc(b'<x:Header id="h"/>')
    yield "nul", doc(b'<Header id="h\x00"/>')
    yield "bad-charref", doc(b'<Header id="&#0;"/>')
    yield "latin1-undeclared", doc(b'<Header id="caf\xe9"/>', b"")
    yield "latin1-declared-utf8", doc(b'<Header id="caf\xe9"/>')
    yield "invalid-utf8-attribute", doc(b'<Header id="h" nameStructure="\xc3\x28\xff"/>')
    yield "latin1-declared", doc(b'<Header id="caf\xe9" nameStructure="\xe9"/>',
                                 b'<?xml version="1.0" encoding="ISO-8859-1"?>\n')
    yield "cdata", doc(b'<Header id="h"><![CDATA[x < y]]></Header>')
    yield "cdata-text", doc(b'<Header id="h"><Text><![CDATA[a]]>b</Text></Header>')
    yield "doctype", b"<!DOCTYPE SCL>\n" + doc(declaration=b"")
    yield "doctype-entity", b'<!DOCTYPE SCL [<!ENTITY e "x">]>\n' + doc(b'<Header id="&e;"/>', b"")
    yield "comment-pi", doc(b'<!-- c --><?pi x?><Header id="h"/>')
    yield "wrong-namespace", b'<SCL xmlns="urn:x" version="2007" revision="B" release="4"/>'
    yield "no-namespace", b"<SCL/>"
    yield "bad-enum", doc(b'<Header id="h" nameStructure="Bogus"/>')
    yield "many-errors", doc(b'<Header id="h"/>' + b'<IED/>' * 150)
    yield "big-lines-schema", doc(b"\n" * 70000 + b'<Header id="h" nameStructure="X"/>')
    yield "big-lines-syntax", doc(b"\n" * 70000 + b'<Header id="h"')
    for bom, codec in ((b"\xff\xfe", "utf-16-le"), (b"\xfe\xff", "utf-16-be"),
                       (b"\xff\xfe\x00\x00", "utf-32-le"), (b"\x00\x00\xfe\xff", "utf-32-be")):
        text = doc(b'<Header id="caf\xc3\xa9"/>', b"").decode("utf-8")
        yield f"{codec}-bom", bom + text.encode(codec)
        yield f"{codec}-no-bom", text.encode(codec)
        declared = '<?xml version="1.0" encoding="%s"?>\n' % codec.upper().replace("-LE", "").replace("-BE", "")
        yield f"{codec}-declared-bom", bom + (declared + text).encode(codec)
        yield f"{codec}-declared-no-bom", (declared + text).encode(codec)
    yield "utf8-bom", b"\xef\xbb\xbf" + doc()


def encoded_value(name):
    """Bytes for SAMPLE in `name` when Python knows the codec, else raw high bytes."""
    try:
        return SAMPLE.encode(name, errors="ignore")
    except (LookupError, ValueError):
        return HIGH


def encodings():
    for name in ENCODINGS:
        declaration = b'<?xml version="1.0" encoding="%s"?>' % name.encode("ascii")
        yield f"encoding:{name}:minimal", declaration + b"<SCL/>"
        if name.upper().startswith(("UTF-16", "UTF-32", "UCS-", "UTF-7", "EBCDIC", "IBM")):
            continue
        value = encoded_value(name).replace(b'"', b"").replace(b"&", b"").replace(b"<", b"")
        yield f"encoding:{name}:value", doc(b'<Header id="h" nameStructure="' + value + b'"/>',
                                            declaration + b"\n")
        yield f"encoding:{name}:high", doc(b'<Header id="h" nameStructure="' + HIGH + b'"/>',
                                           declaration + b"\n")


def large():
    padding = b'<Private type="t">' + b"x" * 200 + b"</Private>\n"
    body = padding * (20_000_000 // len(padding)) + b'<Header id="h" nameStructure="X"/>'
    yield "large-20mb", doc(body)


def cases():
    for label, data in malformed():
        yield label, "SCL2007B4", data
    for label, data in encodings():
        yield label, "SCL2007B4", data
    yield "schema-2003", "SCL2003", b'<SCL ' + NS + b'><Header id="h" nameStructure="X"/></SCL>'
    yield "schema-2007B", "SCL2007B", b'<SCL ' + NS + b' version="2007" revision="B"><Header/></SCL>'
    for label, data in large():
        yield label, "SCL2007B4", data
