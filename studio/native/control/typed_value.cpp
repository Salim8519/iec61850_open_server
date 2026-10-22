// Typed ctlVal values (ADR-0104). The kind names what the user meant; the device's declared
// ctlVal type must agree, otherwise the command is refused before anything is sent.
#include "typed_session.hpp"
#include <cmath>
#include <stdexcept>

namespace {
long long whole(const Json& value) {
    if (value.is_boolean()) return value.get<bool>() ? 1 : 0;
    if (value.is_number_integer()) return value.get<long long>();
    if (value.is_number_float()) {
        double number = value.get<double>();
        if (std::floor(number) == number) return static_cast<long long>(number);
    }
    throw std::runtime_error("This command needs a whole number");
}

int tcmd(const Json& value) {
    if (value.is_string()) {
        auto text = value.get<std::string>();
        if (text == "stop") return 0;
        if (text == "lower") return 1;
        if (text == "higher") return 2;
    }
    throw std::runtime_error("A step command is higher, lower or stop");
}
}

MmsValue* typed_control_value(ControlObjectClient control, const Json& value, const std::string& kind) {
    auto type = ControlObjectClient_getCtlValType(control);
    if (kind == "bool") {
        if (type != MMS_BOOLEAN || !value.is_boolean())
            throw std::runtime_error("This control does not take on/off values");
        return MmsValue_newBoolean(value.get<bool>());
    }
    if (kind == "int" || kind == "enum") {
        if (type != MMS_INTEGER) throw std::runtime_error("This control does not take whole numbers");
        auto number = whole(value);
        if (number < INT32_MIN || number > INT32_MAX) throw std::runtime_error("The number is too large for this control");
        return MmsValue_newIntegerFromInt32(int32_t(number));
    }
    if (kind == "float") {
        // APC: the client library wraps the number into the AnalogueValue structure.
        if (type != MMS_STRUCTURE && type != MMS_FLOAT) throw std::runtime_error("This control does not take analogue values");
        if (!value.is_number()) throw std::runtime_error("This command needs a number");
        return MmsValue_newFloat(value.get<float>());
    }
    if (kind == "step") {
        if (type != MMS_BIT_STRING) throw std::runtime_error("This control does not take higher/lower steps");
        return Dbpos_toMmsValue(nullptr, static_cast<Dbpos>(tcmd(value)));
    }
    throw std::runtime_error("Value kind must be bool, int, enum, float or step");
}

const char* typed_model_name(int model) {
    switch (model) {
    case CONTROL_MODEL_STATUS_ONLY: return "status-only";
    case CONTROL_MODEL_DIRECT_NORMAL: return "direct-normal";
    case CONTROL_MODEL_SBO_NORMAL: return "sbo-normal";
    case CONTROL_MODEL_DIRECT_ENHANCED: return "direct-enhanced";
    case CONTROL_MODEL_SBO_ENHANCED: return "sbo-enhanced";
    default: return "unknown";
    }
}
