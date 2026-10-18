"""Keep the MMS client list of every simulated IED during a Run (ADR-0066)."""
import threading
import time
from studio_app.domain.mms_access import client_view, mark_own_peer, merge_summary

# ScadaBench's own command goes out through studio_probe as an ordinary MMS control.
OWN_COMMAND_S = 20


class MmsAccess:
    def __init__(self, coordinator):
        self.coordinator = coordinator
        self.lock = threading.Lock()
        self.clients = {}
        self.own = {}

    def reset(self):
        with self.lock:
            self.clients, self.own = {}, {}

    def mark_own(self, ied_id, control_ref):
        with self.lock:
            self.own[(ied_id, control_ref)] = time.monotonic() + OWN_COMMAND_S

    def control(self, ied_id, event):
        """True when an MMS control came from another client, not from ScadaBench itself."""
        peer = event.get("peer") or ""
        with self.lock:
            own = self.own.pop((ied_id, event.get("ref")), 0) > time.monotonic()
            if own and peer:
                mark_own_peer(self.clients.setdefault(ied_id, {}), peer, time.monotonic())
        return bool(peer) and not own

    def update(self, ied_id, event):
        """Merge one worker summary and publish the client list (live view only, not the run log)."""
        now = time.monotonic()
        with self.lock:
            clients = merge_summary(self.clients.setdefault(ied_id, {}), event.get("clients") or [], now)
            rows = client_view(clients, now)
        self.coordinator.events.publish("mms_access", ied_id=ied_id, clients=rows)

    def snapshot(self):
        now = time.monotonic()
        with self.lock:
            return {ied_id: client_view(clients, now) for ied_id, clients in self.clients.items()}
