"""Gateway routes: config, points, propose, CSV, commands, logs and the role switch."""
import json


def run(send, projects):
    send("GET", "/tools/gateway/config")
    send("GET", "/tools/gateway")
    for body in ({"protocol": "x"}, {"protocol": "iec104", "listen": "lan"}, {"protocol": "iec104", "settings": {"t1": 5}},
                 {"protocol": "iec104", "settings": {"port": True}}, {"protocol": "dnp3", "settings": {"unsolicited": 1}},
                 {"protocol": "modbus", "settings": {"bogus": 1}}, {"protocol": "iec104", "real_commands": "yes"},
                 {"protocol": "iec104", "settings": {"port": 22404}, "real_commands": False},
                 {"protocol": "modbus", "listen": "network", "settings": {"port": 20502, "unit_id": 7}}, [1], None):
        send("PUT", "/tools/gateway/config", body)
    send("PUT", "/tools/gateway/config", raw="{bad", content_type="application/json")
    send("PUT", "/tools/gateway/config", raw=json.dumps({"protocol": "iec104"}), content_type="text/plain")
    proposals = {}
    for protocol in ("iec104", "modbus", "dnp3", "x", None):
        response = send("POST", "/tools/gateway/propose", {"protocol": protocol})
        if response.status_code == 200:
            proposals[protocol] = response.json["points"]
    for protocol in ("iec104", "modbus", "dnp3"):
        points = [{key: value for key, value in point.items() if key not in ("ied_name", "monitor_only")}
                  for point in proposals[protocol]]
        send("PUT", "/tools/gateway/points", {"protocol": protocol, "points": points[:40]})
        send("GET", f"/tools/gateway/points?protocol={protocol}")
        send("GET", f"/tools/gateway/points.csv?protocol={protocol}")
    bad = [{"kind": "single", "address": True}, {"kind": "float", "address": 1, "scale": 0}, {"kind": "nope"}, 5,
           {"kind": "single", "address": 7, "ied": "missing", "ref": "x"},
           {"kind": "double", "address": 8, "ied": projects.project["ieds"][0]["id"], "ref": "BayACTRL/XCBR1.Pos.stVal"},
           {"kind": "double", "address": 8, "ied": projects.project["ieds"][0]["id"], "ref": "BayACTRL/XSWI1.Pos.stVal"}]
    send("PUT", "/tools/gateway/points", {"protocol": "iec104", "points": bad})
    send("PUT", "/tools/gateway/points", {"protocol": "iec104", "points": "x"})
    send("PUT", "/tools/gateway/points", {"protocol": "iec104", "points": [{"kind": "single"}] * 5001})
    send("GET", "/tools/gateway/points?protocol=bad")
    send("GET", "/tools/gateway/points")
    csv_text = ("﻿type,address,ied,reference,scale,description\r\n"
                "double,20001,BayA,BayACTRL/XCBR1.Pos.stVal,1,Breaker\r\n\r\n"
                "float,30001,BayA,BayACTRL/MMXU1.TotW.mag.f,0.001,\"Power, total\"\n"
                "bogus,1,BayA,x,1,\nsingle,1,Nobody,x,1,\nsingle,x1,BayA,x,1,\nsingle,+5,BayB,BayBCTRL/XCBR1.Loc.stVal,,\n")
    for text in (csv_text, "a,b\n", "", 5, None):
        send("POST", "/tools/gateway/points.csv", {"protocol": "iec104", "text": text})
    send("POST", "/tools/gateway/points.csv", {"protocol": "zz", "text": csv_text})
    command_point = next(point["id"] for point in proposals["iec104"] if point["kind"] == "command_double")
    for body in ({"point_id": command_point, "value": True}, {"point_id": command_point, "value": 1, "confirm": True},
                 {"point_id": 5, "value": True, "confirm": True}, {"point_id": "nope", "value": True, "confirm": True},
                 {"point_id": command_point, "value": False, "confirm": True}):
        send("POST", "/tools/gateway/command", body)
    for query in ("", "?after=0", "?after=x", "?after=%2B1", "?after=1_0", "?after=99999999999999999999999"):
        send("GET", "/tools/gateway/commands" + query)
        send("GET", "/tools/gateway/traffic" + query)
    for body in ({"role": "x"}, {"role": "converter"}, {"role": "master"}, {"role": "gateway"}, {}):
        send("POST", "/tools/gateway/role", body)
    send("GET", "/tools/gateway/config")
