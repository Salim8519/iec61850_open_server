from studio_app.scl.templates import make_template
from studio_app.scl.importer import import_scl


def test_generated_template_has_typed_controls_and_both_report_types():
    result = import_scl(make_template("Feeder01"))[0]
    assert result["name"] == "Feeder01"
    assert len(result["controls"]) == 4
    assert [control["cdc"] for control in result["controls"]].count("DPC") == 3
    assert [control["cdc"] for control in result["controls"]].count("SPC") == 1
    assert all(control["ctl_val_kind"] == "BOOLEAN" for control in result["controls"])
    assert {report["buffered"] for report in result["reports"]} == {True, False}
    assert all(control["model"] == 3 for control in result["controls"])
    assert all(control["sbo_timeout_ms"] == 10000 for control in result["controls"])


def test_imported_enhanced_sbo_retains_thirty_second_timeout():
    source = make_template("ImportedSbo").replace(b"<Val>3</Val>", b"<Val>4</Val>")
    source = source.replace(b"<Val>10000</Val>", b"<Val>30000</Val>")
    result = import_scl(source)[0]
    assert all(control["model"] == 4 for control in result["controls"])
    assert all(control["sbo_timeout_ms"] == 30000 for control in result["controls"])
