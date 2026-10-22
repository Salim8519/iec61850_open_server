// ScadaBench as IEC 60870-5-104 controlling station (master) toward a controlled station
// (the device): connect, general interrogation, commands with select/execute and answers.
#include "client.hpp"
#include "iec104_command_objects.hpp"
#include "iec104_decode.hpp"
#include "traffic.hpp"
#include "value_events.hpp"
#include <atomic>
#include <cstdio>
#include <chrono>
#include <condition_variable>
#include <mutex>
#include <optional>
#include <vector>
extern "C" {
#include "cs104_connection.h"
#include "hal_time.h"
}

namespace {
std::string number(double value) {
    char text[32];
    std::snprintf(text, sizeof text, "%.7g", value);
    return text;
}

struct Answer { int type; int ioa; int cot; bool negative; long serial; };

class Client104 : public ProtocolClient {
public:
    ~Client104() override { disconnect(); }

    Json connect(const Json& settings) override {
        host = settings.at("host").get<std::string>();
        int port = settings.value("port", 2404);
        ca = settings.value("common_address", 1);
        time_tagged = settings.value("time_tagged_commands", false);
        peer = peer_name(host, port);
        connection = CS104_Connection_create(host.c_str(), port);
        struct sCS104_APCIParameters apci = *CS104_Connection_getAPCIParameters(connection);
        apci.k = settings.value("k", 12); apci.w = settings.value("w", 8);
        apci.t0 = settings.value("t0", 10); apci.t1 = settings.value("t1", 15);
        apci.t2 = settings.value("t2", 10); apci.t3 = settings.value("t3", 20);
        CS104_Connection_setAPCIParameters(connection, &apci);
        CS104_Connection_getAppLayerParameters(connection)->originatorAddress = settings.value("originator_address", 0);
        t1_ms = apci.t1 * 1000;
        CS104_Connection_setConnectTimeout(connection, apci.t0 * 1000);
        CS104_Connection_setASDUReceivedHandler(connection, &Client104::on_asdu, this);
        CS104_Connection_setConnectionHandler(connection, &Client104::on_link, this);
        emit_link("connecting", "Connecting to " + peer);
        if (!CS104_Connection_connect(connection)) {
            CS104_Connection_destroy(connection);
            connection = nullptr;
            throw std::runtime_error("No answer from " + peer + " within " + std::to_string(apci.t0) +
                                     " s. Check the address and port, and that the device accepts this PC as its master.");
        }
        open = true;
        CS104_Connection_sendStartDT(connection);
        traffic("tx", peer, "STARTDT act (start data transfer)");
        return {{"host", host}, {"port", port}, {"common_address", ca}};
    }

    Json interrogate() override {
        require_open();
        auto before = mark();
        CS104_Connection_sendInterrogationCommand(connection, CS101_COT_ACTIVATION, ca, IEC60870_QOI_STATION);
        traffic("tx", peer, "C_IC_NA_1 station interrogation, common address " + std::to_string(ca));
        auto confirmed = wait(C_IC_NA_1, 0, before, CS101_COT_ACTIVATION_CON, t1_ms);
        bool ok = confirmed && !confirmed->negative;
        // The values follow as "interrogated"; ACT_TERM arrives later as a master_interrogation event.
        // Integrated totals (M_IT) never answer a general interrogation: ask for them separately.
        std::string counters = ok ? counter_interrogation(before) : "not asked";
        return {{"confirmed", ok}, {"counters", counters},
                {"text", ok ? "confirmed; counters " + counters
                            : confirmed ? "refused by the device" : "no confirmation within t1"}};
    }

    Json command(const Json& request) override {
        require_open();
        auto kind = request.at("kind").get<std::string>();
        if (kind == "clock") return clock_sync();
        int ioa = request.at("address").get<int>();
        double value = request.at("value").get<double>();
        int qualifier = request.value("qualifier", 0);
        if (qualifier < 0 || qualifier > 31) throw std::runtime_error("The command qualifier (QU) is a whole number from 0 to 31");
        // "deactivate" sends one object with cause 8; the device answers deactivation confirmed,
        // never a termination. Otherwise it is an activation (cause 6) with the usual select/execute.
        bool deactivate = request.value("deactivate", false);
        int cot = deactivate ? CS101_COT_DEACTIVATION : CS101_COT_ACTIVATION;
        Json steps = Json::array();
        if (!deactivate && request.value("select", false) && !step(kind, ioa, value, true, qualifier, cot, steps))
            return {{"ok", false}, {"steps", steps}};
        long executed = 0;
        int type = 0;
        if (!step(kind, ioa, value, false, qualifier, cot, steps, &executed, &type)) return {{"ok", false}, {"steps", steps}};
        if (deactivate) return {{"ok", true}, {"steps", steps}, {"terminated", false}};
        bool terminated = wait(type, ioa, executed, CS101_COT_ACTIVATION_TERMINATION, 5000).has_value();
        steps.push_back({{"step", "termination"}, {"ok", terminated},
                         {"text", terminated ? "activation terminated" : "no termination within 5 s"}});
        return {{"ok", true}, {"steps", steps}, {"terminated", terminated}};
    }

    void disconnect() override {
        if (!connection) return;
        CS104_Connection_destroy(connection);
        connection = nullptr;
        open = false;
        emit_link("closed", "Disconnected by you");
    }

private:
    CS104_Connection connection = nullptr;
    std::string host, peer;
    int ca = 1, t1_ms = 15000;
    bool time_tagged = false;
    std::atomic<bool> open{false};
    long serial = 0;
    std::vector<Answer> answers;
    std::mutex lock;
    std::condition_variable arrived;

    void require_open() {
        if (!connection || !open) throw std::runtime_error("Not connected to the device; press Connect first");
    }

    long mark() { std::lock_guard<std::mutex> guard(lock); return serial; }

    // The device's first answer to (type, IOA) after `after`: the wanted cause, or any refusal.
    std::optional<Answer> wait(int type, int ioa, long after, int cot, int timeout_ms) {
        std::unique_lock<std::mutex> guard(lock);
        std::optional<Answer> found;
        arrived.wait_for(guard, std::chrono::milliseconds(timeout_ms), [&] {
            for (const auto& answer : answers) {
                bool refusal = answer.negative || answer.cot >= CS101_COT_UNKNOWN_TYPE_ID;
                if (answer.serial > after && answer.type == type && answer.ioa == ioa && (answer.cot == cot || refusal)) {
                    found = answer;
                    return true;
                }
            }
            return false;
        });
        return found;
    }

    bool step(const std::string& kind, int ioa, double value, bool select, int qualifier, int cot, Json& steps,
              long* sent = nullptr, int* sent_type = nullptr) {
        auto built = iec104_command_object(kind, ioa, value, select, time_tagged, qualifier);
        auto before = mark();
        if (sent) *sent = before;
        if (sent_type) *sent_type = built.type;
        CS104_Connection_sendProcessCommandEx(connection, (CS101_CauseOfTransmission)cot, ca, built.object);
        InformationObject_destroy(built.object);
        bool deactivate = cot == CS101_COT_DEACTIVATION;
        std::string action = deactivate ? " deactivate" : select ? " select" : " execute";
        traffic("tx", peer, std::string(iec104_name(built.type)) + action + " IOA " + std::to_string(ioa)
                + " value " + number(value) + (qualifier ? " QU " + std::to_string(qualifier) : ""));
        int confirm = deactivate ? CS101_COT_DEACTIVATION_CON : CS101_COT_ACTIVATION_CON;
        auto answer = wait(built.type, ioa, before, confirm, t1_ms);
        bool ok = answer && !answer->negative && answer->cot == confirm;
        std::string text = !answer ? "no confirmation within t1"
            : ok ? (deactivate ? "deactivation confirmed" : "confirmed") : refusal_text(*answer, select);
        steps.push_back({{"step", deactivate ? "deactivate" : select ? "select" : "execute"}, {"ok", ok}, {"text", text}});
        return ok;
    }

    // A negative confirmation keeps the cause "activation confirmed" with P/N set; say what it means.
    static std::string refusal_text(const Answer& answer, bool select) {
        if (answer.cot >= CS101_COT_UNKNOWN_TYPE_ID) return "refused by the device (" + iec104_cause(answer.cot) + ")";
        return select ? "refused by the device (negative confirmation: it may accept only direct execute; untick "
                        "Select before operate)"
                      : "refused by the device (negative confirmation)";
    }

    // General counter interrogation (C_CI_NA_1, QCC general request, read without freeze), sent
    // after the station interrogation has ended so devices that handle one request at a time
    // answer both. A device without counters may refuse it; that is not a discovery failure.
    std::string counter_interrogation(long gi_sent) {
        wait(C_IC_NA_1, 0, gi_sent, CS101_COT_ACTIVATION_TERMINATION, 20000);
        auto before = mark();
        CS104_Connection_sendCounterInterrogationCommand(connection, CS101_COT_ACTIVATION, ca,
                                                         IEC60870_QCC_RQT_GENERAL | IEC60870_QCC_FRZ_READ);
        traffic("tx", peer, "C_CI_NA_1 counter interrogation (read all counters), common address " + std::to_string(ca));
        auto answer = wait(C_CI_NA_1, 0, before, CS101_COT_ACTIVATION_CON, t1_ms);
        if (!answer) return "not confirmed within t1";
        return answer->negative || answer->cot != CS101_COT_ACTIVATION_CON ? "refused (the device may have none)" : "confirmed";
    }

    Json clock_sync() {
        auto before = mark();
        struct sCP56Time2a now;
        CP56Time2a_createFromMsTimestamp(&now, Hal_getTimeInMs());
        CS104_Connection_sendClockSyncCommand(connection, ca, &now);
        traffic("tx", peer, "C_CS_NA_1 clock synchronisation with the time of this PC");
        auto answer = wait(C_CS_NA_1, 0, before, CS101_COT_ACTIVATION_CON, t1_ms);
        bool ok = answer && !answer->negative;
        std::string text = ok ? "confirmed" : answer ? "refused by the device" : "no confirmation within t1";
        return {{"ok", ok}, {"steps", Json::array({Json{{"step", "clock"}, {"ok", ok}, {"text", text}}})}};
    }

    static bool on_asdu(void* self, int, CS101_ASDU asdu) {
        static_cast<Client104*>(self)->received(asdu);
        return true;
    }

    void received(CS101_ASDU asdu) {
        int type = CS101_ASDU_getTypeID(asdu), cot = CS101_ASDU_getCOT(asdu);
        auto values = iec104_values(asdu);
        std::string line = std::string(iec104_name(type)) + " " + iec104_cause(cot);
        if (!values.empty()) {
            emit_values(values);
            line += values.size() == 1 ? " IOA " + std::to_string(values[0]["address"].get<int>()) + " = "
                + number(values[0]["value"].get<double>()) : " " + std::to_string(values.size()) + " objects";
        } else {
            auto object = CS101_ASDU_getElement(asdu, 0);
            int ioa = object ? InformationObject_getObjectAddress(object) : 0;
            if (object) InformationObject_destroy(object);
            {
                std::lock_guard<std::mutex> guard(lock);
                answers.push_back({type, ioa, cot, CS101_ASDU_isNegative(asdu), ++serial});
                if (answers.size() > 200) answers.erase(answers.begin(), answers.begin() + 100);
            }
            arrived.notify_all();
            if (type == C_IC_NA_1 && cot == CS101_COT_ACTIVATION_TERMINATION) emit_interrogation("finished");
            line += " IOA " + std::to_string(ioa) + (CS101_ASDU_isNegative(asdu) ? " (negative)" : "");
        }
        traffic("rx", peer, line);
    }

    static void on_link(void* self, CS104_Connection, CS104_ConnectionEvent event) {
        auto client = static_cast<Client104*>(self);
        if (event == CS104_CONNECTION_OPENED) {
            client->open = true;
            emit_link("open", "Connected to " + client->peer);
        } else if (event == CS104_CONNECTION_CLOSED || event == CS104_CONNECTION_FAILED) {
            client->open = false;
            emit_link("closed", "The connection to " + client->peer + " closed");
        } else if (event == CS104_CONNECTION_STARTDT_CON_RECEIVED) {
            traffic("rx", client->peer, "STARTDT con: data transfer started");
        }
    }
};
}

std::unique_ptr<ProtocolClient> make_client_104() { return std::make_unique<Client104>(); }
