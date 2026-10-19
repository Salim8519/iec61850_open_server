"""Golden replay (docs/architecture/go-engine/tests/04-golden-replay.md).

Scenarios under tests/parity/golden/scenarios drive an engine over HTTP; the recorder writes a
transcript (every status, header, body and SSE event); the normaliser removes only ids, wall
times, durations, ports and temp paths; the diff compares everything else exactly. Inputs are
restricted to the public manifest: the recorder refuses any other SCL.
"""
