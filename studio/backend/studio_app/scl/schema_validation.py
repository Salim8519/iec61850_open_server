"""Offline, edition-aware XSD validation using unmodified IEC code components."""
from functools import lru_cache
from pathlib import Path
import sys
import threading
from lxml import etree
from .document import parse_document

LOCK = threading.Lock()
SCHEMAS = (Path(sys._MEIPASS) / "schemas" if getattr(sys, "frozen", False)
           else Path(__file__).resolve().parents[3] / "schemas")


def schema_version(root):
    version, revision, release = root.get("version"), root.get("revision"), root.get("release")
    if version is None or version == "2003":
        return "SCL2003"
    if version == "2007" and revision == "B":
        if release in {None, ""}:
            return "SCL2007B"
        if release == "4":
            return "SCL2007B4"
    return None


@lru_cache(maxsize=3)
def load_schema(version):
    parser = etree.XMLParser(resolve_entities=False, no_network=True, load_dtd=False)
    return etree.XMLSchema(etree.parse(str(SCHEMAS / version / "SCL.xsd"), parser))


def validate_schema(source, root=None):
    """`root` is this source's safely parsed document, when the caller already has it."""
    if isinstance(source, str):
        source = source.encode("utf-8")
    if root is None:
        root = parse_document(source)
    version = schema_version(root)
    if not version:
        return {"status": "unsupported", "schema": None, "errors": [
            {"line": 1, "message": "No bundled XSD matches this SCL version/revision/release"}]}
    parser = etree.XMLParser(resolve_entities=False, no_network=True, load_dtd=False, huge_tree=False)
    try:
        document = etree.fromstring(source, parser)
        if document.getroottree().docinfo.doctype:
            raise ValueError("DTD declarations are not allowed in SCL")
        with LOCK:
            schema = load_schema(version)
            valid = schema.validate(document)
            errors = [{"line": item.line, "message": item.message} for item in list(schema.error_log)[:100]]
        return {"status": "valid" if valid else "invalid", "schema": version, "errors": errors}
    except etree.XMLSyntaxError as error:
        return {"status": "invalid", "schema": version,
                "errors": [{"line": error.lineno, "message": str(error)}]}
