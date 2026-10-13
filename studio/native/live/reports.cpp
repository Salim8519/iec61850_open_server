// Use free report control blocks of an external IED and release them again.
#include "reports.hpp"
#include "mms_json.hpp"
#include <stdexcept>

namespace {
bool owner_empty(MmsValue* owner) {
    if (!owner || MmsValue_getType(owner) != MMS_OCTET_STRING) return true;
    auto size = MmsValue_getOctetStringSize(owner);
    auto data = MmsValue_getOctetStringBuffer(owner);
    for (int i = 0; i < size; ++i) if (data[i]) return false;
    return true;
}

void received(void* parameter, ClientReport report) {
    auto& use = *static_cast<ReportUse*>(parameter);
    auto values = ClientReport_getDataSetValues(report);
    if (!values) return;
    std::lock_guard<std::mutex> guard(use.lock);
    Json flat = Json::object();
    Json reasons = Json::object();
    int count = MmsValue_getArraySize(values);
    for (int index = 0; index < count && index < int(use.members.size()); ++index) {
        auto reason = ClientReport_getReasonForInclusion(report, index);
        if (reason == IEC61850_REASON_NOT_INCLUDED) continue;
        mms_flatten(MmsValue_getElement(values, index), use.member_specs[index], use.members[index], flat);
        reasons[use.members[index]] = int(reason);
    }
    emit_frame({{"event", "live_report"}, {"rcb", use.rcb}, {"values", flat}, {"reasons", reasons},
                {"has_timestamp", ClientReport_hasTimestamp(report)},
                {"source_timestamp", ClientReport_hasTimestamp(report) ? ClientReport_getTimestamp(report) : 0}});
}
}

ReportAccess::~ReportAccess() { release_all(); }

Json ReportAccess::status_of(const std::string& rcb) {
    IedClientError error = IED_ERROR_OK;
    auto block = IedConnection_getRCBValues(session.handle(), &error, rcb.c_str(), nullptr);
    if (!block) return {{"rcb", rcb}, {"readable", false}, {"error", int(error)}, {"free", false}};
    bool buffered = ClientReportControlBlock_isBuffered(block);
    bool enabled = ClientReportControlBlock_getRptEna(block);
    bool reserved = !buffered && ClientReportControlBlock_getResv(block);
    int reserve_time = buffered ? ClientReportControlBlock_getResvTms(block) : 0;
    bool owned = !owner_empty(ClientReportControlBlock_getOwner(block));
    auto dataset = ClientReportControlBlock_getDataSetReference(block);
    Json result = {{"rcb", rcb}, {"readable", true}, {"buffered", buffered}, {"enabled", enabled},
        {"reserved", reserved || reserve_time > 0}, {"owned", owned},
        {"dataset", dataset ? dataset : ""}, {"conf_rev", ClientReportControlBlock_getConfRev(block)},
        {"free", !enabled && !reserved && reserve_time <= 0 && !owned}};
    ClientReportControlBlock_destroy(block);
    return result;
}

Json ReportAccess::status(const Json& request) {
    std::lock_guard<std::mutex> guard(session.io_lock());
    Json result = Json::array();
    for (const auto& rcb : request.at("rcbs")) result.push_back(status_of(rcb.get<std::string>()));
    return {{"blocks", result}};
}

Json ReportAccess::reserve(const Json& request) {
    auto rcb = request.at("rcb").get<std::string>();
    std::lock_guard<std::mutex> guard(session.io_lock());
    if (uses.count(rcb)) throw std::runtime_error("Report block already in use by Studio");
    auto state = status_of(rcb);
    if (!state["free"].get<bool>()) return {{"reserved", false}, {"status", state}};
    auto use = std::make_unique<ReportUse>();
    use->rcb = rcb;
    use->dataset = state["dataset"].get<std::string>();
    auto dataset_ref = use->dataset;
    auto dollar = dataset_ref.find('$');
    if (dollar != std::string::npos) dataset_ref[dollar] = '.';
    IedClientError error = IED_ERROR_OK;
    auto members = IedConnection_getDataSetDirectory(session.handle(), &error, dataset_ref.c_str(), nullptr);
    if (error != IED_ERROR_OK || !members) throw std::runtime_error("Could not read report dataset " + dataset_ref);
    for (auto item = LinkedList_getNext(members); item; item = LinkedList_getNext(item)) {
        std::string entry = static_cast<const char*>(item->data);
        auto bracket = entry.rfind('[');
        std::string ref = entry.substr(0, bracket);
        std::string fc = bracket == std::string::npos ? "ST" : entry.substr(bracket + 1, 2);
        MmsVariableSpecification* spec = nullptr;
        IedClientError spec_error = IED_ERROR_OK;
        auto raw = IedConnection_getVariableSpecification(session.handle(), &spec_error, ref.c_str(),
                                                          FunctionalConstraint_fromString(fc.c_str()));
        if (spec_error == IED_ERROR_OK) spec = raw;
        use->members.push_back(ref);
        use->member_specs.push_back(spec);
    }
    LinkedList_destroy(members);
    auto block = IedConnection_getRCBValues(session.handle(), &error, rcb.c_str(), nullptr);
    if (!block) throw std::runtime_error("Report block disappeared while reserving");
    IedConnection_installReportHandler(session.handle(), rcb.c_str(),
        ClientReportControlBlock_getRptId(block), received, use.get());
    if (!ClientReportControlBlock_isBuffered(block)) {
        ClientReportControlBlock_setResv(block, true);
        IedConnection_setRCBValues(session.handle(), &error, block, RCB_ELEMENT_RESV, true);
    }
    ClientReportControlBlock_setRptEna(block, true);
    IedConnection_setRCBValues(session.handle(), &error, block, RCB_ELEMENT_RPT_ENA, true);
    bool enabled = error == IED_ERROR_OK;
    if (enabled) {
        ClientReportControlBlock_setGI(block, true);
        IedConnection_setRCBValues(session.handle(), &error, block, RCB_ELEMENT_GI, true);
    }
    ClientReportControlBlock_destroy(block);
    if (!enabled) {
        IedConnection_uninstallReportHandler(session.handle(), rcb.c_str());
        throw std::runtime_error("The IED refused to enable report block " + rcb);
    }
    Json result = {{"reserved", true}, {"rcb", rcb}, {"members", use->members}};
    uses[rcb] = std::move(use);
    return result;
}

Json ReportAccess::release(const Json& request) {
    auto rcb = request.at("rcb").get<std::string>();
    std::lock_guard<std::mutex> guard(session.io_lock());
    auto found = uses.find(rcb);
    if (found == uses.end()) return {{"released", false}};
    IedClientError error = IED_ERROR_OK;
    auto block = IedConnection_getRCBValues(session.handle(), &error, rcb.c_str(), nullptr);
    if (block) {
        ClientReportControlBlock_setRptEna(block, false);
        IedConnection_setRCBValues(session.handle(), &error, block, RCB_ELEMENT_RPT_ENA, true);
        if (!ClientReportControlBlock_isBuffered(block)) {
            ClientReportControlBlock_setResv(block, false);
            IedConnection_setRCBValues(session.handle(), &error, block, RCB_ELEMENT_RESV, true);
        }
        ClientReportControlBlock_destroy(block);
    }
    IedConnection_uninstallReportHandler(session.handle(), rcb.c_str());
    for (auto spec : found->second->member_specs) if (spec) MmsVariableSpecification_destroy(spec);
    uses.erase(found);
    return {{"released", error == IED_ERROR_OK}, {"rcb", rcb}};
}

void ReportAccess::release_all() {
    std::vector<std::string> names;
    for (const auto& entry : uses) names.push_back(entry.first);
    for (const auto& name : names) {
        try { release({{"rcb", name}}); } catch (...) {}
    }
}
