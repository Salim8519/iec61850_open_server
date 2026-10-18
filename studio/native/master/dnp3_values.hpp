#pragma once
#include "ipc/framing.hpp"
#include <opendnp3/master/ISOEHandler.h>
#include <string>

// Collects the values of one DNP3 response fragment as master_values records
// {group, type, address, value, quality, time_ms, cause} and sends them when the fragment
// ends. The quality is the raw DNP3 flags byte (bit 0 ONLINE); the backend words it.
class Dnp3Values : public opendnp3::ISOEHandler {
public:
    explicit Dnp3Values(std::string peer) : peer(std::move(peer)) {}
    void BeginFragment(const opendnp3::ResponseInfo& info) override;
    void EndFragment(const opendnp3::ResponseInfo& info) override;
    void Process(const opendnp3::HeaderInfo& info,
                 const opendnp3::ICollection<opendnp3::Indexed<opendnp3::Binary>>& values) override;
    void Process(const opendnp3::HeaderInfo& info,
                 const opendnp3::ICollection<opendnp3::Indexed<opendnp3::DoubleBitBinary>>& values) override;
    void Process(const opendnp3::HeaderInfo& info,
                 const opendnp3::ICollection<opendnp3::Indexed<opendnp3::Analog>>& values) override;
    void Process(const opendnp3::HeaderInfo& info,
                 const opendnp3::ICollection<opendnp3::Indexed<opendnp3::Counter>>& values) override;
    void Process(const opendnp3::HeaderInfo& info,
                 const opendnp3::ICollection<opendnp3::Indexed<opendnp3::FrozenCounter>>& values) override;
    void Process(const opendnp3::HeaderInfo& info,
                 const opendnp3::ICollection<opendnp3::Indexed<opendnp3::BinaryOutputStatus>>& values) override;
    void Process(const opendnp3::HeaderInfo& info,
                 const opendnp3::ICollection<opendnp3::Indexed<opendnp3::AnalogOutputStatus>>& values) override;
    void Process(const opendnp3::HeaderInfo&, const opendnp3::ICollection<opendnp3::Indexed<opendnp3::OctetString>>&) override {}
    void Process(const opendnp3::HeaderInfo&, const opendnp3::ICollection<opendnp3::Indexed<opendnp3::TimeAndInterval>>&) override {}
    void Process(const opendnp3::HeaderInfo&,
                 const opendnp3::ICollection<opendnp3::Indexed<opendnp3::BinaryCommandEvent>>&) override {}
    void Process(const opendnp3::HeaderInfo&,
                 const opendnp3::ICollection<opendnp3::Indexed<opendnp3::AnalogCommandEvent>>&) override {}
    void Process(const opendnp3::HeaderInfo&, const opendnp3::ICollection<opendnp3::DNPTime>&) override {}

private:
    std::string peer;
    bool unsolicited = false;
    Json batch = Json::array();
    template<class T> void add(const char* group, const opendnp3::HeaderInfo& info,
                               const opendnp3::ICollection<opendnp3::Indexed<T>>& values);
};
