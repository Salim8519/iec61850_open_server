"""SCADA master routes without a device: config, ranges, routes, labels, selection, refusals."""


def run(send, projects):
    send("POST", "/tools/gateway/role", {"role": "master"})
    send("GET", "/tools/master/config")
    send("GET", "/tools/master")
    for body in ({"protocol": "x"}, {"host": "bad host!"}, {"protocol": "dnp3", "host": " rtu1.station.local "},
                 {"protocol": "iec104", "settings": {"t0": 300}}, {"protocol": "iec104", "settings": {"w": 20}},
                 {"protocol": "modbus", "settings": {"poll_ms": 100}}, {"protocol": "modbus", "modbus_ranges": "x"},
                 {"protocol": "modbus", "host": "10.0.0.9", "settings": {"port": 20502, "poll_ms": 500},
                  "modbus_ranges": [{"table": "holding_registers", "start": 0, "count": 10, "data_type": "float32"}]},
                 {"protocol": "iec104", "host": "127.0.0.1", "settings": {"port": 22404, "gi_minutes": 0}}):
        send("PUT", "/tools/master/config", body)
    ranges = [{"table": "coils", "start": 0, "count": 2000}, {"table": "input_registers", "start": 65530, "count": 4},
              {"table": "holding_registers", "start": 5, "count": 3, "data_type": "uint32", "word_order": "little"}]
    for value in (ranges, ranges[:1], [{"table": "x"}], [{"table": "coils", "start": True, "count": 1}], [{}] * 21, None):
        send("PUT", "/tools/master/ranges", {"modbus_ranges": value})
    routes = [{"upstream": 50001, "upstream_kind": "command_double", "kind": "double", "address": 7},
              {"upstream": 50002, "upstream_kind": "command_single", "kind": "coil", "address": 3, "select": False}]
    for value in (routes, [{"kind": "x"}], [{"kind": "coil", "upstream_kind": "command_single", "upstream": -1}], "x"):
        send("PUT", "/tools/master/routes", {"routes": value})
    for body in ({"key": "double:2001", "label": "  Feeder 1 breaker  "}, {"key": "single:1", "label": "x" * 90},
                 {"key": "double:2001", "label": " "}, {"key": "k" * 41, "label": "x"}, {"key": 5}):
        send("PUT", "/tools/master/label", body)
    for body in ({"unticked": ["single:1001", "double:2001"]}, {"unticked": "x"}, {"unticked": [5]}):
        send("PUT", "/tools/master/selection", body)
    labels = "Group,Address,Label\nsingle,1001,Trip\ndouble,x,Bad\nnope,5,No\nsingle,1002,\n\nfloat,3001,Power\n"
    for text in (labels, "a,b\n1,2\n", "", None):
        send("POST", "/tools/master/labels.csv", {"text": text})
    send("GET", "/tools/master/points")
    send("GET", "/tools/master/points.csv")
    for body in ({"kind": "double", "address": 1, "value": 2}, {"kind": "nope", "confirm": True},
                 {"kind": "double", "address": True, "value": 2, "confirm": True},
                 {"kind": "double", "address": 1, "value": "2", "confirm": True},
                 {"kind": "single", "address": 1, "value": True, "confirm": True},
                 {"kind": "clock", "confirm": True}):
        send("POST", "/tools/master/command", body)
    for body in ({"table": "coils", "start": 0, "end": 10}, {"table": "x", "start": 0, "end": 1},
                 {"table": "coils", "start": 10, "end": 1}, {"table": "coils", "start": 0, "end": 10000}):
        send("POST", "/tools/master/probe", body)
    for body in ({"data_type": "bool"}, {}, {"data_type": "float32"}):
        send("POST", "/tools/master/probe/ranges", body)
    send("POST", "/tools/master/interrogate")
    send("POST", "/tools/master/disconnect")
    for query in ("", "?after=-1", "?after= 3 "):
        send("GET", "/tools/master/traffic" + query)
        send("GET", "/tools/master/commands" + query)
    send("POST", "/tools/master/connect")
    send("GET", "/tools/master/config")
    send("POST", "/tools/gateway/role", {"role": "gateway"})
