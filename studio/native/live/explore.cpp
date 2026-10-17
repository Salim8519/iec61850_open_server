// One level of the IEC 61850 model per request: devices, nodes of a device, or one node.
#include "explore.hpp"
#include "browse.hpp"
#include <stdexcept>

Json explore_level(MmsSession& session, const Json& request) {
    std::lock_guard<std::mutex> guard(session.io_lock());
    if (!session.handle()) throw std::runtime_error("Not connected");
    auto node = request.value("node", std::string());
    if (!node.empty()) return browse_node(session.handle(), node);
    auto device = request.value("device", std::string());
    return {{"names", browse_names(session.handle(), device)}};
}
