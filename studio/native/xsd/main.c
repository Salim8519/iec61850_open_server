/* studio_xsd: validates SCL documents against the bundled IEC XSDs with the libxml2
 * release lxml 6.0.2 bundles (2.11.9), so the Go engine reports the same messages and
 * line numbers as the Python engine did. One request at a time over the native
 * framing; see README.md for the protocol. */
#include "document.h"
#include "framing.h"
#include "json_read.h"
#include "json_write.h"
#include "schema_cache.h"
#include "validate.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <libxml/globals.h>
#include <libxml/parser.h>
#include <libxml/xmlerror.h>
#include <libxml/xmlversion.h>

#ifndef STUDIO_XSD_BUILD
#define STUDIO_XSD_BUILD "dev"
#endif
#define XSD_PROTOCOL 1

/* lxml _initThreadLogging: generic libxml2 error text is dropped, never printed. */
static void drop_generic_error(void *context, const char *message, ...) {
    (void)context;
    (void)message;
}

static char *copy_text(const char *text) {
    char *copy = (char *)malloc(strlen(text) + 1);
    if (copy) strcpy(copy, text);
    return copy;
}

/* Runtime libxml2 version as "major.minor.patch" from xmlParserVersion ("21109"). */
static void runtime_version(char *text, size_t size) {
    long number = strtol(xmlParserVersion, NULL, 10);
    snprintf(text, size, "%ld.%ld.%ld", number / 10000, (number / 100) % 100, number % 100);
}

static void hello(xsd_buf *out) {
    char version[32];
    runtime_version(version, sizeof version);
    xsd_buf_text(out, "{\"protocol\":");
    xsd_buf_long(out, XSD_PROTOCOL);
    xsd_buf_text(out, ",\"capabilities\":[\"xsd_validate_v1\"],\"libxml2\":");
    xsd_buf_json_string(out, version, strlen(version));
    xsd_buf_text(out, ",\"libxml2_compiled\":\"" LIBXML_DOTTED_VERSION "\",\"build\":");
    xsd_buf_json_string(out, STUDIO_XSD_BUILD, strlen(STUDIO_XSD_BUILD));
    xsd_buf_text(out, ",\"parse_options\":");
    xsd_buf_long(out, XSD_PARSE_OPTIONS);
    xsd_buf_text(out, "}");
}

static int reply(const char *id, const char *result_json, const char *error) {
    xsd_buf out;
    char *payload;
    int status;
    xsd_buf_init(&out);
    xsd_buf_text(&out, "{\"id\":");
    xsd_buf_json_string(&out, id, strlen(id));
    if (error) {
        xsd_buf_text(&out, ",\"ok\":false,\"error\":");
        xsd_buf_json_string(&out, error, strlen(error));
    } else {
        xsd_buf_text(&out, ",\"ok\":true");
        if (result_json) {
            xsd_buf_text(&out, ",\"result\":");
            xsd_buf_text(&out, result_json);
        }
    }
    xsd_buf_text(&out, "}");
    payload = xsd_buf_take(&out);
    if (!payload) return -1;
    if (strlen(payload) > XSD_FRAME_LIMIT) {
        free(payload);
        return reply(id, NULL, "The validation result is larger than the 16 MiB frame limit");
    }
    status = xsd_frame_write(payload, strlen(payload));
    free(payload);
    return status;
}

/* Handles one request; returns 1 to continue, 0 after "stop", -1 on a broken pipe. */
static int handle(const xsd_request *request) {
    xsd_buf result;
    char *error = NULL, *body;
    int status;
    if (request->version != XSD_PROTOCOL) return reply(request->id, NULL, "Protocol version must be 1") < 0 ? -1 : 1;
    if (!request->op) return reply(request->id, NULL, "Missing operation") < 0 ? -1 : 1;
    if (!strcmp(request->op, "stop")) return reply(request->id, NULL, NULL) < 0 ? -1 : 0;
    xsd_buf_init(&result);
    if (!strcmp(request->op, "hello")) {
        hello(&result);
    } else if (!strcmp(request->op, "validate")) {
        if (!request->source_path || !request->schema_path)
            error = copy_text("validate needs source_path and schema_path");
        else
            xsd_validate(request->source_path, request->schema_path, &result, &error);
    } else {
        error = (char *)malloc(strlen(request->op) + 20);
        if (error) sprintf(error, "Unknown operation %s", request->op);
    }
    body = xsd_buf_take(&result);
    if (!body && !error) error = copy_text("Out of memory building the reply");
    status = reply(request->id, error ? NULL : body, error);
    free(body);
    free(error);
    return status < 0 ? -1 : 1;
}

int main(void) {
    int running = 1;
    LIBXML_TEST_VERSION
    xmlInitParser();
    xmlThrDefIndentTreeOutput(1);
#ifdef _MSC_VER
#pragma warning(push)
#pragma warning(disable : 4996) /* deprecated in 2.11; lxml's etree.pyx still calls it */
#endif
    xmlThrDefLineNumbersDefaultValue(1);
#ifdef _MSC_VER
#pragma warning(pop)
#endif
    xmlSetGenericErrorFunc(NULL, drop_generic_error);
    xmlSetStructuredErrorFunc(NULL, NULL);
    if (xsd_frame_init() != 0) {
        fprintf(stderr, "studio_xsd: unable to reserve the frame output\n");
        return 1;
    }
    while (running == 1) {
        char *payload = NULL;
        size_t size = 0;
        const char *error = NULL;
        xsd_request request;
        int status = xsd_frame_read(&payload, &size, &error);
        if (status == 0) break;
        if (status < 0) {
            fprintf(stderr, "studio_xsd: %s\n", error);
            xsd_schema_clear();
            return 1;
        }
        if (xsd_request_parse(payload, size, &request, &error) < 0)
            running = reply(request.id ? request.id : "", NULL, error) < 0 ? -1 : 1;
        else
            running = handle(&request);
        xsd_request_free(&request);
        free(payload);
    }
    xsd_schema_clear();
    xmlCleanupParser();
    return running < 0 ? 1 : 0;
}
