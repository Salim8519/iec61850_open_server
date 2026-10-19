#include "schema_cache.h"

#include "document.h"
#include "error_log.h"
#include "json_write.h"

#include <stdlib.h>
#include <string.h>

typedef struct {
    char *path;
    xmlDocPtr doc; /* lxml keeps the copied schema document alive with the schema */
    xmlSchemaPtr schema;
    unsigned long long used;
} slot;

static slot slots[XSD_SCHEMA_SLOTS];
static unsigned long long clock_tick = 0;

static void release(slot *entry) {
    if (entry->schema) xmlSchemaFree(entry->schema);
    if (entry->doc) xmlFreeDoc(entry->doc);
    free(entry->path);
    memset(entry, 0, sizeof *entry);
}

static char *message(const char *prefix, const char *path, const char *detail) {
    xsd_buf out;
    xsd_buf_init(&out);
    xsd_buf_text(&out, prefix);
    xsd_buf_text(&out, path);
    if (detail) {
        xsd_buf_text(&out, ": ");
        xsd_buf_text(&out, detail);
    }
    return xsd_buf_take(&out);
}

/* lxml _BaseErrorLog._buildExceptionMessage("Document is not valid XML Schema"). */
static char *schema_failure(const xsd_log *log) {
    const xsd_entry *first = xsd_log_first_error(log);
    xsd_buf out;
    xsd_buf_init(&out);
    xsd_buf_text(&out, first ? first->message : "Document is not valid XML Schema");
    if (first && first->line > 0) {
        xsd_buf_text(&out, ", line ");
        xsd_buf_long(&out, first->line);
        if (first->column > 0) {
            xsd_buf_text(&out, ", column ");
            xsd_buf_long(&out, first->column);
        }
    }
    return xsd_buf_take(&out);
}

/* lxml _copyDocRoot: a non-recursive copy of the document plus a deep copy of the
 * root element (a parsed document has no text nodes after the root to carry over). */
static xmlDocPtr copy_document_root(xmlDocPtr doc) {
    xmlDocPtr copy = xmlCopyDoc(doc, 0);
    xmlNodePtr root;
    if (!copy) return NULL;
    root = xmlDocCopyNode(xmlDocGetRootElement(doc), copy, 1);
    if (!root) {
        xmlFreeDoc(copy);
        return NULL;
    }
    xmlDocSetRootElement(copy, root);
    return copy;
}

static int load(slot *entry, const char *utf8_path, char **error) {
    xsd_parse_result parsed;
    xmlSchemaParserCtxtPtr parser;
    xsd_log log;
    if (xsd_parse_file(utf8_path, &parsed) < 0) {
        *error = message("Out of memory loading the bundled XSD ", utf8_path, NULL);
        return -1;
    }
    if (!parsed.doc) {
        *error = message("The bundled XSD could not be read: ", utf8_path, parsed.syntax_message);
        xsd_parse_result_free(&parsed);
        return -1;
    }
    entry->doc = copy_document_root(parsed.doc);
    xsd_parse_result_free(&parsed);
    if (!entry->doc) {
        *error = message("Out of memory loading the bundled XSD ", utf8_path, NULL);
        return -1;
    }
    parser = xmlSchemaNewDocParserCtxt(entry->doc);
    if (!parser) {
        *error = message("Out of memory loading the bundled XSD ", utf8_path, NULL);
        return -1;
    }
    xsd_log_init(&log);
    xmlSchemaSetParserStructuredErrors(parser, xsd_log_receive, &log);
    entry->schema = xmlSchemaParse(parser);
    xmlSchemaFreeParserCtxt(parser);
    if (!entry->schema) {
        char *detail = schema_failure(&log);
        *error = message("The bundled XSD is not a valid XML Schema: ", utf8_path, detail);
        free(detail);
    }
    xsd_log_free(&log);
    return entry->schema ? 0 : -1;
}

xmlSchemaPtr xsd_schema_get(const char *utf8_path, char **error) {
    slot *target = &slots[0];
    size_t index;
    *error = NULL;
    for (index = 0; index < XSD_SCHEMA_SLOTS; index++) {
        if (slots[index].path && !strcmp(slots[index].path, utf8_path)) {
            slots[index].used = ++clock_tick;
            return slots[index].schema;
        }
        if (!slots[index].path) target = &slots[index];
        else if (target->path && slots[index].used < target->used) target = &slots[index];
    }
    release(target); /* least recently used, or an empty slot */
    target->path = (char *)malloc(strlen(utf8_path) + 1);
    if (!target->path) {
        *error = message("Out of memory loading the bundled XSD ", utf8_path, NULL);
        return NULL;
    }
    strcpy(target->path, utf8_path);
    if (load(target, utf8_path, error) < 0) {
        release(target);
        return NULL;
    }
    target->used = ++clock_tick;
    return target->schema;
}

void xsd_schema_clear(void) {
    size_t index;
    for (index = 0; index < XSD_SCHEMA_SLOTS; index++) release(&slots[index]);
}
