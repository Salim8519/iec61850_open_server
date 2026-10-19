"""Bytes, encodings, line endings, sizes and the public files as they are."""
from parity.corpus.base import Case, public, template, utf8

DECLARATION = "<?xml version='1.0' encoding='utf-8'?>"
MAX_XML = 20 * 1024 * 1024


def _declared(text, encoding):
    return text.replace(DECLARATION, f"<?xml version='1.0' encoding='{encoding}'?>", 1)


def _desc(text, value):
    return text.replace('desc="Simulation template; not a manufacturer relay model"', f'desc="{value}"', 1)


def _padded(text, size):
    """The sample IED padded with a comment to exactly `size` bytes."""
    head, tail = text.split("</SCL>")
    filler = size - len(utf8(head)) - len("<!--") - len("-->") - len("</SCL>")
    return utf8(head) + b"<!--" + b"x" * filler + b"-->" + b"</SCL>"


def cases():
    text = template()
    body = text.split("\n", 1)[1]
    yield from (Case("public_" + alias, f"public file {alias} unchanged", alias, public(alias))
                for alias in ("open_substation", "model_substation", "protection_relay", "demo"))
    yield Case("template", "built-in sample IED unchanged", "template", utf8(text))
    yield Case("bom_utf8", "UTF-8 byte order mark", "template", b"\xef\xbb\xbf" + utf8(text))
    yield Case("utf8_no_declaration", "no XML declaration", "template", utf8(body))
    for name, codec in (("utf16le_bom", "utf-16-le"), ("utf16be_bom", "utf-16-be")):
        yield Case(name, f"{codec} with BOM: parser accepts, utf-8-sig decode fails", "template",
                   ("﻿" + _declared(text, "UTF-16")).encode(codec))
    yield Case("latin1_desc", "ISO-8859-1 declared, e-acute in desc", "template",
               _desc(_declared(text, "ISO-8859-1"), "café").encode("latin-1"))
    yield Case("windows1252", "windows-1252 declared (expat: unknown encoding)", "template",
               _desc(_declared(text, "windows-1252"), "café").encode("cp1252"))
    yield Case("utf8_invalid_byte", "declared UTF-8 with an invalid byte", "template",
               utf8(_desc(text, "bad")).replace(b'desc="bad"', b'desc="b\xffd"'))
    yield Case("usascii_charref", "US-ASCII with a numeric character reference", "template",
               utf8(_desc(_declared(text, "US-ASCII"), "caf&#233;")))
    yield Case("crlf", "CRLF between elements", "template", utf8(text.replace("><", ">\r\n<")))
    yield Case("cr_only", "CR-only line endings", "template", utf8(text.replace("><", ">\r<")))
    yield Case("lf", "LF between elements", "template", utf8(text.replace("><", ">\n<")))
    yield Case("tab_newline_attr", "tab and newline inside an attribute value", "template",
               utf8(_desc(text, "a\tb\nc")))
    yield Case("trailing_whitespace", "whitespace after the root", "template", utf8(text + "\n\n  "))
    yield Case("trailing_garbage", "a second element after the root", "template", utf8(text + "<extra/>"))
    yield Case("blank", "a single space", "none", b" ")
    yield Case("empty", "zero bytes", "none", b"")
    yield Case("html_page", "an HTML page", "none", b"<!DOCTYPE html><html><body>Not SCL</body></html>")
    yield Case("zip_renamed", "a ZIP renamed .scd", "none", b"PK\x03\x04" + bytes(26) + b"project.json")
    yield Case("foreign_root", "well-formed XML that is not SCL", "none", utf8(DECLARATION + "<Foo/>"))
    yield Case("crlf_protection_relay", "public relay with CRLF", "protection_relay",
               public("protection_relay").replace(b"\r\n", b"\n").replace(b"\n", b"\r\n"))
    yield Case("bom_open_substation", "public station with a BOM", "open_substation",
               b"\xef\xbb\xbf" + public("open_substation").removeprefix(b"\xef\xbb\xbf"))
    yield Case("size_at_limit", "exactly MAX_XML (20 MB)", "template", _padded(text, MAX_XML), generated=True)
    yield Case("size_over_limit", "MAX_XML + 1 byte", "template", _padded(text, MAX_XML + 1), generated=True)
