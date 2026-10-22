// Test-only DNP3 master (never packaged): an independent opendnp3 master used to check
// the gateway simulator. Usage:
//   studio_test_dnp3_master HOST PORT MASTER_ADDR OUTSTATION_ADDR STEP...
// Steps: scan | crob:INDEX:on|off|pulse_on|pulse_off|trip|close[:direct]
//        | ao:INDEX:VALUE[:v1|v2|v3][:direct] | wait:MS. Without :direct a command is
// select-before-operate. Results are JSON lines.
#include <opendnp3/DNP3Manager.h>
#include <opendnp3/channel/IChannelListener.h>
#include <opendnp3/logging/ILogHandler.h>
#include <opendnp3/logging/LogLevels.h>
#include <opendnp3/master/DefaultMasterApplication.h>
#include <opendnp3/master/ISOEHandler.h>
#include <chrono>
#include <cstdio>
#include <cstdlib>
#include <mutex>
#include <string>
#include <thread>
#include <vector>

using namespace opendnp3;

namespace {
std::mutex output;

void line(const std::string& text) {
    std::lock_guard<std::mutex> guard(output);
    std::printf("%s\n", text.c_str());
    std::fflush(stdout);
}

// "event" is true for an event object (g2, g4, g11, g32, g42), false for a static one.
template<class T>
void emit(const char* kind, const HeaderInfo& info, const ICollection<Indexed<T>>& values, bool numeric) {
    values.ForeachItem([&](const Indexed<T>& item) {
        double value = numeric ? double(item.value.value) : double(int(item.value.value));
        line(std::string("{\"kind\":\"") + kind + "\",\"index\":" + std::to_string(item.index) + ",\"value\":"
             + std::to_string(value) + ",\"flags\":" + std::to_string(int(item.value.flags.value))
             + ",\"event\":" + (info.isEventVariation ? "true" : "false") + "}");
    });
}

class Measurements : public ISOEHandler {
public:
    void BeginFragment(const ResponseInfo&) override {}
    void EndFragment(const ResponseInfo&) override {}
    void Process(const HeaderInfo& info, const ICollection<Indexed<Binary>>& values) override {
        emit("binary", info, values, false);
    }
    void Process(const HeaderInfo& info, const ICollection<Indexed<DoubleBitBinary>>& values) override {
        emit("double", info, values, false);
    }
    void Process(const HeaderInfo& info, const ICollection<Indexed<Analog>>& values) override {
        emit("analog", info, values, true);
    }
    void Process(const HeaderInfo&, const ICollection<Indexed<Counter>>&) override {}
    void Process(const HeaderInfo&, const ICollection<Indexed<FrozenCounter>>&) override {}
    void Process(const HeaderInfo& info, const ICollection<Indexed<BinaryOutputStatus>>& values) override {
        emit("output", info, values, false);
    }
    void Process(const HeaderInfo& info, const ICollection<Indexed<AnalogOutputStatus>>& values) override {
        emit("analog_output", info, values, true);
    }
    void Process(const HeaderInfo&, const ICollection<Indexed<OctetString>>&) override {}
    void Process(const HeaderInfo&, const ICollection<Indexed<TimeAndInterval>>&) override {}
    void Process(const HeaderInfo&, const ICollection<Indexed<BinaryCommandEvent>>&) override {}
    void Process(const HeaderInfo&, const ICollection<Indexed<AnalogCommandEvent>>&) override {}
    void Process(const HeaderInfo&, const ICollection<DNPTime>&) override {}
};

class Quiet : public ILogHandler {
public:
    void log(ModuleId, const char*, LogLevel, char const*, char const*) override {}
};

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

// on/off = latch on/off; pulse_on, pulse_off; trip and close = pulse on with the trip-close code.
ControlRelayOutputBlock crob(const std::string& code) {
    if (code == "pulse_on") return ControlRelayOutputBlock(OperationType::PULSE_ON);
    if (code == "pulse_off") return ControlRelayOutputBlock(OperationType::PULSE_OFF);
    if (code == "trip") return ControlRelayOutputBlock(OperationType::PULSE_ON, TripCloseCode::TRIP);
    if (code == "close") return ControlRelayOutputBlock(OperationType::PULSE_ON, TripCloseCode::CLOSE);
    return ControlRelayOutputBlock(code == "on" ? OperationType::LATCH_ON : OperationType::LATCH_OFF);
}

// g41 var 1 (int32), var 2 (int16) or var 3 (float32).
CommandSet analog(const std::string& value, const std::string& variation, uint16_t index) {
    if (variation == "v1") return CommandSet({WithIndex(AnalogOutputInt32(int32_t(std::atoi(value.c_str()))), index)});
    if (variation == "v2") return CommandSet({WithIndex(AnalogOutputInt16(int16_t(std::atoi(value.c_str()))), index)});
    return CommandSet({WithIndex(AnalogOutputFloat32(float(std::atof(value.c_str()))), index)});
}

class Channel : public IChannelListener {
public:
    void OnStateChange(ChannelState state) override {
        line(std::string("{\"channel\":\"") + ChannelStateSpec::to_string(state) + "\"}");
    }
};
}

int main(int argc, char** argv) {
    if (argc < 5) {
        std::fprintf(stderr, "usage: HOST PORT MASTER_ADDR OUTSTATION_ADDR STEP...\n");
        return 2;
    }
    DNP3Manager manager(1, std::make_shared<Quiet>());
    auto channel = manager.AddTCPClient("master", levels::NORMAL, ChannelRetry::Default(),
                                        {IPEndpoint(argv[1], uint16_t(std::atoi(argv[2])))}, "0.0.0.0",
                                        std::make_shared<Channel>());
    MasterStackConfig config;
    config.master.responseTimeout = TimeDuration::Seconds(3);
    config.master.disableUnsolOnStartup = true;
    config.master.startupIntegrityClassMask = ClassField::None();
    config.link.LocalAddr = uint16_t(std::atoi(argv[3]));
    config.link.RemoteAddr = uint16_t(std::atoi(argv[4]));
    auto handler = std::make_shared<Measurements>();
    auto master = channel->AddMaster("master", handler, DefaultMasterApplication::Create(), config);
    master->Enable();
    std::this_thread::sleep_for(std::chrono::milliseconds(500));
    for (int index = 5; index < argc; ++index) {
        std::string step = argv[index];
        if (step == "scan") {
            master->ScanClasses(ClassField::AllClasses(), handler);
        } else if (step.rfind("wait:", 0) == 0) {
            std::this_thread::sleep_for(std::chrono::milliseconds(std::atoi(step.c_str() + 5)));
            continue;
        } else if (step.rfind("crob:", 0) == 0 || step.rfind("ao:", 0) == 0) {
            auto parts = split(step);
            auto point = uint16_t(std::atoi(parts[1].c_str()));
            bool direct = step.find(":direct") != std::string::npos;
            auto done = [point](const ICommandTaskResult& result) {
                std::string status = "none";
                result.ForeachItem([&](const CommandPointResult& item) { status = CommandStatusSpec::to_string(item.status); });
                line("{\"command\":" + std::to_string(point) + ",\"summary\":\""
                     + TaskCompletionSpec::to_string(result.summary) + "\",\"status\":\"" + status + "\"}");
            };
            CommandSet commands = parts[0] == "crob" ? CommandSet({WithIndex(crob(parts[2]), point)})
                                                     : analog(parts[2], parts.size() > 3 ? parts[3] : "v3", point);
            if (direct) master->DirectOperate(std::move(commands), done);
            else master->SelectAndOperate(std::move(commands), done);
            std::this_thread::sleep_for(std::chrono::milliseconds(1500));
            continue;
        }
        std::this_thread::sleep_for(std::chrono::milliseconds(800));
    }
    manager.Shutdown();
    return 0;
}
