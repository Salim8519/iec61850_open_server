"""SSE event stream with replay cursor and heartbeat."""
import json
from flask import Blueprint, Response, current_app, request

blueprint = Blueprint("events", __name__)


@blueprint.get("/events")
def stream_events():
    bus = current_app.extensions["events"]
    cursor = max(0, int(request.headers.get("Last-Event-ID", request.args.get("after", 0))))

    def stream():
        nonlocal cursor
        yield "retry: 1000\n\n"
        while True:
            events = bus.after(cursor, wait=10)
            if not events:
                yield ": heartbeat\n\n"
            for event in events:
                cursor = event["id"]
                yield f"id: {cursor}\ndata: {json.dumps(event)}\n\n"

    return Response(stream(), mimetype="text/event-stream", headers={"Cache-Control": "no-cache"})
