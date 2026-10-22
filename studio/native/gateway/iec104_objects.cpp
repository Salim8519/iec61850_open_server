// Gateway point -> IEC 104 information object, with IEC 61850 quality mapped to 104 flags.
#include "iec104_objects.hpp"
#include <algorithm>
#include <chrono>
#include <cmath>
#include <cstdio>

namespace {
// IEC 61850 detail bits (libiec61850 order).
constexpr int OVERFLOW_BIT = 4, OLD_DATA = 128, SUBSTITUTED = 1024, OPERATOR_BLOCKED = 4096;

uint64_t stamp(const GatewayPoint& point) {
    if (point.time_ms > 0) return uint64_t(point.time_ms);
    return uint64_t(std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count());
}
}

QualityDescriptor iec104_quality(const GatewayPoint& point) {
    int flags = IEC60870_QUALITY_GOOD;
    if (quality_invalid(point)) flags |= IEC60870_QUALITY_INVALID;
    if (!point.source_ok || quality_questionable(point) || (point.quality & OLD_DATA))
        flags |= IEC60870_QUALITY_NON_TOPICAL;
    if (point.quality & SUBSTITUTED) flags |= IEC60870_QUALITY_SUBSTITUTED;
    if (point.quality & OPERATOR_BLOCKED) flags |= IEC60870_QUALITY_BLOCKED;
    if ((point.quality & OVERFLOW_BIT) && (point.kind == "float" || point.kind == "integer" || point.kind == "step_position"))
        flags |= IEC60870_QUALITY_OVERFLOW;
    return QualityDescriptor(flags);
}

InformationObject iec104_object(const GatewayPoint& point, bool with_time) {
    auto quality = iec104_quality(point);
    sCP56Time2a time;
    CP56Time2a_createFromMsTimestamp(&time, stamp(point));
    if (point.kind == "single") {
        bool on = point.value != 0.0;
        return with_time ? (InformationObject)SinglePointWithCP56Time2a_create(nullptr, point.address, on, quality, &time)
                         : (InformationObject)SinglePointInformation_create(nullptr, point.address, on, quality);
    }
    if (point.kind == "double") {
        auto state = DoublePointValue(std::clamp(int(point.value), 0, 3));
        return with_time ? (InformationObject)DoublePointWithCP56Time2a_create(nullptr, point.address, state, quality, &time)
                         : (InformationObject)DoublePointInformation_create(nullptr, point.address, state, quality);
    }
    if (point.kind == "step_position") {  // M_ST: the tap position, -64..63 (ADR-0107)
        int step = int(std::clamp(std::lround(point.value * point.scale), -64L, 63L));
        return with_time ? (InformationObject)StepPositionWithCP56Time2a_create(nullptr, point.address, step, false,
                                                                               quality, &time)
                         : (InformationObject)StepPositionInformation_create(nullptr, point.address, step, false, quality);
    }
    if (point.kind == "integer") {
        int scaled = int(std::clamp(std::lround(point.value * point.scale), -32768L, 32767L));
        return with_time ? (InformationObject)MeasuredValueScaledWithCP56Time2a_create(nullptr, point.address, scaled,
                                                                                      quality, &time)
                         : (InformationObject)MeasuredValueScaled_create(nullptr, point.address, scaled, quality);
    }
    float value = float(point.value * point.scale);
    return with_time ? (InformationObject)MeasuredValueShortWithCP56Time2a_create(nullptr, point.address, value,
                                                                                 quality, &time)
                     : (InformationObject)MeasuredValueShort_create(nullptr, point.address, value, quality);
}

std::string iec104_value_text(const GatewayPoint& point) {
    static const char* DPI[] = {"intermediate", "OFF", "ON", "bad state"};
    std::string text;
    if (point.kind == "single") text = point.value != 0.0 ? "ON" : "OFF";
    else if (point.kind == "double") text = DPI[std::clamp(int(point.value), 0, 3)];
    else {
        char buffer[48];
        std::snprintf(buffer, sizeof(buffer), "%.6g", point.value * point.scale);
        text = buffer;
    }
    if (quality_invalid(point)) text += " [IV]";
    else if (iec104_quality(point) != IEC60870_QUALITY_GOOD) text += " [quality flags]";
    return text;
}

const char* iec104_type_name(int type_id) {
    switch (type_id) {
    case M_SP_NA_1: return "M_SP_NA_1"; case M_SP_TB_1: return "M_SP_TB_1";
    case M_DP_NA_1: return "M_DP_NA_1"; case M_DP_TB_1: return "M_DP_TB_1";
    case M_ME_NB_1: return "M_ME_NB_1"; case M_ME_TE_1: return "M_ME_TE_1";
    case M_ME_NC_1: return "M_ME_NC_1"; case M_ME_TF_1: return "M_ME_TF_1";
    case C_SC_NA_1: return "C_SC_NA_1"; case C_SC_TA_1: return "C_SC_TA_1";
    case C_DC_NA_1: return "C_DC_NA_1"; case C_DC_TA_1: return "C_DC_TA_1";
    case M_ST_NA_1: return "M_ST_NA_1"; case M_ST_TB_1: return "M_ST_TB_1";
    case C_RC_NA_1: return "C_RC_NA_1"; case C_RC_TA_1: return "C_RC_TA_1";
    case C_SE_NA_1: return "C_SE_NA_1"; case C_SE_TA_1: return "C_SE_TA_1";
    case C_SE_NB_1: return "C_SE_NB_1"; case C_SE_TB_1: return "C_SE_TB_1";
    case C_SE_NC_1: return "C_SE_NC_1"; case C_SE_TC_1: return "C_SE_TC_1";
    case C_IC_NA_1: return "C_IC_NA_1"; case C_CS_NA_1: return "C_CS_NA_1";
    case C_RD_NA_1: return "C_RD_NA_1"; case C_TS_TA_1: return "C_TS_TA_1";
    default: return "other";
    }
}
