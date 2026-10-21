"""ToolMemory against random stored files and updates; prints results and the file bytes."""
import json
import os
import sys
import tempfile
from pathlib import Path

sys.path.insert(0, os.environ["STUDIO_ORACLE_LIB"])
from oracle_lib import case, rng, seed  # noqa: E402
from studio_app.tools.tool_memory import ToolMemory  # noqa: E402

random = rng(seed())
SCALARS = [None, True, False, 0, 102, 499, 500, 60000, 60001, 2 ** 70, -5, 1.5, "", "x", "10.0.0.1", "h" * 65,
           "é\U0001F600", [], {}]


def text():
    return random.choice(["", "LD0/XCBR1.Pos.stVal", "ST", "MX", "r" * 201, "fc" * 3, "é", "BOOLEAN" * 6])


def item():
    if random.random() < 0.1:
        return random.choice(SCALARS)
    data = {"ref": random.choice([text(), random.choice(SCALARS)]), "fc": random.choice([text(), "ST", 3])}
    if random.random() < 0.7:
        data["type"] = random.choice([text(), 5, None, 2.5, True])
    return data


def watches():
    return {random.choice(["10.0.0.%d:102" % random.randint(1, 60), "", "h" * 70, "é:1"]):
            [item() for _ in range(random.randint(0, 4))] if random.random() < 0.9 else random.choice(SCALARS)
            for _ in range(random.randint(0, 60))}


def section():
    data = {}
    for key in ("host", "port", "interval_ms", "range", "choice"):
        if random.random() < 0.6:
            data[key] = random.choice(SCALARS + ["192.168.1.0/24"])
    if random.random() < 0.6:
        data["watch_by_host"] = watches()
    return data


def stored():
    roll = random.random()
    if roll < 0.1:
        return random.choice(["", "not json", "﻿{}", "[1,2]", "null", "{\"explorer\": 5}"])
    return json.dumps({"explorer": section(), "find": section(), "other": 1}, indent=random.choice([None, 2]))


for number in range(int(sys.argv[2]) if len(sys.argv) > 2 else 400):
    folder = Path(tempfile.mkdtemp())
    raw = stored()
    if random.random() < 0.9:
        (folder / "tool-settings.json").write_bytes(raw.encode("utf-8"))
    else:
        raw = None
    ops = []
    for _ in range(random.randint(0, 3)):
        tool = random.choice(["explorer", "find"])
        ops.append([tool, {key: value for key, value in section().items()}])

    def run():
        memory = ToolMemory(folder)
        results = [memory.get()]
        for tool, changes in ops:
            results.append(memory.update(tool, **changes))
        path = folder / "tool-settings.json"
        results.append(path.read_bytes().decode("utf-8") if path.exists() else None)
        return results

    case("memory", [raw, ops], run)
