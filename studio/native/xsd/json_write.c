#include "json_write.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

void xsd_buf_init(xsd_buf *buf) {
    buf->data = NULL;
    buf->size = 0;
    buf->capacity = 0;
    buf->failed = 0;
}

void xsd_buf_free(xsd_buf *buf) {
    free(buf->data);
    xsd_buf_init(buf);
}

static int reserve(xsd_buf *buf, size_t extra) {
    size_t wanted;
    char *grown;
    if (buf->failed) return 0;
    wanted = buf->size + extra + 1;
    if (wanted <= buf->capacity) return 1;
    if (wanted < buf->capacity * 2) wanted = buf->capacity * 2;
    if (wanted < 256) wanted = 256;
    grown = (char *)realloc(buf->data, wanted);
    if (!grown) {
        buf->failed = 1;
        return 0;
    }
    buf->data = grown;
    buf->capacity = wanted;
    return 1;
}

void xsd_buf_append(xsd_buf *buf, const char *text, size_t size) {
    if (!reserve(buf, size)) return;
    memcpy(buf->data + buf->size, text, size);
    buf->size += size;
    buf->data[buf->size] = '\0';
}

void xsd_buf_text(xsd_buf *buf, const char *text) {
    xsd_buf_append(buf, text, strlen(text));
}

void xsd_buf_long(xsd_buf *buf, long long value) {
    char digits[32];
    int length = snprintf(digits, sizeof digits, "%lld", value);
    if (length > 0) xsd_buf_append(buf, digits, (size_t)length);
}

void xsd_buf_json_string(xsd_buf *buf, const char *text, size_t size) {
    static const char hex[] = "0123456789abcdef";
    size_t start = 0, index;
    xsd_buf_append(buf, "\"", 1);
    for (index = 0; index < size; index++) {
        unsigned char c = (unsigned char)text[index];
        char escape[7];
        size_t escape_size = 2;
        if (c >= 0x20 && c != '"' && c != '\\') continue;
        xsd_buf_append(buf, text + start, index - start);
        start = index + 1;
        escape[0] = '\\';
        switch (c) {
        case '"': escape[1] = '"'; break;
        case '\\': escape[1] = '\\'; break;
        case '\n': escape[1] = 'n'; break;
        case '\r': escape[1] = 'r'; break;
        case '\t': escape[1] = 't'; break;
        case '\b': escape[1] = 'b'; break;
        case '\f': escape[1] = 'f'; break;
        default:
            memcpy(escape + 1, "u00", 3);
            escape[4] = hex[c >> 4];
            escape[5] = hex[c & 15];
            escape_size = 6;
        }
        xsd_buf_append(buf, escape, escape_size);
    }
    xsd_buf_append(buf, text + start, size - start);
    xsd_buf_append(buf, "\"", 1);
}

char *xsd_buf_take(xsd_buf *buf) {
    char *data;
    if (buf->failed) {
        xsd_buf_free(buf);
        return NULL;
    }
    if (!buf->data) {
        xsd_buf_append(buf, "", 0);
        if (buf->failed) return NULL;
    }
    data = buf->data;
    xsd_buf_init(buf);
    return data;
}
