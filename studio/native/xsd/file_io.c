#include "file_io.h"

#include <stdio.h>
#include <stdlib.h>
#ifdef _WIN32
#include <windows.h>
#endif

static FILE *open_utf8(const char *utf8_path) {
#ifdef _WIN32
    wchar_t *wide;
    FILE *file = NULL;
    int count = MultiByteToWideChar(CP_UTF8, MB_ERR_INVALID_CHARS, utf8_path, -1, NULL, 0);
    if (count <= 0) return NULL;
    wide = (wchar_t *)malloc((size_t)count * sizeof(wchar_t));
    if (!wide) return NULL;
    if (MultiByteToWideChar(CP_UTF8, MB_ERR_INVALID_CHARS, utf8_path, -1, wide, count) > 0)
        file = _wfopen(wide, L"rb");
    free(wide);
    return file;
#else
    return fopen(utf8_path, "rb");
#endif
}

int xsd_read_file(const char *utf8_path, char **data, size_t *size, const char **error) {
    FILE *file = open_utf8(utf8_path);
    size_t capacity = 1 << 20, used = 0;
    char *buffer;
    *data = NULL;
    *size = 0;
    if (!file) {
        *error = "The source file of the validate request could not be opened";
        return -1;
    }
    buffer = (char *)malloc(capacity);
    while (buffer) {
        size_t got = fread(buffer + used, 1, capacity - used, file);
        used += got;
        if (got == 0) break;
        if (used == capacity) {
            char *grown;
            if (capacity >= XSD_SOURCE_LIMIT) {
                free(buffer);
                fclose(file);
                *error = "The source file of the validate request is larger than 64 MiB";
                return -1;
            }
            grown = (char *)realloc(buffer, capacity * 2);
            if (!grown) free(buffer);
            buffer = grown;
            capacity *= 2;
        }
    }
    if (!buffer) {
        fclose(file);
        *error = "Out of memory reading the source file";
        return -1;
    }
    if (ferror(file)) {
        free(buffer);
        fclose(file);
        *error = "The source file of the validate request could not be read";
        return -1;
    }
    fclose(file);
    *data = buffer;
    *size = used;
    return 0;
}
