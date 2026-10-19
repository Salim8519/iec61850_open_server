"""Drive the Python EventBus (studio_app/runtime/event_bus.py) and print what it returns.

Usage: python -I bus_vectors.py --backend <studio/backend> --seed N --count N --capacity N
One JSON line per operation: publish (kind, payload, the json.dumps of the event) or
after (cursor and the json.dumps of every returned event). time.time() is replaced by a
counter (+0.1234567 per call) that the Go test reproduces.
"""
import argparse
import json
import random
import sys
import types


def value(rng, depth=0):
    pick = rng.random()
    if depth < 2 and pick < 0.15:
        return [value(rng, depth + 1) for _ in range(rng.randint(0, 3))]
    if depth < 2 and pick < 0.3:
        return {rng.choice("abcxyz") + str(i): value(rng, depth + 1) for i in range(rng.randint(0, 3))}
    return rng.choice((None, True, False, 0, -7, 2 ** 70, 1.5, 1e-07, 100.0, float("nan"), "",
                       "text", "é中😀", '"\\\n', 12.345678901234567))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--backend", required=True)
    parser.add_argument("--seed", type=int, default=1)
    parser.add_argument("--count", type=int, default=1000)
    parser.add_argument("--capacity", type=int, default=7)
    args = parser.parse_args()
    sys.path.insert(0, args.backend)
    from studio_app.runtime import event_bus
    clock = [1760900000.0]

    def fake_time():
        clock[0] += 0.1234567
        return clock[0]

    event_bus.time = types.SimpleNamespace(time=fake_time)
    bus = event_bus.EventBus(capacity=args.capacity)
    rng = random.Random(args.seed)
    out = sys.stdout
    for _ in range(args.count):
        if rng.random() < 0.55:
            kind = rng.choice(("signal", "runtime", "project_changed", "é-kind"))
            keys = ["ied_id", "ref", "value", "timestamp", "updates"]
            payload = {key: value(rng) for key in rng.sample(keys, rng.randint(0, 4))}
            event = bus.publish(kind, **payload)
            line = {"op": "publish", "kind": kind, "payload": payload, "data": json.dumps(event)}
        else:
            cursor = rng.randint(0, bus.sequence + 2)
            if rng.random() < 0.2:
                cursor = rng.randint(0, 3)
            line = {"op": "after", "cursor": cursor, "result": [json.dumps(e) for e in bus.after(cursor)]}
        out.write(json.dumps(line) + "\n")


if __name__ == "__main__":
    main()
