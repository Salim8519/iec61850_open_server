# libxml2 2.11.9 configured with the feature set of the libxml2 inside the lxml 6.0.2
# Windows wheel. Reference: lxml/includes/libxml/xmlversion.h shipped in that wheel
# (the header generated when lxml's libxml2 was built). check_features.cmake compares
# the header generated here with STUDIO_XSD_EXPECTED_FEATURES and stops on any drift.

set(BUILD_SHARED_LIBS OFF CACHE BOOL "" FORCE)

# Features that change parsing or validation text are listed first.
set(STUDIO_XSD_LIBXML2_OPTIONS
    ICONV=ON        # encodings beyond UTF-8/16/Latin-1 go through GNU libiconv
    ICU=OFF
    ISO8859X=OFF    # unused when iconv is on; off in the wheel
    ZLIB=ON         # memory parsing never decompresses; kept for an identical build
    LZMA=OFF
    SCHEMAS=ON
    REGEXPS=ON      # XSD pattern facets
    UNICODE=ON
    AUTOMATA=ON
    PATTERN=ON
    VALID=ON
    SAX1=ON
    PUSH=ON
    TREE=ON
    OUTPUT=ON
    LEGACY=ON
    THREADS=ON
    THREAD_ALLOC=OFF
    HTTP=ON
    FTP=ON
    HTML=ON
    C14N=ON
    CATALOG=ON
    XPATH=ON
    XPTR=ON
    XPTR_LOCS=OFF
    XINCLUDE=ON
    READER=ON
    WRITER=ON
    SCHEMATRON=ON
    MODULES=ON
    DEBUG=ON
    MEM_DEBUG=OFF
    RUN_DEBUG=OFF
    EXPR=OFF
    TRIO=OFF
    # Nothing else is built: the helper needs only the library.
    PROGRAMS=OFF
    TESTS=OFF
    PYTHON=OFF)

foreach(option IN LISTS STUDIO_XSD_LIBXML2_OPTIONS)
    string(REPLACE "=" ";" pair "${option}")
    list(GET pair 0 name)
    list(GET pair 1 value)
    set(LIBXML2_WITH_${name} ${value} CACHE BOOL "" FORCE)
endforeach()

# Macro -> 1 (defined) / 0 (not defined) in lxml 6.0.2's xmlversion.h.
set(STUDIO_XSD_EXPECTED_FEATURES
    LIBXML_THREAD_ENABLED=1 LIBXML_THREAD_ALLOC_ENABLED=0 LIBXML_TREE_ENABLED=1
    LIBXML_OUTPUT_ENABLED=1 LIBXML_PUSH_ENABLED=1 LIBXML_READER_ENABLED=1
    LIBXML_PATTERN_ENABLED=1 LIBXML_WRITER_ENABLED=1 LIBXML_SAX1_ENABLED=1
    LIBXML_FTP_ENABLED=1 LIBXML_HTTP_ENABLED=1 LIBXML_VALID_ENABLED=1
    LIBXML_HTML_ENABLED=1 LIBXML_LEGACY_ENABLED=1 LIBXML_C14N_ENABLED=1
    LIBXML_CATALOG_ENABLED=1 LIBXML_XPATH_ENABLED=1 LIBXML_XPTR_ENABLED=1
    LIBXML_XPTR_LOCS_ENABLED=0 LIBXML_XINCLUDE_ENABLED=1 LIBXML_ICONV_ENABLED=1
    LIBXML_ICU_ENABLED=0 LIBXML_ISO8859X_ENABLED=0 LIBXML_DEBUG_ENABLED=1
    LIBXML_DEBUG_RUNTIME=0 LIBXML_UNICODE_ENABLED=1 LIBXML_REGEXP_ENABLED=1
    LIBXML_AUTOMATA_ENABLED=1 LIBXML_EXPR_ENABLED=0 LIBXML_SCHEMAS_ENABLED=1
    LIBXML_SCHEMATRON_ENABLED=1 LIBXML_MODULES_ENABLED=1 LIBXML_ZLIB_ENABLED=1
    LIBXML_LZMA_ENABLED=0)
