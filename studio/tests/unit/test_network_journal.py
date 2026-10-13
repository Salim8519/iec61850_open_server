"""The network journal is append-only and drives a safe clean-up (NET-07/08/11)."""
from studio_app.networking.journal import NetworkJournal

OWNED = {"kind": "alias", "adapter_guid": "{1}", "adapter_index": 7, "adapter_name": "Lab NIC",
         "ip": "192.168.50.21", "prefix": 24, "owned": True}
FOREIGN = {**OWNED, "ip": "192.168.50.10", "owned": False}
RULE = {"kind": "firewall", "name": "SubstationStudio-MMS-0123456789ab", "owned": True}


def test_journal_is_append_only(tmp_path):
    journal = NetworkJournal(tmp_path)
    journal.append("applied", OWNED, "f1")
    first = journal.path.read_text(encoding="utf-8")
    journal.append("removed", OWNED, "f2")
    second = journal.path.read_text(encoding="utf-8")
    assert second.startswith(first) and len(second.splitlines()) == 2
    assert journal.outstanding() == []


def test_cleanup_removes_only_owned_items(tmp_path):
    journal = NetworkJournal(tmp_path)
    journal.append("applied", OWNED, "f1")
    journal.append("applied", FOREIGN, "f1")
    journal.append("applied", RULE, "f1")
    journal.append("failed", {**OWNED, "ip": "192.168.50.22"}, "f1")
    assert {item.get("ip") or item["name"] for item in journal.outstanding()} == {
        "192.168.50.21", "SubstationStudio-MMS-0123456789ab"}


def test_leftovers_are_reported_not_removed(tmp_path):
    journal = NetworkJournal(tmp_path)
    journal.append("applied", OWNED, "f1")
    journal.append("applied", RULE, "f1")
    before = journal.path.read_text(encoding="utf-8")
    leftovers = journal.leftovers(lambda item: item["kind"] == "alias")
    assert leftovers == [OWNED]
    assert journal.path.read_text(encoding="utf-8") == before
