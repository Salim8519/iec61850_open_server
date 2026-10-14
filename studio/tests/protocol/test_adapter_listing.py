"""On Windows the IP Helper listing matches Get-NetAdapter and is fast enough for live refresh."""
import json
import os
import time

import pytest
from studio_app.networking import adapters, ip_helper

pytestmark = pytest.mark.skipif(os.name != "nt", reason="Windows adapters only")


def test_native_list_matches_get_netadapter():
    started = time.perf_counter()
    native = {item["guid"].upper(): item for item in ip_helper.adapters()}
    elapsed = time.perf_counter() - started
    shell = {item["guid"].upper(): item for item in json.loads(adapters._powershell(adapters.SCRIPT, timeout=90))}
    assert native.keys() == shell.keys()
    for guid, item in native.items():
        other = shell[guid]
        assert (item["index"], item["name"], item["mac"]) == (other["index"], other["name"], other["mac"])
        assert (item["status"] == "Up") == (other["status"] == "Up"), item["name"]
        listed = other["addresses"] if isinstance(other["addresses"], list) else [other["addresses"]]
        assert sorted((a["ip"], a["prefix"]) for a in item["addresses"]) == sorted((a["ip"], a["prefix"]) for a in listed)
    assert elapsed < 0.5
