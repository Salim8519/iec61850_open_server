# Static zlib 1.3.1 and GNU libiconv 1.17 for studio_xsd.exe (MSVC x64), built out of tree
# from the unmodified source trees in .local/native. Neither upstream build fits:
# - zlib's own CMakeLists renames zconf.h inside the source tree on out-of-tree builds;
# - GNU libiconv has no MSVC project (configure + make only).
# So both are compiled here from their library sources. `cmake --install` puts
#   include/zlib.h, include/zconf.h, include/iconv.h, lib/zlib.lib, lib/libiconv.lib
# under CMAKE_INSTALL_PREFIX, the folder studio_xsd's ZLIB_*/Iconv_* variables point to.
# Steps and licence notes: studio/native/xsd/README.md.
cmake_minimum_required(VERSION 3.18)
project(StudioXsdDeps LANGUAGES C)

set(_native "${CMAKE_CURRENT_SOURCE_DIR}/../../../../../.local/native")
set(ZLIB_SOURCE "${_native}/zlib-1.3.1" CACHE PATH "zlib 1.3.1 source tree (unmodified)")
set(LIBICONV_SOURCE "${_native}/libiconv-1.17" CACHE PATH "GNU libiconv 1.17 source tree (unmodified)")

if(NOT EXISTS "${ZLIB_SOURCE}/zlib.h" OR NOT EXISTS "${ZLIB_SOURCE}/inflate.c")
    message(FATAL_ERROR "studio_xsd deps: ZLIB_SOURCE (${ZLIB_SOURCE}) is not a zlib source tree")
endif()
file(READ "${ZLIB_SOURCE}/zlib.h" _zlib_header)
if(NOT _zlib_header MATCHES "#define ZLIB_VERSION \"1\\.3\\.1\"")
    message(FATAL_ERROR "studio_xsd deps: ZLIB_SOURCE is not zlib 1.3.1 (the release in lxml 6.0.2)")
endif()
if(NOT EXISTS "${LIBICONV_SOURCE}/lib/iconv.c" OR NOT EXISTS "${LIBICONV_SOURCE}/include/iconv.h.build.in")
    message(FATAL_ERROR "studio_xsd deps: LIBICONV_SOURCE (${LIBICONV_SOURCE}) is not a GNU libiconv source tree")
endif()
file(READ "${LIBICONV_SOURCE}/include/iconv.h.build.in" _iconv_header)
if(NOT _iconv_header MATCHES "#define _LIBICONV_VERSION 0x0111")
    message(FATAL_ERROR "studio_xsd deps: LIBICONV_SOURCE is not GNU libiconv 1.17")
endif()

set(BUILD_SHARED_LIBS OFF)
if(MSVC)
    # Same C runtime as libxml2 and studio_xsd (the CMake default, /MD).
    add_compile_definitions(_CRT_SECURE_NO_DEPRECATE _CRT_NONSTDC_NO_DEPRECATE)
endif()

# zlib: the library objects of zlib's own CMakeLists (zlib_SRCS), with the shipped zconf.h.
add_library(zlib STATIC
    "${ZLIB_SOURCE}/adler32.c" "${ZLIB_SOURCE}/compress.c" "${ZLIB_SOURCE}/crc32.c"
    "${ZLIB_SOURCE}/deflate.c" "${ZLIB_SOURCE}/gzclose.c" "${ZLIB_SOURCE}/gzlib.c"
    "${ZLIB_SOURCE}/gzread.c" "${ZLIB_SOURCE}/gzwrite.c" "${ZLIB_SOURCE}/inflate.c"
    "${ZLIB_SOURCE}/infback.c" "${ZLIB_SOURCE}/inftrees.c" "${ZLIB_SOURCE}/inffast.c"
    "${ZLIB_SOURCE}/trees.c" "${ZLIB_SOURCE}/uncompr.c" "${ZLIB_SOURCE}/zutil.c")
target_include_directories(zlib PUBLIC "${ZLIB_SOURCE}")

# libiconv: the public headers come from the *.build.in templates, filled the way
# configure fills them for a static Windows build (no visibility, no dllimport).
set(HAVE_VISIBILITY 0)
set(DLL_VARIABLE "")
set(EILSEQ "")          # <errno.h> defines EILSEQ; the fallback is never used
set(ICONV_CONST "")
set(USE_MBSTATE_T 1)
set(BROKEN_WCHAR_H 0)
set(HAVE_WCHAR_T 1)
set(_generated "${CMAKE_CURRENT_BINARY_DIR}/libiconv-include")
configure_file("${LIBICONV_SOURCE}/include/iconv.h.build.in" "${_generated}/iconv.h" @ONLY)
configure_file("${LIBICONV_SOURCE}/libcharset/include/localcharset.h.build.in"
    "${_generated}/localcharset.h" @ONLY)
configure_file("${CMAKE_CURRENT_SOURCE_DIR}/libiconv_config.h" "${_generated}/config.h" COPYONLY)

add_library(libiconv STATIC
    "${LIBICONV_SOURCE}/lib/iconv.c"
    "${LIBICONV_SOURCE}/libcharset/lib/localcharset.c")
target_include_directories(libiconv
    PRIVATE "${_generated}" "${LIBICONV_SOURCE}/lib"
    INTERFACE "${_generated}")
# DEFS of lib/Makefile.in that still apply to a static library (no BUILDING_LIBICONV:
# that would mark every function dllexport; no ENABLE_RELOCATABLE: DLL-only).
target_compile_definitions(libiconv PRIVATE IN_LIBRARY NO_XMALLOC)
set_target_properties(libiconv PROPERTIES OUTPUT_NAME libiconv)

install(TARGETS zlib libiconv ARCHIVE DESTINATION lib)
install(FILES "${ZLIB_SOURCE}/zlib.h" "${ZLIB_SOURCE}/zconf.h" "${_generated}/iconv.h"
    DESTINATION include)
