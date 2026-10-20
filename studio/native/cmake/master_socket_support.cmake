# SCADA master connect (speed lane N3): lib60870's Windows socket HAL is compiled unmodified
# under master/socket_60870.c, which replaces only Socket_connect so that a refused TCP connect
# fails at once instead of after the whole IEC 104 t0 (ADR-0006 pattern, vendor tree untouched).
# Only studio_master and the test masters connect; the gateway's slave sockets are unchanged.
# A lib60870 upgrade stops configuration here until the replacement is reviewed again.
if(NOT WIN32)
    return()
endif()
set(master_socket_file "${LIB60870_SOURCE}/src/hal/socket/win32/socket_win32.c")
file(SHA256 "${master_socket_file}" master_socket_hash)
if(NOT master_socket_hash STREQUAL "bc134a207c83972560b626e991bdaa0ad8fb335665ea09c4ccf8fc6c1064976e")
    message(FATAL_ERROR "Pinned lib60870 socket_win32.c changed; review studio/native/master/socket_60870.c")
endif()
foreach(lib60870_target lib60870 lib60870-shared)
    get_target_property(lib60870_sources ${lib60870_target} SOURCES)
    list(FILTER lib60870_sources EXCLUDE REGEX "hal/socket/win32/socket_win32\\.c$")
    list(APPEND lib60870_sources "${CMAKE_CURRENT_SOURCE_DIR}/master/socket_60870.c")
    set_property(TARGET ${lib60870_target} PROPERTY SOURCES "${lib60870_sources}")
    target_include_directories(${lib60870_target} PRIVATE "${LIB60870_SOURCE}/src/hal/socket/win32")
endforeach()
