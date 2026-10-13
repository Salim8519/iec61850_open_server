// Discover the model through MMS only; no engineering source/config is available here.
#include "browse.hpp"
#include "type_tree.hpp"
#include <stdexcept>
#include <string>
#include <vector>

namespace {
using Json = nlohmann::json;

std::vector<std::string> strings(LinkedList list, IedClientError error, const std::string& operation) {
    std::vector<std::string> result;
    if (list) {
        if (error == IED_ERROR_OK) {
            for (auto item = LinkedList_getNext(list); item; item = LinkedList_getNext(item))
                result.emplace_back(static_cast<const char*>(item->data));
        }
        LinkedList_destroy(list);
    }
    if (error != IED_ERROR_OK)
        throw std::runtime_error(operation + ": MMS client error " + std::to_string(int(error)));
    return result;
}

Json directory(IedConnection connection, const std::string& reference, ACSIClass type) {
    IedClientError error = IED_ERROR_OK;
    auto list = IedConnection_getLogicalNodeDirectory(connection, &error, reference.c_str(), type);
    return strings(list, error, reference);
}

Json dataset_members(IedConnection connection, const std::string& reference, const Json& names) {
    Json result = Json::object();
    for (const auto& name : names) {
        auto dataset = reference + "." + name.get<std::string>();
        bool deletable = false;
        IedClientError error = IED_ERROR_OK;
        auto list = IedConnection_getDataSetDirectory(connection, &error, dataset.c_str(), &deletable);
        result[name.get<std::string>()] = {{"members", strings(list, error, dataset)}, {"deletable", deletable}};
    }
    return result;
}
}

nlohmann::json browse_server(IedConnection connection) {
    try {
        IedClientError error = IED_ERROR_OK;
        auto list = IedConnection_getServerDirectory(connection, &error, false);
        auto devices = strings(list, error, "GetServerDirectory");
        Json output = {{"ok", true}, {"devices", Json::array()}};
        for (const auto& device : devices) {
            list = IedConnection_getLogicalDeviceDirectory(connection, &error, device.c_str());
            auto names = strings(list, error, device);
            Json entry = {{"name", device}, {"nodes", Json::array()}};
            for (const auto& name : names) {
                auto reference = device + "/" + name;
                list = IedConnection_getLogicalNodeVariables(connection, &error, reference.c_str());
                auto variables = strings(list, error, reference);
                auto datasets = directory(connection, reference, ACSI_CLASS_DATA_SET);
                entry["nodes"].push_back({{"name", name}, {"variables", variables},
                    {"leaves", discover_leaves(connection, reference, variables)},
                    {"objects", directory(connection, reference, ACSI_CLASS_DATA_OBJECT)},
                    {"datasets", datasets}, {"dataset_members", dataset_members(connection, reference, datasets)},
                    {"buffered_reports", directory(connection, reference, ACSI_CLASS_BRCB)},
                    {"unbuffered_reports", directory(connection, reference, ACSI_CLASS_URCB)}});
            }
            output["devices"].push_back(entry);
        }
        return output;
    } catch (const std::exception& error) {
        return {{"ok", false}, {"error", error.what()}};
    }
}
