#pragma once
#include "server/server.hpp"
#include <atomic>
#include <string>
#include <vector>

// One typed control of a simulated IED (ADR-0104): INC, ENC, APC, BAC, BSC or ISC. The
// coordinator sends it under the start request's optional key "extended_controls"; SPC/DPC
// stay in ControlBinding unchanged.
struct TypedBinding {
    Server* server = nullptr;
    std::string reference;
    std::string status;      // feedback leaf: stVal, valWTr.posVal or mxVal.f
    std::string cdc;
    std::string value_kind;  // int, enum, float or step
    DataAttribute* feedback = nullptr;
    DataAttribute* trans_ind = nullptr;
    DataAttribute* timestamp = nullptr;
    int model = 0;
    int sbo_timeout_ms = 0;
    bool has_limits = false;
    double minimum = 0, maximum = 0, step = 1;
    std::vector<long long> allowed;  // ENC: the ords of the enumeration
    std::atomic<bool> blocked{false};
    std::atomic<bool> interlock_blocked{false};  // refuses when the client asks for interlock check
    std::atomic<bool> synchro_blocked{false};    // refuses when the client asks for synchro check
};

// What one command would do: the new feedback value, or why the IED refuses it.
struct TypedPlan {
    bool valid = false;
    int add_cause = 0;      // ControlAddCause when not valid
    std::string action;     // set, higher, lower or stop
    double target = 0;      // new feedback value (integer kinds hold whole numbers)
    Json ctl_value;         // the value the client sent, for the event log
};

TypedPlan plan_typed(TypedBinding& binding, MmsValue* ctl_value);
// Writes the plan into the model and returns the feedback value as the event reports it.
Json apply_typed(TypedBinding& binding, const TypedPlan& plan, bool& changed, uint64_t now);
void install_typed_control(Server& server, const Json& config);
void set_typed_checks(Server& server, const Json& request);
