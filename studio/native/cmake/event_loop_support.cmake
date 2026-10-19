# Event-driven simulated IED server (workstream G; proposed ADR in
# docs/architecture/go-engine/streams/g-native.md). The MMS server stays single-threaded
# (CONFIG_MMS_SINGLE_THREADED, vendor default); studio_ied drives it with its own loop.
# The vendor tree stays untouched (ADR-0006): socket_studio.c compiles the vendor socket HAL
# under renamed symbols, and the worker reads private scheduling fields of the pinned commit.
# Any change in these files must be reviewed against the loop before the hashes are updated.
set(event_loop_pins
    "hal/socket/win32/socket_win32.c=de8f63edc9e377f6ce6116edf373ecc5300bf3368b6cb2d23358b1bb4b472ea4"
    "src/iec61850/inc_private/reporting.h=edf71ab5a36e36d7175961921b5b428ac201329c324f40a50a2c5a20718c77bd"
    "src/iec61850/inc_private/mms_mapping_internal.h=b30bcce2838dbda7c80bafc942c1a0071d9d38011576041f9f96f9f6027ef51d"
    "src/iec61850/inc_private/ied_server_private.h=c597e620b6ced018e9eb3817b12710d47bc2cb6244203142354e6a823d76e3a2")
foreach(event_loop_pin ${event_loop_pins})
    string(REPLACE "=" ";" event_loop_pair "${event_loop_pin}")
    list(GET event_loop_pair 0 event_loop_file)
    list(GET event_loop_pair 1 event_loop_hash)
    file(SHA256 "${IEC_SOURCE}/${event_loop_file}" event_loop_actual)
    if(NOT event_loop_actual STREQUAL event_loop_hash)
        message(FATAL_ERROR "Pinned ${event_loop_file} changed; review the studio_ied event loop")
    endif()
endforeach()
if(NOT CONFIG_MMS_SINGLE_THREADED OR CONFIG_MMS_THREADLESS_STACK)
    message(FATAL_ERROR "studio_ied's event loop needs the single-threaded MMS server build")
endif()
foreach(hal_target hal hal-shared)
    get_target_property(hal_sources ${hal_target} SOURCES)
    list(FILTER hal_sources EXCLUDE REGEX "socket_win32\\.c$")
    list(APPEND hal_sources "${CMAKE_CURRENT_SOURCE_DIR}/hal/socket_studio.c")
    set_property(TARGET ${hal_target} PROPERTY SOURCES "${hal_sources}")
    target_include_directories(${hal_target} PRIVATE "${IEC_SOURCE}/hal/socket/win32")
endforeach()
