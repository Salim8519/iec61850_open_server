"""The GOOSE engine with a lab adapter: raw publishers on its bus and a raw inbox (ADR-0073)."""
import queue
import socket
import time

from studio_app.runtime.worker_process import WorkerProcess
from goose_engine_rig import BIN
from npcap_peer import ADAPTER, mac_bytes

SHARE_CHECK_SECONDS = 3


class WireRig:
    """studio_goose.exe with a UDP socket standing in for the IEDs and another for studio_live."""

    def __init__(self, ieds):
        self.events = queue.Queue()
        self.engine = WorkerProcess(BIN / "studio_goose.exe", self.events.put)
        opened = self.engine.request("open", ieds=[{"id": ied, "gocb_refs": refs} for ied, refs in ieds.items()])
        self.bus = ("127.0.0.1", opened["port"])
        self.macs = {ied: mac_bytes(mac) for ied, mac in opened["sources"].items()}
        self.inbox = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self.inbox.bind(("127.0.0.1", 0))
        self.inbox.settimeout(0.05)
        self.engine.request("deliver", port=self.inbox.getsockname()[1])
        self.publisher = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)

    def uplink(self, share):
        return self.engine.request("uplink", adapter_guid=ADAPTER, share=share)

    def publish(self, frame):
        self.publisher.sendto(frame, self.bus)

    def inside(self, seconds):
        """(origin, frame) pairs delivered to the decoder's inbox for `seconds`."""
        found, deadline = [], time.monotonic() + seconds
        while time.monotonic() < deadline:
            try:
                data = self.inbox.recv(65536)
            except (socket.timeout, ConnectionResetError):
                continue
            found.append((data[0], data[1:]))
        return found

    def event(self, kind, seconds=6, **match):
        deadline = time.monotonic() + seconds
        while time.monotonic() < deadline:
            try:
                event = self.events.get(timeout=0.1)
            except queue.Empty:
                continue
            if event.get("event") == kind and all(event.get(k) == v for k, v in match.items()):
                return event
        return None

    def wait_sharing(self):
        self.uplink(True)
        return self.event("goose_share", seconds=SHARE_CHECK_SECONDS + 5, state="sharing")

    def close(self):
        self.engine.close()
        self.inbox.close()
        self.publisher.close()


def from_source(frames, mac):
    return [frame for frame in frames if frame[6:12] == mac]
