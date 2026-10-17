"""Hold engineering requests until the background startup recovery has finished.

Registered after security.protect_request, so loopback, session and license checks
always answer first. Session, license and preference calls never wait: the window can
sign in and show the license screen while the project is still opening.
"""
from flask import current_app, jsonify, request

WAIT_SECONDS = 300
NO_WAIT = ("/api/v1/session", "/api/v1/license", "/api/v1/preferences")


def wait_for_project():
    recovery = current_app.extensions.get("startup")
    if recovery is None or not request.path.startswith("/api/") or request.path.startswith(NO_WAIT):
        return None
    if not recovery.wait(WAIT_SECONDS):
        return jsonify(error="ScadaBench is still opening your last project. Try again in a moment.",
                       code="project_opening"), 503
    if recovery.error is not None:
        return jsonify(error=f"ScadaBench stopped to protect your last autosave: {recovery.error}. "
                             "Close ScadaBench, inspect the data folder, then start it again.",
                       code="startup_recovery_failed"), 503
    return None
