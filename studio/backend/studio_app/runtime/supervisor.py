"""Start, initialize, and stop a group of isolated native IED workers."""
from pathlib import Path
from .worker_process import WorkerProcess
from .worker_settings import check_capabilities, start_request
from studio_app.domain.network_settings import DEFAULT_NETWORK
from studio_app.networking.endpoints import preflight
from studio_app.networking.ip_helper import addresses_on
from studio_app.networking.loopback import assign_endpoints


class Supervisor:
    def __init__(self, executable, work_directory, on_event, lab_addresses=addresses_on):
        self.executable = Path(executable)
        self.work_directory = Path(work_directory)
        self.on_event = on_event
        self.lab_addresses = lab_addresses
        self.workers = {}
        self.endpoints = {}

    def start(self, ieds, network=None):
        if self.workers:
            raise ValueError("Simulation is already running")
        if not ieds:
            raise ValueError("Import or create an IED before Run")
        self.add(ieds, network)

    def endpoint(self, ied_id):
        """The address and port this IED's worker confirmed it listens on (lab or loopback)."""
        if ied_id not in self.endpoints:
            raise ValueError("IED is not running")
        return self.endpoints[ied_id]

    def add(self, ieds, network=None, running=()):
        """Start workers for more IEDs; `running` are the IEDs whose workers already listen."""
        if not self.executable.exists():
            raise ValueError("Native IED engine is not built yet")
        if any(ied["id"] in self.workers for ied in ieds):
            raise ValueError("This IED is already running")
        network = network or dict(DEFAULT_NETWORK)
        assign_endpoints([*running, *ieds])
        allowed = self.lab_addresses(network["adapter_guid"]) if network["mode"] == "lab" else ()
        preflight(ieds, network, allowed)
        self.work_directory.mkdir(parents=True, exist_ok=True)
        try:
            for ied in ieds:
                self._launch(ied, network)
        except Exception:
            self.stop_some(ieds)
            raise

    def _launch(self, ied, network):
        config = self.work_directory / (ied["id"] + ".cfg")
        config.write_text(ied["cfg"], encoding="utf-8")
        worker = WorkerProcess(self.executable, lambda event, key=ied["id"]: self.on_event(key, event))
        self.workers[ied["id"]] = worker
        request = start_request(ied, network, ied["controls"], config.resolve())
        started = worker.request("start", **request)
        check_capabilities(started)
        self.endpoints[ied["id"]] = (started.get("ip", request["ip"]), int(started.get("port", request["port"])))
        ied["goose_state"] = started.get("goose", {})
        for signal in ied["signals"]:
            if signal["kind"] in {"Dbpos", "Tcmd", "Quality", "Timestamp"} and signal["value"]:
                worker.request("update", ref=signal["ref"], value=signal["value"])
        ied["state"] = "running"

    def update(self, ied_id, reference, value):
        if ied_id not in self.workers:
            raise ValueError("IED is not running")
        return self.workers[ied_id].request("update", ref=reference, value=value)

    def update_sample(self, ied_id, updates):
        if ied_id not in self.workers:
            raise ValueError("IED is not running")
        return self.workers[ied_id].request("update_batch", updates=updates)

    def read(self, ied_id, reference):
        if ied_id not in self.workers:
            raise ValueError("IED is not running")
        return self.workers[ied_id].request("read", ref=reference)

    def read_versioned(self, ied_id, reference):
        return self.workers[ied_id].request("read_versioned", ref=reference)

    def update_guarded(self, ied_id, updates, guard):
        return self.workers[ied_id].request("update_guarded", updates=updates, guard=guard)

    def stop_some(self, ieds):
        """Stop the workers of these IEDs only; other IEDs keep running."""
        for ied in ieds:
            worker = self.workers.pop(ied["id"], None)
            self.endpoints.pop(ied["id"], None)
            if worker:
                worker.close()
            ied["state"] = "stopped"
            ied.pop("goose_state", None)

    def stop(self, ieds):
        for worker in self.workers.values():
            worker.close()
        self.workers.clear()
        self.endpoints.clear()
        for ied in ieds:
            ied["state"] = "stopped"
            ied.pop("goose_state", None)
