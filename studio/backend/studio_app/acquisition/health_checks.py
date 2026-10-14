"""Read-only health reads on an existing live session (no extra association, no writes)."""
from datetime import datetime, timezone
from studio_app.domain.health_reads import gocb_items, mode_items, parse_answer


def _now():
    return datetime.now(timezone.utc).isoformat(timespec="seconds")


def _read(process, items):
    if not items:
        return {}, {}
    return parse_answer(process.request("read", timeout=30, refs=items))


def read_mode(process, ied):
    values, errors = _read(process, mode_items(ied))
    return {"values": values, "errors": errors, "checked_at": _now(), "source": "mms_read"}


def read_health(process, ied, blocks, access):
    """Report-block settings, GOOSE-block settings and mode, all read through `read`/`rcb_status`.

    `blocks` are the rcb_status answers the session already read for auto_reports access;
    with polling access Studio never reads report blocks, so they are not checked.
    """
    now = _now()
    gocb_values, gocb_errors = _read(process, gocb_items(ied))
    return {
        "rcbs": None if access != "auto_reports" else {"blocks": blocks, "checked_at": now, "source": "mms_read"},
        "gocbs": {"values": gocb_values, "errors": gocb_errors, "checked_at": now, "source": "mms_read"},
        "mode": read_mode(process, ied),
    }
