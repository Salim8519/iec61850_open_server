"""Actionable engineering validation errors."""


class StudioError(ValueError):
    def __init__(self, message, *, code="validation", details=None):
        super().__init__(message)
        self.code = code
        self.details = details or []
