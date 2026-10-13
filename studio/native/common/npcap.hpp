#pragma once
#include <nlohmann/json.hpp>
#include <string>

// Npcap is installed by the user. Every executable delay-loads it, so callers
// must ask npcap_status() before using any GOOSE or Ethernet function.
nlohmann::json npcap_status();
bool npcap_available();
// Resolve a Windows adapter GUID to the numeric interface ID libiec61850 expects.
std::string npcap_interface_for(const std::string& adapter_guid);
nlohmann::json npcap_adapters();
