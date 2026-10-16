"""Publish live/dead/earthed sections of the diagram while running (display only)."""
import time
from studio_app.domain.energization import Sections, describe
from studio_app.domain.interlock_hints import switching_hints

EMPTY = {"nodes": {}, "edges": {}, "hints": {}}
# ScadaBench's own command reaches the worker as an ordinary MMS control; it is not reported as external.
OWN_COMMAND_S = 20


class EnergizationStates:
    def __init__(self, coordinator):
        self.coordinator = coordinator
        self.diagram = None
        self.last = None
        self.own = {}

    def reset(self, diagram):
        self.diagram = diagram
        self.last = None

    def _current(self):
        if self.coordinator.state not in {"running", "paused"} or not self.diagram:
            return None
        sections = Sections(self.diagram, self.coordinator.equipment_states.positions())
        return {**describe(sections), "hints": switching_hints(sections)}

    def publish(self, force=False):
        """Emit one event when the sections changed; never writes a signal."""
        result = self._current()
        if result is None or (not force and result == self.last):
            return
        self.last = result
        self.coordinator.emit("energization", **result)

    def snapshot(self):
        return self._current() or EMPTY

    def mark_own(self, node_id):
        self.own[node_id] = time.monotonic() + OWN_COMMAND_S

    def observe(self, node, action, origin):
        """An external client switched against a hint: say so in Activity (never blocks)."""
        if self.own.pop(node["id"], 0) > time.monotonic():
            return
        hints = ((self.last or EMPTY)["hints"].get(node["id"]) or {}).get(action) or []
        for hint in hints:
            self.coordinator.emit("interlock_hint_observed", node_id=node["id"], action=action,
                                  origin=origin, code=hint["code"],
                                  message=f"{origin} {'closed' if action == 'close' else 'opened'} "
                                          f"{node['data']['label']}. {hint['text']} A real station's "
                                          "interlocking would usually stop this.")
