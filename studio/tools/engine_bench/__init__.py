"""Engine benchmarks S1-S10 (capacity-audit.md section 6), HTTP only, for either engine.

The CLI is studio/tools/benchmark_engines.py. Every scenario starts the product launcher
(Python entrypoint or the Go server) without the dev watcher, drives only the public HTTP
API, samples RAM/CPU of the backend and every child process once per second, and records
provenance. IEDs live on an isolated loopback range (127.66.x.y, high ports), never on the
127.50.0.x:102 addresses the owner's app uses.
"""
