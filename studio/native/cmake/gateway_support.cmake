# Protocol libraries of the gateway simulator (ADR-0057), fetched and pinned by
# studio/tools/fetch_gateway_deps.py. The gateway targets never link libiec61850:
# lib60870 has its own HAL with the same symbol names.
set(LIB60870_SOURCE "${CMAKE_CURRENT_SOURCE_DIR}/../../.local/native/lib60870/lib60870-C" CACHE PATH "Pinned lib60870")
set(OPENDNP3_SOURCE "${CMAKE_CURRENT_SOURCE_DIR}/../../.local/native/opendnp3" CACHE PATH "Pinned opendnp3")
set(OPENDNP3_DEPS "${CMAKE_CURRENT_SOURCE_DIR}/../../.local/native/opendnp3-deps" CACHE PATH "Pinned opendnp3 deps")
foreach(required "${LIB60870_SOURCE}/src/inc/api/cs104_slave.h" "${OPENDNP3_SOURCE}/CMakeLists.txt"
                 "${OPENDNP3_DEPS}/asio/asio/include/asio.hpp")
    if(NOT EXISTS "${required}")
        message(FATAL_ERROR "Gateway libraries are missing (${required}). Run: python studio/tools/fetch_gateway_deps.py")
    endif()
endforeach()

set(BUILD_TESTS OFF CACHE BOOL "" FORCE)
add_subdirectory("${LIB60870_SOURCE}" lib60870 EXCLUDE_FROM_ALL)
set(LIB60870_HEADERS "${LIB60870_SOURCE}/config" "${LIB60870_SOURCE}/src/inc/api"
    "${LIB60870_SOURCE}/src/common/inc" "${LIB60870_SOURCE}/src/hal/inc")

set(DNP3_STATIC_LIBS ON CACHE BOOL "" FORCE)
set(DNP3_TLS OFF CACHE BOOL "" FORCE)
set(DNP3_TESTS OFF CACHE BOOL "" FORCE)
set(DNP3_EXAMPLES OFF CACHE BOOL "" FORCE)
set(FETCHCONTENT_FULLY_DISCONNECTED ON CACHE BOOL "" FORCE)
set(FETCHCONTENT_SOURCE_DIR_ASIO "${OPENDNP3_DEPS}/asio" CACHE PATH "" FORCE)
set(FETCHCONTENT_SOURCE_DIR_EXE4CPP "${OPENDNP3_DEPS}/exe4cpp" CACHE PATH "" FORCE)
set(FETCHCONTENT_SOURCE_DIR_SER4CPP "${OPENDNP3_DEPS}/ser4cpp" CACHE PATH "" FORCE)
add_subdirectory("${OPENDNP3_SOURCE}" opendnp3 EXCLUDE_FROM_ALL)
