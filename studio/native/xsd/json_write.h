/* Growable text buffer with the few JSON writers the helper's replies need. */
#ifndef STUDIO_XSD_JSON_WRITE_H
#define STUDIO_XSD_JSON_WRITE_H

#include <stddef.h>

typedef struct {
    char *data;
    size_t size;
    size_t capacity;
    int failed; /* set when an allocation failed; the content is then unusable */
} xsd_buf;

void xsd_buf_init(xsd_buf *buf);
void xsd_buf_free(xsd_buf *buf);
void xsd_buf_append(xsd_buf *buf, const char *text, size_t size);
void xsd_buf_text(xsd_buf *buf, const char *text);
void xsd_buf_long(xsd_buf *buf, long long value);

/* Appends a JSON string literal. `text` must be valid UTF-8 (the helper only writes
 * strings it produced or validated); quotes, backslashes and control characters are
 * escaped, everything else is copied byte for byte. */
void xsd_buf_json_string(xsd_buf *buf, const char *text, size_t size);

/* Takes ownership of the content as a NUL-terminated string (NULL if failed). */
char *xsd_buf_take(xsd_buf *buf);

#endif
