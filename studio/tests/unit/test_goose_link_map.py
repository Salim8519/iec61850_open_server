"""GOOSE link map from SCL: vendor flags, remote publishers and address hints (LAYER-01..06)."""
from pathlib import Path

from goose_samples import GE, document, gse, named_input, packed_input, publisher, subscriber

from studio_app.domain.goose_link_live import with_live
from studio_app.domain.goose_link_map import build_link_map
from studio_app.scl.document import parse_document
from studio_app.scl.goose_inventory import block_inventory
from studio_app.scl.goose_sources import read_goose_sources
from studio_app.scl.goose_vendor import packed_input_hint
from studio_app.scl.importer import import_scl

SAMPLE = (Path(__file__).resolve().parents[3] / "scd/open_substation.scd").read_text(encoding="utf-8")


def station(extra_inputs=(), function="Enabled"):
    pub = publisher("PUB", [("GoCB01", "DS1", function), ("GoCB02", "", "Disabled")])
    sub = subscriber("SUB", [named_input("PUB", "GoCB01"), named_input("PUB", "GoCB02", do="Ind2",
                                                                      int_addr="GGIO3.Ind2.stVal"), *extra_inputs])
    return document([pub, sub], [gse("PUB", "GoCB01", "01-0C-CD-01-00-01", "0000"),
                                 gse("PUB", "GoCB02", "01-0C-CD-01-00-02", "0000")])


def project(*names):
    return [{"id": f"id-{name}", "name": name, "role": "simulated"} for name in names]


def link_map(text, *names):
    return build_link_map(read_goose_sources([(text, list(names))]), project(*names))


def test_vendor_disabled_block_is_not_sent_and_vendor_data_is_kept():
    blocks = block_inventory(parse_document(station()))
    assert blocks[("PUB", "LD0", "GoCB01")]["status"] == "sendable"
    disabled = blocks[("PUB", "LD0", "GoCB02")]
    assert disabled["status"] == "disabled_in_file"
    assert disabled["vendor"]["http://www.gedigitalenergy.com/multilin"]["Function"] == "Disabled"
    text = SAMPLE.replace('xmlns="http://www.iec.ch/61850/2003/SCL"', 'xmlns="http://www.iec.ch/61850/2003/SCL" '
                          f'xmlns:ge="{GE}"', 1).replace('name="gcbEvents" type="GOOSE"',
                                                         'name="gcbEvents" type="GOOSE" ge:Function="Disabled"')
    imported = import_scl(text.encode(), "t.scd", ["IED2_PTOC"])[0]
    assert imported["goose"]["publishes"] == []
    assert any("switched off in the file" in warning for warning in imported["warnings"])
    assert 'ge:Function="Disabled"' in imported["source"]


def test_link_to_disabled_block_is_kept_with_its_reason():
    result = link_map(station(), "PUB", "SUB")
    by_block = {link["block"]: link for link in result["links"]}
    assert by_block["GoCB01"]["status"] == "declared" and by_block["GoCB01"]["match"] == "named"
    assert by_block["GoCB02"]["status"] == "disabled_in_file"
    warning = next(item for item in result["diagnostics"] if item["code"] == "disabled_in_file")
    assert "never sent" in warning["message"] and warning["hint"]


def test_packed_vendor_intaddr_is_only_a_hint():
    assert packed_input_hint("CON|BCU_9|1|01:0C:CD:01:00:00|Bool|x") == {
        "ld_inst": "CON", "publisher": "BCU_9", "index": "1", "mac": "01-0C-CD-01-00-00",
        "value_type": "Bool", "vendor": "packed intAddr"}
    for ordinary in ("GGIO3.Ind1.stVal", "Op1", "", "a|b|c", "CON|BCU|1|not-a-mac|Bool"):
        assert packed_input_hint(ordinary) is None


def test_publisher_outside_every_file_is_remote_not_a_problem():
    result = link_map(station([packed_input("FARBCU")]), "SUB")
    remote = next(link for link in result["links"] if link["publisher"] == "FARBCU")
    assert remote["status"] == "remote" and remote["mac"] == "01-0C-CD-01-00-00"
    assert remote["match"] == "vendor_hint" and remote["gocb_ref"] is None
    assert next(ied for ied in result["ieds"] if ied["name"] == "FARBCU")["remote"] is True
    assert not any(item["level"] == "problem" for item in result["diagnostics"])
    assert any(item["code"] == "remote_publisher" for item in result["diagnostics"])


def test_bare_spare_inputs_are_ignored():
    spare = '<ExtRef intAddr="GGIO3.Ind9.stVal" desc="spare"/>'
    result = link_map(station([spare]), "SUB")
    assert sum(len(link["inputs"]) for link in result["links"]) == 2


def test_appid_zero_duplicate_appid_and_shared_mac_are_hints_not_failures():
    pub_a = publisher("PA", [("GoCB01", "DS1", None)])
    pub_b = publisher("PB", [("GoCB01", "DS1", None)])
    pub_c = publisher("PC", [("GoCB01", "DS1", None)])
    text = document([pub_a, pub_b, pub_c], [gse("PA", "GoCB01", "01-0C-CD-01-00-00", "0000"),
                                            gse("PB", "GoCB01", "01-0C-CD-01-00-00", "0000"),
                                            gse("PC", "GoCB01", "01-0C-CD-01-00-07", "3001"),
                                            gse("PC", "GoCB09", "01-0C-CD-01-00-08", "3001")])
    codes = {item["code"]: item for item in link_map(text, "PA", "PB", "PC")["diagnostics"]}
    assert "2 GOOSE block(s) use APPID 0000" in codes["app_id_zero"]["message"]
    assert codes["shared_mac"]["level"] == "hint" and "01-0C-CD-01-00-00" in codes["shared_mac"]["message"]
    assert "duplicate_app_id" not in codes  # The second PC block has no GSEControl, so it is not sendable.
    assert all(item["hint"] for item in codes.values())


def test_stub_ied_in_several_files_is_read_once_preferring_its_own_file():
    stub = publisher("PUB", [("GoCB01", "DS1", "Enabled")])
    full = publisher("PUB", [("GoCB01", "DS1", "Enabled"), ("GoCB03", "DS1", "Enabled")])
    address = [gse("PUB", "GoCB01", "01-0C-CD-01-00-01", "0001"), gse("PUB", "GoCB03", "01-0C-CD-01-00-03", "0003")]
    first = document([subscriber("SUB", [named_input("PUB", "GoCB01")]), stub], address)
    second = document([full], address)
    declarations = read_goose_sources([(first, ["SUB"]), (second, ["PUB"])])
    assert ("PUB", "LD0", "GoCB03") in declarations["blocks"]
    result = build_link_map(declarations, project("SUB", "PUB"))
    assert [link["publisher"] for link in result["links"]] == ["PUB"]
    assert sum(ied["name"] == "PUB" for ied in result["ieds"]) == 1


def test_live_join_never_marks_a_declared_link_received():
    result = link_map(station(), "PUB", "SUB")
    stopped = with_live(result, [], False)
    assert {link["block"]: link["live"] for link in stopped["links"]} == {"GoCB01": {"state": "stopped"}, "GoCB02": None}
    running = with_live(result, [], True)
    assert all(link["live"] is None for link in running["links"])
    ref = next(link["gocb_ref"] for link in result["links"] if link["block"] == "GoCB01")
    seen = with_live(result, [{"gocb_ref": ref, "state": "timeout", "problem": "timeout", "st_num": 4}], True)
    assert seen["summary"]["SUB"]["not_heard"] == 1 and seen["summary"]["SUB"]["not_sent"] == 1
    assert seen["summary"]["PUB"]["sends_to"] == 1


def test_link_map_is_deterministic():
    text = station([packed_input("FARBCU")])
    assert link_map(text, "PUB", "SUB") == link_map(text, "PUB", "SUB")
