// Lets an MMS client change a measurement's db or zeroDb where the SCD allows it (ADR-0118):
// valKind Set or not given. Every other configuration (FC CF) value stays refused, as
// libiec61850's default write policy says; this only adds a handler to the named attributes.
#include "server.hpp"
#include <cstdint>

namespace {
MmsDataAccessError written(DataAttribute*, MmsValue* value, ClientConnection, void* parameter) {
    const auto& reference = *static_cast<const std::string*>(parameter);
    int64_t number = 0;
    switch (MmsValue_getType(value)) {
    case MMS_UNSIGNED: number = MmsValue_toUint32(value); break;
    case MMS_INTEGER: number = MmsValue_toInt64(value); break;
    default: return DATA_ACCESS_ERROR_TYPE_INCONSISTENT;
    }
    // IEC 61850-7-3: db and zeroDb are 0..100000 (0.001 % of the measuring range).
    if (number < 0 || number > 100000) return DATA_ACCESS_ERROR_OBJECT_VALUE_INVALID;
    emit_frame({{"event", "deadband_written"}, {"ref", reference}, {"value", number}});
    return DATA_ACCESS_ERROR_SUCCESS;
}
}

int install_deadband_writes(Server& server, const Json& references) {
    int installed = 0;
    for (const auto& entry : references) {
        auto reference = std::make_unique<std::string>(entry.get<std::string>());
        DataAttribute* attribute = nullptr;
        try { attribute = server.attribute(*reference); } catch (const std::exception&) { continue; }
        IedServer_handleWriteAccess(server.handle, attribute, written, reference.get());
        server.deadband_refs.push_back(std::move(reference));
        ++installed;
    }
    return installed;
}
