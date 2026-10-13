"""Patterns found in real vendor CIDs import instead of rejecting the file (change 0053)."""
from studio_app.scl.importer import import_scl
from studio_app.scl.templates import make_template


def variant(old, new):
    source = make_template("Vendor").decode("utf-8")
    assert old in source
    return source.replace(old, new, 1)


def test_phycomaddr_attributes_import():
    source = variant('<DA name="t" bType="Timestamp" fc="ST" /></DOType>',
                     '<DA name="t" bType="Timestamp" fc="ST" /><DA name="dstAddress" bType="PhyComAddr" fc="ST" />'
                     '</DOType>')
    model = import_scl(source)[0]
    signals = [item for item in model["signals"] if item["kind"] == "PhyComAddr"]
    assert signals and all(item["ref"].endswith(".dstAddress") for item in signals)
    assert "DA(dstAddress 0 29 " in model["cfg"]


def test_unnamed_report_block_is_skipped_with_warning():
    source = variant('<ReportControl name="EventsRCB"', '<ReportControl name="" rptID="X" datSet="Events" '
                     'confRev="1" buffered="false" indexed="false"><RptEnabled max="1" /></ReportControl>'
                     '<ReportControl name="EventsRCB"')
    model = import_scl(source)[0]
    assert [report["name"] for report in model["reports"]] == ["EventsRCB", "Buffered"]
    assert any("have no name" in warning for warning in model["warnings"])
