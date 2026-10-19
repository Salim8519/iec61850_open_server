"""Parity harness (Go engine plan, tests/03-parity-harness.md).

STUDIO_ENGINE=python (default) leaves every test exactly as it was. python-http and go spawn
a real backend process per create_app call and translate the in-process seams the tests use
into the seams file and the parity inspect API, so the same test files judge both engines.
Test-only code: the product backend (studio/backend, frozen) gains nothing from this package.
"""
