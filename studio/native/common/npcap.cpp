// Locate the user-installed Npcap runtime and map adapters to pcap device indexes.
#include "npcap.hpp"
#include <algorithm>
#include <cctype>
#include <mutex>
#include <stdexcept>
#ifdef _WIN32
#include <winsock2.h>
#include <windows.h>
#define HAVE_REMOTE
#include <pcap.h>
#endif

namespace {
using Json = nlohmann::json;
std::once_flag load_once;
bool loaded = false;
std::string failure = "Npcap is not installed";

void load() {
#ifdef _WIN32
    wchar_t system[MAX_PATH];
    UINT length = GetSystemDirectoryW(system, MAX_PATH);
    if (!length || length >= MAX_PATH) { failure = "Windows system folder unavailable"; return; }
    std::wstring folder = std::wstring(system) + L"\\Npcap";
    std::wstring library = folder + L"\\wpcap.dll";
    if (GetFileAttributesW(library.c_str()) == INVALID_FILE_ATTRIBUTES) return;
    SetDllDirectoryW(folder.c_str());
    HMODULE handle = LoadLibraryW(library.c_str());
    if (!handle) { failure = "Npcap is installed but could not be loaded"; return; }
    loaded = true;
    failure.clear();
#endif
}

std::string upper(std::string text) {
    std::transform(text.begin(), text.end(), text.begin(), [](unsigned char c) { return char(std::toupper(c)); });
    return text;
}
}

bool npcap_available() {
    std::call_once(load_once, load);
    return loaded;
}

Json npcap_status() {
    bool available = npcap_available();
    Json result = {{"available", available}, {"reason", failure}};
#ifdef _WIN32
    if (available) result["version"] = pcap_lib_version();
#endif
    return result;
}

Json npcap_adapters() {
    Json result = Json::array();
#ifdef _WIN32
    if (!npcap_available()) return result;
    char error[PCAP_ERRBUF_SIZE];
    pcap_if_t* devices = nullptr;
    if (pcap_findalldevs_ex(const_cast<char*>(PCAP_SRC_IF_STRING), nullptr, &devices, error) == -1)
        throw std::runtime_error(std::string("Npcap adapter list failed: ") + error);
    int index = 0;
    for (auto device = devices; device; device = device->next, ++index) {
        std::string name = device->name;
        auto open = name.find('{');
        auto close = name.find('}', open == std::string::npos ? 0 : open);
        std::string guid = open != std::string::npos && close != std::string::npos ?
            upper(name.substr(open, close - open + 1)) : "";
        result.push_back({{"index", index}, {"name", name}, {"guid", guid},
                          {"description", device->description ? device->description : ""}});
    }
    pcap_freealldevs(devices);
#endif
    return result;
}

std::string npcap_interface_for(const std::string& adapter_guid) {
    auto wanted = upper(adapter_guid);
    for (const auto& adapter : npcap_adapters()) {
        if (adapter["guid"] == wanted) return std::to_string(adapter["index"].get<int>());
    }
    throw std::runtime_error("The selected network adapter is not visible to Npcap");
}
