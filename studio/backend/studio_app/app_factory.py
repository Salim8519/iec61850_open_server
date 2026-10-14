"""Compose Flask routes and long-lived application services."""
import secrets
from flask import Flask, jsonify, send_from_directory
from pydantic import ValidationError
from werkzeug.exceptions import HTTPException
from studio_app.settings import Settings
from studio_app.contracts.errors import StudioError
from studio_app.projects.service import ProjectService
from studio_app.projects.startup_recovery import recover_startup
from studio_app.runtime.event_bus import EventBus
from studio_app.simulation.coordinator import Coordinator
from studio_app.api import projects, scl, signals, runtime, events, security, scenarios, ieds, traces, equipment
from studio_app.api import test_panel, network, live, sld, signal_flow, health, test_mode
from studio_app.networking.service import NetworkService


def create_app(settings=None, token=None, secret_key=None, admin_origin=None):
    settings = settings or Settings.defaults()
    app = Flask(__name__, static_folder=None)
    app.config.update(SECRET_KEY=secret_key or secrets.token_hex(32), LAUNCH_TOKEN=token or secrets.token_urlsafe(32),
                      MAX_CONTENT_LENGTH=100 * 1024 * 1024, SESSION_COOKIE_HTTPONLY=True,
                      SESSION_COOKIE_SAMESITE="Strict", SESSION_COOKIE_NAME="studio_session",
                      ADMIN_ORIGIN=admin_origin.rstrip("/") if admin_origin else None)
    bus = EventBus()
    project_service = ProjectService(settings.directory, lambda revision, source:
                                     bus.publish("project_changed", revision=revision, source=source))
    recovery = recover_startup(project_service)
    coordinator = Coordinator(project_service, bus, settings.native_executable, settings.directory)
    network_service = NetworkService(project_service, settings.directory, settings.native_executable)
    app.extensions.update(projects=project_service, events=bus, runtime=coordinator, network=network_service,
                          studio_settings=settings, startup_recovery=recovery)
    app.before_request(security.protect_request)
    for module in (projects, scl, signals, runtime, events, security, scenarios, ieds, traces,
                   test_panel, equipment, network, live, sld, signal_flow, health, test_mode):
        app.register_blueprint(module.blueprint, url_prefix="/api/v1")

    @app.errorhandler(StudioError)
    def engineering_error(error):
        status = 409 if error.code == "conflict" else 400
        return jsonify(error=str(error), code=error.code, details=error.details), status

    @app.errorhandler(ValidationError)
    def contract_error(error):
        return jsonify(error="Invalid request", details=error.errors(include_context=False, include_input=False)), 400

    @app.errorhandler(ValueError)
    @app.errorhandler(KeyError)
    def input_error(error):
        return jsonify(error=str(error)), 400

    @app.errorhandler(RuntimeError)
    @app.errorhandler(TimeoutError)
    def runtime_error(error):
        return jsonify(error=str(error)), 503

    @app.errorhandler(HTTPException)
    def http_error(error):
        return jsonify(error=error.description), error.code

    @app.get("/")
    def index():
        if not (settings.frontend / "index.html").exists():
            return "Substation Studio frontend build is not available yet.", 503
        return send_from_directory(settings.frontend, "index.html")

    @app.get("/assets/<path:filename>")
    def asset(filename):
        return send_from_directory(settings.frontend / "assets", filename)

    return app
