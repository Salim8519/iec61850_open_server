"""Find IEDs preview, tool memory, explorer refusals, activity, test panel and trace preview."""


def run(send, projects):
    for body in ({"range": "10.0.0.0/24"}, {"range": "10.0.0.0/8"}, {"range": "192.168.1.20-40", "port": True},
                 {"range": "x"}, {"range": "1.2.3.4", "port": 0}, {}, [1]):
        send("POST", "/tools/scan/preview", body)
    send("POST", "/tools/scan/start", {"range": "10.0.0.0/8"})
    send("GET", "/tools/memory")
    for body in ({"choice": "Lab adapter"}, {"choice": 5}, {"choice": "x" * 201}):
        send("PUT", "/tools/memory/find", body)
    send("GET", "/tools/memory")
    send("GET", "/tools/scan")
    send("POST", "/tools/scan/stop")
    for body in ({"host": "x"}, {"host": "10.0.0.1", "port": 0}, {"host": "10.0.0.1", "port": "102"}, None):
        send("POST", "/tools/explorer/connect", body)
    send("GET", "/tools/explorer/devices")
    send("GET", "/tools/explorer/nodes?device=LD0")
    send("GET", "/tools/explorer/node?ref=LD0")
    send("GET", "/tools/explorer/node?ref=LD0/XCBR1")
    for body in ({"items": "x"}, {"items": [{"ref": "a", "fc": "ST"}]}, {"items": []}, {"items": [{"ref": 1, "fc": "ST"}]}):
        send("POST", "/tools/explorer/read", body)
    for body in ({"items": [{"ref": "a", "fc": "ST"}, {"ref": "a", "fc": "ST", "type": 5}]},
                 {"items": [{"ref": "a", "fc": "ST"}], "interval_ms": 100}, {"items": [{"ref": "r%d" % n, "fc": "ST"}
                                                                                     for n in range(201)]}):
        send("PUT", "/tools/explorer/watch", body)
    for body in ({"on": True}, {"on": False}):
        send("POST", "/tools/explorer/polling", body)
    send("POST", "/tools/explorer/disconnect")
    send("GET", "/tools/explorer/watch.csv")
    for body in ({"ref": 5}, {"ref": "LD0/XCBR1.Pos", "value": True}, {"ref": "LD0/XCBR1.Pos", "value": True, "confirm": True}):
        send("POST", "/tools/explorer/operate", body)
    send("GET", "/activity")
    ied = projects.project["ieds"][0]
    writable = next(signal for signal in ied["signals"] if signal["writable"] and signal["fc"] != "CF")
    entry = {"id": "c1", "label": " Trip ", "ied_id": ied["id"], "ref": writable["ref"], "duration_ms": 500}
    for body in ([entry], [dict(entry, id="bad id")], {"a": 1}, [entry, dict(entry, label="")]):
        send("PUT", "/project/test-panel", body)
    send("PUT", "/project/test-panel", raw="[]", content_type="text/plain")
    send("PUT", "/project/test-panel", raw="[", content_type="application/json")
    send("GET", "/project/test-panel")
    for text in ("t,v\r\n0,1\r\n1,2\r\n2,3\r\n3,4\r\n4,5\r\n5,6\r\n", "\xef\xbb\xbft,v\n0,1\n", "t,v\n", "a\xff,b\n1,2\n",
                 "t\n1\n", "t,v\n0,1,2\n"):
        send("POST", "/traces/preview", file=text)
    send("POST", "/traces/preview", {"x": 1})
