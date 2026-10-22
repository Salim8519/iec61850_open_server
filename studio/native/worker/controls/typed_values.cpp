// Typed command values (ADR-0104): what a ctlVal means for each class and how the simulated
// IED's feedback follows it. INC/ENC/ISC: the status takes the value. APC: mxVal takes it.
// BSC: Tcmd higher/lower moves valWTr.posVal one step within its limits. BAC: Tcmd moves
// mxVal by stepSize within its limits.
#include "typed_binding.hpp"
#include <cmath>

namespace {
// ctlVal as a number; APC ctlVal arrives as the AnalogueValue structure.
bool numeric(MmsValue* value, double& out, bool& is_float) {
    if (!value) return false;
    if (MmsValue_getType(value) == MMS_STRUCTURE) {
        if (MmsValue_getArraySize(value) < 1) return false;
        for (int i = 0; i < int(MmsValue_getArraySize(value)); ++i) {
            auto element = MmsValue_getElement(value, i);
            if (element && MmsValue_getType(element) == MMS_FLOAT) {
                out = MmsValue_toFloat(element); is_float = true; return true;
            }
        }
        return false;
    }
    if (MmsValue_getType(value) == MMS_INTEGER) { out = double(MmsValue_toInt64(value)); is_float = false; return true; }
    if (MmsValue_getType(value) == MMS_FLOAT) { out = MmsValue_toFloat(value); is_float = true; return true; }
    return false;
}

double current_value(TypedBinding& binding) {
    auto value = IedServer_getAttributeValue(binding.server->handle, binding.feedback);
    if (!value) return 0;
    if (MmsValue_getType(value) == MMS_FLOAT) return MmsValue_toFloat(value);
    return double(MmsValue_toInt64(value));
}

bool in_limits(const TypedBinding& binding, double target) {
    return !binding.has_limits || (target >= binding.minimum && target <= binding.maximum);
}

TypedPlan refuse(TypedPlan plan, int cause) {
    plan.valid = false;
    plan.add_cause = cause;
    return plan;
}
}

TypedPlan plan_typed(TypedBinding& binding, MmsValue* ctl_value) {
    TypedPlan plan;
    if (!ctl_value) return refuse(plan, ADD_CAUSE_INCONSISTENT_PARAMETERS);
    if (binding.value_kind == "step") {
        if (MmsValue_getType(ctl_value) != MMS_BIT_STRING) return refuse(plan, ADD_CAUSE_INCONSISTENT_PARAMETERS);
        int command = int(Dbpos_fromMmsValue(ctl_value));  // Tcmd: 0 stop, 1 lower, 2 higher, 3 reserved
        static const char* names[] = {"stop", "lower", "higher", "reserved"};
        plan.ctl_value = names[command & 3];
        plan.action = names[command & 3];
        double now = current_value(binding);
        if (command == 0) { plan.target = now; plan.valid = true; return plan; }
        if (command == 3) return refuse(plan, ADD_CAUSE_INCONSISTENT_PARAMETERS);
        plan.target = now + (command == 2 ? binding.step : -binding.step);
        if (!in_limits(binding, plan.target)) return refuse(plan, ADD_CAUSE_STEP_LIMIT);
        plan.valid = true;
        return plan;
    }
    double number = 0;
    bool is_float = false;
    if (!numeric(ctl_value, number, is_float)) return refuse(plan, ADD_CAUSE_INCONSISTENT_PARAMETERS);
    bool wants_float = binding.value_kind == "float";
    if (wants_float != is_float) return refuse(plan, ADD_CAUSE_INCONSISTENT_PARAMETERS);
    plan.action = "set";
    plan.target = number;
    plan.ctl_value = is_float ? Json(number) : Json(static_cast<long long>(number));
    if (binding.value_kind == "enum") {
        bool known = false;
        for (auto ord : binding.allowed) known = known || double(ord) == number;
        if (!known) return refuse(plan, ADD_CAUSE_INCONSISTENT_PARAMETERS);
    } else if (!std::isfinite(number) || !in_limits(binding, number)) {
        return refuse(plan, ADD_CAUSE_INCONSISTENT_PARAMETERS);
    }
    plan.valid = true;
    return plan;
}

Json apply_typed(TypedBinding& binding, const TypedPlan& plan, bool& changed, uint64_t now) {
    auto server = binding.server->handle;
    auto current = IedServer_getAttributeValue(server, binding.feedback);
    bool is_float = current && MmsValue_getType(current) == MMS_FLOAT;
    changed = current_value(binding) != plan.target;
    if (changed) {
        if (is_float) IedServer_updateFloatAttributeValue(server, binding.feedback, float(plan.target));
        else IedServer_updateInt32AttributeValue(server, binding.feedback, int32_t(plan.target));
        if (binding.timestamp) IedServer_updateUTCTimeAttributeValue(server, binding.timestamp, now);
        ++binding.server->revisions[binding.feedback];
    }
    // The simulated drive reaches the new position at once: never left "in transition".
    if (binding.trans_ind) IedServer_updateBooleanAttributeValue(server, binding.trans_ind, false);
    if (is_float) return Json(double(float(plan.target)));
    return Json(static_cast<long long>(plan.target));
}
