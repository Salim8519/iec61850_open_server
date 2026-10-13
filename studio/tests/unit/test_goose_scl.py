"""GOOSE publishing and subscriptions come only from the SCD (GOOSE-03/04/07)."""
from pathlib import Path

from studio_app.scl.importer import import_scl

SCD = (Path(__file__).resolve().parents[3] / "scd/open_substation.scd").read_text(encoding="utf-8")


def load(text, names):
    return {ied["name"]: ied for ied in import_scl(text.encode("utf-8"), "test.scd", names)}


def test_gc_line_uses_scd_values():
    ied = load(SCD, ["IED2_PTOC"])["IED2_PTOC"]
    control = ied["goose"]["publishes"][0]
    assert (control["mac"], control["app_id"], control["vlan_id"], control["vlan_priority"]) == (
        "01-0C-CD-01-00-02", 0x1002, 1, 4)
    assert (control["go_id"], control["conf_rev"], control["dataset"]) == ("events", 2, "Events")
    assert control["gocb_ref"] == "IED2_PTOCGenericIO/LLN0$GO$gcbEvents"
    lines = ied["cfg"].splitlines()
    start = lines.index("GC(gcbEvents events Events 2 0 -1 -1){")
    assert lines[start + 1:start + 3] == ["PA(4 1 4098 010ccd010002);", "}"]


def test_control_without_address_is_not_published():
    text = SCD.replace('<GSE ldInst="GenericIO" cbName="gcbEvents">', '<GSE ldInst="GenericIO" cbName="renamed">', 1)
    ied = load(text, ["IED1_XCBR"])["IED1_XCBR"]
    assert ied["goose"]["publishes"] == []
    assert "GC(" not in ied["cfg"]
    assert any("no network address (GSE)" in warning and "gcbEvents" in warning for warning in ied["warnings"])


def test_unknown_source_block_is_diagnosed():
    ied = load(SCD, ["IED1_XCBR"])["IED1_XCBR"]
    assert ied["goose"]["subscribes"] == []
    assert any("IED2_PTOC/GenericIO/GoCB" in item for item in ied["goose"]["diagnostics"])


def test_extref_resolves_publisher_block():
    text = SCD.replace('srcLNClass="LLN0" srcCBName="GoCB"', 'srcLNClass="LLN0" srcCBName="gcbEvents"')
    ied = load(text, ["IED1_XCBR"])["IED1_XCBR"]
    assert ied["goose"]["diagnostics"] == []
    subscription = ied["goose"]["subscribes"][0]
    assert subscription["gocb_ref"] == "IED2_PTOCGenericIO/LLN0$GO$gcbEvents"
    assert subscription["publisher"] == "IED2_PTOC" and subscription["app_id"] == 0x1002
    assert subscription["inputs"][0]["int_addr"] == "Tr"


def test_scd_address_is_read_without_rewriting_the_source():
    ied = load(SCD, ["IED2_PTOC"])["IED2_PTOC"]
    assert ied["scl_address"]["ip"] == "10.0.0.3" and ied["scl_address"]["subnet"] == "255.255.255.0"
    assert ied["source"] == SCD
