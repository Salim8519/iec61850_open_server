#include "validate.h"

#include "document.h"
#include "error_log.h"
#include "file_io.h"
#include "schema_cache.h"

#include <stdlib.h>
#include <string.h>
#include <libxml/xmlschemas.h>

static char *copy_text(const char *text) {
    size_t size = strlen(text);
    char *copy = (char *)malloc(size + 1);
    if (copy) memcpy(copy, text, size + 1);
    return copy;
}

static void error_item(xsd_buf *out, long long line, const char *message) {
    xsd_buf_text(out, "{\"line\":");
    xsd_buf_long(out, line);
    xsd_buf_text(out, ",\"message\":");
    xsd_buf_json_string(out, message, strlen(message));
    xsd_buf_text(out, "}");
}

static void syntax_result(xsd_buf *out, const xsd_parse_result *parsed) {
    xsd_buf_text(out, "{\"phase\":\"parse\",\"doctype\":false,\"status\":\"invalid\","
                      "\"error_count\":1,\"errors\":[");
    error_item(out, parsed->syntax_line, parsed->syntax_message);
    xsd_buf_text(out, "]}");
}

/* XMLSchema.__call__: a fresh validation context per document, every structured
 * error recorded in order (warnings included, as lxml's error_log keeps them). */
static int schema_result(xsd_buf *out, xmlDocPtr doc, xmlSchemaPtr schema, char **error) {
    xmlSchemaValidCtxtPtr context = xmlSchemaNewValidCtxt(schema);
    xsd_log log;
    size_t index, shown;
    int status;
    if (!context) {
        *error = copy_text("Out of memory creating the XML Schema validation context");
        return -1;
    }
    xsd_log_init(&log);
    xmlSchemaSetValidStructuredErrors(context, xsd_log_receive, &log);
    status = xmlSchemaValidateDoc(context, doc);
    xmlSchemaFreeValidCtxt(context);
    if (status == -1 || log.failed) {
        xsd_log_free(&log);
        *error = copy_text(status == -1 ? "Internal error in XML Schema validation."
                                        : "Out of memory recording XML Schema errors");
        return -1;
    }
    xsd_buf_text(out, "{\"phase\":\"schema\",\"doctype\":false,\"status\":");
    xsd_buf_text(out, status == 0 ? "\"valid\"" : "\"invalid\"");
    xsd_buf_text(out, ",\"error_count\":");
    xsd_buf_long(out, (long long)log.count);
    xsd_buf_text(out, ",\"errors\":[");
    shown = log.count < XSD_MAX_ERRORS ? log.count : XSD_MAX_ERRORS;
    for (index = 0; index < shown; index++) {
        if (index) xsd_buf_text(out, ",");
        error_item(out, log.items[index].line, log.items[index].message);
    }
    xsd_buf_text(out, "]}");
    xsd_log_free(&log);
    return 0;
}

int xsd_validate(const char *source_path, const char *schema_path, xsd_buf *out, char **error) {
    char *source = NULL;
    size_t size = 0;
    const char *read_error = NULL;
    xsd_parse_result parsed;
    xmlSchemaPtr schema;
    int status = 0;
    *error = NULL;
    if (xsd_read_file(source_path, &source, &size, &read_error) < 0) {
        *error = copy_text(read_error);
        return -1;
    }
    status = xsd_parse_memory(source, size, &parsed);
    free(source);
    if (status < 0) {
        *error = copy_text("Out of memory parsing the SCL document");
        return -1;
    }
    if (!parsed.doc) {
        syntax_result(out, &parsed);
    } else if (parsed.doctype) {
        xsd_buf_text(out, "{\"phase\":\"parse\",\"doctype\":true}");
    } else {
        /* Python loads the schema only after a successful parse and DOCTYPE check. */
        schema = xsd_schema_get(schema_path, error);
        status = schema ? schema_result(out, parsed.doc, schema, error) : -1;
    }
    xsd_parse_result_free(&parsed);
    return status;
}
