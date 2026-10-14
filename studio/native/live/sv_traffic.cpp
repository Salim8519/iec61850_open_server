// Layer-2 Sampled Values reception and supervision (listen only).
#include "sv_traffic.hpp"
#include "npcap.hpp"
extern "C" {
#include "hal_time.h"
}
#include <chrono>
#include <cstdio>
#include <stdexcept>

namespace {
void parse_mac(const std::string& text, uint8_t* mac) {
    unsigned int parts[6];
    if (std::sscanf(text.c_str(), "%x-%x-%x-%x-%x-%x", &parts[0], &parts[1], &parts[2],
                    &parts[3], &parts[4], &parts[5]) != 6)
        throw std::runtime_error("Invalid Sampled Values MAC address " + text);
    for (int i = 0; i < 6; ++i) mac[i] = static_cast<uint8_t>(parts[i]);
}

void received(SVSubscriber, void* parameter, SVSubscriber_ASDU asdu) {
    auto pair = static_cast<std::pair<SvTraffic*, SvWatch*>*>(parameter);
    pair->first->on_asdu(pair->second, asdu);
}

std::map<SvWatch*, std::unique_ptr<std::pair<SvTraffic*, SvWatch*>>> contexts;
}

SvTraffic::~SvTraffic() { stop(); }

Json SvTraffic::start(const Json& request) {
    if (receiver) throw std::runtime_error("Sampled Values listener already running");
    if (!npcap_available()) throw std::runtime_error("Npcap is not installed, so Sampled Values cannot be received");
    if (request.at("streams").size() > STREAM_LIMIT)
        throw std::runtime_error("At most " + std::to_string(STREAM_LIMIT) + " Sampled Values streams are supervised");
    auto interface_id = npcap_interface_for(request.at("adapter_guid").get<std::string>());
    receiver = SVReceiver_create();
    SVReceiver_setInterfaceId(receiver, interface_id.c_str());
    for (const auto& item : request.at("streams")) {
        auto watch = std::make_unique<SvWatch>();
        watch->key = item.at("key").get<std::string>();
        watch->mac = item.at("mac").get<std::string>();
        watch->app_id = item.at("app_id").get<uint16_t>();
        watch->layout = item.value("layout", "raw");
        watch->per_cycle = item.value("per_cycle", 0);
        watch->counter.wrap = item.value("wrap", 0);
        watch->channels = item.value("channels", std::vector<std::string>{});
        watch->scale = item.value("scale", std::vector<double>{});
        watch->cycles.resize(watch->channels.size());
        uint8_t mac[6];
        parse_mac(watch->mac, mac);
        watch->subscriber = SVSubscriber_create(mac, watch->app_id);
        auto context = std::make_unique<std::pair<SvTraffic*, SvWatch*>>(this, watch.get());
        SVSubscriber_setListener(watch->subscriber, received, context.get());
        contexts[watch.get()] = std::move(context);
        SVReceiver_addSubscriber(receiver, watch->subscriber);
        watches[watch->key] = std::move(watch);
    }
    SVReceiver_start(receiver);
    if (!SVReceiver_isRunning(receiver)) {
        SVReceiver_destroy(receiver);
        receiver = nullptr;
        throw std::runtime_error("Could not open the network adapter for Sampled Values");
    }
    headers.start(interface_id);
    running = true;
    supervisor = std::thread([this] { supervise(); });
    return {{"listening", true}, {"streams", watches.size()}};
}

void SvTraffic::on_asdu(SvWatch* watch, SVSubscriber_ASDU asdu) {
    std::lock_guard<std::mutex> guard(lock);
    int smp_cnt = SVSubscriber_ASDU_getSmpCnt(asdu);
    watch->counter.next(smp_cnt);
    watch->asdus++;
    watch->window_asdus++;
    watch->last_rx_ms = Hal_getTimeInMs();
    watch->sv_id = SVSubscriber_ASDU_getSvId(asdu) ? SVSubscriber_ASDU_getSvId(asdu) : "";
    watch->dataset = SVSubscriber_ASDU_hasDatSet(asdu) && SVSubscriber_ASDU_getDatSet(asdu)
        ? SVSubscriber_ASDU_getDatSet(asdu) : "";
    watch->conf_rev = SVSubscriber_ASDU_getConfRev(asdu);
    watch->smp_rate = SVSubscriber_ASDU_hasSmpRate(asdu) ? SVSubscriber_ASDU_getSmpRate(asdu) : -1;
    watch->smp_mod = SVSubscriber_ASDU_hasSmpMod(asdu) ? SVSubscriber_ASDU_getSmpMod(asdu) : -1;
    watch->smp_synch = SVSubscriber_ASDU_getSmpSynch(asdu);
    watch->data_size = SVSubscriber_ASDU_getDataSize(asdu);
    bool interpreted = watch->layout == "9-2LE" && watch->data_size == 64 && watch->per_cycle > 0;
    bool invalid = false;
    for (size_t channel = 0; channel < watch->channels.size() && int(channel * 8 + 8) <= watch->data_size; ++channel) {
        int32_t value = SVSubscriber_ASDU_getINT32(asdu, int(channel * 8));
        Quality quality = SVSubscriber_ASDU_getQuality(asdu, int(channel * 8 + 4));
        if ((quality & 0x3) == QUALITY_VALIDITY_INVALID) invalid = true;
        if (interpreted) {
            double scale = channel < watch->scale.size() ? watch->scale[channel] : 1.0;
            watch->cycles[channel].add(value * scale, smp_cnt % watch->per_cycle, watch->per_cycle);
        }
    }
    if (invalid) watch->invalid++;
    if (!interpreted && watch->raw.empty() == false) return;
    if (!interpreted) {
        for (int index = 0; index + 4 <= watch->data_size && index < 64; index += 4)
            watch->raw.push_back(SVSubscriber_ASDU_getINT32(asdu, index));
    }
}

Json SvTraffic::report(SvWatch& watch, uint64_t now, const std::map<std::string, SvHeader>& seen) {
    double seconds = (now - watch.window_start_ms) / 1000.0;
    if (seconds > 0) watch.rate = watch.window_asdus / seconds;
    watch.window_asdus = 0;
    watch.window_start_ms = now;
    auto header = seen.find(sv_key(watch.app_id, watch.mac));
    Json values = Json::array();
    for (size_t channel = 0; channel < watch.cycles.size(); ++channel) {
        const auto& cycle = watch.cycles[channel];
        if (cycle.ready)
            values.push_back({{"name", watch.channels[channel]}, {"rms", cycle.rms},
                              {"magnitude", cycle.magnitude}, {"angle", cycle.angle}});
    }
    Json event = {{"event", "sv_rx"}, {"key", watch.key}, {"state", watch.state}, {"asdus", watch.asdus},
        {"rate", watch.rate}, {"sv_id", watch.sv_id}, {"dataset", watch.dataset}, {"conf_rev", watch.conf_rev},
        {"smp_rate", watch.smp_rate}, {"smp_mod", watch.smp_mod}, {"smp_synch", watch.smp_synch},
        {"lost", watch.counter.lost}, {"restarts", watch.counter.restarts}, {"invalid_quality", watch.invalid},
        {"data_size", watch.data_size}, {"values", values}, {"raw", watch.raw},
        {"simulate", header != seen.end() && header->second.simulate}};
    watch.raw.clear();
    return event;
}

void SvTraffic::supervise() {
    std::set<std::string> expected;
    {
        std::lock_guard<std::mutex> guard(lock);
        for (auto& entry : watches) {
            expected.insert(sv_key(entry.second->app_id, entry.second->mac));
            entry.second->window_start_ms = Hal_getTimeInMs();
        }
    }
    while (running) {
        std::this_thread::sleep_for(std::chrono::milliseconds(500));
        auto seen = headers.snapshot();
        std::lock_guard<std::mutex> guard(lock);
        auto now = Hal_getTimeInMs();
        for (auto& entry : watches) {
            auto& watch = *entry.second;
            if (watch.asdus) watch.state = now > watch.last_rx_ms + 1000 ? "timeout" : "receiving";
            emit_frame(report(watch, now, seen));
        }
        for (const auto& item : seen) {
            if (expected.count(item.first) || unknown.count(item.first)) continue;
            if (unknown.size() >= UNKNOWN_LIMIT) {
                if (!unknown_capped) emit_frame({{"event", "sv_unknown_cap"}, {"limit", UNKNOWN_LIMIT}});
                unknown_capped = true;
                continue;
            }
            unknown.insert(item.first);
            const auto& header = item.second;
            emit_frame({{"event", "sv_unknown"}, {"app_id", header.app_id}, {"mac", header.dst_mac},
                        {"src_mac", header.src_mac}, {"sv_id", header.sv_id}, {"simulate", header.simulate},
                        {"vlan_id", header.vlan_id >= 0 ? Json(header.vlan_id) : Json(nullptr)}});
        }
    }
}

void SvTraffic::stop() {
    running = false;
    if (supervisor.joinable()) supervisor.join();
    headers.stop();
    if (receiver) {
        SVReceiver_stop(receiver);
        SVReceiver_destroy(receiver);
        receiver = nullptr;
    }
    std::lock_guard<std::mutex> guard(lock);
    contexts.clear();
    watches.clear();
    unknown.clear();
    unknown_capped = false;
}
