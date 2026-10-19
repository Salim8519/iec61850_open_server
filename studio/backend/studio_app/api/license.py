"""License status and activation endpoints; always reachable so users can activate."""
from flask import Blueprint, current_app, jsonify, request
from studio_app.licensing.certificate import CertificateError
from studio_app.licensing.client import LicenseServerError
from studio_app.licensing.service import LicenseActionError
from studio_app.version import __version__

blueprint = Blueprint("license", __name__)
CONFLICTS = {"bound_elsewhere", "transfer_limit", "trial_used", "email_required", "email_used"}


def _service():
    return current_app.extensions["license"]


def _run(action):
    try:
        return jsonify(action())
    except LicenseServerError as error:
        status = 503 if error.code in {"offline", "no_server", "server_error", "bad_time"} else \
            409 if error.code in CONFLICTS else 400
        return jsonify(error=str(error), code=error.code), status
    except (LicenseActionError, CertificateError) as error:
        return jsonify(error=str(error), code=getattr(error, "code", "bad_certificate")), 400


def _body():
    body = request.get_json(silent=True)
    return body if isinstance(body, dict) else {}


@blueprint.get("/license/status")
def status():
    return jsonify({**_service().status(), "app_version": __version__})


@blueprint.post("/license/trial")
def trial():
    body = _body()
    return _run(lambda: _service().start_trial(body.get("email") or None, body.get("code") or None))


@blueprint.post("/license/trial/email-code")
def trial_email_code():
    return _run(lambda: _service().request_trial_code(_body().get("email", "")))


@blueprint.post("/license/activate")
def activate():
    body = _body()
    return _run(lambda: _service().activate(body.get("license_key", ""), move=bool(body.get("move"))))


@blueprint.post("/license/activate-offline")
def activate_offline():
    return _run(lambda: _service().activate_offline(_body().get("certificate", "")))


@blueprint.post("/license/check-online")
def check_online():
    return _run(lambda: _service().check_online_now())


@blueprint.put("/license/config")
def config():
    return _run(lambda: _service().set_server_url(_body().get("server_url", "")))
