"""Run one vector against the frozen Python backend; the Go test has the same table."""
import json
import tempfile
from pathlib import Path
from unittest import mock

from studio_app.projects import bulk_removal, list_filters, migration_backup, service as service_module
from studio_app.projects.document import compact, expand
from studio_app.projects.validation import validate_diagram, validate_sources
from studio_app.projects.test_panel import validate_panel
from studio_app.projects.mapping_migration import migrate_diagram
from studio_app.projects.removal import impacts, without_ieds
from studio_app.projects.ied_settings import endpoint_conflicts, update_ied, update_network
from studio_app.projects.goose_reach import save_reach
from studio_app.projects.preferences import PreferenceStore
from studio_app.projects.service import ProjectService
from studio_app.domain.scenario import validate_scenarios
from studio_app.domain.source import source_summary
from studio_app.domain.network_settings import validate_ied_settings, validate_live, validate_network
from studio_app.domain.test_mode import validate_test_mode
from studio_app.domain.gateway_rules import empty_gateway, normalize_gateway
from studio_app.domain.master_rules import normalize_master
from studio_app.networking.loopback import assign_endpoints
from vectors_common import Stub, fixed_uuid, reduce_project


def _with_service(call):
    def run(a, ieds):
        stub = Stub(a["project"])
        result = call(stub, a)
        return {"result": result, "project": stub.project}
    return run


def _bulk_remove(a, ieds):
    with tempfile.TemporaryDirectory() as folder:
        stub = Stub(a["project"], Path(folder))
        with mock.patch.object(bulk_removal, "uuid4", fixed_uuid("f" * 32)):
            result = bulk_removal.remove(stub, a["keys"], a["fingerprint"])
        result.pop("recovery_file")
        return {"result": result, "project": stub.project}


def _store(a, folder):
    store = PreferenceStore(folder)
    if a.get("stored_text") is not None:
        store.path.write_text(a["stored_text"], encoding="utf-8")
    return store


def _preferences_get(a, ieds):
    with tempfile.TemporaryDirectory() as folder:
        return _store(a, folder).get()


def _preferences_update(a, ieds):
    with tempfile.TemporaryDirectory() as folder:
        store = _store(a, folder)
        result = store.update(a["changes"])
        return {"result": result, "file": store.path.read_text(encoding="utf-8")}


def _assign(a, ieds):
    assign_endpoints(a["ieds"])
    return a["ieds"]


def _snapshot(a, ieds):
    return Stub(a["project"], notice=a["notice"], session_id=a["session_id"], revision=a["revision"]).snapshot()


def _new_project(a, ieds):
    with tempfile.TemporaryDirectory() as folder:
        with mock.patch.object(service_module, "uuid4", fixed_uuid(a["id"])):
            return ProjectService(Path(folder)).project


def payload_of(document):
    import io
    import zipfile
    stream = io.BytesIO()
    with zipfile.ZipFile(stream, "w", zipfile.ZIP_DEFLATED) as archive:
        archive.writestr("project.json", json.dumps(document).encode("utf-8"))
    return stream.getvalue()


def _restore(a, ieds):
    with tempfile.TemporaryDirectory() as folder:
        projects = ProjectService(Path(folder))
        with mock.patch.object(service_module, "uuid4", fixed_uuid(a["new_id"])), \
                mock.patch.object(migration_backup, "uuid4", fixed_uuid("e" * 32)):
            projects.restore(payload_of(a["document"]), persist=False)
        notice = projects.migration_notice
        if notice and notice.get("backup"):
            notice = {**notice, "backup": "@backup"}
        return {"project": reduce_project(projects.project), "notice": notice}


RUNNERS = {
    "compact": lambda a, ieds: compact(a["project"]),
    "expand": lambda a, ieds: expand(a["document"]),
    "validate_diagram": lambda a, ieds: validate_diagram(a["diagram"], ieds),
    "validate_sources": lambda a, ieds: validate_sources(a["sources"], ieds),
    "validate_scenarios": lambda a, ieds: validate_scenarios(a["scenarios"], ieds),
    "validate_panel": lambda a, ieds: validate_panel(a["entries"], ieds),
    "migrate_diagram": lambda a, ieds: list(migrate_diagram(a["diagram"], ieds, a["version"])),
    "impacts": lambda a, ieds: impacts(a["project"], set(a["ids"])),
    "without_ieds": lambda a, ieds: without_ieds(a["project"], set(a["ids"])),
    "removal_preview": lambda a, ieds: Stub(a["project"]).removal_preview(a["key"]),
    "bulk_preview": lambda a, ieds: bulk_removal.preview(Stub(a["project"]), a["keys"]),
    "bulk_remove": _bulk_remove,
    "validate_network": lambda a, ieds: validate_network(a["value"]),
    "validate_ied_settings": lambda a, ieds: validate_ied_settings(a["value"]),
    "validate_live": lambda a, ieds: validate_live(a["value"]),
    "validate_test_mode": lambda a, ieds: validate_test_mode(a["value"]),
    "update_network": _with_service(lambda s, a: update_network(s, a["value"])),
    "update_ied": _with_service(lambda s, a: update_ied(s, a["key"], a["changes"])),
    "save_reach": _with_service(lambda s, a: save_reach(s, a["reach"], a["uplink"], a["share"])),
    "endpoint_conflicts": lambda a, ieds: endpoint_conflicts(a["ieds"], a["network"]),
    "normalize_gateway": lambda a, ieds: normalize_gateway(a["raw"]),
    "normalize_master": lambda a, ieds: normalize_master(a["raw"]),
    "empty_gateway": lambda a, ieds: empty_gateway(),
    "preferences_get": _preferences_get,
    "preferences_update": _preferences_update,
    "list_filters_clean": lambda a, ieds: list_filters.clean_stored(a["stored"]),
    "list_filters_merge": lambda a, ieds: list_filters.merge_update(a["current"], a["changes"]),
    "assign_endpoints": _assign,
    "snapshot": _snapshot,
    "new_project": _new_project,
    "source_summary": lambda a, ieds: source_summary(a["spec"]),
    "restore": _restore,
}
