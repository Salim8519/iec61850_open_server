"""Apply a typed MMS command, then drive the selected simulated feedback."""
from collections import defaultdict
from studio_app.domain.equipment_control import SWITCHES, expected_cdc, feedback_targets, semantic_command
from studio_app.runtime.mms_client import operate as mms_operate
from studio_app.domain.network_settings import command_refusal


def start_position(data, action):
    """Persisted start position after a command: switches use open/close directly."""
    if data["kind"] in SWITCHES:
        return {"initial_position": "closed" if action == "close" else "open"}
    return {"initial_position": "open"}


class EquipmentControl:
    def __init__(self, coordinator):
        self.coordinator = coordinator

    def _validate_target(self, binding):
        coordinator = self.coordinator
        key = (binding["ied_id"], binding["ref"])
        signal = coordinator.index.get(key)
        if (not signal or not signal["writable"] or signal["fc"] == "CF"
                or binding["ied_id"] not in coordinator.supervisor.workers):
            raise ValueError("Mapped equipment state point cannot be updated while running")
        return signal

    def _resolve(self, node_id, value, action):
        coordinator = self.coordinator
        node = coordinator.projects.find_node(node_id)
        data = node["data"]
        semantic, wire_value = semantic_command(data, action, value)
        binding = data.get("control_binding")
        if not binding:
            raise ValueError("Equipment has no command mapping")
        ied = coordinator.projects.find_ied(binding["ied_id"])
        refusal = command_refusal(ied)
        if refusal:
            raise ValueError(refusal)
        control = next((item for item in ied["controls"] if item["ref"] == binding["ref"]), None)
        if not control or control.get("cdc") != expected_cdc(data["kind"]):
            raise ValueError("Equipment command mapping is incompatible")
        targets = feedback_targets(data, data.get("state_mapping"), semantic)
        native_status = (ied["id"], control["status"])
        planned = [(item, target) for item, target in targets
                   if ((item["ied_id"], item["ref"]) != native_status
                       or target != wire_value)]
        for item, _ in planned:
            self._validate_target(item)
        return node, ied, control, planned, semantic, wire_value

    def operate(self, node_id, value=None, action=None):
        coordinator = self.coordinator
        with coordinator.cycle_lock:
            if coordinator.state not in {"running", "paused"}:
                raise ValueError("Run the IEDs before operating equipment")
            node = coordinator.projects.find_node(node_id)
            data = node["data"]
            if data["kind"] == "PUSH_BUTTON" and data["push_mode"] == "latching_status":
                return self._toggle(node, action)
            if data["kind"] == "PUSH_BUTTON" and data["push_mode"] == "self_resetting":
                return self._momentary_status(node, action)
            node, ied, control, planned, semantic, wire_value = self._resolve(
                node_id, value, action)
            coordinator.energization.mark_own(node_id)
            coordinator.mms_access.mark_own(ied["id"], control["ref"])
            supervisor = coordinator.supervisor
            result = mms_operate(supervisor.executable, ied, control["ref"], wire_value,
                                 supervisor.endpoint(ied["id"]))
            try:
                applied, changed = self._apply(planned)
            except Exception as error:
                coordinator.emit("equipment_feedback_failed", node_id=node_id,
                                 control_ref=control["ref"], message=str(error))
                raise RuntimeError(
                    "MMS command was applied, but simulated equipment feedback failed") from error
            coordinator.emit("equipment_feedback_applied", node_id=node_id,
                             control_ref=control["ref"], target_count=len(applied),
                             changed_count=changed,
                             action=semantic, wire_value=wire_value,
                             origin="studio_api")
            coordinator.projects.node_data(node_id, start_position(data, semantic))
            coordinator.equipment_states.publish_node(node_id)
            return {"command": result, "feedback": applied,
                    "node_id": node["id"], "action": semantic,
                    "wire_value": wire_value}

    def _momentary_status(self, node, action):
        if action not in {None, "press"}:
            raise ValueError("Action is incompatible with this equipment")
        mapping = node["data"].get("state_mapping")
        if not mapping or mapping.get("mode") != "direct":
            raise ValueError("Assign a Boolean state point before pressing this button")
        binding = mapping["source"]
        signal = self._validate_target(binding)
        if signal["kind"] != "BOOLEAN" or not isinstance(signal.get("value"), bool):
            raise ValueError("Self-resetting push button requires a Boolean state point")
        rest = bool(node["data"].get("push_rest_value", False))
        duration = node["data"].get("push_reset_ms", 1000)
        if isinstance(duration, bool) or not isinstance(duration, int):
            raise ValueError("Self-resetting push button requires a reset delay in milliseconds")
        started = self.coordinator.pulses.start(binding["ied_id"], binding["ref"], not rest, duration)
        self.coordinator.projects.node_data(
            node["id"], {"initial_position": "closed" if rest else "open"})
        self.coordinator.emit("equipment_button_applied", node_id=node["id"],
                              action="press", value=started["value"])
        return {"node_id": node["id"], "action": "press", "wire_value": None,
                "feedback": [{"ied_id": binding["ied_id"], "ref": binding["ref"],
                              "value": started["value"]}]}

    def _toggle(self, node, action):
        if action not in {None, "press"}:
            raise ValueError("Action is incompatible with this equipment")
        mapping = node["data"].get("state_mapping")
        if not mapping or mapping.get("mode") != "direct":
            raise ValueError("Assign a Boolean state point before pressing this button")
        binding = mapping["source"]
        signal = self._validate_target(binding)
        if signal["kind"] != "BOOLEAN" or not isinstance(signal.get("value"), bool):
            raise ValueError("Latching push button requires a Boolean state point")
        value = not signal["value"]
        result = self.coordinator.sample_writer.apply(
            binding["ied_id"], [(binding["ref"], value, None)], "equipment_button")
        actual = result[binding["ref"]]
        self.coordinator.projects.node_data(
            node["id"], {"initial_position": "closed" if value else "open"})
        self.coordinator.emit("equipment_button_applied", node_id=node["id"],
                              action="press", value=actual)
        return {"node_id": node["id"], "action": "press", "wire_value": None,
                "feedback": [{"ied_id": binding["ied_id"], "ref": binding["ref"],
                              "value": actual}]}

    def _apply(self, planned):
        previous = {(binding["ied_id"], binding["ref"]):
                    self.coordinator.index[(binding["ied_id"], binding["ref"])].get("value")
                    for binding, _ in planned}
        grouped = defaultdict(list)
        for binding, value in planned:
            grouped[binding["ied_id"]].append((binding["ref"], value, None))
        applied = []
        for ied_id, items in grouped.items():
            result = self.coordinator.sample_writer.apply(
                ied_id, items, "control_feedback")
            applied.extend({"ied_id": ied_id, "ref": reference, "value": result[reference]}
                           for reference, _, _ in items)
        changed = sum(previous[(item["ied_id"], item["ref"])] != item["value"]
                      for item in applied)
        return applied, changed
