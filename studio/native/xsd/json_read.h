/* Reads one request frame: a JSON object whose interesting members are flat strings
 * and integers. Unknown members of any type are skipped. */
#ifndef STUDIO_XSD_JSON_READ_H
#define STUDIO_XSD_JSON_READ_H

#include <stddef.h>

typedef struct {
    char *id;          /* request id, echoed in the reply ("" when absent) */
    char *op;          /* "hello", "validate" or "stop" */
    char *schema_path; /* validate: absolute path of the bundled SCL.xsd (UTF-8) */
    char *source_path; /* validate: absolute path of the SCL bytes to validate (UTF-8) */
    long long version; /* protocol version, must be 1 */
} xsd_request;

/* Returns 0 on success; on failure returns -1 with a static English message in
 * `error`. The request must be released with xsd_request_free in both cases. */
int xsd_request_parse(const char *text, size_t size, xsd_request *request, const char **error);
void xsd_request_free(xsd_request *request);

#endif
