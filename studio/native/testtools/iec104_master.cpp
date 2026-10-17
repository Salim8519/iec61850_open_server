// Test-only IEC 104 master (never packaged): an independent lib60870 client used to
// check the gateway simulator. Usage:
//   studio_test_iec104_master HOST PORT CA STEP...
// Steps: gi | read:IOA | sc:IOA:0|1[:select] | dc:IOA:1|2[:select] | wait:MS
// Every received ASDU is printed as one JSON line.
#include <chrono>
#include <cstdio>
#include <cstdlib>
#include <string>
#include <thread>
#include <vector>
extern "C" {
#include "cs104_connection.h"
#include "hal_time.h"
}

namespace {
std::vector<std::string> split(const std::string& text) {
    std::vector<std::string> parts;
    size_t start = 0, found;
    while ((found = text.find(':', start)) != std::string::npos) {
        parts.push_back(text.substr(start, found - start));
        start = found + 1;
    }
    parts.push_back(text.substr(start));
    return parts;
}

bool received(void*, int, CS101_ASDU asdu) {
    int type = CS101_ASDU_getTypeID(asdu);
    std::printf("{\"type\":%d,\"cot\":%d,\"negative\":%s,\"objects\":[", type, int(CS101_ASDU_getCOT(asdu)),
                CS101_ASDU_isNegative(asdu) ? "true" : "false");
    for (int index = 0; index < CS101_ASDU_getNumberOfElements(asdu); ++index) {
        auto object = CS101_ASDU_getElement(asdu, index);
        if (!object) continue;
        double value = 0;
        int quality = 0;
        if (type == M_SP_NA_1 || type == M_SP_TB_1) {
            value = SinglePointInformation_getValue((SinglePointInformation)object);
            quality = SinglePointInformation_getQuality((SinglePointInformation)object);
        } else if (type == M_DP_NA_1 || type == M_DP_TB_1) {
            value = DoublePointInformation_getValue((DoublePointInformation)object);
            quality = DoublePointInformation_getQuality((DoublePointInformation)object);
        } else if (type == M_ME_NC_1 || type == M_ME_TF_1) {
            value = MeasuredValueShort_getValue((MeasuredValueShort)object);
            quality = MeasuredValueShort_getQuality((MeasuredValueShort)object);
        } else if (type == M_ME_NB_1 || type == M_ME_TE_1) {
            value = MeasuredValueScaled_getValue((MeasuredValueScaled)object);
            quality = MeasuredValueScaled_getQuality((MeasuredValueScaled)object);
        }
        std::printf("%s{\"ioa\":%d,\"value\":%g,\"quality\":%d}", index ? "," : "",
                    InformationObject_getObjectAddress(object), value, quality);
        InformationObject_destroy(object);
    }
    std::printf("]}\n");
    std::fflush(stdout);
    return true;
}
}

int main(int argc, char** argv) {
    if (argc < 4) {
        std::fprintf(stderr, "usage: HOST PORT CA STEP...\n");
        return 2;
    }
    int ca = std::atoi(argv[3]);
    auto connection = CS104_Connection_create(argv[1], std::atoi(argv[2]));
    CS104_Connection_setConnectTimeout(connection, 3000);
    CS104_Connection_setASDUReceivedHandler(connection, received, nullptr);
    if (!CS104_Connection_connect(connection)) {
        std::printf("{\"error\":\"connect failed\"}\n");
        CS104_Connection_destroy(connection);
        return 1;
    }
    CS104_Connection_sendStartDT(connection);
    std::this_thread::sleep_for(std::chrono::milliseconds(200));
    for (int index = 4; index < argc; ++index) {
        auto step = split(argv[index]);
        if (step[0] == "gi") {
            CS104_Connection_sendInterrogationCommand(connection, CS101_COT_ACTIVATION, ca, IEC60870_QOI_STATION);
        } else if (step[0] == "read") {
            CS104_Connection_sendReadCommand(connection, ca, std::atoi(step[1].c_str()));
        } else if (step[0] == "sc" || step[0] == "dc") {
            int ioa = std::atoi(step[1].c_str()), state = std::atoi(step[2].c_str());
            bool select = step.size() > 3 && step[3] == "select";
            auto command = step[0] == "sc" ? (InformationObject)SingleCommand_create(nullptr, ioa, state != 0, select, 0)
                                           : (InformationObject)DoubleCommand_create(nullptr, ioa, state, select, 0);
            CS104_Connection_sendProcessCommandEx(connection, CS101_COT_ACTIVATION, ca, command);
            InformationObject_destroy(command);
        } else if (step[0] == "wait") {
            std::this_thread::sleep_for(std::chrono::milliseconds(std::atoi(step[1].c_str())));
            continue;
        }
        std::this_thread::sleep_for(std::chrono::milliseconds(300));
    }
    CS104_Connection_destroy(connection);
    return 0;
}
