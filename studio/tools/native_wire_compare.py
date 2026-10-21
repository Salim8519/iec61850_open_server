"""Compare what two studio_ied builds put on the wire for the same scripted session.

Usage: python studio/tools/native_wire_compare.py BASE_BIN NEW_BIN
Each build serves a synthetic template IED behind a recording TCP proxy; studio_probe (from
BASE_BIN) browses, reads, operates and receives reports through it while the worker gets
the same IPC updates. Server-to-client MMS bytes must match after masking timestamps
(UtcTime, BinaryTime). GOOSE frames sent to a stand-in engine must match after masking t.
127.67.x.y, ports >= 20000, synthetic/public SCL only (workstream G evidence).
"""
import json
import socket
import subprocess
import sys
import tempfile
import threading
import time
from pathlib import Path

from native_bench_rig import GooseSink, Worker, goose_model, import_scl, make_template

IED = "Wire"
LISTEN, TARGET = ("127.67.0.20", 20020), ("127.67.0.21", 20021)


class Proxy:
    """Forwards one MMS client at a time and records the server's bytes per connection."""

    def __init__(self):
        self.server = socket.create_server(LISTEN)
        self.streams = []
        threading.Thread(target=self._accept, daemon=True).start()

    def _accept(self):
        while True:
            try:
                client, _ = self.server.accept()
            except OSError:
                return
            upstream = socket.create_connection(TARGET)
            record = bytearray()
            self.streams.append(record)
            threading.Thread(target=self._pump, args=(client, upstream, None), daemon=True).start()
            threading.Thread(target=self._pump, args=(upstream, client, record), daemon=True).start()

    @staticmethod
    def _pump(source, sink, record):
        try:
            while data := source.recv(65536):
                if record is not None:
                    record.extend(data)
                sink.sendall(data)
        except OSError:
            pass
        finally:
            for side in (source, sink):
                try:
                    side.shutdown(socket.SHUT_RDWR)
                except OSError:
                    pass

    def close(self):
        self.server.close()


def mask(stream):
    """Zero UtcTime (0x91 0x08) and BinaryTime (0x8c 0x06) contents in BER data."""
    data = bytearray(stream)
    for tag, size in ((0x91, 8), (0x8C, 6)):
        index = data.find(bytes([tag, size]))
        while index >= 0:
            data[index + 2:index + 2 + size] = bytes(size)
            index = data.find(bytes([tag, size]), index + 2 + size)
    return bytes(data)


def probe(binary, *args, timeout=20):
    command = [str(Path(binary) / "studio_probe.exe"), LISTEN[0], str(LISTEN[1]), *args]
    return subprocess.run(command, capture_output=True, text=True, timeout=timeout).stdout.strip()


def mms_session(binary, probe_binary, folder):
    source = make_template(IED).replace(b'intgPd="1000"', b'intgPd="0"')
    model = import_scl(source)[0]
    worker = Worker(binary, model, folder, *TARGET)
    proxy = Proxy()
    outputs = []
    try:
        status = f"{IED}CTRL/XCBR1.Pos.stVal"
        outputs.append(probe(probe_binary, "browse"))
        outputs.append(probe(probe_binary, "read", status, "ST"))
        outputs.append(probe(probe_binary, "control", f"{IED}CTRL/XCBR1.Pos", "true"))
        outputs.append(probe(probe_binary, "read", status, "ST"))
        reports = subprocess.Popen([str(Path(probe_binary) / "studio_probe.exe"), *map(str, LISTEN), "reports",
                                    f"{IED}CTRL/LLN0.RP.EventsRCB"], stdout=subprocess.PIPE, text=True)
        reports.stdout.readline()
        for value in (1, 2, 1):
            time.sleep(0.3)
            worker.request("update", ref=status, value=value)
        outputs.append(reports.communicate(timeout=20)[0].strip())
        time.sleep(0.3)
        return outputs, [mask(stream) for stream in proxy.streams]
    finally:
        proxy.close()
        worker.close()


def goose_payloads(frames):
    """GOOSE APDUs with the t field (tag 0x84, 8 bytes) zeroed; Ethernet header kept."""
    result = []
    for _, frame in frames:
        data = bytearray(frame)
        index = data.find(b"\x84\x08")
        if index >= 0:
            data[index + 2:index + 10] = bytes(8)
        result.append(bytes(data))
    return result


def goose_session(binary, folder):
    sink = GooseSink()
    worker = Worker(binary, goose_model(), folder, "127.67.0.22", 20022, sink.interface())
    try:
        time.sleep(1.5)
        for value in (2, 1, 2):
            worker.request("update", ref="IED1_XCBRGenericIO/XCBR1.Pos.stVal", value=value)
            time.sleep(0.6)
        return goose_payloads(sink.frames)
    finally:
        worker.close()
        sink.close()


def main(arguments):
    base, new = arguments[0], arguments[1]
    with tempfile.TemporaryDirectory(prefix="g_wire_") as folder:
        runs = {name: mms_session(binary, base, folder) for name, binary in (("base", base), ("new", new))}
        goose = {name: goose_session(binary, folder) for name, binary in (("base", base), ("new", new))}
    (base_out, base_streams), (new_out, new_streams) = runs["base"], runs["new"]
    same_streams = [a == b for a, b in zip(base_streams, new_streams)]
    print(f"MMS connections: base {len(base_streams)}, new {len(new_streams)}; identical after masking: {same_streams}")
    print(f"MMS bytes: base {[len(s) for s in base_streams]}, new {[len(s) for s in new_streams]}")
    reports = [len(json.loads(output[-1]).get("reports", [])) for output in (base_out, new_out)]
    print(f"probe outputs identical: {base_out == new_out}; reports received base/new: {reports}")
    count = min(len(goose["base"]), len(goose["new"]))
    distinct = sorted(set(goose["base"][:count]) ^ set(goose["new"][:count]))
    print(f"GOOSE frames: base {len(goose['base'])}, new {len(goose['new'])}; "
          f"first {count} identical in order: {goose['base'][:count] == goose['new'][:count]}; "
          f"distinct frames: {len(distinct)}")
    ok = (len(base_streams) == len(new_streams) and all(same_streams) and base_out == new_out
          and goose["base"][:count] == goose["new"][:count])
    print("WIRE IDENTICAL" if ok else "WIRE DIFFERS")
    return 0 if ok else 1


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
