// ScadaBench as DNP3 master toward an outstation (the device): integrity poll, class 1/2/3
// event polls, unsolicited events, CROB and analog output commands (opendnp3).
#include "client.hpp"
#include "dnp3_values.hpp"
#include "traffic.hpp"
#include "value_events.hpp"
#include <opendnp3/DNP3Manager.h>
#include <opendnp3/channel/IChannelListener.h>
#include <opendnp3/logging/ILogHandler.h>
#include <opendnp3/logging/LogLevels.h>
#include <opendnp3/master/DefaultMasterApplication.h>
#include <opendnp3/master/ITaskCallback.h>
#include <atomic>
#include <chrono>
#include <future>

using namespace opendnp3;

namespace {
constexpr int CONNECT_SECONDS = 10;

std::string words(const char* code) {
    std::string text(code);
    for (auto& letter : text) letter = letter == '_' ? ' ' : char(std::tolower(static_cast<unsigned char>(letter)));
    return text;
}

class Quiet : public ILogHandler {
public:
    void log(ModuleId, const char*, LogLevel, char const*, char const*) override {}
};

class Link : public IChannelListener {
public:
    explicit Link(std::string peer) : peer(std::move(peer)) {}
    void OnStateChange(ChannelState state) override {
        if (state == ChannelState::OPEN) {
            open = true;
            opened.store(true);
            emit_link("open", "Connected to " + peer);
        } else if (state == ChannelState::CLOSED && open && !closing) {
            open = false;
            emit_link("closed", "The connection to " + peer + " closed");
        }
    }
    std::string peer;
    std::atomic<bool> open{false}, opened{false}, closing{false};
};

class PollDone : public ITaskCallback {
public:
    explicit PollDone(std::string peer) : peer(std::move(peer)) {}
    void OnStart() override {}
    void OnComplete(TaskCompletion result) override {
        bool ok = result == TaskCompletion::SUCCESS;
        traffic("info", peer, "Integrity poll " + (ok ? std::string("finished") : words(TaskCompletionSpec::to_string(result))));
        emit_interrogation(ok ? "finished" : "failed: " + words(TaskCompletionSpec::to_string(result)));
    }
    void OnDestroyed() override {}
    std::string peer;
};

class ClientDnp3 : public ProtocolClient {
public:
    ~ClientDnp3() override { disconnect(); }

    Json connect(const Json& settings) override {
        auto host = settings.at("host").get<std::string>();
        int port = settings.value("port", 20000);
        peer = peer_name(host, port);
        link = std::make_shared<Link>(peer);
        values = std::make_shared<Dnp3Values>(peer);
        manager = std::make_unique<DNP3Manager>(1, std::make_shared<Quiet>());
        auto channel = manager->AddTCPClient("master", levels::NOTHING,
                                             ChannelRetry(TimeDuration::Seconds(1), TimeDuration::Seconds(10)),
                                             {IPEndpoint(host, uint16_t(port))}, "0.0.0.0", link);
        MasterStackConfig config;
        config.master.responseTimeout = TimeDuration::Seconds(5);
        bool unsolicited = settings.value("unsolicited", true);
        config.master.disableUnsolOnStartup = !unsolicited;
        config.master.unsolClassMask = unsolicited ? ClassField::AllEventClasses() : ClassField::None();
        config.master.startupIntegrityClassMask = ClassField::None();  // the backend asks right after connecting
        config.link.LocalAddr = uint16_t(settings.value("master_address", 1));
        config.link.RemoteAddr = uint16_t(settings.value("outstation_address", 10));
        emit_link("connecting", "Connecting to " + peer);
        master = channel->AddMaster("master", values, DefaultMasterApplication::Create(), config);
        int events = settings.value("event_seconds", 5);
        if (events > 0) master->AddClassScan(ClassField::AllEventClasses(), TimeDuration::Seconds(events), values);
        master->Enable();
        auto until = std::chrono::steady_clock::now() + std::chrono::seconds(CONNECT_SECONDS);
        bool refused = false;  // a connect attempt failed at once (RST): nothing listens there
        for (int round = 1; !link->opened && !refused && std::chrono::steady_clock::now() < until; ++round) {
            std::this_thread::sleep_for(std::chrono::milliseconds(50));
            if (round % 4 == 0) refused = channel->GetStatistics().channel.numOpenFail > 0 && !link->opened;
        }
        if (refused) {
            disconnect();
            throw std::runtime_error(refused_text(peer, "outstation"));
        }
        if (!link->opened) {
            disconnect();
            throw std::runtime_error("No answer from " + peer + " within " + std::to_string(CONNECT_SECONDS)
                                     + " s. Check the address and port, and that the outstation accepts this PC as its master.");
        }
        return {{"host", host}, {"port", port}, {"outstation_address", config.link.RemoteAddr}};
    }

    Json interrogate() override {
        require_open();
        master->ScanClasses(ClassField::AllClasses(), values, TaskConfig::With(std::make_shared<PollDone>(peer)));
        traffic("tx", peer, "Integrity poll (class 0, 1, 2 and 3)");
        return {{"confirmed", true}, {"text", "sent (integrity poll)"}};
    }

    // The outstation's status, plus the step that failed when it was not the answer itself
    // (state SUCCESS only means a matching answer arrived, so it is not shown as "success").
    static std::string refusal(const CommandPointResult& item) {
        std::string text = "refused: " + words(CommandStatusSpec::to_string(item.status));
        if (item.status == CommandStatus::NOT_SUPPORTED) text += " (no such control point, or this control is not allowed)";
        switch (item.state) {
        case CommandPointState::SELECT_FAIL: return text + ", at select; try direct operate";
        case CommandPointState::SELECT_SUCCESS: return "selected, but the operate did not complete";
        case CommandPointState::OPERATE_FAIL: return text + ", at operate";
        case CommandPointState::SELECT_MISMATCH: return text + ", the select answer did not match the request";
        case CommandPointState::INIT: return "no answer for this point";
        default: return text;
        }
    }

    Json command(const Json& request) override {
        require_open();
        auto kind = request.at("kind").get<std::string>();
        auto index = uint16_t(request.at("address").get<int>());
        double value = request.at("value").get<double>();
        bool select = request.value("select", false);
        auto result = std::make_shared<std::promise<Json>>();
        auto done = [result](const ICommandTaskResult& answer) {
            Json step = {{"ok", false}, {"text", "no answer (" + words(TaskCompletionSpec::to_string(answer.summary)) + ")"}};
            answer.ForeachItem([&](const CommandPointResult& item) {
                bool ok = item.state == CommandPointState::SUCCESS && item.status == CommandStatus::SUCCESS;
                step = {{"ok", ok}, {"text", ok ? std::string("accepted by the outstation") : refusal(item)}};
            });
            try { result->set_value(step); } catch (const std::future_error&) {}
        };
        auto send = [&](CommandSet&& set) {  // CommandSet is move-only
            if (select) master->SelectAndOperate(std::move(set), done);
            else master->DirectOperate(std::move(set), done);
        };
        std::string line;
        if (kind == "binary_output") {
            auto code = request.value("code", std::string("latch"));
            auto opType = code == "pulse" ? (value ? OperationType::PULSE_ON : OperationType::PULSE_OFF)
                : code == "trip_close" ? OperationType::PULSE_ON
                : (value ? OperationType::LATCH_ON : OperationType::LATCH_OFF);
            auto tcc = code == "trip_close" ? (value ? TripCloseCode::CLOSE : TripCloseCode::TRIP) : TripCloseCode::NUL;
            uint8_t count = uint8_t(request.value("count", 1));
            uint32_t onTime = request.value("on_time_ms", 100), offTime = request.value("off_time_ms", 100);
            ControlRelayOutputBlock crob(opType, tcc, false, count, onTime, offTime);
            send(CommandSet({WithIndex(crob, index)}));
            line = "CROB " + words(OperationTypeSpec::to_string(crob.opType))
                   + (crob.tcc != TripCloseCode::NUL ? " " + words(TripCloseCodeSpec::to_string(crob.tcc)) : "")
                   + " count " + std::to_string(count) + " on " + std::to_string(onTime) + " ms off "
                   + std::to_string(offTime) + " ms";
        } else if (kind == "analog_output_int16") {
            send(CommandSet({WithIndex(AnalogOutputInt16(int16_t(value)), index)}));
            line = "Analog output (16-bit integer) " + std::to_string(int16_t(value));
        } else if (kind == "analog_output_int") {
            send(CommandSet({WithIndex(AnalogOutputInt32(int32_t(value)), index)}));
            line = "Analog output (32-bit integer) " + std::to_string(int32_t(value));
        } else if (kind == "analog_output_float") {
            send(CommandSet({WithIndex(AnalogOutputFloat32(float(value)), index)}));
            line = "Analog output (float) " + std::to_string(value);
        } else {
            throw std::runtime_error("Unknown DNP3 command kind " + kind);
        }
        traffic("tx", peer, line + " index " + std::to_string(index) + (select ? " (select before operate)" : " (direct operate)"));
        auto future = result->get_future();
        Json step = future.wait_for(std::chrono::seconds(15)) == std::future_status::ready ? future.get()
            : Json{{"ok", false}, {"text", "no answer within 15 s"}};
        step["step"] = select ? "select and operate" : "direct operate";
        traffic("rx", peer, "Command " + step["text"].get<std::string>());
        return {{"ok", step["ok"]}, {"steps", Json::array({step})}};
    }

    void disconnect() override {
        if (!manager) return;
        link->closing = true;
        manager->Shutdown();
        manager.reset();
        master.reset();
        emit_link("closed", "Disconnected by you");
    }

private:
    std::unique_ptr<DNP3Manager> manager;
    std::shared_ptr<IMaster> master;
    std::shared_ptr<Link> link;
    std::shared_ptr<Dnp3Values> values;
    std::string peer;

    void require_open() {
        if (!manager || !link->open) throw std::runtime_error("Not connected to the outstation; press Connect first");
    }
};
}

std::unique_ptr<ProtocolClient> make_client_dnp3() { return std::make_unique<ClientDnp3>(); }
