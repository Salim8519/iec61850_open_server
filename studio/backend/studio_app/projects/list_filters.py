"""Remembered list filters (change 0093): which chips and dropdowns a user last chose per list.

Display-only choices. They never change project data, so damaged or unknown entries are
dropped instead of failing; a new choice for one list replaces only that list's entry.
"""
import re

LIST_ID = re.compile(r"^[a-z][a-z0-9-]{0,39}$")
KEY = re.compile(r"^[a-z][a-zA-Z0-9_]{0,31}$")
MAX_LISTS = 40
MAX_KEYS = 16
MAX_TEXT = 120


def _clean_values(values):
    if not isinstance(values, dict):
        return None
    clean = {}
    for key, value in list(values.items())[:MAX_KEYS]:
        if not isinstance(key, str) or not KEY.match(key):
            continue
        if isinstance(value, bool) or (isinstance(value, str) and len(value) <= MAX_TEXT):
            clean[key] = value
    return clean


def clean_stored(stored):
    """Every valid list entry of a stored value; anything else is ignored."""
    if not isinstance(stored, dict):
        return {}
    result = {}
    for list_id, values in stored.items():
        clean = _clean_values(values) if isinstance(list_id, str) and LIST_ID.match(list_id) else None
        if clean is not None and len(result) < MAX_LISTS:
            result[list_id] = clean
    return result


def merge_update(current, changes):
    """Validate a PUT body part strictly (clear errors for a programming mistake) and merge it."""
    if not isinstance(changes, dict):
        raise ValueError("list_filters must be an object of lists")
    merged = dict(current)
    for list_id, values in changes.items():
        if not isinstance(list_id, str) or not LIST_ID.match(list_id):
            raise ValueError(f"Unknown list name: {list_id!r}")
        clean = _clean_values(values)
        if clean is None or len(clean) != len(values):
            raise ValueError(f"Filters of {list_id} must be short text or true/false values")
        merged[list_id] = clean
    if len(merged) > MAX_LISTS:
        raise ValueError("Too many remembered lists")
    return merged
