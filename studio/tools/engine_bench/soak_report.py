"""Turn soak sample rows into trends: least-squares slopes per hour and first/last window drift."""
from engine_bench.sse_clients import percentile

SERIES = ("backend_ws_mb", "backend_private_mb", "backend_handles", "backend_threads",
          "ied_ws_mb", "ied_private_mb", "ied_handles", "ied_threads", "other_native_ws_mb",
          "data_mb", "stdout_kb", "free_mb")


def slope_per_hour(points):
    """Least-squares slope of (seconds, value) in value per hour; None for fewer than 3 points."""
    if len(points) < 3:
        return None
    count = len(points)
    mean_t = sum(t for t, _ in points) / count
    mean_v = sum(v for _, v in points) / count
    spread = sum((t - mean_t) ** 2 for t, _ in points)
    if not spread:
        return None
    return round(3600 * sum((t - mean_t) * (v - mean_v) for t, v in points) / spread, 3)


def trends(rows, warmup=600):
    result = {}
    for key in SERIES:
        every = [(row["t"], row[key]) for row in rows if row.get(key) is not None]
        later = [(t, v) for t, v in every if t >= warmup]
        result[key] = {"first": every[0][1] if every else None, "last": every[-1][1] if every else None,
                       "min": min((v for _, v in every), default=None), "max": max((v for _, v in every), default=None),
                       "slope_per_hour_all": slope_per_hour(every), "slope_per_hour_after_warmup": slope_per_hour(later)}
    return result


def window(calls, start, end):
    """calls: (t_since_start, seconds, error). Latency stats inside [start, end)."""
    chosen = [call for call in calls if start <= call[0] < end]
    times = [seconds for _, seconds, error in chosen if not error]
    return {"count": len(chosen), "errors": sum(1 for call in chosen if call[2]),
            "p50_ms": _ms(percentile(times, 0.5)), "p95_ms": _ms(percentile(times, 0.95)),
            "max_ms": _ms(max(times, default=None))}


def drift(calls, duration, span=600):
    return {"first_10_min": window(calls, 0, span), "last_10_min": window(calls, duration - span, duration + 1),
            "whole_run": window(calls, 0, duration + 1)}


def _ms(value):
    return None if value is None else round(value * 1000, 2)
