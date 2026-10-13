"""Mirror applied native MMS controls into explicitly mapped equipment state."""
from collections import defaultdict
import queue
import threading

from studio_app.domain.equipment_control import feedback_targets, semantic_command


class NativeControlFeedback:
    def __init__(self, coordinator):
        self.coordinator = coordinator
        self.lock = threading.RLock()
        self.generation = 0
        self.by_control = defaultdict(list)
        self.items = queue.Queue()
        self.thread = threading.Thread(target=self._run, daemon=True)
        self.thread.start()

    def reset(self, diagram):
        grouped = defaultdict(list)
        for node in diagram["nodes"]:
            binding = node["data"].get("control_binding")
            if binding:
                grouped[(binding["ied_id"], binding["ref"])].append(node)
        with self.lock:
            self.generation += 1
            self.by_control = grouped

    def clear(self):
        with self.lock:
            self.generation += 1
            self.by_control = defaultdict(list)

    def enqueue(self, ied_id, event):
        ctl_value = event.get("ctl_value")
        if not isinstance(ctl_value, bool):
            self.coordinator.emit("equipment_feedback_failed", ied_id=ied_id,
                                  control_ref=event.get("ref"),
                                  message="Native control event has no Boolean ctlVal")
            return
        with self.lock:
            generation = self.generation
        self.items.put((generation, ied_id, dict(event)))

    def _run(self):
        while True:
            item = self.items.get()
            if item is None:
                return
            generation, ied_id, event = item
            self._process(generation, ied_id, event)

    def _process(self, generation, ied_id, event):
        with self.lock:
            if generation != self.generation:
                return
            nodes = list(self.by_control.get((ied_id, event["ref"]), ()))
        with self.coordinator.cycle_lock:
            with self.lock:
                if generation != self.generation:
                    return
            if self.coordinator.state not in {"running", "paused"}:
                return
            for node in nodes:
                self._apply_node(node, ied_id, event)
            self.coordinator.equipment_states.publish_changed(
                [(ied_id, event.get("status_ref"))])

    def _apply_node(self, node, ied_id, event):
        coordinator = self.coordinator
        data = node["data"]
        try:
            action, wire_value = semantic_command(data, value=event["ctl_value"])
            targets = feedback_targets(data, data.get("state_mapping"), action)
            native_status = (ied_id, event.get("status_ref"))
            grouped = defaultdict(list)
            for binding, value in targets:
                key = (binding["ied_id"], binding["ref"])
                signal = coordinator.index.get(key)
                if ((key != native_status or value != event.get("value"))
                        and (not signal or signal.get("value") != value)):
                    grouped[binding["ied_id"]].append((binding["ref"], value, None))
            changed = 0
            for target_ied, items in grouped.items():
                result = coordinator.sample_writer.apply(
                    target_ied, items, "control_feedback")
                changed += sum(reference in result for reference, _, _ in items)
            coordinator.emit("equipment_feedback_observed", node_id=node["id"],
                             control_ref=event["ref"], target_count=len(targets),
                             changed_count=changed, action=action,
                             wire_value=wire_value,
                             raw_wire_value=event.get("raw_ctl_value", wire_value),
                             compatibility=event.get("compatibility"),
                             origin="native_control")
        except Exception as error:
            coordinator.emit("equipment_feedback_failed", node_id=node["id"],
                             control_ref=event.get("ref"), origin="native_control",
                             message=str(error))
        finally:
            coordinator.equipment_states.publish_node(node["id"])

    def close(self):
        self.clear()
        self.items.put(None)
        self.thread.join(timeout=3)
