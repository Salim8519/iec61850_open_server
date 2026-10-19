"""Per process type summary of Sampler rows: one entry per image name (python.exe, studio_ied.exe...)."""


def by_type(rows):
    """{name: processes, ws/private mean+peak totals (MB), per-process ws mean (MB), CPU % of one core, handles}."""
    result = {}
    for name in sorted({row["name"].lower() for row in rows}):
        chosen = [row for row in rows if row["name"].lower() == name]
        times = sorted({row["t"] for row in chosen})
        ws = [sum(row["ws"] for row in chosen if row["t"] == t) for t in times]
        private = [sum(row["private"] for row in chosen if row["t"] == t) for t in times]
        firsts, lasts = {}, {}
        for row in chosen:
            if row["pid"] not in firsts or row["t"] < firsts[row["pid"]]["t"]:
                firsts[row["pid"]] = row
            if row["pid"] not in lasts or row["t"] > lasts[row["pid"]]["t"]:
                lasts[row["pid"]] = row
        span = (times[-1] - times[0]) if len(times) > 1 else 0
        used = sum(lasts[pid]["cpu"] - firsts[pid]["cpu"] for pid in firsts)
        processes = len(firsts)
        result[name] = {
            "processes": processes, "samples": len(times),
            "ws_total_mean_mb": _mb(sum(ws) / len(ws)) if ws else 0, "ws_total_peak_mb": _mb(max(ws, default=0)),
            "ws_per_process_mean_mb": _mb(sum(ws) / len(ws) / processes) if ws and processes else 0,
            "private_total_mean_mb": _mb(sum(private) / len(private)) if private else 0,
            "cpu_percent_one_core": round(100 * used / span, 2) if span else None,
            "handles_last": sum(row["handles"] for row in chosen if row["t"] == times[-1]) if times else 0}
    return result


def _mb(value):
    return round(value / 2**20, 1)
