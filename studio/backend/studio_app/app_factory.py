"""Compose Flask routes and long-lived application services."""
import secrets
import sys
import threading
from flask import Flask, jsonify
from pydantic import ValidationError
from werkzeug.exceptions import HTTPException
from studio_app.settings import Settings
from studio_app.contracts.errors import StudioError
from studio_app.projects.service import ProjectService
from studio_app.projects.preferences import PreferenceStore
from studio_app.projects.background_recovery import BackgroundRecovery
from studio_app.api import startup_gate
from studio_app.runtime.event_bus import EventBus
from studio_app.simulation.coordinator import Coordinator
from studio_app.api import projects, scl, signals, runtime, events, security, scenarios, ieds, traces, equipment
from studio_app.api import test_panel, network, live, sld, signal_flow, health, test_mode, license, preferences, tools, updates
from studio_app.api import gateway, gateway_master, goose_share
from studio_app.gateway.converter import ConverterSource
from studio_app.gateway.manager import GatewayManager
from studio_app.gateway.master_manager import MasterManager
from studio_app.licensing.service import LicenseService, default_home
from studio_app.networking.gateway_firewall import GatewayFirewall
from studio_app.networking.service import NetworkService
from studio_app.launcher.frontend_assets import FrontendAssets
from studio_app.tools.explorer import ExplorerService
from studio_app.tools.ied_scan import ScanService
from studio_app.tools.control_client import ControlClient
from studio_app.tools.tool_memory import ToolMemory
from studio_app.updates.service import UpdateService
from studio_app.licensing import release_key


def create_app(settings=None, token=None, secret_key=None, admin_origin=None, license_service=None, update_service=None):
    settings = settings or Settings.defaults()
    license_service = license_service or LicenseService(
        default_home(), settings.directory, enforce=bool(getattr(sys, "frozen", False)),
        native_folder=settings.native_executable.parent)
    update_service = update_service or UpdateService(settings.directory, lambda: getattr(
        license_service, "server_url", lambda: release_key.DEFAULT_SERVER_URL)())
    app = Flask(__name__, static_folder=None)
    app.config.update(SECRET_KEY=secret_key or secrets.token_hex(32), LAUNCH_TOKEN=token or secrets.token_urlsafe(32),
                      MAX_CONTENT_LENGTH=100 * 1024 * 1024, SESSION_COOKIE_HTTPONLY=True,
                      SESSION_COOKIE_SAMESITE="Strict", SESSION_COOKIE_NAME="studio_session",
                      ADMIN_ORIGIN=admin_origin.rstrip("/") if admin_origin else None)
    bus = EventBus()
    project_service = ProjectService(settings.directory, lambda revision, source:
                                     bus.publish("project_changed", revision=revision, source=source))
    recovery = BackgroundRecovery(project_service)
    coordinator = Coordinator(project_service, bus, settings.native_executable, settings.directory)
    network_service = NetworkService(project_service, settings.directory, settings.native_executable)
    control = ControlClient(settings.native_executable.with_name("studio_control.exe"))
    memory = ToolMemory(settings.directory)
    gateway_manager = GatewayManager(coordinator, project_service, settings.native_executable, control)
    master_manager = MasterManager(project_service, settings.native_executable, coordinator.emit)
    gateway_manager.converter = ConverterSource(master_manager)
    gateway_firewall = GatewayFirewall(network_service, gateway_manager.executable)
    app.extensions.update(projects=project_service, events=bus, runtime=coordinator, network=network_service,
                          studio_settings=settings, startup=recovery,
                          license=license_service, preferences=PreferenceStore(settings.directory),
                          explorer=ExplorerService(settings.live_executable, bus.publish, memory, control),
                          scanner=ScanService(settings.live_executable, memory=memory), tool_memory=memory,
                          gateway=gateway_manager, master=master_manager,
                          gateway_firewall=gateway_firewall, updates=update_service)
    app.before_request(security.protect_request)
    app.before_request(startup_gate.wait_for_project)  # After the security and license checks.
    for module in (projects, scl, signals, runtime, events, security, scenarios, ieds, traces,
                   test_panel, equipment, network, live, sld, signal_flow, health, test_mode, license,
                   preferences, tools, gateway, gateway_master, updates, goose_share):
        app.register_blueprint(module.blueprint, url_prefix="/api/v1")
    license_service.start_background_verify()
    update_service.start_background()
    recovery.start()
    threading.Thread(target=_resume_gateway, args=(recovery, gateway_manager, master_manager), name="gateway-resume", daemon=True).start()

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

    frontend = FrontendAssets(settings.frontend)

    @app.get("/")
    def index():
        if not frontend.available():
            return "ScadaBench frontend build is not available yet.", 503
        return frontend.index()

    @app.get("/assets/<path:filename>")
    def asset(filename):
        return frontend.asset(filename)

    return app


def _resume_gateway(recovery, gateway_manager, master_manager):
    """Owner rule: a gateway or master that was running when the app closed starts again by itself."""
    recovery.wait()
    if recovery.error is None:
        master_manager.autoconnect()  # first: a protocol converter serves what the master reads
        gateway_manager.autostart()
