#include "json_read.h"

#include "json_write.h"

#include <stdlib.h>
#include <string.h>

typedef struct {
    const char *at;
    const char *end;
    const char *error;
} cursor;

static void skip_space(cursor *c) {
    while (c->at < c->end && (*c->at == ' ' || *c->at == '\t' || *c->at == '\n' || *c->at == '\r'))
        c->at++;
}

static int fail(cursor *c, const char *message) {
    if (!c->error) c->error = message;
    return -1;
}

static int hex4(cursor *c, unsigned *value) {
    int index;
    *value = 0;
    if (c->end - c->at < 4) return fail(c, "Truncated \\u escape in request");
    for (index = 0; index < 4; index++) {
        char h = *c->at++;
        unsigned digit;
        if (h >= '0' && h <= '9') digit = (unsigned)(h - '0');
        else if (h >= 'a' && h <= 'f') digit = (unsigned)(h - 'a' + 10);
        else if (h >= 'A' && h <= 'F') digit = (unsigned)(h - 'A' + 10);
        else return fail(c, "Invalid \\u escape in request");
        *value = *value * 16 + digit;
    }
    return 0;
}

static void put_utf8(xsd_buf *out, unsigned code) {
    char bytes[4];
    size_t size;
    if (code < 0x80) { bytes[0] = (char)code; size = 1; }
    else if (code < 0x800) {
        bytes[0] = (char)(0xC0 | (code >> 6)); bytes[1] = (char)(0x80 | (code & 0x3F)); size = 2;
    } else if (code < 0x10000) {
        bytes[0] = (char)(0xE0 | (code >> 12)); bytes[1] = (char)(0x80 | ((code >> 6) & 0x3F));
        bytes[2] = (char)(0x80 | (code & 0x3F)); size = 3;
    } else {
        bytes[0] = (char)(0xF0 | (code >> 18)); bytes[1] = (char)(0x80 | ((code >> 12) & 0x3F));
        bytes[2] = (char)(0x80 | ((code >> 6) & 0x3F)); bytes[3] = (char)(0x80 | (code & 0x3F)); size = 4;
    }
    xsd_buf_append(out, bytes, size);
}

static int unicode_escape(cursor *c, xsd_buf *out) {
    unsigned code, low;
    if (hex4(c, &code) < 0) return -1;
    if (code >= 0xD800 && code <= 0xDBFF) {
        if (c->end - c->at < 6 || c->at[0] != '\\' || c->at[1] != 'u')
            return fail(c, "Unpaired surrogate in request");
        c->at += 2;
        if (hex4(c, &low) < 0) return -1;
        if (low < 0xDC00 || low > 0xDFFF) return fail(c, "Unpaired surrogate in request");
        code = 0x10000 + ((code - 0xD800) << 10) + (low - 0xDC00);
    } else if (code >= 0xDC00 && code <= 0xDFFF) {
        return fail(c, "Unpaired surrogate in request");
    }
    put_utf8(out, code);
    return 0;
}

/* Parses a string literal; stores a malloc'd copy in *value when value is not NULL. */
static int parse_string(cursor *c, char **value) {
    xsd_buf out;
    xsd_buf_init(&out);
    if (c->at >= c->end || *c->at != '"') return fail(c, "Expected a string in request");
    c->at++;
    while (c->at < c->end && *c->at != '"') {
        const char *run = c->at;
        while (c->at < c->end && *c->at != '"' && *c->at != '\\') c->at++;
        xsd_buf_append(&out, run, (size_t)(c->at - run));
        if (c->at < c->end && *c->at == '\\') {
            char e;
            if (++c->at >= c->end) break;
            e = *c->at++;
            switch (e) {
            case '"': case '\\': case '/': xsd_buf_append(&out, &e, 1); break;
            case 'b': xsd_buf_append(&out, "\b", 1); break;
            case 'f': xsd_buf_append(&out, "\f", 1); break;
            case 'n': xsd_buf_append(&out, "\n", 1); break;
            case 'r': xsd_buf_append(&out, "\r", 1); break;
            case 't': xsd_buf_append(&out, "\t", 1); break;
            case 'u':
                if (unicode_escape(c, &out) < 0) { xsd_buf_free(&out); return -1; }
                break;
            default: xsd_buf_free(&out); return fail(c, "Invalid escape in request");
            }
        }
    }
    if (c->at >= c->end) { xsd_buf_free(&out); return fail(c, "Unterminated string in request"); }
    c->at++;
    if (value) {
        *value = xsd_buf_take(&out);
        if (!*value) return fail(c, "Out of memory reading the request");
    } else {
        xsd_buf_free(&out);
    }
    return 0;
}

static int parse_integer(cursor *c, long long *value) {
    int negative = 0, digits = 0;
    long long result = 0;
    if (c->at < c->end && *c->at == '-') { negative = 1; c->at++; }
    while (c->at < c->end && *c->at >= '0' && *c->at <= '9') {
        if (result > 100000000000000LL) return fail(c, "Number too large in request");
        result = result * 10 + (*c->at++ - '0');
        digits++;
    }
    if (!digits) return fail(c, "Expected a number in request");
    if (c->at < c->end && (*c->at == '.' || *c->at == 'e' || *c->at == 'E'))
        return fail(c, "Expected an integer in request");
    *value = negative ? -result : result;
    return 0;
}

static int skip_value(cursor *c, int depth) {
    const char *start;
    skip_space(c);
    if (depth > 32) return fail(c, "Request nests too deeply");
    if (c->at >= c->end) return fail(c, "Truncated request");
    if (*c->at == '"') return parse_string(c, NULL);
    if (*c->at == '{' || *c->at == '[') {
        char close = *c->at == '{' ? '}' : ']';
        int object = close == '}';
        c->at++;
        skip_space(c);
        if (c->at < c->end && *c->at == close) { c->at++; return 0; }
        for (;;) {
            if (object) {
                skip_space(c);
                if (parse_string(c, NULL) < 0) return -1;
                skip_space(c);
                if (c->at >= c->end || *c->at++ != ':') return fail(c, "Expected ':' in request");
            }
            if (skip_value(c, depth + 1) < 0) return -1;
            skip_space(c);
            if (c->at < c->end && *c->at == ',') { c->at++; continue; }
            if (c->at < c->end && *c->at == close) { c->at++; return 0; }
            return fail(c, "Malformed request");
        }
    }
    if (c->end - c->at >= 4 && (!strncmp(c->at, "true", 4) || !strncmp(c->at, "null", 4))) {
        c->at += 4;
        return 0;
    }
    if (c->end - c->at >= 5 && !strncmp(c->at, "false", 5)) { c->at += 5; return 0; }
    start = c->at;
    while (c->at < c->end && *c->at && strchr("+-0123456789.eE", *c->at)) c->at++;
    return c->at > start ? 0 : fail(c, "Malformed request");
}

static int member(cursor *c, xsd_request *request, const char *key) {
    char **target = NULL;
    if (!strcmp(key, "id")) target = &request->id;
    else if (!strcmp(key, "op")) target = &request->op;
    else if (!strcmp(key, "schema_path")) target = &request->schema_path;
    else if (!strcmp(key, "source_path")) target = &request->source_path;
    else if (!strcmp(key, "version")) return parse_integer(c, &request->version);
    if (!target) return skip_value(c, 1);
    free(*target);
    *target = NULL;
    return parse_string(c, target);
}

int xsd_request_parse(const char *text, size_t size, xsd_request *request, const char **error) {
    cursor c = {text, text + size, NULL};
    memset(request, 0, sizeof *request);
    skip_space(&c);
    if (c.at >= c.end || *c.at++ != '{') { *error = "Request must be a JSON object"; return -1; }
    skip_space(&c);
    if (c.at < c.end && *c.at == '}') c.at++;
    else for (;;) {
        char *key = NULL;
        int status;
        skip_space(&c);
        if (parse_string(&c, &key) < 0) break;
        skip_space(&c);
        if (c.at >= c.end || *c.at++ != ':') { free(key); fail(&c, "Expected ':' in request"); break; }
        skip_space(&c);
        status = member(&c, request, key);
        free(key);
        if (status < 0) break;
        skip_space(&c);
        if (c.at < c.end && *c.at == ',') { c.at++; continue; }
        if (c.at < c.end && *c.at == '}') { c.at++; break; }
        fail(&c, "Malformed request");
        break;
    }
    if (c.error) { *error = c.error; return -1; }
    if (!request->id) request->id = (char *)calloc(1, 1);
    if (!request->id) { *error = "Out of memory reading the request"; return -1; }
    return 0;
}

void xsd_request_free(xsd_request *request) {
    free(request->id);
    free(request->op);
    free(request->schema_path);
    free(request->source_path);
    memset(request, 0, sizeof *request);
}
