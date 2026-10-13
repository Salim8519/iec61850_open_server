"""Project authoritative equipment states from one or more exact IED signals."""
from collections import defaultdict
from studio_app.domain.equipment_state import binding_keys, evaluate_state


class EquipmentStates:
    def __init__(self, coordinator):
        self.coordinator = coordinator
        self.nodes = {}
        self.by_source = defaultdict(set)
        self.last = {}

    def reset(self, diagram):
        self.nodes = {node["id"]: node["data"]["state_mapping"]
                      for node in diagram["nodes"] if node["data"].get("state_mapping")}
        for node in diagram["nodes"]:
            data = node["data"]
            if data["kind"] != "RELAY":
                continue
            for entry in data.get("relay_statuses") or []:
                self.nodes[f"{node['id']}:{entry['id']}"] = {"mode": "direct", "source": entry["source"]}
        self.by_source = defaultdict(set)
        self.last = {}
        for node_id, mapping in self.nodes.items():
            for key in binding_keys(mapping):
                self.by_source[key].add(node_id)

    def _read(self, key):
        signal = self.coordinator.index.get(key)
        return signal.get("value") if signal else None

    def _publish(self, node_id, force=False):
        value = evaluate_state(self.nodes[node_id], self._read)
        if force or self.last.get(node_id) != value:
            self.last[node_id] = value
            self.coordinator.emit("equipment_state", node_id=node_id, value=value)

    def publish_all(self):
        for node_id in self.nodes:
            self._publish(node_id, True)

    def publish_node(self, node_id):
        if node_id in self.nodes:
            self._publish(node_id, True)

    def snapshot(self):
        if self.coordinator.state not in {"running", "paused"}:
            return []
        return [{"node_id": node_id, "value": evaluate_state(mapping, self._read)}
                for node_id, mapping in self.nodes.items()]

    def publish_changed(self, keys):
        affected = set()
        for key in keys:
            affected.update(self.by_source.get(key, ()))
        for node_id in affected:
            self._publish(node_id)
