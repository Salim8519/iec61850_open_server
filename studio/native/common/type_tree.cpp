// Expand server-returned MMS type descriptions, including array elements.
#include "type_tree.hpp"
#include <memory>
#include <set>
#include <stdexcept>
extern "C" {
#include "mms_type_spec.h"
}

namespace {
using Json = nlohmann::json;

void leaves(MmsVariableSpecification* specification, const std::string& reference,
            const std::string& fc, Json& output, int depth) {
    if (!specification || depth > 64 || output.size() >= 100000)
        throw std::runtime_error("MMS type tree exceeds probe limits or contains a missing specification");
    auto type = MmsVariableSpecification_getType(specification);
    if (type == MMS_STRUCTURE) {
        auto count = MmsVariableSpecification_getSize(specification);
        for (int index = 0; index < count; ++index) {
            auto child = MmsVariableSpecification_getChildSpecificationByIndex(specification, index);
            auto name = child ? MmsVariableSpecification_getName(child) : nullptr;
            if (!name) throw std::runtime_error("Unnamed MMS structure member");
            leaves(child, reference + "." + name, fc, output, depth + 1);
        }
    } else if (type == MMS_ARRAY) {
        auto count = MmsVariableSpecification_getSize(specification);
        if (count < 0 || count > 10000) throw std::runtime_error("MMS array exceeds probe limit");
        auto element = MmsVariableSpecification_getArrayElementSpecification(specification);
        for (int index = 0; index < count; ++index)
            leaves(element, reference + "(" + std::to_string(index) + ")", fc, output, depth + 1);
    } else {
        output.push_back({{"ref", reference}, {"fc", fc}, {"mms_type", int(type)}});
    }
}
}

nlohmann::json discover_leaves(IedConnection connection, const std::string& reference,
                              const std::vector<std::string>& variables) {
    std::set<std::string> constraints;
    for (const auto& variable : variables) constraints.insert(variable.substr(0, variable.find('$')));
    Json output = Json::array();
    for (const auto& constraint : constraints) {
        auto fc = FunctionalConstraint_fromString(constraint.c_str());
        if (fc == IEC61850_FC_NONE) continue;
        IedClientError error = IED_ERROR_OK;
        auto raw = IedConnection_getVariableSpecification(connection, &error, reference.c_str(), fc);
        std::unique_ptr<MmsVariableSpecification, decltype(&MmsVariableSpecification_destroy)>
            specification(raw, MmsVariableSpecification_destroy);
        if (error != IED_ERROR_OK || !raw)
            throw std::runtime_error(reference + "[" + constraint + "]: type discovery error "
                                     + std::to_string(int(error)));
        leaves(raw, reference, constraint, output, 0);
    }
    return output;
}
