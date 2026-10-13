"""Access application services from request handlers."""
from flask import current_app
from studio_app.contracts.errors import StudioError


def projects():
    return current_app.extensions["projects"]


def runtime():
    return current_app.extensions["runtime"]


def require_stopped():
    if runtime().state != "stopped":
        raise StudioError("Stop the simulation before editing its structure")
