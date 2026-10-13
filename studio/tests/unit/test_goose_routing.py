"""Received GOOSE only drives LGOS supervision in simulated subscribers (GOOSE-08)."""
from studio_app.domain.goose_supervision import dataset_values, lgos_updates, watch_list


def signal(ref, kind):
    return {"ref": ref, "kind": kind, "fc": "ST"}


INDEX = {item["ref"]: item for item in [
    signal("S/LGOS1.St.stVal", "BOOLEAN"), signal("S/LGOS1.SimSt.stVal", "BOOLEAN"),
    signal("S/LGOS1.LastStNum.stVal", "INT32"), signal("S/LGOS1.ConfRevNum.stVal", "INT32"),
    signal("S/PTRC1.Tr.general", "BOOLEAN"), signal("S/XCBR1.Pos.stVal", "Dbpos")]}


def test_only_lgos_attributes_are_written():
    received = {"state": "receiving", "test": False, "st_num": 7, "conf_rev": 2, "values": [True, 2]}
    updates = dict(lgos_updates("S/LGOS1", received, INDEX))
    assert updates == {"S/LGOS1.St.stVal": True, "S/LGOS1.SimSt.stVal": False,
                       "S/LGOS1.LastStNum.stVal": 7, "S/LGOS1.ConfRevNum.stVal": 2}
    assert all(ref.startswith("S/LGOS1.") for ref in updates)
    assert lgos_updates(None, received, INDEX) == []
    assert dict(lgos_updates("S/LGOS1", {"state": "timeout"}, INDEX)) == {
        "S/LGOS1.St.stVal": False, "S/LGOS1.SimSt.stVal": False}


def test_watch_list_joins_publishers_and_subscribers():
    control = {"gocb_ref": "P/LLN0$GO$g", "app_id": 1, "mac": "01-0C-CD-01-00-01", "go_id": "g",
               "dataset_ref": "P/LLN0$DS", "conf_rev": 1}
    ieds = [{"id": "p", "name": "P", "role": "real", "goose": {"publishes": [control], "subscribes": []}},
            {"id": "s", "name": "S", "role": "simulated",
             "goose": {"publishes": [], "subscribes": [{**control, "lgos": "S/LGOS1", "inputs": [{}]}]}},
            {"id": "o", "name": "O", "role": "off", "goose": {"publishes": [control], "subscribes": []}}]
    watch = watch_list(ieds)["P/LLN0$GO$g"]
    assert watch["publisher"]["role"] == "real" and watch["subscribers"][0]["lgos"] == "S/LGOS1"


def test_dataset_values_follow_scd_order():
    ied = {"datasets": [{"ref": "P/LLN0$DS", "fcda_refs": ["P/XCBR1.Pos", "P/PTRC1.Tr.general"]}],
           "signals": [{"ref": "P/XCBR1.Pos.stVal", "fc": "ST"}, {"ref": "P/XCBR1.Pos.q", "fc": "ST"},
                       {"ref": "P/XCBR1.Pos.t", "fc": "ST"}, {"ref": "P/PTRC1.Tr.general", "fc": "ST"}]}
    control = {"dataset_ref": "P/LLN0$DS", "members": ["XCBR1$ST$Pos", "PTRC1$ST$Tr$general"]}
    mapped = dataset_values(ied, control, [[{"bits": 2}, {"bits": 13}, 5], True])
    assert mapped == {"P/XCBR1.Pos.stVal": {"bits": 2}, "P/XCBR1.Pos.q": {"bits": 13}, "P/XCBR1.Pos.t": 5,
                      "P/PTRC1.Tr.general": True}
    assert dataset_values(ied, control, [True]) == {}
