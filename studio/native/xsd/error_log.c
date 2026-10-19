#include "error_log.h"

#include "json_write.h"

#include <limits.h>
#include <stdlib.h>
#include <string.h>
#include <libxml/parser.h>
#include <libxml/tree.h>

void xsd_log_init(xsd_log *log) {
    memset(log, 0, sizeof *log);
}

void xsd_log_free(xsd_log *log) {
    size_t index;
    for (index = 0; index < log->count; index++) {
        free(log->items[index].message);
        free(log->items[index].filename);
    }
    free(log->items);
    xsd_log_init(log);
}

/* Length of the strict UTF-8 sequence at text (Python's codec rules), 0 if invalid. */
static size_t utf8_sequence(const unsigned char *text, size_t left) {
    unsigned char c = text[0];
    unsigned char low = 0x80, high = 0xBF;
    size_t size, index;
    if (c < 0x80) return 1;
    if (c >= 0xC2 && c <= 0xDF) size = 2;
    else if (c >= 0xE0 && c <= 0xEF) {
        size = 3;
        if (c == 0xE0) low = 0xA0;
        if (c == 0xED) high = 0x9F;
    } else if (c >= 0xF0 && c <= 0xF4) {
        size = 4;
        if (c == 0xF0) low = 0x90;
        if (c == 0xF4) high = 0x8F;
    } else return 0;
    if (left < size) return 0;
    if (text[1] < low || text[1] > high) return 0;
    for (index = 2; index < size; index++)
        if (text[index] < 0x80 || text[index] > 0xBF) return 0;
    return size;
}

static int valid_utf8(const char *text, size_t size) {
    size_t at = 0;
    while (at < size) {
        size_t step = utf8_sequence((const unsigned char *)text + at, size - at);
        if (!step) return 0;
        at += step;
    }
    return 1;
}

static char *copy_bytes(const char *text, size_t size) {
    char *copy = (char *)malloc(size + 1);
    if (!copy) return NULL;
    memcpy(copy, text, size);
    copy[size] = '\0';
    return copy;
}

char *xsd_lxml_message(const char *raw) {
    static const char hex[] = "0123456789abcdef";
    size_t size, index;
    xsd_buf out;
    if (!raw || raw[0] == '\0' || (raw[0] == '\n' && raw[1] == '\0'))
        return copy_bytes("unknown error", 13);
    size = strlen(raw);
    if (raw[size - 1] == '\n') size--;
    if (valid_utf8(raw, size)) return copy_bytes(raw, size);
    xsd_buf_init(&out); /* ASCII with backslashreplace */
    for (index = 0; index < size; index++) {
        unsigned char c = (unsigned char)raw[index];
        if (c < 0x80) {
            xsd_buf_append(&out, raw + index, 1);
        } else {
            char escape[4] = {'\\', 'x', hex[c >> 4], hex[c & 15]};
            xsd_buf_append(&out, escape, 4);
        }
    }
    return xsd_buf_take(&out);
}

/* lxml _decodeFilename: UTF-8, or Latin-1 when the bytes are not UTF-8. */
static char *decode_filename(const char *raw) {
    size_t size = strlen(raw), index;
    xsd_buf out;
    if (valid_utf8(raw, size)) return copy_bytes(raw, size);
    xsd_buf_init(&out);
    for (index = 0; index < size; index++) {
        unsigned char c = (unsigned char)raw[index];
        char bytes[2] = {(char)(0xC0 | (c >> 6)), (char)(0x80 | (c & 0x3F))};
        if (c < 0x80) xsd_buf_append(&out, raw + index, 1);
        else xsd_buf_append(&out, bytes, 2);
    }
    return xsd_buf_take(&out);
}

void xsd_log_receive(void *data, xmlErrorPtr error) {
    xsd_log *log = (xsd_log *)data;
    xsd_entry entry;
    if (!log || !error) return;
    if (log->count == log->capacity) {
        size_t capacity = log->capacity ? log->capacity * 2 : 16;
        xsd_entry *grown = (xsd_entry *)realloc(log->items, capacity * sizeof *grown);
        if (!grown) { log->failed = 1; return; }
        log->items = grown;
        log->capacity = capacity;
    }
    entry.domain = error->domain;
    entry.code = error->code;
    entry.level = (int)error->level;
    entry.line = error->line;
    entry.column = error->int2;
    if (error->node) { /* lxml: a node's line wins only beyond INT_MAX */
        long node_line = xmlGetLineNo((const xmlNode *)error->node);
        if (node_line > INT_MAX) entry.line = node_line;
    }
    entry.message = xsd_lxml_message(error->message);
    entry.filename = error->file ? decode_filename(error->file) : copy_bytes("<string>", 8);
    if (!entry.message || !entry.filename) {
        free(entry.message);
        free(entry.filename);
        log->failed = 1;
        return;
    }
    log->items[log->count++] = entry;
}

void xsd_log_receive_parser(void *data, xmlErrorPtr error) {
    xmlParserCtxtPtr context = (xmlParserCtxtPtr)data;
    if (context && context->_private) xsd_log_receive(context->_private, error);
}

const xsd_entry *xsd_log_first_error(const xsd_log *log) {
    size_t index;
    for (index = 0; index < log->count; index++)
        if (log->items[index].level >= XML_ERR_ERROR) return &log->items[index];
    return NULL;
}
