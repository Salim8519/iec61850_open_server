// Observe genuine MMS reports; report callbacks never retain stack-owned values.
#include "reports.hpp"
#include <chrono>
#include <iostream>
#include <mutex>
#include <thread>

using Json = nlohmann::json;
struct Observations {
    std::mutex mutex;
    Json reports = Json::array();
};

static void received(void* parameter, ClientReport report) {
    auto& observations = *static_cast<Observations*>(parameter);
    auto values = ClientReport_getDataSetValues(report);
    Json entries = Json::array();
    for (int index = 0; index < MmsValue_getArraySize(values); ++index) {
        auto reason = ClientReport_getReasonForInclusion(report, index);
        if (!reason) continue;
        char buffer[1024];
        MmsValue_printToBuffer(MmsValue_getElement(values, index), buffer, sizeof(buffer));
        entries.push_back({{"index", index}, {"reason", reason}, {"value", buffer}});
    }
    std::lock_guard<std::mutex> lock(observations.mutex);
    observations.reports.push_back({{"entries", entries}});
}

Json receive_reports(IedConnection connection, const char* reference) {
    IedClientError error;
    auto rcb = IedConnection_getRCBValues(connection, &error, reference, nullptr);
    if (!rcb) return {{"ok", false}, {"error", int(error)}};
    Observations observations;
    IedConnection_installReportHandler(connection, reference,
        ClientReportControlBlock_getRptId(rcb), received, &observations);
    ClientReportControlBlock_setRptEna(rcb, true);
    IedConnection_setRCBValues(connection, &error, rcb, RCB_ELEMENT_RPT_ENA, true);
    bool enabled = error == IED_ERROR_OK;
    if (enabled) {
        ClientReportControlBlock_setGI(rcb, true);
        IedConnection_setRCBValues(connection, &error, rcb, RCB_ELEMENT_GI, true);
        std::cout << Json({{"ready", error == IED_ERROR_OK}}).dump() << std::endl;
        std::this_thread::sleep_for(std::chrono::milliseconds(1800));
        ClientReportControlBlock_setRptEna(rcb, false);
        IedClientError cleanup_error;
        IedConnection_setRCBValues(connection, &cleanup_error, rcb, RCB_ELEMENT_RPT_ENA, true);
    }
    IedConnection_uninstallReportHandler(connection, reference);
    ClientReportControlBlock_destroy(rcb);
    std::lock_guard<std::mutex> lock(observations.mutex);
    return {{"ok", enabled && error == IED_ERROR_OK}, {"reports", observations.reports}};
}
