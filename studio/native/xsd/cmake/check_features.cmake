# Stops the configure step when the libxml2 just configured is not 2.11.9 or its
# feature switches differ from lxml 6.0.2's (STUDIO_XSD_EXPECTED_FEATURES).

set(studio_xsd_header "")
foreach(candidate
        "${CMAKE_CURRENT_BINARY_DIR}/libxml2/libxml/xmlversion.h"
        "${CMAKE_CURRENT_BINARY_DIR}/libxml2/include/libxml/xmlversion.h")
    if(EXISTS "${candidate}")
        set(studio_xsd_header "${candidate}")
        break()
    endif()
endforeach()
if(NOT studio_xsd_header)
    file(GLOB_RECURSE studio_xsd_found "${CMAKE_CURRENT_BINARY_DIR}/libxml2/*xmlversion.h")
    list(LENGTH studio_xsd_found studio_xsd_count)
    if(studio_xsd_count EQUAL 0)
        message(FATAL_ERROR "studio_xsd: the generated libxml/xmlversion.h was not found")
    endif()
    list(GET studio_xsd_found 0 studio_xsd_header)
endif()

file(READ "${studio_xsd_header}" studio_xsd_text)
if(NOT studio_xsd_text MATCHES "#define LIBXML_DOTTED_VERSION \"2\\.11\\.9\"")
    message(FATAL_ERROR "studio_xsd: LIBXML2_SOURCE is not libxml2 2.11.9 (${studio_xsd_header}). "
        "The helper must use the libxml2 release bundled by lxml 6.0.2.")
endif()

set(studio_xsd_drift "")
foreach(feature IN LISTS STUDIO_XSD_EXPECTED_FEATURES)
    string(REPLACE "=" ";" pair "${feature}")
    list(GET pair 0 macro)
    list(GET pair 1 expected)
    # The generated header writes "#if 1" or "#if 0" right before "#define <macro>".
    if(studio_xsd_text MATCHES "#if ([01])[ \t\r]*\n[ \t]*#define ${macro}[ \t\r\n]")
        set(actual "${CMAKE_MATCH_1}")
    else()
        set(actual "missing")
    endif()
    if(NOT actual STREQUAL expected)
        list(APPEND studio_xsd_drift "${macro}: expected ${expected}, configured ${actual}")
    endif()
endforeach()
if(studio_xsd_drift)
    string(REPLACE ";" "\n  " studio_xsd_lines "${studio_xsd_drift}")
    message(FATAL_ERROR "studio_xsd: libxml2 features differ from lxml 6.0.2:\n  ${studio_xsd_lines}")
endif()
message(STATUS "studio_xsd: libxml2 2.11.9 with lxml 6.0.2 feature set (${studio_xsd_header})")
