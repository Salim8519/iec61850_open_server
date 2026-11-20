"""Median, worst and share of the total for the steps of many rounds, and a Markdown table."""
from statistics import median


def describe(rounds, names):
    """rounds: [{"steps": {name: s}, "facts": {"total": s}}]; one row per step plus the total."""
    totals = [r["facts"]["total"] for r in rounds]
    middle = median(totals) if totals else 0
    rows = []
    for name in names:
        values = [r["steps"].get(name, 0.0) for r in rounds]
        value = median(values) if values else 0
        rows.append({"step": name, "median": round(value, 3), "worst": round(max(values, default=0), 3),
                     "share": round(100 * value / middle, 1) if middle else 0})
    rows.append({"step": "total (click to answer)", "median": round(middle, 3),
                 "worst": round(max(totals, default=0), 3), "share": 100.0})
    return rows


def fact(rounds, key):
    values = [r["facts"][key] for r in rounds if r["facts"].get(key) is not None]
    return {"median": round(median(values), 3), "worst": round(max(values), 3)} if values else None


def markdown(rows):
    lines = ["| Step | Median s | Worst s | Share of median total |", "|---|---|---|---|"]
    for row in rows:
        lines.append(f"| {row['step']} | {row['median']:.3f} | {row['worst']:.3f} | {row['share']:.1f} % |")
    return "\n".join(lines)
