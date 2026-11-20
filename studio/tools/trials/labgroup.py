"""Building blocks for the Lab-card ownership trials (N3, N4, N5, N9, N10 and the cross-removal
finding, change 0222). Not part of verify.py.

This group owns ports 24000-24999 and Lab addresses 198.18.24.0/24 only; other agents' engines
may run at the same time, so every check reads Windows' own list of the Lab card for our /24
and the lab journal of our own data folder (which engine removed an address).

    engine = LabInstance(server=BASE).start()          # or server=TRIAL, scratch=<same folder>
    setup_small(engine)                                 # 8 IEDs on 198.18.24.10-17, MMS port 24102
    on_card()                                           # {"198.18.24.10", ...} as Windows lists it
    engine.quit()                                       # graceful exit (Ctrl+Break), or engine.kill()
"""
import json
import os
from pathlib import Path
import signal
import subprocess
import time

import instance
import projects
import winaddr
from instance import Instance

RANGE = "198.18.24"
PREFIX = RANGE + "."
IED_PORT = 24102
GATEWAY_PORT = 24404
SNMP_PORT = 24161
BASE = projects.REPO / ".local/trials/c/base/studio_server.exe"    # today's code (master before 0222)
TRIAL = projects.REPO / "studio/build/engine/studio_server.exe"    # the working tree
RESULTS = projects.REPO / ".local/trials/results"
projects.RANGE, projects.IED_PORT = RANGE, IED_PORT  # this group's addresses and MMS port


def free_port():
    return instance.free_port(24000, 24999)


class LabInstance(Instance):
    """An Instance in its own process group (so quit() can send Ctrl+Break, the app's normal
    exit) that can reopen the same data folder (scratch=) after a kill."""

    def __init__(self, server=BASE, scratch=None, port=None):
        super().__init__(port=port or free_port(), server=server, keep=True, shared=True)
        if scratch:
            self.scratch, self.data = Path(scratch), Path(scratch) / "data"

    def start(self, timeout=60):
        original = instance.subprocess.Popen

        def grouped(*args, **kwargs):
            kwargs["creationflags"] = kwargs.get("creationflags", 0) | subprocess.CREATE_NEW_PROCESS_GROUP
            return original(*args, **kwargs)

        home = self.scratch / "home"
        if home.exists():  # a relaunch: Instance.start creates it again
            os.rename(home, self.scratch / f"home-{time.time_ns()}")
        instance.subprocess.Popen = grouped
        try:
            return super().start(timeout)
        finally:
            instance.subprocess.Popen = original

    def quit(self, timeout=30):
        """The app's own exit: Ctrl+Break reaches the server as os.Interrupt and Close runs."""
        os.kill(self.process.pid, signal.CTRL_BREAK_EVENT)
        self.process.wait(timeout)
        self.job.close()
        self.job, self.process = None, None

    def kill(self):
        """Task Manager's End task: the server and every worker die at once, nothing runs Close."""
        self.job.close()  # the Job kills the server and its workers
        self.process.wait(10)
        self.job, self.process = None, None

    def remove(self):
        import shutil
        shutil.rmtree(self.scratch, ignore_errors=True)


def on_card():
    """Our /24's addresses on the Lab card as Windows lists them now."""
    return set(winaddr.addresses(PREFIX, winaddr.lab_card_index()))


def clean_card(addresses):
    """Take this group's leftover addresses (killed trial engines whose data folder is gone) off the
    Lab card through the installed service (labsetup/trial_cleanup_test.go, tag trial)."""
    wanted = [ip for ip in addresses if ip.startswith(PREFIX)]
    if not wanted:
        return
    env = {**os.environ, "TRIAL_REMOVE": ",".join(wanted)}
    tmp = str(projects.REPO / ".local/gotmp")
    env.update(GOTMPDIR=tmp, TEMP=tmp, TMP=tmp)
    subprocess.run(["go", "test", "-tags", "trial", "-count=1", "-run", "TestTrialRemoveLeftovers",
                    "./internal/labsetup/"], cwd=str(projects.REPO / "studio/engine"), env=env, check=True,
                   capture_output=True)


def journal(data_dir, since=0.0):
    """Lab journal records of one data folder after `since` (Unix seconds): (event, ip, at)."""
    path = Path(data_dir) / "network/lab_journal.jsonl"
    found = []
    if not path.is_file():
        return found
    from datetime import datetime
    for line in path.read_text(encoding="utf-8").splitlines():
        if not line.strip():
            continue
        record = json.loads(line)
        at = datetime.fromisoformat(record["at"]).timestamp()
        if at >= since:
            found.append((record["event"], record["item"].get("ip", ""), at))
    return found


def setup_small(engine):
    """The 8-IED project on the Lab card at 198.18.24.10-17; returns {ied_id: ip}."""
    projects.open_in(engine, projects.build("small"))
    return projects.use_lab_card(engine)


def ready_gateway(engine, port=GATEWAY_PORT):
    """Station gateway role, points proposed and saved for IEC 104 on `port`, so Start works."""
    engine.json("POST", "/api/v1/tools/gateway/role", {"role": "gateway"})
    config = engine.json("GET", "/api/v1/tools/gateway/config")
    settings = dict((config.get("settings") or {}).get("iec104") or {}, port=port)
    engine.json("PUT", "/api/v1/tools/gateway/config", {"protocol": "iec104", "settings": settings})
    proposal = engine.json("POST", "/api/v1/tools/gateway/propose", {"protocol": "iec104"})
    engine.json("PUT", "/api/v1/tools/gateway/points", {"protocol": "iec104", "points": proposal["points"][:20]})


def add_snmp_device(engine, ip, port=SNMP_PORT):
    """The project's only SNMP device: a network switch at ip:port on the Lab card; its id."""
    for old in engine.json("GET", "/api/v1/tools/snmp/config").get("devices") or []:
        engine.json("DELETE", f"/api/v1/tools/snmp/devices/{old['id']}")
    device = engine.json("POST", "/api/v1/tools/snmp/devices", {"profile": "switch"})["device"]
    device["port"] = port
    engine.json("PUT", f"/api/v1/tools/snmp/devices/{device['id']}", {"device": device})
    engine.json("PUT", f"/api/v1/network/address/snmp/{device['id']}", {"ip": ip})
    return device["id"]


def save(name, result):
    RESULTS.mkdir(parents=True, exist_ok=True)
    path = RESULTS / f"{name}-{time.strftime('%Y%m%d-%H%M%S')}.json"
    path.write_text(json.dumps(result, indent=1), encoding="utf-8")
    return path
