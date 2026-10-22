// IEC 104 command handling: activation confirmation, negative answers and termination.
#include "iec104_commands.hpp"
#include "iec104_command_decode.hpp"
#include "iec104_objects.hpp"
#include "traffic.hpp"

namespace {
void reply(IMasterConnection connection, CS101_ASDU asdu, CS101_CauseOfTransmission cause, bool negative) {
    CS101_ASDU_setCOT(asdu, cause);
    CS101_ASDU_setNegative(asdu, negative);
    IMasterConnection_sendASDU(connection, asdu);
}

std::string expected(const GatewayPoint& point) {
    if (point.kind == "command_step") return "regulating step command (C_RC_NA_1 47 / C_RC_TA_1 60)";
    if (point.kind == "command_setpoint") {
        if (point.format == "normalized") return "normalized set point (C_SE_NA_1 48 / C_SE_TA_1 61)";
        if (point.format == "scaled") return "scaled set point (C_SE_NB_1 49 / C_SE_TB_1 62)";
        return "float set point (C_SE_NC_1 50 / C_SE_TC_1 63)";
    }
    return point.kind == "command_single" ? "single command" : "double command";
}
}

bool Iec104Commands::handles(int type_id, bool typed) { return iec104_command_type(type_id, typed); }

std::optional<GatewayPoint> Iec104Commands::command_at(int ioa) const {
    auto point = table.at("command", ioa);
    return point ? point : table.at("setpoint", ioa);
}

void Iec104Commands::handle(IMasterConnection connection, CS101_ASDU asdu, const std::string& peer) {
    int type_id = CS101_ASDU_getTypeID(asdu);
    auto command = iec104_decode_command(asdu);
    if (!command.valid) {
        traffic("rx", peer, std::string(iec104_type_name(type_id)) + " without information object: ignored");
        return;
    }
    auto cause = CS101_ASDU_getCOT(asdu);
    bool cancel = cause == CS101_COT_DEACTIVATION;
    // The Python-era channel keeps its words; with typed commands a cancel says so (ADR-0107).
    const auto what = cancel && bridge.typed ? command.base + " (cancel)" : command.what;
    if (cancel && !bridge.typed) {  // the Python-era channel: the outstation confirms a cancel itself
        traffic("rx", peer, what + ": cancel confirmed");
        reply(connection, asdu, CS101_COT_DEACTIVATION_CON, false);
        return;
    }
    if (!cancel && cause != CS101_COT_ACTIVATION) {
        traffic("rx", peer, what + ": wrong cause of transmission, refused (COT 45)");
        reply(connection, asdu, CS101_COT_UNKNOWN_COT, true);
        return;
    }
    auto point = command_at(command.ioa);
    if (!point) {
        traffic("rx", peer, what + ": no command at this IOA, refused (COT 47)");
        reply(connection, asdu, CS101_COT_UNKNOWN_IOA, true);
        return;
    }
    bool legacy_pair = is_command(point->kind) && point->kind != "command_step" && point->kind != "command_setpoint"
                       && command.kind != "command_step" && command.kind != "command_setpoint";
    if (point->kind != command.kind || (command.kind == "command_setpoint" && point->format != command.format)) {
        traffic("rx", peer, what + ": this IOA expects a " + (legacy_pair
            ? std::string(command.kind == "command_single" ? "double" : "single") + " command (COT 44)"
            : expected(*point) + ", refused (COT 44)"));
        reply(connection, asdu, CS101_COT_UNKNOWN_TYPE_ID, true);
        return;
    }
    auto confirm = cancel ? CS101_COT_DEACTIVATION_CON : CS101_COT_ACTIVATION_CON;
    bool bad_double = command.kind == "command_double" && (command.state == 0 || command.state == 3);
    if (point->monitor_only || bad_double) {
        // The owner's switch decides (change 0089: the old text said real IEDs are "never" commanded).
        traffic("rx", peer, what + (point->monitor_only
            ? ": real IED, and commands to real IEDs are off in the gateway settings (negative)"
            : ": invalid double command state (negative)"));
        reply(connection, asdu, confirm, true);
        return;
    }
    if (!bridge.typed && command.select) {
        traffic("rx", peer, what + ": select confirmed");
        reply(connection, asdu, CS101_COT_ACTIVATION_CON, false);
        return;
    }
    traffic("rx", peer, what + (cancel ? ": asking ScadaBench" : command.select ? ": select, asking ScadaBench"
                                                                      : ": operating the simulated IED"));
    auto answer = bridge.typed
        ? bridge.ask_step(point->id, command.value, Json{{"type_id", type_id}, {"select", command.select},
                     {"cot", int(cause)}, {"test", CS101_ASDU_isTest(asdu)}}, "iec104", peer)
        : bridge.ask(point->id, command.state == 2, "iec104", peer);
    if (!answer.ok) {
        traffic("tx", peer, what + ": negative confirmation, " + answer.reason);
        reply(connection, asdu, confirm, true);
        return;
    }
    reply(connection, asdu, confirm, false);
    if (cancel || command.select) {
        traffic("tx", peer, what + (cancel ? ": cancel confirmed" : ": select confirmed"));
        return;
    }
    reply(connection, asdu, CS101_COT_ACTIVATION_TERMINATION, false);
    traffic("tx", peer, what + ": confirmed and terminated");
}
