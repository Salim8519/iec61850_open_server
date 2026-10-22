// Decode IEC 60870-5-104 monitoring information objects received by ScadaBench as master.
#include "iec104_decode.hpp"

namespace {
uint64_t stamp(CP56Time2a time) { return time ? CP56Time2a_toMsTimestamp(time) : 0; }

struct Decoded { const char* group; double value; int quality; uint64_t time_ms; };

bool decode(int type, InformationObject object, Decoded& out) {
    switch (type) {
    case M_SP_NA_1: case M_SP_TB_1: {
        auto point = (SinglePointInformation)object;
        out = {"single", SinglePointInformation_getValue(point) ? 1.0 : 0.0, SinglePointInformation_getQuality(point),
               type == M_SP_TB_1 ? stamp(SinglePointWithCP56Time2a_getTimestamp((SinglePointWithCP56Time2a)object)) : 0};
        return true; }
    case M_DP_NA_1: case M_DP_TB_1: {
        auto point = (DoublePointInformation)object;
        out = {"double", double(DoublePointInformation_getValue(point)), DoublePointInformation_getQuality(point),
               type == M_DP_TB_1 ? stamp(DoublePointWithCP56Time2a_getTimestamp((DoublePointWithCP56Time2a)object)) : 0};
        return true; }
    case M_ST_NA_1: case M_ST_TB_1: {
        auto point = (StepPositionInformation)object;
        out = {"step", double(StepPositionInformation_getValue(point)), StepPositionInformation_getQuality(point),
               type == M_ST_TB_1 ? stamp(StepPositionWithCP56Time2a_getTimestamp((StepPositionWithCP56Time2a)object)) : 0};
        return true; }
    case M_BO_NA_1: case M_BO_TB_1: {
        auto point = (BitString32)object;
        out = {"bitstring", double(BitString32_getValue(point)), BitString32_getQuality(point),
               type == M_BO_TB_1 ? stamp(Bitstring32WithCP56Time2a_getTimestamp((Bitstring32WithCP56Time2a)object)) : 0};
        return true; }
    case M_ME_NA_1: case M_ME_TD_1: {
        auto point = (MeasuredValueNormalized)object;
        out = {"normalized", MeasuredValueNormalized_getValue(point), MeasuredValueNormalized_getQuality(point),
               type == M_ME_TD_1 ? stamp(MeasuredValueNormalizedWithCP56Time2a_getTimestamp(
                   (MeasuredValueNormalizedWithCP56Time2a)object)) : 0};
        return true; }
    case M_ME_NB_1: case M_ME_TE_1: {
        auto point = (MeasuredValueScaled)object;
        out = {"scaled", double(MeasuredValueScaled_getValue(point)), MeasuredValueScaled_getQuality(point),
               type == M_ME_TE_1 ? stamp(MeasuredValueScaledWithCP56Time2a_getTimestamp(
                   (MeasuredValueScaledWithCP56Time2a)object)) : 0};
        return true; }
    case M_ME_NC_1: case M_ME_TF_1: {
        auto point = (MeasuredValueShort)object;
        out = {"float", MeasuredValueShort_getValue(point), MeasuredValueShort_getQuality(point),
               type == M_ME_TF_1 ? stamp(MeasuredValueShortWithCP56Time2a_getTimestamp(
                   (MeasuredValueShortWithCP56Time2a)object)) : 0};
        return true; }
    case M_IT_NA_1: case M_IT_TB_1: {
        auto reading = IntegratedTotals_getBCR((IntegratedTotals)object);
        out = {"counter", double(BinaryCounterReading_getValue(reading)), BinaryCounterReading_isInvalid(reading) ? 0x80 : 0,
               type == M_IT_TB_1 ? stamp(IntegratedTotalsWithCP56Time2a_getTimestamp(
                   (IntegratedTotalsWithCP56Time2a)object)) : 0};
        return true; }
    default:
        return false;
    }
}
}

Json iec104_values(CS101_ASDU asdu) {
    Json values = Json::array();
    int type = CS101_ASDU_getTypeID(asdu);
    std::string cause = iec104_cause(CS101_ASDU_getCOT(asdu));
    for (int index = 0; index < CS101_ASDU_getNumberOfElements(asdu); ++index) {
        auto object = CS101_ASDU_getElement(asdu, index);
        if (!object) continue;
        Decoded decoded{};
        if (decode(type, object, decoded)) {
            values.push_back({{"group", decoded.group}, {"type", iec104_name(type)},
                              {"address", InformationObject_getObjectAddress(object)}, {"value", decoded.value},
                              {"quality", decoded.quality}, {"time_ms", decoded.time_ms}, {"cause", cause}});
        }
        InformationObject_destroy(object);
    }
    return values;
}

const char* iec104_name(int type_id) {
    switch (type_id) {
    case M_SP_NA_1: return "M_SP_NA_1"; case M_SP_TB_1: return "M_SP_TB_1";
    case M_DP_NA_1: return "M_DP_NA_1"; case M_DP_TB_1: return "M_DP_TB_1";
    case M_ST_NA_1: return "M_ST_NA_1"; case M_ST_TB_1: return "M_ST_TB_1";
    case M_BO_NA_1: return "M_BO_NA_1"; case M_BO_TB_1: return "M_BO_TB_1";
    case M_ME_NA_1: return "M_ME_NA_1"; case M_ME_TD_1: return "M_ME_TD_1";
    case M_ME_NB_1: return "M_ME_NB_1"; case M_ME_TE_1: return "M_ME_TE_1";
    case M_ME_NC_1: return "M_ME_NC_1"; case M_ME_TF_1: return "M_ME_TF_1";
    case M_IT_NA_1: return "M_IT_NA_1"; case M_IT_TB_1: return "M_IT_TB_1";
    case M_EI_NA_1: return "M_EI_NA_1";
    case C_SC_NA_1: return "C_SC_NA_1"; case C_DC_NA_1: return "C_DC_NA_1"; case C_RC_NA_1: return "C_RC_NA_1";
    case C_RC_TA_1: return "C_RC_TA_1";
    case C_SE_NA_1: return "C_SE_NA_1"; case C_SE_NB_1: return "C_SE_NB_1"; case C_SE_NC_1: return "C_SE_NC_1";
    case C_IC_NA_1: return "C_IC_NA_1"; case C_CI_NA_1: return "C_CI_NA_1"; case C_CS_NA_1: return "C_CS_NA_1";
    case C_SC_TA_1: return "C_SC_TA_1"; case C_DC_TA_1: return "C_DC_TA_1";
    case C_SE_TA_1: return "C_SE_TA_1"; case C_SE_TB_1: return "C_SE_TB_1"; case C_SE_TC_1: return "C_SE_TC_1";
    default: return "other type";
    }
}

std::string iec104_cause(int cot) {
    switch (cot) {
    case CS101_COT_PERIODIC: return "periodic";
    case CS101_COT_BACKGROUND_SCAN: return "background scan";
    case CS101_COT_SPONTANEOUS: return "spontaneous";
    case CS101_COT_INITIALIZED: return "initialised";
    case CS101_COT_REQUEST: return "requested";
    case CS101_COT_ACTIVATION: return "activation";
    case CS101_COT_ACTIVATION_CON: return "activation confirmed";
    case CS101_COT_DEACTIVATION: return "deactivation";
    case CS101_COT_DEACTIVATION_CON: return "deactivation confirmed";
    case CS101_COT_ACTIVATION_TERMINATION: return "activation terminated";
    case CS101_COT_RETURN_INFO_REMOTE: return "return (remote command)";
    case CS101_COT_RETURN_INFO_LOCAL: return "return (local command)";
    case CS101_COT_INTERROGATED_BY_STATION: return "interrogated";
    case CS101_COT_REQUESTED_BY_GENERAL_COUNTER: return "counter interrogated";
    case CS101_COT_UNKNOWN_TYPE_ID: return "unknown type";
    case CS101_COT_UNKNOWN_COT: return "unknown cause";
    case CS101_COT_UNKNOWN_CA: return "unknown common address";
    case CS101_COT_UNKNOWN_IOA: return "unknown object address";
    default: return "cause " + std::to_string(cot);
    }
}
