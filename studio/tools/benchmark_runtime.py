"""Record real coordinator/worker load; short runs are explicitly not acceptance."""
import argparse
import json
import os
from pathlib import Path
import platform
import sys
import threading
import time
from uuid import uuid4

STUDIO = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(STUDIO / "backend"))
from studio_app.projects.service import ProjectService
from studio_app.runtime.event_bus import EventBus
from studio_app.simulation.coordinator import Coordinator
from load_fixture import populate
from process_memory import working_set
from benchmark_provenance import capture, changed_files


class MetricsBus(EventBus):
    def __init__(self):
        super().__init__()
        self.metric_lock = threading.Lock()
        self.applied = 0
        self.failures = 0
        self.late_cycles = 0
        self.skipped_ticks = 0
        self.maximum_lateness = 0

    def publish(self, kind, **payload):
        with self.metric_lock:
            self.applied += kind == "signal" and payload.get("source") == "generator"
            self.failures += kind in {"source_failed", "worker_failed", "automation_failed"}
            if kind == "automation_late":
                self.late_cycles += 1
                self.skipped_ticks += payload["skipped_ticks"]
                self.maximum_lateness = max(self.maximum_lateness, payload["lateness_seconds"])
        return super().publish(kind, **payload)

    def metrics(self):
        with self.metric_lock:
            return {"applied": self.applied, "failures": self.failures,
                    "late_cycles": self.late_cycles, "skipped_ticks": self.skipped_ticks,
                    "maximum_reported_lateness_seconds": self.maximum_lateness}


def benchmark(seconds, port=18120):
    directory = STUDIO.parent / ".local/benchmarks" / uuid4().hex
    directory.mkdir(parents=True)
    projects = ProjectService(directory)
    ieds = populate(projects, port=port)
    provenance = capture(STUDIO)
    (directory / "provenance-start.json").write_text(json.dumps(provenance, indent=2), encoding="utf-8")
    bus = MetricsBus()
    runtime = Coordinator(projects, bus, STUDIO / "native/build/Release/studio_ied.exe", directory)
    result = {"hardware": platform.processor(), "logical_cpus": os.cpu_count(),
              "ieds": len(ieds), "attributes": sum(len(ied["signals"]) for ied in ieds),
              "requested_seconds": seconds, "target_updates_per_second": 1000,
              "directory": str(directory), "ui_verified": False, "report_loss_verified": False,
              "port": port, "source_sha256": provenance["sha256"]}
    print(json.dumps({"started": result}), flush=True)
    try:
        start = time.monotonic()
        runtime.start()
        result["startup_seconds"] = time.monotonic() - start
        start = time.monotonic()
        previous = 0
        peak = 0
        samples = []
        while time.monotonic() - start < seconds:
            time.sleep(min(1, max(0, seconds - (time.monotonic() - start))))
            metrics = bus.metrics()
            total = metrics["applied"]
            memory = working_set(os.getpid()) + sum(working_set(worker.process.pid)
                     for worker in runtime.supervisor.workers.values())
            peak = max(peak, memory)
            sample = {"elapsed": time.monotonic() - start, **metrics,
                      "interval_updates": total - previous, "working_set_bytes": memory,
                      "runtime_state": runtime.state}
            samples.append(sample)
            previous = total
            if len(samples) % 10 == 0:
                print(json.dumps(sample), flush=True)
            (directory / "progress.json").write_text(json.dumps(sample), encoding="utf-8")
        runtime.pause()
        elapsed = time.monotonic() - start
        result.update(elapsed_seconds=elapsed, **bus.metrics(),
                      updates_per_second=bus.applied / elapsed, peak_working_set_bytes=peak, samples=samples)
    finally:
        runtime.close()
        result["workers_stopped"] = not runtime.supervisor.workers
        final_provenance = capture(STUDIO)
        (directory / "provenance-end.json").write_text(json.dumps(final_provenance, indent=2), encoding="utf-8")
        result["changed_files_during_run"] = changed_files(provenance, final_provenance)
        (directory / "result.json").write_text(json.dumps(result, indent=2), encoding="utf-8")
    print(json.dumps({key: value for key, value in result.items() if key != "samples"}), flush=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--seconds", type=int, default=30)
    parser.add_argument("--port", type=int, default=18120)
    args = parser.parse_args()
    if not 1 <= args.seconds <= 3600:
        parser.error("Use 1 to 3600 seconds; full-duration acceptance requires 3600")
    if not 1024 <= args.port <= 65535:
        parser.error("Benchmark port must be between 1024 and 65535")
    benchmark(args.seconds, args.port)
