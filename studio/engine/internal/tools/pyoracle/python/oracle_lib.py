"""Shared helpers for the Python oracle scripts of stream E (see oracle.go).

Each script imports this module, calls the frozen studio_app function and prints one JSON
line per case: {"fn", "args", "result"} or {"fn", "args", "error": "<Class>: <text>"}.
"""
import json
import os
import random
import sys

sys.path.insert(0, os.environ["STUDIO_BACKEND"])


def plain(value):
    """Tuples, sets and frozensets become lists (sets sorted) so json.dumps can print them."""
    if isinstance(value, (set, frozenset)):
        return sorted(plain(item) for item in value)
    if isinstance(value, (list, tuple)):
        return [plain(item) for item in value]
    if isinstance(value, dict):
        return {key if isinstance(key, str) else json.dumps(plain(key)): plain(item) for key, item in value.items()}
    return value


def case(fn, args, call):
    """Run call() and print the case line; exceptions are recorded with their class name."""
    line = {"fn": fn, "args": plain(args)}
    try:
        line["result"] = plain(call())
    except Exception as error:  # the class decides the HTTP status, the text is shown to users
        line["error"] = f"{type(error).__name__}: {error}"
    print(json.dumps(line))


def rng(seed):
    return random.Random(seed)


def seed():
    return int(sys.argv[1]) if len(sys.argv) > 1 else 61850
