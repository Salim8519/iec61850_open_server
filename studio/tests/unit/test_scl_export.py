"""Edited exports must preserve unrelated engineering XML and round-trip values."""
import pytest
from lxml import etree
from studio_app.contracts.errors import StudioError
from studio_app.scl.exporter import NS, export_ied
from studio_app.scl.importer import import_scl
from studio_app.scl.templates import make_template
from studio_app.scl.schema_validation import validate_schema


def test_typed_edits_round_trip_without_changing_original():
    source = make_template("ExportBay")
    model = import_scl(source)[0]
    point = next(item for item in model["signals"] if item["kind"] == "FLOAT32")
    edited = export_ied(source, "ExportBay", {point["ref"]: 42.5})
    actual = import_scl(edited)[0]
    assert validate_schema(edited)["status"] == "valid"
    assert next(item for item in actual["signals"] if item["ref"] == point["ref"])["value"] == 42.5
    assert next(item for item in import_scl(source)[0]["signals"]
                if item["ref"] == point["ref"])["value"] == point["value"]


def test_vendor_extensions_comments_and_unselected_ieds_survive():
    root = etree.fromstring(make_template("First"))
    peer = etree.fromstring(make_template("Peer")).find(NS + "IED")
    root.insert(list(root).index(root.find(NS + "DataTypeTemplates")), peer)
    private = etree.Element(NS + "Private", type="vendor-test")
    etree.SubElement(private, "{urn:vendor:test}extension").text = "keep me"
    root.insert(0, private)
    root.insert(0, etree.Comment("keep this comment"))
    source = etree.tostring(root)
    edited = export_ied(source, "First", {})
    restored = etree.fromstring(edited)
    assert b"keep this comment" in edited
    assert restored.find(f"{NS}Private/{{urn:vendor:test}}extension").text == "keep me"
    assert etree.tostring(restored.findall(NS + "IED")[1]) == etree.tostring(peer)
    assert [item["name"] for item in import_scl(edited)] == ["First", "Peer"]


def test_unknown_reference_invalid_value_and_unsupported_schema_are_rejected():
    source = make_template("ExportBay")
    with pytest.raises(StudioError, match="Unknown export reference"):
        export_ied(source, "ExportBay", {"wrong/ref": 1})
    point = next(item for item in import_scl(source)[0]["signals"] if item["kind"] == "FLOAT32")
    with pytest.raises(StudioError, match="finite number"):
        export_ied(source, "ExportBay", {point["ref"]: float("nan")})
    with pytest.raises(StudioError, match="schema-valid"):
        export_ied(source.replace(b'release="4"', b'release="99"'), "ExportBay", {})


def test_enum_label_and_timestamp_are_serialized():
    source = make_template("ExportBay")
    signals = import_scl(source)[0]["signals"]
    enum = next(item for item in signals if item["kind"] == "Enum")
    timestamp = next(item for item in signals if item["kind"] == "Timestamp")
    value = next(iter(enum["enums"].values()))
    output = export_ied(source, "ExportBay", {enum["ref"]: value, timestamp["ref"]: 1720000000123})
    assert b"2024-07-03T09:46:40.123+00:00" in output
    with pytest.raises(StudioError, match="declared enumeration"):
        export_ied(source, "ExportBay", {enum["ref"]: 999999})
