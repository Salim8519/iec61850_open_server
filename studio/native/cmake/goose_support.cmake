# Add the unmodified upstream GOOSE and Windows Ethernet sources to the targets.
# Upstream only enables them when an SDK sits inside its own source tree; Studio
# keeps the vendor tree untouched (ADR-0006) and injects the pinned SDK instead.
set(goose_sources
    "${IEC_SOURCE}/src/goose/goose_subscriber.c"
    "${IEC_SOURCE}/src/goose/goose_receiver.c"
    "${IEC_SOURCE}/src/goose/goose_publisher.c")
foreach(goose_target iec61850 iec61850-shared)
    target_sources(${goose_target} PRIVATE ${goose_sources})
    get_target_property(goose_defs ${goose_target} COMPILE_DEFINITIONS)
    if(goose_defs)
        list(REMOVE_ITEM goose_defs EXCLUDE_ETHERNET_WINDOWS)
        set_property(TARGET ${goose_target} PROPERTY COMPILE_DEFINITIONS "${goose_defs}")
    endif()
    target_include_directories(${goose_target} PRIVATE "${NPCAP_SDK}/Include")
endforeach()
get_property(goose_dir_defs DIRECTORY "${IEC_SOURCE}/src" PROPERTY COMPILE_DEFINITIONS)
list(REMOVE_ITEM goose_dir_defs EXCLUDE_ETHERNET_WINDOWS)
set_property(DIRECTORY "${IEC_SOURCE}/src" PROPERTY COMPILE_DEFINITIONS "${goose_dir_defs}")
# Studio's Ethernet HAL (ADR-0073): ethernet_dispatch.c routes "udp:" interface ids to
# the loopback GOOSE engine backend and numeric ids to the vendor Npcap HAL, which
# ethernet_pcap.c compiles unmodified under Pcap_* names.
set(studio_hal_sources
    "${CMAKE_CURRENT_SOURCE_DIR}/hal/ethernet_dispatch.c"
    "${CMAKE_CURRENT_SOURCE_DIR}/hal/ethernet_pcap.c"
    "${CMAKE_CURRENT_SOURCE_DIR}/hal/ethernet_udp.c")
foreach(hal_target hal hal-shared)
    target_sources(${hal_target} PRIVATE ${studio_hal_sources})
    target_include_directories(${hal_target} PRIVATE "${NPCAP_SDK}/Include"
        "${IEC_SOURCE}/hal/ethernet/win32" "${CMAKE_CURRENT_SOURCE_DIR}/hal")
    target_link_libraries(${hal_target} "${NPCAP_SDK}/Lib/x64/wpcap.lib" "${NPCAP_SDK}/Lib/x64/Packet.lib")
endforeach()

# Npcap is installed by the user, never bundled. Delay-loading lets every
# executable start without it; the worker probes it before enabling GOOSE.
function(studio_delay_load_npcap target)
    target_link_libraries(${target} PRIVATE delayimp)
    target_link_options(${target} PRIVATE "/DELAYLOAD:wpcap.dll" "/DELAYLOAD:Packet.dll")
endfunction()
message(STATUS "Substation Studio: pinned Npcap SDK injected; layer-2 GOOSE is compiled in "
               "(the upstream 'winpcap not found' line above refers only to its in-tree slot)")
