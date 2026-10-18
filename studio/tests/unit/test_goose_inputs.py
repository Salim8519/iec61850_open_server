"""A simulated subscriber applies received GOOSE only to data its ExtRef binds (GOOSE-12..15)."""
from pathlib import Path

from studio_app.domain.goose_inputs import accept_frame, bind_inputs, input_updates
from studio_app.scl.importer import import_scl

SCD = (Path(__file__).resolve().parents[3] / "scd/open_substation.scd").read_text(encoding="utf-8")
WRONG = ('intAddr="Tr" desc="Trip" iedName="IED2_PTOC" ldInst="GenericIO" lnClass="PTRC" lnInst="1" doName="Tr" '
         'daName="general" serviceType="GOOSE" srcLDInst="GenericIO" srcLNClass="LLN0" srcCBName="GoCB"')


def subscriber(int_addr="CILO1.EnaCls.stVal"):
    text = SCD.replace(WRONG, WRONG.replace('intAddr="Tr"', f'intAddr="{int_addr}"')
                       .replace('srcCBName="GoCB"', 'srcCBName="gcbEvents"'))
    ied = next(item for item in import_scl(text.encode(), "t.scd", ["IED1_XCBR", "IED2_PTOC"])
               if item["name"] == "IED1_XCBR")
    return ied, ied["goose"]["subscribes"][0]


def test_input_position_in_dataset_is_read_from_scl():
    _, subscription = subscriber()
    item = subscription["inputs"][0]
    assert (item["member"], item["member_path"], item["ld"]) == (1, "", "IED1_XCBRGenericIO")
    assert subscription["member_count"] == 2


def test_relative_intaddr_binds_to_the_subscribers_own_point():
    ied, subscription = subscriber()
    bindings, problems = bind_inputs(ied, subscription)
    assert problems == []
    assert bindings[0]["target"] == "IED1_XCBRGenericIO/CILO1.EnaCls.stVal" and bindings[0]["kind"] == "BOOLEAN"
    assert input_updates(bindings, [False, True]) == ([("IED1_XCBRGenericIO/CILO1.EnaCls.stVal", True)], [])


def test_vendor_intaddr_is_shown_but_not_applied():
    for int_addr in ("Tr", "CON|IED2_PTOC|1|01:0C:CD:01:00:02|Bool", "Op Trip"):
        ied, subscription = subscriber(int_addr)
        bindings, problems = bind_inputs(ied, subscription)
        assert bindings == [] and "not applied" in problems[0]


def test_frames_with_problems_never_drive_inputs():
    good = {"state": "receiving", "problem": "", "values": [True, False]}
    assert accept_frame(good, 2) == ""
    assert "conf_rev_mismatch" in accept_frame({**good, "state": "problem", "problem": "conf_rev_mismatch"}, 2)
    assert "dataset_mismatch" in accept_frame({**good, "state": "problem", "problem": "dataset_mismatch"}, 2)
    assert "size" in accept_frame({**good, "values": [True]}, 2)
    assert "no dataset values" in accept_frame({"state": "receiving", "problem": ""}, 2)


def test_value_types_are_converted_or_reported():
    binding = {"target": "X/GGIO1.Ind1.stVal", "kind": "BOOLEAN", "member": 0, "path": "", "desc": "d",
               "source_ref": "P/GGIO1.Ind1.stVal"}
    dbpos = {**binding, "target": "X/XCBR1.Pos.stVal", "kind": "Dbpos", "member": 1}
    updates, problems = input_updates([binding, dbpos], [True, {"bits": 2, "le": 1, "be": 2}])
    assert problems == [] and updates == [("X/GGIO1.Ind1.stVal", True), ("X/XCBR1.Pos.stVal", 2)]
    updates, problems = input_updates([binding], [7])
    assert updates == [] and "true/false" in problems[0]


def test_structured_member_needs_the_publisher_layout():
    binding = {"target": "X/GGIO1.Ind1.stVal", "kind": "BOOLEAN", "member": 0, "path": "stVal", "desc": "d",
               "source_ref": "P/GGIO1.Ind1.stVal"}
    updates, problems = input_updates([binding], [[True, 0, 0]])
    assert updates == [] and "structure" in problems[0]
    updates, problems = input_updates([binding], [[True, 0, 0]], {"P/GGIO1.Ind1.stVal": True})
    assert updates == [("X/GGIO1.Ind1.stVal", True)]
