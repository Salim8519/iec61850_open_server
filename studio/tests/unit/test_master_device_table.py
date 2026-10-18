"""The discovered table belongs to one device, and each protocol names its own read
(change 0089: Modbus rows stayed under IEC 104 and Modbus said "Interrogation failed")."""
from types import SimpleNamespace

from studio_app.domain.master_rules import empty_master
from studio_app.gateway.master_manager import ASK_WORDS, MasterManager


def config(protocol, host="192.168.98.130"):
    return {**empty_master(), "protocol": protocol, "host": host}


def test_points_of_another_device_are_not_shown():
    modbus = config("modbus")
    manager = SimpleNamespace(device=("modbus", "192.168.98.130", modbus["settings"]["modbus"]["port"]),
                              table={"holding_register:0": {}})
    assert MasterManager._table_for(manager, modbus) == manager.table
    assert MasterManager._table_for(manager, config("iec104")) == {}
    assert MasterManager._table_for(manager, config("modbus", "10.0.0.9")) == {}
    assert MasterManager._table_for(SimpleNamespace(device=None, table={}), modbus) == {}


def test_each_protocol_names_its_own_read():
    assert ASK_WORDS == {"iec104": "General interrogation", "dnp3": "Integrity poll", "modbus": "Reading the ranges"}
