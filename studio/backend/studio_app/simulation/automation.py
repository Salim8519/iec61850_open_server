"""Evaluate sources and fan out bounded batches across independent native IEDs."""
from collections import defaultdict
from concurrent.futures import ThreadPoolExecutor
from .generators import value_at


class Automation:
    def __init__(self, coordinator):
        self.coordinator = coordinator
        self.pool = ThreadPoolExecutor(max_workers=20, thread_name_prefix="studio-source")

    def advance(self, elapsed):
        coordinator = self.coordinator
        grouped = defaultdict(list)
        with coordinator.lock:
            items = list(coordinator.sources.items())
        for (ied_id, reference), spec in items:
            if spec["mode"] == "manual" or spec.get("overridden"):
                continue
            try:
                value = value_at(spec, elapsed)
                if coordinator.signal(ied_id, reference)["kind"].startswith("INT"):
                    value = int(value)
                grouped[ied_id].append((reference, value, None))
            except Exception as error:
                self.failed(ied_id, [(reference, None, None)], error)
        futures = [self.pool.submit(self.apply, ied_id, points) for ied_id, points in grouped.items()]
        for future in futures:
            future.result()

    def apply(self, ied_id, points):
        for offset in range(0, len(points), 80):
            batch = points[offset:offset + 80]
            if self.coordinator.state != "running":
                return
            try:
                self.coordinator.sample_writer.apply(ied_id, batch, "generator")
            except Exception as error:
                self.failed(ied_id, batch, error)

    def failed(self, ied_id, points, error):
        for reference, _, _ in points:
            spec = self.coordinator.sources.get((ied_id, reference))
            if spec:
                spec["overridden"] = True
            self.coordinator.emit("source_failed", ied_id=ied_id, ref=reference, message=str(error))

    def close(self):
        self.pool.shutdown(wait=True, cancel_futures=True)
