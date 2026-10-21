#include "syntax_error.h"

#include "json_write.h"

#include <string.h>

/* CPython's my_basename() in Objects/exceptions.c, with the Windows separator. */
static const char *basename_of(const char *filename) {
    const char *slash = strrchr(filename, '\\');
    return slash ? slash + 1 : filename;
}

/* repr() of a bytes object, as an f-string renders it ("b'...'"). */
static void bytes_repr(xsd_buf *out, const char *text, size_t size) {
    static const char hex[] = "0123456789abcdef";
    char quote = '\'';
    size_t index;
    if (memchr(text, '\'', size) && !memchr(text, '"', size)) quote = '"';
    xsd_buf_append(out, "b", 1);
    xsd_buf_append(out, &quote, 1);
    for (index = 0; index < size; index++) {
        unsigned char c = (unsigned char)text[index];
        char escape[4] = {'\\', 'x', hex[c >> 4], hex[c & 15]};
        if (c == (unsigned char)quote || c == '\\') { escape[1] = (char)c; xsd_buf_append(out, escape, 2); }
        else if (c == '\t') xsd_buf_append(out, "\\t", 2);
        else if (c == '\n') xsd_buf_append(out, "\\n", 2);
        else if (c == '\r') xsd_buf_append(out, "\\r", 2);
        else if (c < 0x20 || c >= 0x7F) xsd_buf_append(out, escape, 4);
        else xsd_buf_append(out, text + index, 1);
    }
    xsd_buf_append(out, &quote, 1);
}

static int python_space(char c) {
    return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\x0b' || c == '\x0c';
}

/* lxml's fallback when its error log is empty: ctxt.lastError.message.strip() is a
 * bytes object, so the exception text carries its repr. */
static void last_error_text(xsd_buf *out, const xmlParserCtxt *context, long long *line) {
    const char *message = context->lastError.message;
    size_t start = 0, end;
    if (!message) {
        xsd_buf_text(out, "None");
        *line = 0;
        return;
    }
    end = strlen(message);
    while (start < end && python_space(message[start])) start++;
    while (end > start && python_space(message[end - 1])) end--;
    *line = context->lastError.line;
    if (*line > 0) {
        xsd_buf_text(out, "line ");
        xsd_buf_long(out, *line);
        xsd_buf_text(out, ": ");
    }
    bytes_repr(out, message + start, end - start);
}

char *xsd_syntax_error(const xsd_log *log, const xmlParserCtxt *context, long long *line) {
    xsd_buf out;
    const xsd_entry *first;
    xsd_buf_init(&out);
    if (log->count == 0) {
        /* _raiseParseError: no log entries -> lastError; filename is None. */
        last_error_text(&out, context, line);
        xsd_buf_text(&out, " (line ");
        xsd_buf_long(&out, *line);
        xsd_buf_text(&out, ")");
        return xsd_buf_take(&out);
    }
    first = xsd_log_first_error(log);
    if (!first) {
        /* _buildParseException without a first error: default text, line 0, no file. */
        *line = 0;
        xsd_buf_text(&out, "Document is not well formed (line 0)");
        return xsd_buf_take(&out);
    }
    *line = first->line;
    xsd_buf_text(&out, first->message);
    if (first->line > 0) {
        xsd_buf_text(&out, ", line ");
        xsd_buf_long(&out, first->line);
        if (first->column > 0) {
            xsd_buf_text(&out, ", column ");
            xsd_buf_long(&out, first->column);
        }
    }
    /* SyntaxError.__str__: "msg (basename(filename), line lineno)". */
    xsd_buf_text(&out, " (");
    xsd_buf_text(&out, basename_of(first->filename));
    xsd_buf_text(&out, ", line ");
    xsd_buf_long(&out, first->line);
    xsd_buf_text(&out, ")");
    return xsd_buf_take(&out);
}
