cmake_minimum_required(VERSION 3.15)
project(SubstationStudioNative LANGUAGES C CXX)
set(CMAKE_CXX_STANDARD 17)
option(STUDIO_ENFORCE_LICENSE "Release builds only: workers refuse to run without a valid license" OFF)
set(STUDIO_RELEASE_KEY_DIR "${CMAKE_CURRENT_SOURCE_DIR}/release" CACHE PATH "Folder with release_key.hpp")
set(IEC_SOURCE "${CMAKE_CURRENT_SOURCE_DIR}/../../.local/native/libiec61850" CACHE PATH "Pinned IEC source")
set(JSON_SOURCE "${CMAKE_CURRENT_SOURCE_DIR}/../../.local/native/json" CACHE PATH "Pinned JSON source")
set(BUILD_EXAMPLES OFF CACHE BOOL "" FORCE)
set(BUILD_PYTHON_BINDINGS OFF CACHE BOOL "" FORCE)
include(cmake/goose_options.cmake)
add_subdirectory("${IEC_SOURCE}" libiec61850 EXCLUDE_FROM_ALL)
include(cmake/parser_array_fix.cmake)
include(cmake/goose_support.cmake)
include(cmake/sv_support.cmake)
set(STUDIO_HEADERS
    "${IEC_SOURCE}/src/iec61850/inc" "${IEC_SOURCE}/src/mms/inc"
    "${IEC_SOURCE}/src/common/inc" "${IEC_SOURCE}/hal/inc"
    "${IEC_SOURCE}/src/logging" "${IEC_SOURCE}/src/goose" "${IEC_SOURCE}/src/r_session"
    "${IEC_SOURCE}/src/sampled_values"
    "${CMAKE_CURRENT_BINARY_DIR}/libiec61850/config"
    "${JSON_SOURCE}/single_include" "${NPCAP_SDK}/Include" common)
# License guard: compiled into every build; enforcement exists only when the option is ON.
add_library(studio_license STATIC common/license_guard.cpp common/integrity.cpp
    third_party/monocypher/monocypher.c third_party/monocypher/monocypher-ed25519.c)
target_include_directories(studio_license PUBLIC common third_party/monocypher "${JSON_SOURCE}/single_include")
target_include_directories(studio_license PRIVATE "${STUDIO_RELEASE_KEY_DIR}")
# module_key.hpp (module-integrity public key) always lives here; added after the
# configurable release-key dir so release_key.hpp still resolves to the chosen key.
target_include_directories(studio_license PRIVATE "${CMAKE_CURRENT_SOURCE_DIR}/release")
target_compile_definitions(studio_license PRIVATE _CRT_SECURE_NO_WARNINGS NOMINMAX)
if(STUDIO_ENFORCE_LICENSE)
    target_compile_definitions(studio_license PRIVATE STUDIO_ENFORCE_LICENSE=1)
endif()
if(WIN32)
    target_link_libraries(studio_license PUBLIC bcrypt advapi32)
endif()
add_executable(studio_license_check testtools/license_check.cpp)
target_link_libraries(studio_license_check PRIVATE studio_license)
add_executable(studio_ied worker/main.cpp worker/ipc/framing.cpp
    worker/server/server.cpp worker/updates/values.cpp worker/controls/controls.cpp
    worker/goose/publishing.cpp common/npcap.cpp)
target_include_directories(studio_ied PRIVATE ${STUDIO_HEADERS} worker)
add_executable(studio_live live/main.cpp live/goose_traffic.cpp live/mms_session.cpp
    live/reports.cpp live/sv_traffic.cpp live/sv_header.cpp live/explore.cpp live/identify.cpp
    worker/ipc/framing.cpp
    common/npcap.cpp common/mms_json.cpp
    common/browse.cpp common/type_tree.cpp)
target_include_directories(studio_live PRIVATE ${STUDIO_HEADERS} worker live)
add_executable(studio_probe probe/main.cpp probe/reports.cpp common/browse.cpp
    common/type_tree.cpp probe/sbo_controls.cpp)
target_include_directories(studio_probe PRIVATE ${STUDIO_HEADERS})
# Control client of the Tools place (ADR-0057): the only tools program that operates IEDs.
add_executable(studio_control control/main.cpp control/control_session.cpp control/control_texts.cpp
    worker/ipc/framing.cpp)
target_include_directories(studio_control PRIVATE ${STUDIO_HEADERS} worker control)
# Test-only Sampled Values publisher (never packaged): the only target with SV publishing code.
add_executable(studio_sv_test_publisher testtools/sv_test_publisher.cpp worker/ipc/framing.cpp
    common/npcap.cpp "${IEC_SOURCE}/src/sampled_values/sv_publisher.c")
target_include_directories(studio_sv_test_publisher PRIVATE ${STUDIO_HEADERS} worker)
target_include_directories(studio_sv_test_publisher PRIVATE "${IEC_SOURCE}/src/sampled_values"
    "${IEC_SOURCE}/src/mms/inc_private")
foreach(studio_target studio_ied studio_live studio_probe studio_control studio_sv_test_publisher)
    target_compile_definitions(${studio_target} PRIVATE _CRT_SECURE_NO_WARNINGS NOMINMAX)
    target_link_libraries(${studio_target} PRIVATE iec61850 studio_license)
    if(WIN32)
        target_link_libraries(${studio_target} PRIVATE ws2_32 iphlpapi bcrypt)
        studio_delay_load_npcap(${studio_target})
    endif()
endforeach()
# Gateway simulator (ADR-0057): part of every build, like GOOSE; OFF only for diagnostics.
option(STUDIO_GATEWAY "Build the IEC 104 / Modbus TCP / DNP3 gateway simulator worker" ON)
if(STUDIO_GATEWAY)
    include(cmake/gateway_support.cmake)
    add_executable(studio_gateway gateway/main.cpp gateway/point_table.cpp gateway/command_bridge.cpp
        gateway/traffic.cpp gateway/iec104_objects.cpp gateway/iec104_commands.cpp gateway/server_104.cpp
        gateway/modbus_pdu.cpp gateway/modbus_server.cpp gateway/dnp3_outstation.cpp worker/ipc/framing.cpp)
    target_include_directories(studio_gateway PRIVATE gateway worker ${LIB60870_HEADERS} "${JSON_SOURCE}/single_include")
    target_compile_definitions(studio_gateway PRIVATE _CRT_SECURE_NO_WARNINGS NOMINMAX)
    target_link_libraries(studio_gateway PRIVATE lib60870 opendnp3 studio_license ws2_32)
    # Test-only independent masters (never packaged).
    add_executable(studio_test_iec104_master testtools/iec104_master.cpp)
    target_include_directories(studio_test_iec104_master PRIVATE ${LIB60870_HEADERS})
    target_link_libraries(studio_test_iec104_master PRIVATE lib60870 ws2_32)
    add_executable(studio_test_dnp3_master testtools/dnp3_master.cpp)
    target_link_libraries(studio_test_dnp3_master PRIVATE opendnp3)
endif()
