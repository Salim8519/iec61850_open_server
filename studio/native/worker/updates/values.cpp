#include "values.hpp"
#include <stdexcept>

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

static MmsValue* prepare_value(Server& server, const std::string& reference, const Json& value) {
    auto attribute = server.attribute(reference);
    IedServer_lockDataModel(server.handle);
    auto current = IedServer_getAttributeValue(server.handle, attribute);
    auto candidate = current ? MmsValue_clone(current) : nullptr;
    IedServer_unlockDataModel(server.handle);
    if (!candidate) throw std::runtime_error("Attribute has no value or could not be cloned");
    bool supported = true;
    try {
        switch (MmsValue_getType(candidate)) {
        case MMS_BOOLEAN: MmsValue_setBoolean(candidate, value.get<bool>()); break;
        case MMS_INTEGER: MmsValue_setInt64(candidate, value.get<int64_t>()); break;
        case MMS_UNSIGNED: MmsValue_setUint32(candidate, value.get<uint32_t>()); break;
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
    std::vector<std::pair<DataAttribute*, Value>> candidates;
    // Validate every entry before changing any live attribute.
    for (const auto& item : updates) {
        auto reference = item.at("ref").get<std::string>();
        auto attribute = server.attribute(reference);
        candidates.emplace_back(attribute, Value(prepare_value(server, reference, item.at("value")), MmsValue_delete));
    }
    auto guarded = guard.is_null() ? nullptr : server.attribute(guard.at("ref").get<std::string>());
    auto expected = guard.is_null() ? 0 : guard.at("revision").get<uint64_t>();
    IedServer_lockDataModel(server.handle);
    if (guarded && server.revisions[guarded] != expected) {
        IedServer_unlockDataModel(server.handle);
        throw std::runtime_error("Signal changed; stale pulse action cancelled");
    }
    for (auto& candidate : candidates) {
        IedServer_updateAttributeValue(server.handle, candidate.first, candidate.second.get());
        ++server.revisions[candidate.first];
    }
    Json result = Json::object();
    for (const auto& item : updates) {
        auto reference = item.at("ref").get<std::string>();
        result[reference] = value_json(server, server.attribute(reference));
    }
    auto revision = guarded ? server.revisions[guarded] : 0;
    IedServer_unlockDataModel(server.handle);
    return guarded ? Json{{"values", result}, {"revision", revision}} : result;
}

void update_value(Server& server, const std::string& reference, const Json& value) {
    update_batch(server, Json::array({{{"ref", reference}, {"value", value}}}));
}
