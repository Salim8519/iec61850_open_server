"""Gateway point lists as CSV, so they can match the SCADA engineer's own point list."""
import csv
import io
from .gateway_points import KINDS

COLUMNS = ["type", "address", "ied", "reference", "scale", "description"]
MAX_BYTES = 2 * 1024 * 1024


def export_csv(points, ieds):
    names = {ied["id"]: ied["name"] for ied in ieds}
    stream = io.StringIO()
    writer = csv.writer(stream, lineterminator="\r\n")
    writer.writerow(COLUMNS)
    for point in sorted(points, key=lambda item: (item["kind"], item["address"])):
        writer.writerow([point["kind"], point["address"], names.get(point["ied"], point["ied"]), point["ref"],
                         f"{point.get('scale', 1.0):g}", point.get("description", "")])
    return stream.getvalue()


def import_csv(text, ieds):
    """Return (points, problems). Rows are matched to IEDs by name; nothing is invented."""
    if len(text.encode("utf-8")) > MAX_BYTES:
        raise ValueError("The CSV file is larger than 2 MB")
    rows = list(csv.reader(io.StringIO(text.lstrip("﻿"))))
    if not rows or [cell.strip().lower() for cell in rows[0]][:4] != COLUMNS[:4]:
        raise ValueError("The first row must be: " + ",".join(COLUMNS))
    by_name = {ied["name"]: ied["id"] for ied in ieds}
    points, problems = [], []
    for line, row in enumerate(rows[1:], 2):
        if not any(cell.strip() for cell in row):
            continue
        cells = [cell.strip() for cell in row] + [""] * (len(COLUMNS) - len(row))
        kind, address, ied_name, reference, scale, description = cells[:6]
        if kind not in KINDS:
            problems.append(f"Line {line}: unknown type '{kind}'. Use one of: {', '.join(KINDS)}")
            continue
        if ied_name not in by_name:
            problems.append(f"Line {line}: no IED named '{ied_name}' in this project")
            continue
        try:
            number = int(address)
            factor = float(scale) if scale else 1.0
        except ValueError:
            problems.append(f"Line {line}: address must be a whole number and scale a number")
            continue
        points.append({"kind": kind, "address": number, "ied": by_name[ied_name], "ref": reference,
                       "scale": factor, "description": description})
    return points, problems
