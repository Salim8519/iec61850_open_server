"""Restrict administration to authenticated, same-origin loopback sessions."""
import hmac
import ipaddress
from urllib.parse import urlsplit
from flask import Blueprint, current_app, jsonify, request, session

blueprint = Blueprint("security", __name__)


@blueprint.post("/session")
def establish_session():
    supplied = (request.get_json(silent=True) or {}).get("token", "")
    if not isinstance(supplied, str) or not hmac.compare_digest(supplied, current_app.config["LAUNCH_TOKEN"]):
        return jsonify(error="Invalid launch session. Reopen the app from its launcher."), 401
    session["authenticated"] = True
    return jsonify(ok=True)


def _loopback_caller(address):
    """Lab mode adds LAN addresses to this PC; the admin API must still refuse them."""
    try:
        return address is None or ipaddress.ip_address(address).is_loopback
    except ValueError:
        return False


def protect_request():
    host = request.host.split(":")[0]
    if host not in {"127.0.0.1", "localhost"} or not _loopback_caller(request.remote_addr):
        return jsonify(error="Administration is available on loopback only"), 403
    origin = request.headers.get("Origin")
    dev_origin = current_app.config.get("ADMIN_ORIGIN")
    frontend = request.headers.get("X-Studio-Frontend") or origin
    wrong_dev_frontend = dev_origin and frontend and frontend.rstrip("/") != dev_origin
    wrong_packaged_origin = not dev_origin and origin and urlsplit(origin).netloc != request.host
    if wrong_dev_frontend or wrong_packaged_origin:
        message = ("This is not the active ScadaBench browser tab. Reopen ScadaBench from start-studio.cmd."
                   if dev_origin else "Cross-origin administration is not permitted")
        return jsonify(error=message, active_frontend=dev_origin), 403
    if not request.path.startswith("/api/") or request.path == "/api/v1/session":
        return None
    token = request.headers.get("Authorization", "").removeprefix("Bearer ")
    if not session.get("authenticated") and not hmac.compare_digest(token, current_app.config["LAUNCH_TOKEN"]):
        launcher = "start-studio.cmd" if dev_origin else "ScadaBench.exe"
        return jsonify(error=f"This browser tab has no ScadaBench session. Open the full address printed by "
                             f"{launcher} (it ends with #session=...)."), 401
    return _require_license()


def _require_license():
    """Packaged builds answer 402 until this computer is licensed; development never checks."""
    service = current_app.extensions.get("license")
    if service is None or not service.enforce or request.path.startswith("/api/v1/license"):
        return None
    if service.is_licensed():
        return None
    return jsonify(error="A license is required. Start the free trial or activate your key.",
                   code="license_required"), 402
