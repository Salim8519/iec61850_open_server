# Layer-2 GOOSE needs the Npcap SDK at build time. The SDK is a pinned build input
# (studio/tools/fetch_npcap_sdk.py); development and release builds must be equal,
# so a missing SDK stops configuration instead of silently omitting GOOSE.
set(NPCAP_SDK "${CMAKE_CURRENT_SOURCE_DIR}/../../.local/native/npcap-sdk" CACHE PATH "Pinned Npcap SDK")
if(NOT EXISTS "${NPCAP_SDK}/Lib/x64/wpcap.lib" OR NOT EXISTS "${NPCAP_SDK}/Include/pcap.h")
    message(FATAL_ERROR "Npcap SDK missing at ${NPCAP_SDK}. Run: python studio/tools/fetch_npcap_sdk.py")
endif()
set(CONFIG_INCLUDE_GOOSE_SUPPORT ON CACHE BOOL "" FORCE)
set(CONFIG_IEC61850_L2_GOOSE ON CACHE BOOL "" FORCE)
# Sampled Values are received only (ADR-0038); no product executable publishes SV.
set(CONFIG_IEC61850_L2_SMV ON CACHE BOOL "" FORCE)
set(CONFIG_IEC61850_R_GOOSE OFF CACHE BOOL "" FORCE)
set(CONFIG_IEC61850_R_SMV OFF CACHE BOOL "" FORCE)
