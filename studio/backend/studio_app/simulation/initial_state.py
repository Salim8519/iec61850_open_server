"""Apply saved equipment start positions to mapped switch state points at run start."""
from collections import defaultdict
from studio_app.domain.equipment_control import SWITCHES, feedback_targets

ACTION = {"open": "open", "closed": "close"}


def apply_initial_states(coordinator):
    """Write each mapped CB/DS/ES start position into its IED before publishing."""
    nodes = coordinator.projects.project["diagram"]["nodes"]
    planned = defaultdict(list)
    for node in nodes:
        data = node["data"]
        if data.get("kind") not in SWITCHES:
            continue
        targets = feedback_targets(data, data.get("state_mapping"),
                                   ACTION[data.get("initial_position", "open")])
        for binding, value in targets:
            planned[binding["ied_id"]].append((binding["ref"], value))
    for ied_id, items in planned.items():
        if ied_id not in coordinator.sample_writer.locks:
            continue
        coordinator.sample_writer.apply(ied_id, [(ref, value, None) for ref, value in items],
                                        "initial_state")
