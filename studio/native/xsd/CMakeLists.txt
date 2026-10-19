# studio_xsd.exe: SCL XSD validation with libxml2 2.11.9, the release lxml 6.0.2 bundles
# (ADR-0074, go-engine architecture section 1). A separate project from studio/native so
# the IEC workers never depend on libxml2. Build steps: README.md in this folder.
cmake_minimum_required(VERSION 3.18)
project(StudioXsd LANGUAGES C)

set(LIBXML2_SOURCE "${CMAKE_CURRENT_SOURCE_DIR}/../../../.local/native/libxml2-2.11.9"
    CACHE PATH "Pinned libxml2 2.11.9 source tree (libxml2-2.11.9.tar.xz, unmodified)")
set(STUDIO_XSD_BUILD "dev" CACHE STRING "Build id reported by the hello operation")

if(NOT EXISTS "${LIBXML2_SOURCE}/xmlschemas.c" OR NOT EXISTS "${LIBXML2_SOURCE}/CMakeLists.txt")
    message(FATAL_ERROR "studio_xsd: LIBXML2_SOURCE (${LIBXML2_SOURCE}) is not a libxml2 source tree. "
        "Unpack libxml2-2.11.9.tar.xz and pass -DLIBXML2_SOURCE=<folder>.")
endif()

# Same switches as the libxml2 inside the lxml 6.0.2 wheel; GNU libiconv and zlib are
# found through the standard Iconv_INCLUDE_DIR/Iconv_LIBRARY and
# ZLIB_INCLUDE_DIR/ZLIB_LIBRARY cache variables.
include(cmake/lxml_features.cmake)
add_subdirectory("${LIBXML2_SOURCE}" libxml2 EXCLUDE_FROM_ALL)
include(cmake/check_features.cmake)

add_executable(studio_xsd
    main.c
    framing.c
    json_read.c
    json_write.c
    file_io.c
    error_log.c
    syntax_error.c
    document.c
    schema_cache.c
    validate.c)
if(TARGET LibXml2::LibXml2)
    target_link_libraries(studio_xsd PRIVATE LibXml2::LibXml2)
else()
    target_link_libraries(studio_xsd PRIVATE LibXml2)
endif()
target_compile_definitions(studio_xsd PRIVATE
    LIBXML_STATIC
    _CRT_SECURE_NO_WARNINGS
    STUDIO_XSD_BUILD="${STUDIO_XSD_BUILD}")
if(MSVC)
    target_compile_options(studio_xsd PRIVATE /W4 /utf-8)
else()
    target_compile_options(studio_xsd PRIVATE -Wall -Wextra)
endif()
