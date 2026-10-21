"""Write generated/misc.json: loopback, Run preflight, Npcap, firewall/PowerShell readers and ARP probe.

Usage (from the worktree root):
  studio/.venv/Scripts/python.exe -I -B studio/engine/internal/netcfg/testdata/make_misc.py
Sockets, PowerShell and SendARP are fakes; nothing on the PC is read or changed.
"""
import subprocess
import sys
import tempfile
import types
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from vector_common import GATEWAY, dumps, outcome, write  # noqa: E402
from studio_app.networking import adapters, arp_probe, endpoints, firewall_rules  # noqa: E402
from studio_app.networking.loopback import assign_endpoints  # noqa: E402
from studio_app.networking.npcap import npcap_status  # noqa: E402

BUSY = "Only one usage of each socket address (protocol/network address/port) is normally permitted"


def loopback_cases():
    lists = [["", "", ""], ["127.50.0.2", "", "127.50.0.3", ""], [], ["10.0.0.1"],
             ["127.50.0.%d" % n for n in range(2, 254)] + ["", ""], ["127.50.0.%d" % n for n in range(2, 255)] + [""]]
    cases = []
    for ips in lists:
        ieds = [{"ip": ip} if ip else {} for ip in ips]
        result = outcome(lambda: assign_endpoints(ieds))
        cases.append({"ips": ips, "after": [ied.get("ip", "") for ied in ieds], **result})
    return cases


def preflight_cases():
    lab = {"mode": "lab", "adapter_label": "Lab NIC"}
    loop = {"mode": "loopback", "adapter_label": ""}

    def ied(name="Relay", ip="127.50.0.240", port=18140, lab_ip=""):
        return {"name": name, "ip": ip, "port": port, "lab_ip": lab_ip}
    inputs = [
        ([ied(ip="192.168.10.21")], loop, [], []), ([ied()], loop, [], []),
        ([ied(lab_ip="192.168.77.21")], lab, ["192.168.77.99"], []), ([ied(lab_ip="")], lab, [], []),
        ([ied(lab_ip="192.168.77.21")], {"mode": "lab"}, [], []),
        ([ied(), {**ied(), "name": "Other"}], loop, [], []), ([ied(lab_ip="192.168.77.21")], lab, ["192.168.77.21"], []),
        ([ied(port=0)], loop, [], []), ([ied(port=65536)], loop, [], []), ([ied(port="102")], loop, [], []),
        ([ied(port=" 1_02 ")], loop, [], []), ([ied(port="x")], loop, [], []), ([ied(port=102.9)], loop, [], []),
        ([ied(port=True)], loop, [], []), ([ied(ip="::1")], loop, [], []), ([ied(ip="fe80::1%3")], loop, [], []),
        ([ied(ip="localhost")], loop, [], []), ([ied(ip="")], loop, [], []), ([ied(ip="127.050.0.1")], loop, [], []),
        ([ied(), ied(name="B", ip="127.50.0.241")], loop, [], [["127.50.0.241", 18140]]),
        ([ied(lab_ip="192.168.77.21")], {"mode": "loopback", "adapter_label": "X"}, ["192.168.77.21"], []),
        ([{"name": "NoPort", "ip": "127.0.0.9"}], loop, [], []),
    ]
    cases = []
    for ieds, network, addresses, busy in inputs:
        binds = []

        class Socket:
            def __init__(self, *args):
                pass

            def __enter__(self):
                return self

            def __exit__(self, *args):
                return False

            def setsockopt(self, *args):
                pass

            def bind(self, endpoint):
                binds.append(list(endpoint))
                if list(endpoint) in busy:
                    raise OSError(10048, BUSY)
        endpoints.socket = types.SimpleNamespace(socket=Socket, AF_INET=2, SOCK_STREAM=1, SOL_SOCKET=0xFFFF,
                                                 SO_EXCLUSIVEADDRUSE=-5)
        result = outcome(lambda: endpoints.preflight(ieds, network, set(addresses)))
        cases.append({"ieds": ieds, "network": network, "addresses": addresses, "busy": busy,
                      "bind_error": str(OSError(10048, BUSY)), "binds": binds, **result})
    return cases


def npcap_cases():
    cases = []
    for layout in ("file", "missing", "folder"):
        with tempfile.TemporaryDirectory() as root:
            library = Path(root) / "System32" / "Npcap" / "wpcap.dll"
            if layout == "file":
                library.parent.mkdir(parents=True)
                library.write_bytes(b"MZ")
            elif layout == "folder":
                library.mkdir(parents=True)
            status = npcap_status(root)
            status["path"] = status["path"].replace(root, "<ROOT>")
            cases.append({"layout": layout, "out": dumps(status)})
    return cases


def recorder(stdout, calls):
    def run(script, timeout=20):
        calls.append({"script": script, "timeout": timeout})
        return stdout
    return run


def reader_cases():
    cases = []
    outputs = ["", "  \r\n", "a\r\n\r\n b \n", '{"name": "a", "display": "b"}', '[{"name": "a"}, {"name": 5, "display": null}]',
               "[]", "not json", '[{"display": "x"}]', '{"gateway": ["a", "b"], "disabled": "xy"}', '{"gateway": null}',
               "[1]", '{"gateway": [], "disabled": ["n1"]}']
    readers = {"gateway_rules": lambda run: firewall_rules.list_gateway_rules(run),
               "blocking": lambda run: firewall_rules.blocking_rules(GATEWAY, run),
               "blocking_quote": lambda run: firewall_rules.blocking_rules("C:\\O'Brien\\g.exe", run),
               "state": lambda run: [sorted(s) for s in firewall_rules.gateway_state(["a'b", "TCP x"], run)],
               "state_empty": lambda run: [sorted(s) for s in firewall_rules.gateway_state([], run)],
               "studio_rules": lambda run: adapters.list_studio_rules(run)}
    for name, reader in readers.items():
        for text in outputs:
            calls = []
            cases.append({"reader": name, "stdout": text, **outcome(lambda: reader(recorder(text, calls))),
                          "calls": calls})
    return cases


def shell_cases():
    cases = []
    for code, stderr in ((0, ""), (1, "first\r\nsecond line\r\n"), (1, ""), (1, "  \n\n"), (2, "only")):
        calls = []

        def fake_run(argv, capture_output, text, timeout, creationflags):
            calls.append({"argv": argv, "timeout": timeout, "flags": creationflags})
            return subprocess.CompletedProcess(argv, code, "[]", stderr)
        for module in (adapters, firewall_rules):
            module.subprocess.run = fake_run
            cases.append({"module": module.__name__.rsplit(".", 1)[1], "exit_code": code, "stderr": stderr,
                          **outcome(lambda: module._powershell("Get-Thing", timeout=7), str), "calls": list(calls)})
            calls.clear()
    fallback = '[{"guid": "x", "name": "Ethernet", "description": "NIC", "status": "Up", "addresses": {"ip": "10.0.0.5", "prefix": 8}}]'

    def broken():
        raise OSError(50, "not supported")
    native = outcome(lambda: adapters.list_adapters(run=lambda script: fallback if script == adapters.SCRIPT else "", native=broken))
    return cases, {"fallback": fallback, **native, "script": adapters.SCRIPT, "rules_script": adapters.RULES_SCRIPT}


def probe_cases():
    cases = []
    for addresses in (["192.168.50.21", "", "192.168.50.22", "192.168.50.21"], [], ["x"], ["10.0.0.1"] * 20, [""]):
        asked = []

        def answers(address):
            asked.append(address)
            return address.endswith(".21") or address == "10.0.0.1"
        result = outcome(lambda: sorted(arp_probe.addresses_in_use(addresses, probe=answers)))
        cases.append({"addresses": addresses, "asked": sorted(asked), **result})
    return cases


def main():
    shells, fallback = shell_cases()
    write("misc.json", {"loopback": loopback_cases(), "preflight": preflight_cases(), "npcap": npcap_cases(),
                        "readers": reader_cases(), "shell": shells, "fallback": fallback, "probe": probe_cases()})


if __name__ == "__main__":
    main()
