// DNP3 values received by ScadaBench as master, in the same records as the IEC 104 decoder.
#include "dnp3_values.hpp"
#include "traffic.hpp"
#include "value_events.hpp"
#include <opendnp3/gen/GroupVariation.h>

using namespace opendnp3;

namespace {
template<class T> double number(const T& value) { return double(value.value); }
double number(const DoubleBitBinary& value) { return double(static_cast<int>(value.value)); }
double number(const Binary& value) { return value.value ? 1.0 : 0.0; }
double number(const BinaryOutputStatus& value) { return value.value ? 1.0 : 0.0; }
}

void Dnp3Values::BeginFragment(const ResponseInfo& info) {
    unsolicited = info.unsolicited;
    batch = Json::array();
}

void Dnp3Values::EndFragment(const ResponseInfo&) {
    if (batch.empty()) return;
    traffic("rx", peer, std::string(unsolicited ? "Unsolicited response" : "Response") + " with "
                            + std::to_string(batch.size()) + (batch.size() == 1 ? " value" : " values"));
    emit_values(batch);
    batch = Json::array();
}

template<class T> void Dnp3Values::add(const char* group, const HeaderInfo& info,
                                        const ICollection<Indexed<T>>& values) {
    const char* cause = unsolicited ? "unsolicited" : info.isEventVariation ? "event" : "integrity poll";
    auto type = GroupVariationSpec::to_human_string(info.gv);
    values.ForeachItem([&](const Indexed<T>& item) {
        batch.push_back({{"group", group}, {"type", type}, {"address", item.index}, {"value", number(item.value)},
                         {"quality", int(item.value.flags.value)},
                         {"time_ms", item.value.time.value ? Json(item.value.time.value) : Json()}, {"cause", cause}});
    });
}

void Dnp3Values::Process(const HeaderInfo& info, const ICollection<Indexed<Binary>>& values) { add("binary", info, values); }
void Dnp3Values::Process(const HeaderInfo& info, const ICollection<Indexed<DoubleBitBinary>>& values) {
    add("double_binary", info, values);
}
void Dnp3Values::Process(const HeaderInfo& info, const ICollection<Indexed<Analog>>& values) { add("analog", info, values); }
void Dnp3Values::Process(const HeaderInfo& info, const ICollection<Indexed<Counter>>& values) { add("counter", info, values); }
void Dnp3Values::Process(const HeaderInfo& info, const ICollection<Indexed<FrozenCounter>>& values) {
    add("frozen_counter", info, values);
}
void Dnp3Values::Process(const HeaderInfo& info, const ICollection<Indexed<BinaryOutputStatus>>& values) {
    add("binary_output", info, values);
}
void Dnp3Values::Process(const HeaderInfo& info, const ICollection<Indexed<AnalogOutputStatus>>& values) {
    add("analog_output", info, values);
}
