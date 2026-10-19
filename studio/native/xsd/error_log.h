/* Collects libxml2 structured errors the way lxml 6.0.2's _ErrorLog/_LogEntry does
 * (xmlerror.pxi), so the helper reports the same text and line numbers. */
#ifndef STUDIO_XSD_ERROR_LOG_H
#define STUDIO_XSD_ERROR_LOG_H

#include <stddef.h>
#include <libxml/xmlerror.h>

typedef struct {
    int domain;
    int code;      /* lxml: entry.type */
    int level;     /* XML_ERR_WARNING / XML_ERR_ERROR / XML_ERR_FATAL */
    long long line;
    int column;    /* lxml: entry.column = error->int2 */
    char *message; /* lxml: entry.message, already decoded to valid UTF-8 */
    char *filename;/* lxml: entry.filename, "<string>" when libxml2 gave none */
} xsd_entry;

typedef struct {
    xsd_entry *items;
    size_t count;
    size_t capacity;
    int failed; /* an allocation failed while recording; the log is incomplete */
} xsd_log;

void xsd_log_init(xsd_log *log);
void xsd_log_free(xsd_log *log);

/* xmlStructuredErrorFunc for validation/schema contexts: `data` is the xsd_log. */
void xsd_log_receive(void *data, xmlErrorPtr error);

/* xmlStructuredErrorFunc for parser contexts (sax->serror): `data` is the
 * xmlParserCtxt whose _private member points at the xsd_log (lxml's
 * _receiveParserError/_forwardParserError). */
void xsd_log_receive_parser(void *data, xmlErrorPtr error);

/* First entry of level XML_ERR_ERROR or worse (lxml _first_error), or NULL. */
const xsd_entry *xsd_log_first_error(const xsd_log *log);

/* lxml _LogEntry.message for a raw libxml2 message: "unknown error" for NULL, "" or
 * "\n"; otherwise one trailing newline removed and the bytes decoded as strict
 * UTF-8, falling back to ASCII with backslashreplace. Returns malloc'd UTF-8. */
char *xsd_lxml_message(const char *raw);

#endif
