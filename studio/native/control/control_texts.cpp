// Plain-English names of control models and AddCause values (IEC 61850-7-2).
#include "control_session.hpp"

const char* control_model_name(int model) {
    switch (model) {
    case CONTROL_MODEL_STATUS_ONLY: return "status-only";
    case CONTROL_MODEL_DIRECT_NORMAL: return "direct-with-normal-security";
    case CONTROL_MODEL_SBO_NORMAL: return "sbo-with-normal-security";
    case CONTROL_MODEL_DIRECT_ENHANCED: return "direct-with-enhanced-security";
    case CONTROL_MODEL_SBO_ENHANCED: return "sbo-with-enhanced-security";
    default: return "unknown";
    }
}

const char* add_cause_text(int cause) {
    static const char* texts[] = {
        "unknown", "not supported", "blocked by switching hierarchy (local/remote)", "select failed",
        "invalid position", "position already reached", "parameter change in execution", "step limit",
        "blocked by mode (Beh/Mod)", "blocked by process", "blocked by interlocking", "blocked by synchrocheck",
        "command already in execution", "blocked by health", "1-of-n control", "aborted by cancel",
        "time limit over", "aborted by trip", "object not selected", "object already selected",
        "no access authority", "ended with overshoot", "aborted due to deviation", "aborted by communication loss",
        "aborted by command", "none", "inconsistent parameters", "locked by another client"};
    return cause >= 0 && cause < int(sizeof(texts) / sizeof(texts[0])) ? texts[cause] : "unknown";
}
