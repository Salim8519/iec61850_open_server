// Test-only DNP3 master (never packaged): an independent opendnp3 master used to check
// the gateway simulator. Usage:
//   studio_test_dnp3_master HOST PORT MASTER_ADDR OUTSTATION_ADDR STEP...
// Steps: scan | crob:INDEX:on|off[:direct] | wait:MS. Results are JSON lines.
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

using namespace opendnp3;

namespace {
std::mutex output;

void line(const std::string& text) {
    std::lock_guard<std::mutex> guard(output);
    std::printf("%s\n", text.c_str());
    std::fflush(stdout);
}

template<class T> void emit(const char* kind, const ICollection<Indexed<T>>& values, bool numeric) {
    values.ForeachItem([&](const Indexed<T>& item) {
        double value = numeric ? double(item.value.value) : double(int(item.value.value));
        line(std::string("{\"kind\":\"") + kind + "\",\"index\":" + std::to_string(item.index) + ",\"value\":"
             + std::to_string(value) + ",\"flags\":" + std::to_string(int(item.value.flags.value)) + "}");
    });
}

class Measurements : public ISOEHandler {
public:
    void BeginFragment(const ResponseInfo&) override {}
    void EndFragment(const ResponseInfo&) override {}
    void Process(const HeaderInfo&, const ICollection<Indexed<Binary>>& values) override { emit("binary", values, false); }
    void Process(const HeaderInfo&, const ICollection<Indexed<DoubleBitBinary>>& values) override {
        emit("double", values, false);
    }
    void Process(const HeaderInfo&, const ICollection<Indexed<Analog>>& values) override { emit("analog", values, true); }
    void Process(const HeaderInfo&, const ICollection<Indexed<Counter>>&) override {}
    void Process(const HeaderInfo&, const ICollection<Indexed<FrozenCounter>>&) override {}
    void Process(const HeaderInfo&, const ICollection<Indexed<BinaryOutputStatus>>& values) override {
        emit("output", values, false);
    }
    void Process(const HeaderInfo&, const ICollection<Indexed<AnalogOutputStatus>>&) override {}
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
        } else if (step.rfind("crob:", 0) == 0) {
            auto rest = step.substr(5);
            auto point = uint16_t(std::atoi(rest.c_str()));
            bool on = rest.find(":on") != std::string::npos;
            bool direct = rest.find(":direct") != std::string::npos;
            ControlRelayOutputBlock crob(on ? OperationType::LATCH_ON : OperationType::LATCH_OFF);
            auto done = [point](const ICommandTaskResult& result) {
                std::string status = "none";
                result.ForeachItem([&](const CommandPointResult& item) { status = CommandStatusSpec::to_string(item.status); });
                line("{\"command\":" + std::to_string(point) + ",\"summary\":\""
                     + TaskCompletionSpec::to_string(result.summary) + "\",\"status\":\"" + status + "\"}");
            };
            if (direct) master->DirectOperate(CommandSet({WithIndex(crob, point)}), done);
            else master->SelectAndOperate(CommandSet({WithIndex(crob, point)}), done);
            std::this_thread::sleep_for(std::chrono::milliseconds(1500));
            continue;
        }
        std::this_thread::sleep_for(std::chrono::milliseconds(800));
    }
    manager.Shutdown();
    return 0;
}
