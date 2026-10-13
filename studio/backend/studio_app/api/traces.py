"""Read-only CSV column preview; source activation uses the existing source API."""
from flask import Blueprint, jsonify, request
from studio_app.domain.csv_trace import MAX_BYTES, read_rows

blueprint = Blueprint("traces", __name__)


@blueprint.post("/traces/preview")
def preview():
    payload = request.files["file"].read(MAX_BYTES + 1)
    if len(payload) > MAX_BYTES:
        raise ValueError("CSV trace exceeds 5 MB")
    text = payload.decode("utf-8-sig")
    columns, rows = read_rows(text)
    return jsonify(columns=columns, rows=rows[:5], row_count=len(rows))
