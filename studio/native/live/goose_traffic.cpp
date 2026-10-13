// Layer-2 GOOSE reception and supervision for the GOOSE traffic view.
#include "goose_traffic.hpp"
#include "mms_json.hpp"
#include "npcap.hpp"
extern "C" {
#include "hal_time.h"
}
#include <chrono>
#include <cstdio>
#include <stdexcept>

namespace {
Json tree(const MmsValue* value, int depth = 0) {
    if (!value || depth > 32) return nullptr;
    auto type = MmsValue_getType(value);
    if (type != MMS_STRUCTURE && type != MMS_ARRAY) return mms_scalar_json(value);
    Json items = Json::array();
    auto mutable_value = const_cast<MmsValue*>(value);
    for (int i = 0; i < MmsValue_getArraySize(mutable_value); ++i)
        items.push_back(tree(MmsValue_getElement(mutable_value, i), depth + 1));
    return items;
}

void parse_mac(const std::string& text, uint8_t* mac) {
    unsigned int parts[6];
    if (std::sscanf(text.c_str(), "%x-%x-%x-%x-%x-%x", &parts[0], &parts[1], &parts[2],
                    &parts[3], &parts[4], &parts[5]) != 6)
        throw std::runtime_error("Invalid GOOSE MAC address " + text);
    for (int i = 0; i < 6; ++i) mac[i] = static_cast<uint8_t>(parts[i]);
}

void matched(GooseSubscriber subscriber, void* parameter) {
    auto pair = static_cast<std::pair<GooseTraffic*, GooseWatch*>*>(parameter);
    pair->first->on_frame(subscriber, pair->second);
}

void observed(GooseSubscriber subscriber, void* parameter) {
    static_cast<GooseTraffic*>(parameter)->on_observed(subscriber);
}

std::map<GooseWatch*, std::unique_ptr<std::pair<GooseTraffic*, GooseWatch*>>> contexts;
}

GooseTraffic::~GooseTraffic() { stop(); }

Json GooseTraffic::start(const Json& request) {
    if (receiver) throw std::runtime_error("GOOSE listener already running");
    if (!npcap_available()) throw std::runtime_error("Npcap is not installed, so GOOSE cannot be received");
    auto interface_id = npcap_interface_for(request.at("adapter_guid").get<std::string>());
    receiver = GooseReceiver_create();
    GooseReceiver_setInterfaceId(receiver, interface_id.c_str());
    for (const auto& item : request.at("subscriptions")) {
        auto watch = std::make_unique<GooseWatch>();
        watch->key = item.at("key").get<std::string>();
        watch->gocb_ref = item.at("gocb_ref").get<std::string>();
        watch->expected_conf_rev = item.value("conf_rev", 0u);
        watch->expected_dataset = item.value("dataset_ref", "");
        watch->expected_go_id = item.value("go_id", "");
        watch->subscriber = GooseSubscriber_create(const_cast<char*>(watch->gocb_ref.c_str()), nullptr);
        uint8_t mac[6];
        parse_mac(item.at("mac").get<std::string>(), mac);
        GooseSubscriber_setDstMac(watch->subscriber, mac);
        GooseSubscriber_setAppId(watch->subscriber, item.at("app_id").get<uint16_t>());
        auto context = std::make_unique<std::pair<GooseTraffic*, GooseWatch*>>(this, watch.get());
        GooseSubscriber_setListener(watch->subscriber, matched, context.get());
        contexts[watch.get()] = std::move(context);
        GooseReceiver_addSubscriber(receiver, watch->subscriber);
        watches[watch->key] = std::move(watch);
    }
    auto observer = GooseSubscriber_create(const_cast<char*>(""), nullptr);
    GooseSubscriber_setObserver(observer);
    GooseSubscriber_setListener(observer, observed, this);
    GooseReceiver_addSubscriber(receiver, observer);
    GooseReceiver_start(receiver);
    if (!GooseReceiver_isRunning(receiver)) {
        GooseReceiver_destroy(receiver);
        receiver = nullptr;
        throw std::runtime_error("Could not open the network adapter for GOOSE");
    }
    running = true;
    supervisor = std::thread([this] { supervise(); });
    return {{"listening", true}, {"interface", interface_id}, {"subscriptions", watches.size()}};
}

void GooseTraffic::on_frame(GooseSubscriber subscriber, GooseWatch* watch) {
    std::lock_guard<std::mutex> guard(lock);
    bool valid = GooseSubscriber_isValid(subscriber);
    uint32_t st = GooseSubscriber_getStNum(subscriber);
    uint32_t conf = GooseSubscriber_getConfRev(subscriber);
    std::string dataset = GooseSubscriber_getDataSet(subscriber) ? GooseSubscriber_getDataSet(subscriber) : "";
    std::string problem;
    if (!valid) problem = "invalid_frame";
    else if (watch->expected_conf_rev && conf != watch->expected_conf_rev) problem = "conf_rev_mismatch";
    else if (!watch->expected_dataset.empty() && dataset != watch->expected_dataset) problem = "dataset_mismatch";
    else if (GooseSubscriber_needsCommission(subscriber)) problem = "needs_commission";
    bool changed = !watch->frames || st != watch->st_num || problem != watch->problem;
    watch->frames++;
    watch->st_num = st;
    watch->sq_num = GooseSubscriber_getSqNum(subscriber);
    watch->tal_ms = GooseSubscriber_getTimeAllowedToLive(subscriber);
    watch->last_rx_ms = Hal_getTimeInMs();
    watch->problem = problem;
    auto state = problem.empty() ? "receiving" : "problem";
    if (!changed && watch->state == state) return;
    watch->state = state;
    Json event = {{"event", "goose_rx"}, {"key", watch->key}, {"gocb_ref", watch->gocb_ref},
        {"st_num", st}, {"sq_num", watch->sq_num}, {"conf_rev", conf}, {"dataset", dataset},
        {"go_id", GooseSubscriber_getGoId(subscriber) ? GooseSubscriber_getGoId(subscriber) : ""},
        {"test", GooseSubscriber_isTest(subscriber)}, {"tal_ms", watch->tal_ms},
        {"source_timestamp", GooseSubscriber_getTimestamp(subscriber)}, {"state", state},
        {"problem", problem}, {"frames", watch->frames}};
    if (valid) event["values"] = tree(GooseSubscriber_getDataSetValues(subscriber));
    emit_frame(event);
}

void GooseTraffic::on_observed(GooseSubscriber subscriber) {
    std::string ref = GooseSubscriber_getGoCbRef(subscriber) ? GooseSubscriber_getGoCbRef(subscriber) : "";
    std::lock_guard<std::mutex> guard(lock);
    for (const auto& entry : watches) if (entry.second->gocb_ref == ref) return;
    if (!unknown.insert(ref).second || unknown.size() > 500) return;
    uint8_t mac[6];
    GooseSubscriber_getDstMac(subscriber, mac);
    char text[18];
    std::snprintf(text, sizeof(text), "%02X-%02X-%02X-%02X-%02X-%02X", mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]);
    emit_frame({{"event", "goose_unknown"}, {"gocb_ref", ref}, {"app_id", GooseSubscriber_getAppId(subscriber)},
                {"mac", text}, {"dataset", GooseSubscriber_getDataSet(subscriber) ? GooseSubscriber_getDataSet(subscriber) : ""}});
}

void GooseTraffic::supervise() {
    while (running) {
        std::this_thread::sleep_for(std::chrono::milliseconds(250));
        std::lock_guard<std::mutex> guard(lock);
        auto now = Hal_getTimeInMs();
        Json summary = Json::array();
        for (auto& entry : watches) {
            auto& watch = *entry.second;
            if (watch.frames && watch.state != "timeout" && now > watch.last_rx_ms + watch.tal_ms + 50) {
                watch.state = "timeout";
                emit_frame({{"event", "goose_rx"}, {"key", watch.key}, {"gocb_ref", watch.gocb_ref},
                            {"state", "timeout"}, {"problem", "timeout"}, {"st_num", watch.st_num},
                            {"sq_num", watch.sq_num}, {"frames", watch.frames}});
            }
            summary.push_back({{"key", watch.key}, {"frames", watch.frames}, {"sq_num", watch.sq_num},
                               {"age_ms", watch.frames ? now - watch.last_rx_ms : 0}});
        }
        static int ticks = 0;
        if (++ticks % 8 == 0) emit_frame({{"event", "goose_stats"}, {"watches", summary}});
    }
}

void GooseTraffic::stop() {
    running = false;
    if (supervisor.joinable()) supervisor.join();
    if (receiver) {
        GooseReceiver_stop(receiver);
        GooseReceiver_destroy(receiver);
        receiver = nullptr;
    }
    std::lock_guard<std::mutex> guard(lock);
    contexts.clear();
    watches.clear();
    unknown.clear();
}
