"""Harness errors."""


class InProcessOnly(Exception):
    """A test reached a Python internal the parity inspect API does not mirror.

    Raised loudly (not an AttributeError) so an unported internal access fails the test
    instead of silently passing under an HTTP engine.
    """
