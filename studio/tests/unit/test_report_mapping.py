"""Report-first discovery uses qualified identities, exact boundaries and FCs."""
from copy import deepcopy
from xml.etree.ElementTree import tostring
import pytest
from studio_app.scl.document import parse_document
from studio_app.scl.importer import import_scl
from studio_app.scl.templates import make_template
from studio_app.scl.report_catalogue import report_members


def test_duplicate_local_names_and_cross_device_membership():
    root = parse_document(make_template("Mapping"))
    server = root.find("IED/AccessPoint/Server")
    second = deepcopy(server.find("LDevice"))
    second.set("inst", "OTHER")
    second.set("ldName", "FunctionalDomain")
    server.append(second)
    model = import_scl(tostring(root))[0]
    assert len({report["ref"] for report in model["reports"]}) == 4
    assert len({dataset["ref"] for dataset in model["datasets"]}) == 2
    # The copied dataset still explicitly references CTRL, not its owning OTHER LD.
    members = report_members(model, "FunctionalDomain/LLN0.BR.Buffered")
    assert members == report_members(model, "MappingCTRL/LLN0.BR.Buffered")
    assert all(ref.startswith("MappingCTRL/") for ref in members)


def test_structured_members_preserve_fc_and_description():
    source = make_template("Mapping").replace(b'doName="Pos" daName="stVal"', b'doName="Pos"')
    source = source.replace(b'<DOI name="Pos">', b'<DOI name="Pos" desc="Breaker position">')
    model = import_scl(source)[0]
    members = report_members(model, "MappingCTRL/LLN0.BR.Buffered")
    assert "MappingCTRL/XCBR1.Pos.stVal" in members
    assert "MappingCTRL/XCBR1.Pos.q" in members
    assert "MappingCTRL/XCBR1.Pos.ctlModel" not in members
    assert next(s for s in model["signals"] if s["ref"] == "MappingCTRL/XCBR1.Pos.stVal")["description"] == "Breaker position"
    with pytest.raises(ValueError):
        report_members(model, "Buffered")


def test_indexed_member_and_unresolved_diagnostic():
    source = make_template("Mapping").replace(b'type="StudioMV"', b'type="Wrapper"')
    source = source.replace(b'</DataTypeTemplates>', b'<DOType id="Wrapper" cdc="MV">'
        b'<SDO name="samples" type="StudioMV" count="3" /></DOType></DataTypeTemplates>')
    source = source.replace(b'doName="AnIn1"', b'doName="AnIn1.samples" ix="1"')
    model = import_scl(source)[0]
    members = report_members(model, "MappingCTRL/LLN0.RP.EventsRCB")
    assert "MappingCTRL/GGIO1.AnIn1.samples(1).mag.f" in members
    assert "MappingCTRL/GGIO1.AnIn1.samples(0).mag.f" not in members
    broken = import_scl(source.replace(b'doName="AnIn1.samples"', b'doName="Missing.samples"'))[0]
    assert len(broken["reports"][0]["unresolved"]) == 1
