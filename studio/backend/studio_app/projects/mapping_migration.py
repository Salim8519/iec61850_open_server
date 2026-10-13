"""Convert legacy flat equipment mappings without inventing engineering values."""
import copy

SWITCHES = {"CB", "DS", "ES"}
LEGACY_FIELDS = {"ied_id", "status_ref", "control_ref", "report_ref", "status_kind", "position_values"}


def _indexes(ieds):
    signals = {(ied["id"], signal["ref"]): signal for ied in ieds for signal in ied["signals"]}
    controls = {(ied["id"], control["ref"]): control for ied in ieds for control in ied["controls"]}
    return signals, controls


def _signal_binding(data):
    return {"ied_id": data["ied_id"], "ref": data["status_ref"],
            "report_ref": data.get("report_ref") or None}


def _relay_entries(result, notices):
    for node in result.get("nodes", []):
        data = node.get("data", {})
        if data.get("kind") != "RELAY":
            continue
        mapping = data.pop("state_mapping", None)
        if not mapping:
            continue
        label = data.get("label", node.get("id", "relay"))
        source = mapping.get("source") if mapping.get("mode") in {"direct", "single_to_dbpos"} else None
        if source:
            data["relay_statuses"] = [{"id": "s1", "label": label, "source": source}]
            notices.append(f"{label}: existing state mapping became one named relay status")
        else:
            notices.append(f"{label}: removed unsupported state mapping while upgrading relays")


def _button_edges(result, notices):
    buttons = {node.get("id") for node in result.get("nodes", [])
               if node.get("data", {}).get("kind") == "PUSH_BUTTON"}
    if not buttons:
        return
    edges = result.get("edges", [])
    kept = [edge for edge in edges
            if edge.get("source") not in buttons and edge.get("target") not in buttons]
    removed = len(edges) - len(kept)
    if removed:
        result["edges"] = kept
        notices.append(f"{removed} wire{'s' if removed != 1 else ''} to push buttons removed: "
                       "buttons no longer accept connections")


def migrate_diagram(value, ieds, version):
    result = copy.deepcopy(value)
    if version >= 10:
        return result, []
    notices = []
    _button_edges(result, notices)
    if version >= 9:
        return result, notices
    _relay_entries(result, notices)
    if version <= 3:
        signals, controls = _indexes(ieds)
        for node in result.get("nodes", []):
            data = node.get("data", {})
            kind = data.get("kind")
            label = data.get("label", node.get("id", "equipment"))
            ied_id, status_ref = data.get("ied_id"), data.get("status_ref")
            signal = signals.get((ied_id, status_ref))
            if signal:
                binding = _signal_binding(data)
                if kind in SWITCHES and signal["kind"] == "BOOLEAN":
                    data["state_mapping"] = {"mode": "single_to_dbpos", "source": binding}
                elif kind not in SWITCHES or signal["kind"] == "Dbpos":
                    data["state_mapping"] = {"mode": "direct", "source": binding}
                else:
                    notices.append(
                        f"{label}: removed unsupported {signal['kind']} switch state mapping")
            control_ref = data.get("control_ref")
            control = controls.get((ied_id, control_ref))
            if control and kind in SWITCHES and control.get("cdc") == "DPC":
                data["control_binding"] = {"ied_id": ied_id, "ref": control_ref}
            elif control_ref:
                notices.append(f"{label}: removed incompatible control mapping")
            for field in LEGACY_FIELDS:
                data.pop(field, None)
    for node in result.get("nodes", []):
        data = node.get("data", {})
        binding = data.get("control_binding")
        if binding:
            binding.setdefault("compatibility", "standard")
        if version >= 7:
            continue
        old_command = data.pop("trip_polarity", "operate_true" if version <= 4 else "operate_false")
        if data.get("kind") != "TRIP_RELAY":
            continue
        label = data.get("label", node.get("id", "equipment"))
        data["trip_command_polarity"] = old_command if version == 6 else "operate_false"
        data["trip_state_polarity"] = "operated_true"
        if version <= 5 and old_command == "operate_true":
            notices.append(f"{label}: Trip command polarity changed to Operate=0 / Reset=1")
        if version == 6:
            notices.append(f"{label}: Trip state polarity set to Operated=1 / Reset=0")
    return result, notices
