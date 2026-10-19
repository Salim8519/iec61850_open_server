/* Compiled bundled XSDs, loaded once per process like lxml's
 * functools.lru_cache(maxsize=3) around load_schema() in schema_validation.py. */
#ifndef STUDIO_XSD_SCHEMA_CACHE_H
#define STUDIO_XSD_SCHEMA_CACHE_H

#include <libxml/xmlschemas.h>

#define XSD_SCHEMA_SLOTS 3

/* Returns the compiled schema for the SCL.xsd at utf8_path, loading it on first use
 * (xsd:include/xsd:import resolve relative to that file). On failure returns NULL
 * and a malloc'd English message in *error. */
xmlSchemaPtr xsd_schema_get(const char *utf8_path, char **error);

/* Frees every cached schema (used before exit). */
void xsd_schema_clear(void);

#endif
