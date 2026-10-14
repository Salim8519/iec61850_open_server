"""Export a station health result as CSV rows or a self-contained HTML snapshot."""
import csv
import html
import io

COLUMNS = ("ied", "role", "area", "item", "result", "message", "observed_at", "source", "stale")
WORDS = {"ok": "OK", "warning": "Check", "problem": "Problem", "not_checked": "Not checked"}


def rows(health):
    for entry in health["ieds"]:
        for check in entry["checks"]:
            yield {"ied": entry["name"], "role": entry["role"], **{key: check.get(key) for key in COLUMNS[2:]}}


def csv_text(health):
    buffer = io.StringIO()
    writer = csv.DictWriter(buffer, fieldnames=COLUMNS, lineterminator="\n")
    writer.writeheader()
    for row in rows(health):
        writer.writerow({key: "" if row[key] is None else row[key] for key in COLUMNS})
    return buffer.getvalue()


def html_report(health, project_name, run_id):
    """A printable snapshot; values are escaped and nothing is loaded from the network."""
    def cell(value):
        return f"<td>{html.escape('' if value is None else str(value))}</td>"
    body = "".join("<tr>" + "".join(cell(WORDS.get(row[key], row[key]) if key == "result" else row[key])
                                    for key in COLUMNS) + "</tr>" for row in rows(health))
    head = "".join(f"<th>{name.replace('_', ' ').capitalize()}</th>" for name in COLUMNS)
    return ("<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\">"
            f"<title>Station health - {html.escape(project_name)}</title><style>"
            "body{font:14px system-ui,sans-serif;margin:24px;color:#0f172a}table{border-collapse:collapse;width:100%}"
            "th,td{border:1px solid #a9b1bd;padding:6px 8px;text-align:left;vertical-align:top}"
            "th{background:#eef1f5}</style></head><body>"
            f"<h1>Station health: {html.escape(project_name)}</h1>"
            f"<p>Overall: <strong>{WORDS[health['verdict']]}</strong> · checked {html.escape(health['checked_at'])}"
            f" · run {html.escape(run_id)}</p>"
            f"<table><thead><tr>{head}</tr></thead><tbody>{body}</tbody></table></body></html>")
