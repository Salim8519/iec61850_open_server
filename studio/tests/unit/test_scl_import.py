from pathlib import Path
import pytest
from studio_app.scl.importer import import_scl
from studio_app.contracts.errors import StudioError

ROOT = Path(__file__).resolve().parents[3]


def test_reference_substation_has_four_ieds_and_resolved_controls():
    source = (ROOT / "scd/open_substation.scd").read_bytes()
    models = import_scl(source)
    assert len(models) == 4
    breaker = next(model for model in models if model["name"] == "IED1_XCBR")
    assert any(control["ref"].endswith("CSWI1.Pos") for control in breaker["controls"])
    assert any(signal["ref"].endswith("XCBR1.Pos.stVal") for signal in breaker["signals"])
    assert breaker["source"].encode() == source
    assert breaker["reports"]


def test_unsafe_external_entity_is_rejected():
    xml = b'<!DOCTYPE SCL [<!ENTITY x SYSTEM "file:///C:/Windows/win.ini">]><SCL>&x;</SCL>'
    with pytest.raises(StudioError, match="unsafe XML"):
        import_scl(xml)


def test_missing_types_is_actionable():
    with pytest.raises(StudioError, match="DataTypeTemplates"):
        import_scl(b"<SCL/>")
