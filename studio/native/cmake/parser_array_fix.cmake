# Apply a narrow correction to a generated copy, retaining upstream source/license.
set(parser_original "${IEC_SOURCE}/src/iec61850/server/model/config_file_parser.c")
file(SHA256 "${parser_original}" parser_sha)
if(NOT parser_sha STREQUAL "79c2ccb95dd0d8bc09690acffbb50cbce16a6b4bbff07c5414e1a5b941c0409c")
    message(FATAL_ERROR "Pinned config parser changed; review the top-level array correction")
endif()
file(READ "${parser_original}" parser_text)
set(parser_marker "(ModelNode*)DataObject_create(nameString, (ModelNode*)currentLN, arrayElements);")
string(REPLACE "${parser_marker}" "${parser_marker}
                        if (arrayElements > 0)
                        {
                            inArray = true;
                            currentArrayNode = currentModelNode;
                        }" parser_text "${parser_text}")
string(REPLACE "else if (indendation == 4)" "else if (indendation == 4 && !inArray)"
       parser_text "${parser_text}")
string(REPLACE "else if (indendation > 4)" "else if (indendation > 3)"
       parser_text "${parser_text}")
set(parser_generated "${CMAKE_CURRENT_BINARY_DIR}/generated/config_file_parser.c")
file(WRITE "${parser_generated}" "${parser_text}")
foreach(parser_target iec61850 iec61850-shared)
    get_target_property(parser_sources ${parser_target} SOURCES)
    list(FILTER parser_sources EXCLUDE REGEX "config_file_parser\\.c$")
    list(APPEND parser_sources "${parser_generated}")
    set_property(TARGET ${parser_target} PROPERTY SOURCES "${parser_sources}")
endforeach()
