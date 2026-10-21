#include "values.hpp"
#include <cstdint>
#include <stdexcept>
#include <string>

static Json value_json(Server& server, DataAttribute* attribute) {
    auto value = IedServer_getAttributeValue(server.handle, attribute);
    Json result = nullptr;
    if (value) {
        switch (MmsValue_getType(value)) {
        case MMS_BOOLEAN: result = MmsValue_getBoolean(value); break;
        case MMS_INTEGER: result = MmsValue_toInt64(value); break;
        case MMS_UNSIGNED: result = MmsValue_toUint32(value); break;
        case MMS_FLOAT: result = MmsValue_toDouble(value); break;
        case MMS_BIT_STRING:
            result = attribute->type == IEC61850_CODEDENUM ?
                MmsValue_getBitStringAsIntegerBigEndian(value) : MmsValue_getBitStringAsInteger(value);
            break;
        case MMS_VISIBLE_STRING: result = MmsValue_toString(value); break;
        case MMS_UTC_TIME: result = MmsValue_getUtcTimeInMs(value); break;
        default: break;
        }
    }
    return result;
}

Json read_value(Server& server, const std::string& reference) {
    auto attribute = server.attribute(reference);
    IedServer_lockDataModel(server.handle);
    auto result = value_json(server, attribute);
    IedServer_unlockDataModel(server.handle);
    return result;
}

Json read_versioned(Server& server, const std::string& reference) {
    auto attribute = server.attribute(reference);
    IedServer_lockDataModel(server.handle);
    Json result = {{"value", value_json(server, attribute)}, {"revision", server.revisions[attribute]}};
    IedServer_unlockDataModel(server.handle);
    return result;
}

// libiec61850 integer setters silently ignore values wider than the attribute (for example
// MmsValue_setInt64 on an INT32 or Enum), so pick the setter of the declared type.
static void set_integer(MmsValue* target, DataAttributeType type, int64_t value) {
    auto check = [value](int64_t low, int64_t high) {
        if (value < low || value > high) throw std::runtime_error("Value " + std::to_string(value) + " is out of range");
    };
    switch (type) {
    case IEC61850_INT8: check(INT8_MIN, INT8_MAX); MmsValue_setInt8(target, int8_t(value)); break;
    case IEC61850_INT16: check(INT16_MIN, INT16_MAX); MmsValue_setInt16(target, int16_t(value)); break;
    case IEC61850_INT64: MmsValue_setInt64(target, value); break;
    default: check(INT32_MIN, INT32_MAX); MmsValue_setInt32(target, int32_t(value)); break;
    }
    if (MmsValue_toInt64(target) != value) throw std::runtime_error("The device model could not store this integer");
}

static void set_unsigned(MmsValue* target, DataAttributeType type, uint32_t value) {
    switch (type) {
    case IEC61850_INT8U: if (value > UINT8_MAX) throw std::runtime_error("Value is out of range"); MmsValue_setUint8(target, uint8_t(value)); break;
    case IEC61850_INT16U: if (value > UINT16_MAX) throw std::runtime_error("Value is out of range"); MmsValue_setUint16(target, uint16_t(value)); break;
    default: MmsValue_setUint32(target, value); break;
    }
    if (MmsValue_toUint32(target) != value) throw std::runtime_error("The device model could not store this integer");
}

static MmsValue* prepare_value(Server& server, DataAttribute* attribute, const Json& value) {
    IedServer_lockDataModel(server.handle);
    auto current = IedServer_getAttributeValue(server.handle, attribute);
    auto candidate = current ? MmsValue_clone(current) : nullptr;
    IedServer_unlockDataModel(server.handle);
    if (!candidate) throw std::runtime_error("Attribute has no value or could not be cloned");
    bool supported = true;
    try {
        switch (MmsValue_getType(candidate)) {
        case MMS_BOOLEAN: MmsValue_setBoolean(candidate, value.get<bool>()); break;
        case MMS_INTEGER: set_integer(candidate, attribute->type, value.get<int64_t>()); break;
        case MMS_UNSIGNED: set_unsigned(candidate, attribute->type, value.get<uint32_t>()); break;
        case MMS_FLOAT:
            if (attribute->type == IEC61850_FLOAT32) MmsValue_setFloat(candidate, value.get<float>());
            else MmsValue_setDouble(candidate, value.get<double>());
            break;
        case MMS_BIT_STRING:
            if (attribute->type == IEC61850_CODEDENUM)
                MmsValue_setBitStringFromIntegerBigEndian(candidate, value.get<uint32_t>());
            else MmsValue_setBitStringFromInteger(candidate, value.get<uint32_t>());
            break;
        case MMS_VISIBLE_STRING:
            MmsValue_setVisibleString(candidate, value.get<std::string>().c_str()); break;
        case MMS_UTC_TIME: MmsValue_setUtcTimeMs(candidate, value.get<uint64_t>()); break;
        default: supported = false;
        }
    } catch (...) { MmsValue_delete(candidate); throw; }
    if (!supported) { MmsValue_delete(candidate); throw std::runtime_error("Unsupported scalar type"); }
    return candidate;
}

Json update_batch(Server& server, const Json& updates, const Json& guard) {
    if (!updates.is_array() || updates.empty() || updates.size() > 256)
        throw std::runtime_error("Update batch must contain 1 to 256 attributes");
    using Value = std::unique_ptr<MmsValue, decltype(&MmsValue_delete)>;
    struct Candidate {
        std::string reference;
        DataAttribute* attribute;
        Value value;
    };
    std::vector<Candidate> candidates;
    candidates.reserve(updates.size());
    // Validate every entry before changing any live attribute; each reference resolves once.
    for (const auto& item : updates) {
        auto reference = item.at("ref").get<std::string>();
        auto attribute = server.attribute(reference);
        Value value(prepare_value(server, attribute, item.at("value")), MmsValue_delete);
        candidates.push_back({std::move(reference), attribute, std::move(value)});
    }
    auto guarded = guard.is_null() ? nullptr : server.attribute(guard.at("ref").get<std::string>());
    auto expected = guard.is_null() ? 0 : guard.at("revision").get<uint64_t>();
    IedServer_lockDataModel(server.handle);
    if (guarded && server.revisions[guarded] != expected) {
        IedServer_unlockDataModel(server.handle);
        throw std::runtime_error("Signal changed; stale pulse action cancelled");
    }
    for (auto& candidate : candidates) {
        IedServer_updateAttributeValue(server.handle, candidate.attribute, candidate.value.get());
        ++server.revisions[candidate.attribute];
    }
    Json result = Json::object();
    for (const auto& candidate : candidates)
        result[candidate.reference] = value_json(server, candidate.attribute);
    auto revision = guarded ? server.revisions[guarded] : 0;
    IedServer_unlockDataModel(server.handle);
    server.loop.kick();  // reports queued by this update go out on the loop's next tick
    return guarded ? Json{{"values", result}, {"revision", revision}} : result;
}

void update_value(Server& server, const std::string& reference, const Json& value) {
    update_batch(server, Json::array({{{"ref", reference}, {"value", value}}}));
}
