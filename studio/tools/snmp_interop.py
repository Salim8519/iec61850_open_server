r"""Independent-client interop for studio_snmp.exe (ADR-0092) using pysnmp 7.1.21.

A: pysnmp manager -> ScadaBench agent: v1, v2c and v3 with every auth x priv, GET, walk, SET.
B: ScadaBench agent traps/informs (v1, v2c, v2c inform, v3 trap, v3 inform) -> pysnmp receiver.
C: ScadaBench manager -> pysnmp agent (v2c and v3 Reeder/Blumenthal AES-256): open, walk, set.
Loopback 127.66.50.x, ports >= 20000. Not part of verify.py: pysnmp is a test-only tool, so run it
from its own virtual environment (python -m venv snmp-venv; snmp-venv\Scripts\pip install pysnmp==7.1.21):
    snmp-venv\Scripts\python studio/tools/snmp_interop.py studio/native/build/Release/studio_snmp.exe
"""
import asyncio, json, os, struct, subprocess, sys, threading, queue
from pysnmp.hlapi.v3arch.asyncio import (SnmpEngine, UsmUserData, CommunityData, UdpTransportTarget, ContextData,
    ObjectType, ObjectIdentity, OctetString, get_cmd, set_cmd, walk_cmd, bulk_walk_cmd,
    usmHMACMD5AuthProtocol, usmHMACSHAAuthProtocol, usmHMAC128SHA224AuthProtocol, usmHMAC192SHA256AuthProtocol,
    usmHMAC256SHA384AuthProtocol, usmHMAC384SHA512AuthProtocol, usmDESPrivProtocol, usmAesCfb128Protocol,
    usmAesBlumenthalCfb192Protocol, usmAesBlumenthalCfb256Protocol, usmAesCfb192Protocol, usmAesCfb256Protocol)
from pysnmp.entity import engine as lowengine, config
from pysnmp.carrier.asyncio.dgram import udp
from pysnmp.entity.rfc3413 import ntfrcv, cmdrsp, context
from pysnmp.proto.api import v2c

AUTH = {"md5": usmHMACMD5AuthProtocol, "sha": usmHMACSHAAuthProtocol, "sha224": usmHMAC128SHA224AuthProtocol,
        "sha256": usmHMAC192SHA256AuthProtocol, "sha384": usmHMAC256SHA384AuthProtocol, "sha512": usmHMAC384SHA512AuthProtocol}
PRIV = {"des": usmDESPrivProtocol, "aes128": usmAesCfb128Protocol, "aes192": usmAesBlumenthalCfb192Protocol,
        "aes256": usmAesBlumenthalCfb256Protocol, "aes192c": usmAesCfb192Protocol, "aes256c": usmAesCfb256Protocol}
AGENT, RX, PYAGENT = ("127.66.50.1", 20261), ("127.66.50.2", 20262), ("127.66.50.3", 20263)
ENGINE = "8000000005aabbccdd11"
results = []


def check(name, ok, detail=""):
    results.append((name, ok))
    print(("PASS " if ok else "FAIL ") + name + (f"  [{detail}]" if detail else ""), flush=True)


class Worker:
    def __init__(self, exe):
        self.p = subprocess.Popen([os.path.abspath(exe)], stdin=subprocess.PIPE, stdout=subprocess.PIPE)
        self.replies, self.events, self.n = {}, queue.Queue(), 0
        threading.Thread(target=self.read, daemon=True).start()

    def read(self):
        while True:
            head = self.p.stdout.read(4)
            if len(head) < 4:
                return
            frame = json.loads(self.p.stdout.read(struct.unpack("<I", head)[0]))
            if "event" in frame:
                self.events.put(frame)
            else:
                self.replies[frame["id"]].put(frame)

    def call(self, op, **fields):
        self.n += 1
        key = f"r{self.n}"
        self.replies[key] = queue.Queue()
        data = json.dumps({"version": 1, "id": key, "op": op, **fields}).encode()
        self.p.stdin.write(struct.pack("<I", len(data)) + data)
        self.p.stdin.flush()
        reply = self.replies[key].get(timeout=60)
        if not reply["ok"]:
            raise RuntimeError(reply["error"])
        return reply.get("result")


def users():
    out = [{"name": "plain", "level": "noAuthNoPriv"}]
    for auth in AUTH:
        out.append({"name": f"a_{auth}", "level": "authNoPriv", "auth": auth, "auth_password": "authpassword"})
        for priv in PRIV:
            out.append({"name": f"{auth}_{priv}", "level": "authPriv", "auth": auth, "auth_password": "authpassword",
                        "priv": priv, "priv_password": "privpassword"})
    out.append({"name": "trapper", "level": "authPriv", "auth": "sha256", "auth_password": "authpassword",
                "priv": "aes128", "priv_password": "privpassword"})
    return out


POINTS = [{"oid": "1.3.6.1.2.1.1.1.0", "type": "octets", "value": "ScadaBench interop switch"},
          {"oid": "1.3.6.1.2.1.1.2.0", "type": "oid", "value": "1.3.6.1.4.1.99999.1"},
          {"oid": "1.3.6.1.2.1.1.3.0", "type": "timeticks", "value": "0", "behaviour": "uptime"},
          {"oid": "1.3.6.1.2.1.1.5.0", "type": "octets", "value": "sw1", "writable": True},
          {"oid": "1.3.6.1.2.1.1.6.0", "type": "octets", "value": "Room 1", "writable": True},
          {"oid": "1.3.6.1.2.1.2.2.1.6.1", "type": "octets", "value": "02 5c 42 00 00 01", "hex": True},
          {"oid": "1.3.6.1.2.1.2.2.1.8.1", "type": "integer", "value": "1"},
          {"oid": "1.3.6.1.2.1.4.20.1.1.10.0.0.1", "type": "ipaddress", "value": "10.0.0.1"},
          {"oid": "1.3.6.1.2.1.31.1.1.1.6.1", "type": "counter64", "value": "18446744073709551000"},
          {"oid": "1.3.6.1.2.1.31.1.1.1.15.1", "type": "gauge32", "value": "1000"}]


async def walk_count(snmp, auth, target, bulk):
    count, error = 0, None
    walker = bulk_walk_cmd(snmp, auth, target, ContextData(), 0, 25, ObjectType(ObjectIdentity("1.3.6.1")),
                           lexicographicMode=False) if bulk else walk_cmd(snmp, auth, target, ContextData(),
                           ObjectType(ObjectIdentity("1.3.6.1")), lexicographicMode=False)
    async for err, status, _, binds in walker:
        if err or status:
            error = str(err or status.prettyPrint())
            break
        count += len(binds)
    return count, error


async def part_a():
    target = await UdpTransportTarget.create(AGENT, timeout=2, retries=1)
    for version, model in (("v1", 0), ("v2c", 1)):
        snmp = SnmpEngine()
        count, error = await walk_count(snmp, CommunityData("public", mpModel=model), target, model == 1)
        want = len(POINTS) - (1 if model == 0 else 0)  # v1 has no Counter64
        check(f"A {version} walk", count == want and not error, f"{count}/{want} {error or ''}")
        err, status, _, binds = await set_cmd(snmp, CommunityData("private", mpModel=model), target, ContextData(),
                                              ObjectType(ObjectIdentity("1.3.6.1.2.1.1.6.0"), OctetString(f"Bay {version}")))
        err2, _, _, read = await get_cmd(snmp, CommunityData("public", mpModel=model), target, ContextData(),
                                         ObjectType(ObjectIdentity("1.3.6.1.2.1.1.6.0")))
        check(f"A {version} set+get", not err and not status and str(read[0][1]) == f"Bay {version}", str(err or status or read))
        err, status, _, _ = await set_cmd(snmp, CommunityData("private", mpModel=model), target, ContextData(),
                                          ObjectType(ObjectIdentity("1.3.6.1.2.1.1.1.0"), OctetString("x")))
        check(f"A {version} read-only refused", bool(status), status.prettyPrint())
    for user in users():
        snmp = SnmpEngine()
        if user["level"] == "noAuthNoPriv":
            auth = UsmUserData(user["name"])
        elif user["level"] == "authNoPriv":
            auth = UsmUserData(user["name"], "authpassword", authProtocol=AUTH[user["auth"]])
        else:
            auth = UsmUserData(user["name"], "authpassword", "privpassword", authProtocol=AUTH[user["auth"]],
                               privProtocol=PRIV[user["priv"]])
        count, error = await walk_count(snmp, auth, target, True)
        err, status, _, binds = await set_cmd(snmp, auth, target, ContextData(),
                                              ObjectType(ObjectIdentity("1.3.6.1.2.1.1.5.0"), OctetString(user["name"])))
        check(f"A v3 {user['name']} walk+set", count == len(POINTS) and not error and not err and not status,
              f"{count} {error or err or status or ''}")
    snmp = SnmpEngine()
    bad = UsmUserData("sha256_aes256", "wrongpassword", "privpassword", authProtocol=AUTH["sha256"],
                      privProtocol=PRIV["aes256"])
    err, _, _, _ = await get_cmd(snmp, bad, target, ContextData(), ObjectType(ObjectIdentity("1.3.6.1.2.1.1.1.0")))
    check("A v3 wrong password refused", bool(err), str(err))


async def part_b(worker):
    received = []
    rx = lowengine.SnmpEngine()
    config.add_transport(rx, udp.DOMAIN_NAME, udp.UdpTransport().open_server_mode(RX))
    config.add_v1_system(rx, "area", "public")
    for engine_id in (v2c.OctetString(hexValue=ENGINE), None):
        extra = {"securityEngineId": engine_id} if engine_id is not None else {}
        config.add_v3_user(rx, "trapper", config.USM_AUTH_HMAC192_SHA256, "authpassword", config.USM_PRIV_CFB128_AES,
                           "privpassword", **extra)
    ntfrcv.NotificationReceiver(rx, lambda e, ref, ce, cn, binds, ctx: received.append(
        {str(name): value.prettyPrint() for name, value in binds}))
    traps = await asyncio.to_thread(worker.call, "agent_trap", device="d1", trap={
        "notification": "1.3.6.1.6.3.1.1.5.3", "varbinds": [{"oid": "1.3.6.1.2.1.2.2.1.1.3", "type": "integer", "value": "3"}]})
    for result in traps["results"]:
        check(f"B send {result['target']}", result["ok"], result["text"])
    await asyncio.sleep(1)
    link_down = [r for r in received if r.get("1.3.6.1.6.3.1.1.4.1.0") == "1.3.6.1.6.3.1.1.5.3"]
    check("B pysnmp received all 5 notifications", len(link_down) == 5, f"{len(received)} received")
    check("B values intact", all(r.get("1.3.6.1.2.1.2.2.1.1.3") == "3" for r in link_down))
    rx.close_dispatcher()


async def part_c(worker):
    ag = lowengine.SnmpEngine(snmpEngineID=v2c.OctetString(hexValue="80000000050102030405"))
    config.add_transport(ag, udp.DOMAIN_NAME, udp.UdpTransport().open_server_mode(PYAGENT))
    config.add_v1_system(ag, "area", "public")
    config.add_v3_user(ag, "reeder", config.USM_AUTH_HMAC384_SHA512, "authpassword", config.USM_PRIV_CFB256_AES, "privpassword")
    config.add_v3_user(ag, "blum", config.USM_AUTH_HMAC96_SHA, "authpassword", config.USM_PRIV_CFB256_AES_BLUMENTHAL, "privpassword")
    config.add_vacm_user(ag, 2, "area", "noAuthNoPriv", (1, 3, 6, 1), (1, 3, 6, 1))
    for name in ("reeder", "blum"):
        config.add_vacm_user(ag, 3, name, "authPriv", (1, 3, 6, 1), (1, 3, 6, 1))
    snmp_context = context.SnmpContext(ag)
    for responder in (cmdrsp.GetCommandResponder, cmdrsp.NextCommandResponder, cmdrsp.BulkCommandResponder,
                      cmdrsp.SetCommandResponder):
        responder(ag, snmp_context)
    targets = [("v2c", {"version": "v2c", "community": "public"}),
               ("v3 sha512/aes256c", {"version": "v3", "user": {"name": "reeder", "level": "authPriv", "auth": "sha512",
                "auth_password": "authpassword", "priv": "aes256c", "priv_password": "privpassword"}}),
               ("v3 sha/aes256", {"version": "v3", "user": {"name": "blum", "level": "authPriv", "auth": "sha",
                "auth_password": "authpassword", "priv": "aes256", "priv_password": "privpassword"}})]
    for name, fields in targets:
        target = {"host": PYAGENT[0], "port": PYAGENT[1], "timeout_ms": 2000, "retries": 1, "user": {"name": "", "level": ""}, **fields}
        try:
            opened = await asyncio.to_thread(worker.call, "read_open", target=target)
            descr = opened["system"][0]["value"]
            walked = await asyncio.to_thread(worker.call, "read_walk", root="1.3.6.1.2.1.1")
            check(f"C {name} open+walk", "PySNMP" in descr and len(walked["values"]) > 5 and not walked["problem"],
                  f"{len(walked['values'])} values, sysDescr {descr[:40]!r}")
            written = await asyncio.to_thread(worker.call, "read_set", point={"oid": "1.3.6.1.2.1.1.4.0", "type": "octets",
                                                                             "value": f"set by {name}"})
            check(f"C {name} set", written["values"][0]["value"] == f"set by {name}", str(written))
        except Exception as error:  # noqa: BLE001
            check(f"C {name}", False, str(error))
    ag.close_dispatcher()


async def main():
    worker = Worker(sys.argv[1])
    targets = [{"host": RX[0], "port": RX[1], "version": "v1", "community": "public"},
               {"host": RX[0], "port": RX[1], "version": "v2c", "community": "public"},
               {"host": RX[0], "port": RX[1], "version": "v2c", "community": "public", "inform": True},
               {"host": RX[0], "port": RX[1], "version": "v3", "user": "trapper"},
               {"host": RX[0], "port": RX[1], "version": "v3", "user": "trapper", "inform": True}]
    device = {"id": "d1", "name": "Interop", "profile": "empty", "ip": AGENT[0], "port": AGENT[1], "v1": True, "v2c": True,
              "v3": True, "read_community": "public", "write_community": "private", "users": users(), "engine_id": ENGINE,
              "traps": targets, "points": POINTS}
    started = await asyncio.to_thread(worker.call, "agents_start", devices=[device])
    check("agent started", started["started"][0]["ok"], str(started))
    await part_a()
    await part_b(worker)
    await part_c(worker)
    worker.call("stop")
    failed = [name for name, ok in results if not ok]
    print(f"\n{len(results) - len(failed)}/{len(results)} passed" + (f"; failed: {failed}" if failed else ""))
    return 1 if failed else 0

if __name__ == "__main__":
    sys.exit(asyncio.run(main()))
