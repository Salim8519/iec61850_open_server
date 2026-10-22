// Test-only IEC 104 master (never packaged): an independent lib60870 client used to
// check the gateway simulator. Usage:
//   studio_test_iec104_master HOST PORT CA STEP...
// Steps: gi | read:IOA | sc:IOA:0|1 | dc:IOA:1|2 | rc:IOA:1|2 (step lower/higher)
//        | sen:IOA:VALUE (normalized) | seb:IOA:VALUE (scaled) | sec:IOA:VALUE (float) | wait:MS
// Command options after the value, in any order: select (S/E = 1), time (the type with a
// CP56Time2a tag), cancel (COT 8 deactivation). Every received ASDU is printed as one JSON line.
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

bool has(const std::vector<std::string>& step, const char* option) {
    for (size_t index = 3; index < step.size(); ++index)
        if (step[index] == option) return true;
    return false;
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
        } else if (type == M_ST_NA_1 || type == M_ST_TB_1) {
            value = StepPositionInformation_getValue((StepPositionInformation)object);
            quality = StepPositionInformation_getQuality((StepPositionInformation)object);
        }
        std::printf("%s{\"ioa\":%d,\"value\":%g,\"quality\":%d}", index ? "," : "",
                    InformationObject_getObjectAddress(object), value, quality);
        InformationObject_destroy(object);
    }
    std::printf("]}\n");
    std::fflush(stdout);
    return true;
}

// The command object of one step, or nullptr for an unknown step.
InformationObject command(const std::vector<std::string>& step) {
    int ioa = std::atoi(step[1].c_str());
    double value = std::atof(step[2].c_str());
    bool select = has(step, "select"), tagged = has(step, "time");
    sCP56Time2a time;
    CP56Time2a_createFromMsTimestamp(&time, Hal_getTimeInMs());
    const auto& kind = step[0];
    if (kind == "sc") return tagged ? (InformationObject)SingleCommandWithCP56Time2a_create(nullptr, ioa, value != 0, select, 0, &time)
                                    : (InformationObject)SingleCommand_create(nullptr, ioa, value != 0, select, 0);
    if (kind == "dc") return tagged ? (InformationObject)DoubleCommandWithCP56Time2a_create(nullptr, ioa, int(value), select, 0, &time)
                                    : (InformationObject)DoubleCommand_create(nullptr, ioa, int(value), select, 0);
    if (kind == "rc") {
        auto state = StepCommandValue(int(value));
        return tagged ? (InformationObject)StepCommandWithCP56Time2a_create(nullptr, ioa, state, select, 0, &time)
                      : (InformationObject)StepCommand_create(nullptr, ioa, state, select, 0);
    }
    if (kind == "sen") return tagged ? (InformationObject)SetpointCommandNormalizedWithCP56Time2a_create(
                                           nullptr, ioa, float(value), select, 0, &time)
                                     : (InformationObject)SetpointCommandNormalized_create(nullptr, ioa, float(value), select, 0);
    if (kind == "seb") return tagged ? (InformationObject)SetpointCommandScaledWithCP56Time2a_create(
                                           nullptr, ioa, int(value), select, 0, &time)
                                     : (InformationObject)SetpointCommandScaled_create(nullptr, ioa, int(value), select, 0);
    if (kind == "sec") return tagged ? (InformationObject)SetpointCommandShortWithCP56Time2a_create(
                                           nullptr, ioa, float(value), select, 0, &time)
                                     : (InformationObject)SetpointCommandShort_create(nullptr, ioa, float(value), select, 0);
    return nullptr;
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
        } else if (step[0] == "wait") {
            std::this_thread::sleep_for(std::chrono::milliseconds(std::atoi(step[1].c_str())));
            continue;
        } else if (step.size() >= 3) {
            auto object = command(step);
            if (!object) continue;
            auto cause = has(step, "cancel") ? CS101_COT_DEACTIVATION : CS101_COT_ACTIVATION;
            CS104_Connection_sendProcessCommandEx(connection, cause, ca, object);
            InformationObject_destroy(object);
        }
        std::this_thread::sleep_for(std::chrono::milliseconds(300));
    }
    CS104_Connection_destroy(connection);
    return 0;
}
