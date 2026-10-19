/* One "validate" request: the libxml2 half of scl/schema_validation.validate_schema. */
#ifndef STUDIO_XSD_VALIDATE_H
#define STUDIO_XSD_VALIDATE_H

#include "json_write.h"

/* Python keeps list(schema.error_log)[:100]. */
#define XSD_MAX_ERRORS 100

/* Appends the result object to `out`:
 *   {"phase":"parse","doctype":true}
 *       lxml parsed the bytes and the document has a DOCTYPE (Python then raises
 *       ValueError; the engine decides that, not the helper)
 *   {"phase":"parse","doctype":false,"status":"invalid","error_count":1,
 *    "errors":[{"line":L,"message":"<str(XMLSyntaxError)>"}]}
 *   {"phase":"schema","doctype":false,"status":"valid"|"invalid","error_count":N,
 *    "errors":[{"line":L,"message":"<libxml2 message>"}, ... at most 100]}
 * Returns 0, or -1 with a malloc'd English message in *error. */
int xsd_validate(const char *source_path, const char *schema_path, xsd_buf *out, char **error);

#endif
