"""What is running right now outside the IED Run: tools and sequences, for the running dots.

Only plain facts; the page words them. Reading never starts or warms anything.
"""


def activity(runtime, gateway, explorer, scanner):
    status = gateway.status()
    opened = status.get("opened") or {}
    scan = scanner.state
    return {
        "gateway": {"state": status["state"], "protocol": status["protocol"], "port": opened.get("port"),
                    "clients": len(status["clients"])},
        "explorer": {"state": explorer.state, "host": explorer.endpoint[0] if explorer.endpoint else "",
                     "polling": explorer.polling, "watched": len(explorer.watch)},
        "scan": {"state": scan["state"], "probed": scan.get("probed", 0), "total": scan.get("total", 0)},
        "sequences": sum(1 for run in runtime.scenario_runner.snapshot() if run["state"] == "running"),
    }
