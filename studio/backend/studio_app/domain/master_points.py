"""The point table the SCADA master discovered on the device, in plain words.

A point is keyed by "<group>:<address>" (a 104 IOA, a DNP3 index or a Modbus address),
because one address may carry a status and a measurement in different groups.
Everything discovered is shown and watched; unticked points are only hidden from the
watch list and the CSV (owner rule: discover all by default).
"""
import csv
import io

GROUPS = ("single", "double", "step", "bitstring", "normalized", "scaled", "float", "counter",
          "binary", "double_binary", "analog", "frozen_counter", "binary_output", "analog_output",
          "coil", "discrete_input", "holding_register", "input_register")
GROUP_NAMES = {
    "single": "Single point", "double": "Double point", "step": "Step position", "bitstring": "Bitstring 32",
    "normalized": "Measured (normalised)", "scaled": "Measured (scaled)", "float": "Measured (float)",
    "counter": "Integrated total", "binary": "Binary input", "double_binary": "Double-bit binary input",
    "analog": "Analog input", "binary_output": "Binary output status", "analog_output": "Analog output status",
    "coil": "Coil", "discrete_input": "Discrete input", "holding_register": "Holding register",
    "input_register": "Input register", "frozen_counter": "Frozen counter"}
DNP3_NAMES = {"counter": "Counter"}  # "Integrated total" is the IEC 104 word
# What the device can be commanded with for a point of this group (the user checks the address).
COMMAND_FOR = {"iec104": {"single": "single", "double": "double", "normalized": "setpoint_normalized",
                          "scaled": "setpoint_scaled", "float": "setpoint_float"},
               "dnp3": {"binary_output": "binary_output", "analog_output": "analog_output_float"},
               "modbus": {"coil": "coil", "holding_register": "register"}}
REGISTER_WRITE = {"float32": "registers_float", "int32": "registers_int32", "uint32": "registers_int32"}
IEC104_QUALITY = ((0x80, "invalid"), (0x40, "not topical"), (0x20, "substituted"), (0x10, "blocked"),
                  (0x01, "overflow"))
LINK_LOST = "not current (link lost)"
DOUBLE_TEXT = {0: "intermediate (00)", 1: "open (OFF)", 2: "closed (ON)", 3: "bad state (11)"}


def key(value):
    return f"{value['group']}:{value['address']}"


def merge(table, values):
    """Fold a batch of master_values into the table (dict key -> row); returns new keys."""
    added = []
    for value in values:
        if value.get("group") not in GROUPS or not isinstance(value.get("address"), int):
            continue
        name = key(value)
        row = table.get(name)
        if row is None:
            row = table[name] = {"key": name, "group": value["group"], "address": value["address"], "changes": 0}
            added.append(name)
        elif row["value"] != value.get("value") or row["quality"] != int(value.get("quality", 0)):
            row["changes"] += 1  # a repeated interrogation of the same value is not a change
        row.update(type=value.get("type", ""), value=value.get("value"), quality=int(value.get("quality", 0)),
                   time_ms=value.get("time_ms"), cause=value.get("cause", ""))
    return added


# IEEE 1815 flag bits 5 and 6 mean different things per object; bits 6-7 of binary and
# double-bit inputs and bit 7 of outputs carry the state itself, never a quality (change 0089:
# an open double-bit input was shown as "reference error").
DNP3_BIT5 = {"binary": "chatter filter", "double_binary": "chatter filter", "counter": "rollover",
             "frozen_counter": "rollover", "analog": "over range", "analog_output": "over range"}
DNP3_BIT6 = {"counter": "discontinuity", "frozen_counter": "discontinuity", "analog": "reference error",
             "analog_output": "reference error"}


def quality_words(protocol, quality, group=None):
    """Raw quality bits -> words; an empty list means good."""
    if protocol == "iec104":
        return [word for bit, word in IEC104_QUALITY if quality & bit]
    if protocol == "dnp3":
        words = [] if quality & 0x01 else ["offline"]
        words += [word for bit, word in ((0x02, "restart"), (0x04, "communication lost"),
                                         (0x08, "remote forced"), (0x10, "local forced")) if quality & bit]
        for bit, meaning in ((0x20, DNP3_BIT5), (0x40, DNP3_BIT6)):
            if quality & bit and group in meaning:
                words.append(meaning[group])
        return words
    return ["no answer"] if quality else []


def value_text(row):
    value = row.get("value")
    if row["group"] in ("double", "double_binary"):
        return DOUBLE_TEXT.get(value, str(value))
    if row["group"] in ("single", "binary", "binary_output", "coil", "discrete_input"):
        return "ON" if value else "OFF"
    if isinstance(value, float):
        return f"{value:.6g}"
    return "" if value is None else str(value)


def rows(table, protocol, unticked, labels, link_open):
    """Display rows sorted by group then address; values are 'stale' while the link is down."""
    hidden = set(unticked)
    order = {group: number for number, group in enumerate(GROUPS)}
    result = []
    for row in sorted(table.values(), key=lambda item: (order[item["group"]], item["address"])):
        words = quality_words(protocol, row["quality"], row["group"])
        if not link_open:  # a last-known value is never presented as good
            words = [LINK_LOST, *words]
        result.append({**row, "group_name": group_name(protocol, row["group"]), "label": labels.get(row["key"], ""),
                       "ticked": row["key"] not in hidden, "value_text": value_text(row),
                       "quality_text": ", ".join(words) if words else "good", "good": not words,
                       "stale": not link_open, "command": _command(protocol, row)})
    return result


def group_name(protocol, group):
    return (DNP3_NAMES.get(group) if protocol == "dnp3" else None) or GROUP_NAMES[group]


def _command(protocol, row):
    command = COMMAND_FOR.get(protocol, {}).get(row["group"])
    return REGISTER_WRITE.get(row.get("type"), command) if command == "register" else command


def export_csv(display_rows):
    out = io.StringIO()
    writer = csv.writer(out, lineterminator="\r\n")
    writer.writerow(["group", "address", "label", "value", "quality", "watched"])
    for row in display_rows:
        writer.writerow([row["group"], row["address"], row["label"], row["value_text"], row["quality_text"],
                         "yes" if row["ticked"] else "no"])
    return out.getvalue()


def import_labels(text):
    """CSV with group, address and label columns -> ({key: label}, problems)."""
    labels, problems = {}, []
    reader = csv.DictReader(io.StringIO(text))
    if not reader.fieldnames or not {"group", "address", "label"} <= {name.strip().lower() for name in reader.fieldnames}:
        return {}, ["The CSV needs the columns group, address and label (export one to see the layout)"]
    for number, raw in enumerate(reader, 2):
        row = {name.strip().lower(): (value or "").strip() for name, value in raw.items() if name}
        if row["group"] not in GROUPS or not row["address"].isdigit():
            problems.append(f"Line {number}: unknown group '{row['group']}' or address '{row['address']}'")
        elif row["label"]:
            labels[f"{row['group']}:{int(row['address'])}"] = row["label"][:80]
    return labels, problems
