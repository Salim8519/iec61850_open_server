cmake_minimum_required(VERSION 3.15)
project(SubstationStudioNative LANGUAGES C CXX)
set(CMAKE_CXX_STANDARD 17)
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
add_executable(studio_ied worker/main.cpp worker/ipc/framing.cpp
    worker/server/server.cpp worker/updates/values.cpp worker/controls/controls.cpp
    worker/goose/publishing.cpp common/npcap.cpp)
target_include_directories(studio_ied PRIVATE ${STUDIO_HEADERS} worker)
add_executable(studio_live live/main.cpp live/goose_traffic.cpp live/mms_session.cpp
    live/reports.cpp live/sv_traffic.cpp live/sv_header.cpp worker/ipc/framing.cpp
    common/npcap.cpp common/mms_json.cpp
    common/browse.cpp common/type_tree.cpp)
target_include_directories(studio_live PRIVATE ${STUDIO_HEADERS} worker live)
add_executable(studio_probe probe/main.cpp probe/reports.cpp common/browse.cpp
    common/type_tree.cpp probe/sbo_controls.cpp)
target_include_directories(studio_probe PRIVATE ${STUDIO_HEADERS})
# Test-only Sampled Values publisher (never packaged): the only target with SV publishing code.
add_executable(studio_sv_test_publisher testtools/sv_test_publisher.cpp worker/ipc/framing.cpp
    common/npcap.cpp "${IEC_SOURCE}/src/sampled_values/sv_publisher.c")
target_include_directories(studio_sv_test_publisher PRIVATE ${STUDIO_HEADERS} worker)
target_include_directories(studio_sv_test_publisher PRIVATE "${IEC_SOURCE}/src/sampled_values"
    "${IEC_SOURCE}/src/mms/inc_private")
foreach(studio_target studio_ied studio_live studio_probe studio_sv_test_publisher)
    target_compile_definitions(${studio_target} PRIVATE _CRT_SECURE_NO_WARNINGS NOMINMAX)
    target_link_libraries(${studio_target} PRIVATE iec61850)
    if(WIN32)
        target_link_libraries(${studio_target} PRIVATE ws2_32 iphlpapi bcrypt)
        studio_delay_load_npcap(${studio_target})
    endif()
endforeach()
