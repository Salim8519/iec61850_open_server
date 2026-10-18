// IEC 104 command handling: activation confirmation, negative answers and termination.
#include "iec104_commands.hpp"
#include "iec104_objects.hpp"
#include "traffic.hpp"

namespace {
void reply(IMasterConnection connection, CS101_ASDU asdu, CS101_CauseOfTransmission cause, bool negative) {
    CS101_ASDU_setCOT(asdu, cause);
    CS101_ASDU_setNegative(asdu, negative);
    IMasterConnection_sendASDU(connection, asdu);
}
}

bool Iec104Commands::handles(int type_id) {
    return type_id == C_SC_NA_1 || type_id == C_SC_TA_1 || type_id == C_DC_NA_1 || type_id == C_DC_TA_1;
}

void Iec104Commands::handle(IMasterConnection connection, CS101_ASDU asdu, const std::string& peer) {
    int type_id = CS101_ASDU_getTypeID(asdu);
    std::string name = iec104_type_name(type_id);
    auto object = CS101_ASDU_getElement(asdu, 0);
    if (!object) {
        traffic("rx", peer, name + " without information object: ignored");
        return;
    }
    int ioa = InformationObject_getObjectAddress(object);
    bool single = type_id == C_SC_NA_1 || type_id == C_SC_TA_1;
    bool select = single ? SingleCommand_isSelect((SingleCommand)object) : DoubleCommand_isSelect((DoubleCommand)object);
    int state = single ? (SingleCommand_getState((SingleCommand)object) ? 2 : 1)
                       : DoubleCommand_getState((DoubleCommand)object);
    InformationObject_destroy(object);
    std::string what = name + " IOA " + std::to_string(ioa) + " " + (state == 2 ? "ON" : state == 1 ? "OFF" : "invalid")
                       + (select ? " (select)" : " (execute)");
    auto cause = CS101_ASDU_getCOT(asdu);
    if (cause == CS101_COT_DEACTIVATION) {
        traffic("rx", peer, what + ": cancel confirmed");
        reply(connection, asdu, CS101_COT_DEACTIVATION_CON, false);
        return;
    }
    if (cause != CS101_COT_ACTIVATION) {
        traffic("rx", peer, what + ": wrong cause of transmission, refused (COT 45)");
        reply(connection, asdu, CS101_COT_UNKNOWN_COT, true);
        return;
    }
    auto point = table.at("command", ioa);
    if (!point) {
        traffic("rx", peer, what + ": no command at this IOA, refused (COT 47)");
        reply(connection, asdu, CS101_COT_UNKNOWN_IOA, true);
        return;
    }
    if ((point->kind == "command_single") != single) {
        traffic("rx", peer, what + ": this IOA expects a " + (single ? "double" : "single") + " command (COT 44)");
        reply(connection, asdu, CS101_COT_UNKNOWN_TYPE_ID, true);
        return;
    }
    if (point->monitor_only || state == 0 || state == 3) {
        // The owner's switch decides (change 0089: the old text said real IEDs are "never" commanded).
        traffic("rx", peer, what + (point->monitor_only
            ? ": real IED, and commands to real IEDs are off in the gateway settings (negative)"
            : ": invalid double command state (negative)"));
        reply(connection, asdu, CS101_COT_ACTIVATION_CON, true);
        return;
    }
    if (select) {
        traffic("rx", peer, what + ": select confirmed");
        reply(connection, asdu, CS101_COT_ACTIVATION_CON, false);
        return;
    }
    traffic("rx", peer, what + ": operating the simulated IED");
    auto answer = bridge.ask(point->id, state == 2, "iec104", peer);
    if (!answer.ok) {
        traffic("tx", peer, what + ": negative confirmation, " + answer.reason);
        reply(connection, asdu, CS101_COT_ACTIVATION_CON, true);
        return;
    }
    reply(connection, asdu, CS101_COT_ACTIVATION_CON, false);
    reply(connection, asdu, CS101_COT_ACTIVATION_TERMINATION, false);
    traffic("tx", peer, what + ": confirmed and terminated");
}
