# Constant-time lookups in the MMS value cache (ADR-0133), applied to a generated copy like the
# parser correction (ADR-0006); the vendor tree stays untouched. The vendor cache keeps its
# linked-list map (it still owns keys and entries, and destroys them); every lookup asks
# studio/native/libfix/value_cache_index.c first, which returns the same entry, and a lookup
# that found a cache entry reuses that entry's stored type instead of scanning the domain again
# (the entry's typeSpec was found with the same name when it was inserted).
# Any change in mms_value_cache.c must be reviewed before the hash is updated.
set(value_cache_original "${IEC_SOURCE}/src/mms/iso_mms/server/mms_value_cache.c")
file(SHA256 "${value_cache_original}" value_cache_sha)
if(NOT value_cache_sha STREQUAL "15bf0012aafb2bcb0214430ab15ab5ccfae4799950145a0d6e4c59ab11d46ad4")
    message(FATAL_ERROR "Pinned mms_value_cache.c changed; review the value cache index (ADR-0133)")
endif()
file(READ "${value_cache_original}" value_cache_text)
string(REPLACE "\r\n" "\n" value_cache_text "${value_cache_text}")  # the pinned file uses CRLF

# Replace text exactly `expected` times, or stop the configuration.
function(value_cache_replace search replacement expected)
    string(FIND "${value_cache_text}" "${search}" first)
    set(text "${value_cache_text}")
    set(count 0)
    while(NOT first EQUAL -1)
        math(EXPR count "${count} + 1")
        string(LENGTH "${search}" search_length)
        math(EXPR rest_start "${first} + ${search_length}")
        string(SUBSTRING "${text}" ${rest_start} -1 text)
        string(FIND "${text}" "${search}" first)
    endwhile()
    if(NOT count EQUAL expected)
        message(FATAL_ERROR "value cache index: '${search}' found ${count} times, expected ${expected}")
    endif()
    string(REPLACE "${search}" "${replacement}" value_cache_text "${value_cache_text}")
    set(value_cache_text "${value_cache_text}" PARENT_SCOPE)
endfunction()

value_cache_replace("#include \"stack_config.h\""
    "#include \"stack_config.h\"\n#include \"value_cache_index.h\"\n#define STUDIO_CACHE_GET(cache, key) (StudioCacheIndex_ready((cache)->index) ? StudioCacheIndex_get((cache)->index, (key)) : Map_getEntry((cache)->map, (key)))" 1)
value_cache_replace("    Map map;\n};" "    Map map;\n    StudioCacheIndex index;\n};" 1)
value_cache_replace("self->map = StringMap_create();"
    "self->map = StringMap_create();\n        self->index = StudioCacheIndex_create();" 1)
value_cache_replace("Map_addEntry(self->map, StringUtils_copyString(itemId), cacheEntry);"
    "char* key = StringUtils_copyString(itemId);\n        Map_addEntry(self->map, key, cacheEntry);\n        StudioCacheIndex_add(self->index, key, cacheEntry);" 1)
value_cache_replace("Map_getEntry(self->map, " "STUDIO_CACHE_GET(self, " 4)
value_cache_replace("MmsDomain_getNamedVariable(self->domain, parentId)" "cacheEntry->typeSpec" 2)
value_cache_replace("Map_deleteDeep(self->map, true,"
    "StudioCacheIndex_destroy(self->index);\n        Map_deleteDeep(self->map, true," 1)

set(value_cache_generated "${CMAKE_CURRENT_BINARY_DIR}/generated/mms_value_cache.c")
file(WRITE "${value_cache_generated}" "${value_cache_text}")
foreach(value_cache_target iec61850 iec61850-shared)
    get_target_property(value_cache_sources ${value_cache_target} SOURCES)
    list(FILTER value_cache_sources EXCLUDE REGEX "mms_value_cache\\.c$")
    list(APPEND value_cache_sources "${value_cache_generated}"
        "${CMAKE_CURRENT_SOURCE_DIR}/libfix/value_cache_index.c")
    set_property(TARGET ${value_cache_target} PROPERTY SOURCES "${value_cache_sources}")
    target_include_directories(${value_cache_target} PRIVATE "${CMAKE_CURRENT_SOURCE_DIR}/libfix"
        "${IEC_SOURCE}/src/mms/iso_mms/server")
endforeach()
