// Typed JSON for values received from external IEDs or GOOSE frames.
#include "mms_json.hpp"
#include <stdexcept>

using Json = nlohmann::json;

Json mms_scalar_json(const MmsValue* value) {
    if (!value) return nullptr;
    auto mutable_value = const_cast<MmsValue*>(value);
    switch (MmsValue_getType(value)) {
    case MMS_BOOLEAN: return MmsValue_getBoolean(mutable_value);
    case MMS_INTEGER: return MmsValue_toInt64(mutable_value);
    case MMS_UNSIGNED: return MmsValue_toUint32(mutable_value);
    case MMS_FLOAT: return MmsValue_toDouble(mutable_value);
    case MMS_BIT_STRING:
        return {{"bits", MmsValue_getBitStringSize(mutable_value)},
                {"le", MmsValue_getBitStringAsInteger(mutable_value)},
                {"be", MmsValue_getBitStringAsIntegerBigEndian(mutable_value)}};
    case MMS_VISIBLE_STRING:
    case MMS_STRING: return std::string(MmsValue_toString(mutable_value));
    case MMS_UTC_TIME: return MmsValue_getUtcTimeInMs(mutable_value);
    case MMS_BINARY_TIME: return MmsValue_getBinaryTimeAsUtcMs(mutable_value);
    case MMS_OCTET_STRING: {
        std::string hex;
        static const char digits[] = "0123456789abcdef";
        auto size = MmsValue_getOctetStringSize(mutable_value);
        auto data = MmsValue_getOctetStringBuffer(mutable_value);
        for (int i = 0; i < size; ++i) { hex += digits[data[i] >> 4]; hex += digits[data[i] & 15]; }
        return {{"octets", hex}};
    }
    case MMS_DATA_ACCESS_ERROR:
        return {{"access_error", int(MmsValue_getDataAccessError(mutable_value))}};
    default: return {{"unsupported_mms_type", int(MmsValue_getType(value))}};
    }
}

void mms_flatten(const MmsValue* value, MmsVariableSpecification* spec,
                 const std::string& reference, Json& output, int depth) {
    if (!value || depth > 64 || output.size() > 100000) return;
    auto type = MmsValue_getType(value);
    auto mutable_value = const_cast<MmsValue*>(value);
    if (type == MMS_STRUCTURE || type == MMS_ARRAY) {
        int size = MmsValue_getArraySize(mutable_value);
        for (int index = 0; index < size; ++index) {
            MmsVariableSpecification* child = nullptr;
            std::string name;
            if (spec && type == MMS_STRUCTURE) {
                child = MmsVariableSpecification_getChildSpecificationByIndex(spec, index);
                if (child && MmsVariableSpecification_getName(child)) name = "." + std::string(MmsVariableSpecification_getName(child));
            } else if (spec && type == MMS_ARRAY) {
                child = MmsVariableSpecification_getArrayElementSpecification(spec);
            }
            if (name.empty()) name = type == MMS_ARRAY ? "(" + std::to_string(index) + ")" : "#" + std::to_string(index);
            mms_flatten(MmsValue_getElement(mutable_value, index), child, reference + name, output, depth + 1);
        }
        return;
    }
    output[reference] = mms_scalar_json(value);
}
