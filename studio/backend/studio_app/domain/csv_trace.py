"""Parse bounded, explicit-column measurement traces; never evaluate cell content."""
import csv
import io
import math
from bisect import bisect_right
from .signal import validate_value

MAX_BYTES = 5 * 1024 * 1024
MAX_ROWS = 100000


def read_rows(text):
    if not isinstance(text, str) or len(text.encode("utf-8")) > MAX_BYTES:
        raise ValueError("CSV trace must be UTF-8 text up to 5 MB")
    try:
        reader = csv.reader(io.StringIO(text.lstrip("\ufeff")), strict=True)
        headers = next(reader)
        if not 2 <= len(headers) <= 256 or any(not item.strip() for item in headers):
            raise ValueError("CSV needs 2 to 256 named columns")
        if len(set(headers)) != len(headers):
            raise ValueError("CSV column names must be unique")
        rows = []
        for line, row in enumerate(reader, start=2):
            if not row:
                continue
            if len(row) != len(headers):
                raise ValueError(f"CSV row {line} has the wrong number of columns")
            rows.append(row)
            if len(rows) > MAX_ROWS:
                raise ValueError("CSV exceeds 100,000 data rows")
    except (csv.Error, StopIteration) as error:
        raise ValueError("CSV is empty or malformed") from error
    if not rows:
        raise ValueError("CSV has no data rows")
    return headers, rows


def validate_trace(signal, spec):
    text = spec.get("csv_text")
    headers, rows = read_rows(text)
    time_column, value_column = spec.get("time_column"), spec.get("value_column")
    if time_column not in headers or value_column not in headers or time_column == value_column:
        raise ValueError("Choose different existing time and value columns")
    interpolation = spec.get("interpolation", "hold")
    repeat = spec.get("repeat", False)
    if interpolation not in {"hold", "linear"} or not isinstance(repeat, bool):
        raise ValueError("Choose hold/linear interpolation and a true/false repeat flag")
    if interpolation == "linear" and not signal["kind"].startswith("FLOAT"):
        raise ValueError("Linear playback requires a floating-point measurement")
    time_index, value_index = headers.index(time_column), headers.index(value_column)
    points = []
    for line, row in enumerate(rows, start=2):
        try:
            at, value = float(row[time_index]), float(row[value_index])
            if signal["kind"].startswith("INT") and value.is_integer():
                value = int(value)
            value = validate_value(signal, value)
        except (ValueError, OverflowError) as error:
            raise ValueError(f"Invalid numeric CSV sample at row {line}: {error}") from error
        if not math.isfinite(at) or not 0 <= at <= 86400:
            raise ValueError(f"CSV time at row {line} must be 0 to 86400 seconds")
        if points and at <= points[-1][0]:
            raise ValueError(f"CSV times must strictly increase (row {line})")
        points.append([at, value])
    if points[0][0] != 0:
        raise ValueError("CSV time must start at 0 seconds")
    if repeat and points[-1][0] == 0:
        raise ValueError("A repeating trace needs a positive duration")
    filename = spec.get("filename", "trace.csv")
    if not isinstance(filename, str) or len(filename) > 255:
        raise ValueError("CSV filename must be at most 255 characters")
    return {"mode": "csv", "filename": filename, "csv_text": text,
            "time_column": time_column, "value_column": value_column,
            "interpolation": interpolation, "repeat": repeat, "points": points,
            "row_count": len(points), "duration": points[-1][0], "overridden": False}


def trace_value(spec, elapsed):
    points = spec["points"]
    if spec["repeat"]:
        elapsed %= spec["duration"]
    index = max(0, bisect_right(points, elapsed, key=lambda point: point[0]) - 1)
    at, value = points[index]
    if spec["interpolation"] == "hold" or index == len(points) - 1:
        return value
    end, next_value = points[index + 1]
    return value + (next_value - value) * max(0, (elapsed - at) / (end - at))
