"""Discover advertised MMS model references without supplying them to the client."""
import json
import os
from pathlib import Path
import subprocess
import pytest
from studio_app.runtime.worker_process import WorkerProcess
from studio_app.scl.importer import import_scl
from studio_app.scl.templates import make_template

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
pytestmark = pytest.mark.skipif(not (BIN / "studio_probe.exe").exists(), reason="Native build required")


@pytest.mark.parametrize("functional_name,array", [(False, "none"), (True, "none"),
    (False, "nested"), (False, "top")])
def test_discovered_leaves_datasets_and_reports_match_import(tmp_path, functional_name, array):
    source = make_template("BrowseTest")
    if functional_name:
        source = source.replace(b'<LDevice inst="CTRL"', b'<LDevice ldName="FunctionalDomain" inst="CTRL"')
    if array == "top":
        source = source.replace(b'<DO name="AnIn2"', b'<DO count="3" name="AnIn2"')
        source = source.replace(b'<DO count="3" name="AnIn2" type="StudioMV" />',
            b'<DO count="3" name="AnIn2" type="StudioMV" /><DO name="AnIn3" type="StudioMV" />')
    elif array == "nested":
        source = source.replace(b'type="StudioMV"', b'type="BrowseWrapper"')
        source = source.replace(b'</DataTypeTemplates>',
            b'<DOType id="BrowseWrapper" cdc="MV"><SDO name="samples" type="StudioMV" count="3" />'
            b'</DOType></DataTypeTemplates>')
        source = source.replace(b'doName="AnIn1"', b'doName="AnIn1.samples" ix="0"')
    model = import_scl(source)[0]
    config = tmp_path / "browse.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    worker = WorkerProcess(BIN / "studio_ied.exe", lambda event: None)
    try:
        worker.request("start", cfg=str(config), ip="127.50.0.217", port=18109,
                       controls=model["controls"], edition=2)
        result = subprocess.run([str(BIN / "studio_probe.exe"), "127.50.0.217", "18109", "browse"],
                                capture_output=True, text=True, timeout=20)
        assert result.returncode == 0, result.stdout + result.stderr
        output = json.loads(result.stdout)
        assert output["ok"]
        observed = set()
        nodes = {}
        for device in output["devices"]:
            for node in device["nodes"]:
                reference = device["name"] + "/" + node["name"]
                nodes[reference] = node
                observed.update((leaf["ref"], leaf["fc"]) for leaf in node["leaves"])
        expected = {(signal["ref"], signal["fc"]) for signal in model["signals"]}
        assert expected <= observed, sorted(expected - observed)
        if array == "top":
            assert ("BrowseTestCTRL/GGIO1.AnIn2(2).mag.f", "MX") in observed
        elif array == "nested":
            assert ("BrowseTestCTRL/GGIO1.AnIn2.samples(2).mag.f", "MX") in observed
        domain = "FunctionalDomain" if functional_name else "BrowseTestCTRL"
        assert set(device["name"] for device in output["devices"]) == {domain}
        lln0 = nodes[domain + "/LLN0"]
        assert lln0["datasets"] == ["Events"]
        assert lln0["buffered_reports"] == ["Buffered"]
        assert lln0["unbuffered_reports"] == ["EventsRCB"]
        analog = "/GGIO1.AnIn1.samples(0).mag.f[MX]" if array == "nested" else "/GGIO1.AnIn1.mag.f[MX]"
        assert lln0["dataset_members"]["Events"] == {"deletable": False, "members": [
            domain + "/XCBR1.Pos.stVal[ST]", domain + "/XSWI1.Pos.stVal[ST]",
            domain + "/XSWI2.Pos.stVal[ST]", domain + analog,
            domain + "/GGIO1.SPCSO1.stVal[ST]"]}
        assert "Pos" in nodes[domain + "/XCBR1"]["objects"]
        if array != "none":
            array_path = "/GGIO1.AnIn2" + (".samples" if array == "nested" else "")
            for index in range(3):
                ref = domain + array_path + f"({index}).mag.f"
                worker.request("update", ref=ref, value=40.5 + index)
            for index in range(3):
                ref = domain + array_path + f"({index}).mag.f"
                read = subprocess.run([str(BIN / "studio_probe.exe"), "127.50.0.217", "18109", "read", ref, "MX"],
                                      capture_output=True, text=True, timeout=10)
                assert read.returncode == 0, read.stdout + read.stderr
                assert float(json.loads(read.stdout)["value"]) == 40.5 + index
    finally:
        worker.close()
