#include "framing.h"

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#ifdef _WIN32
#include <fcntl.h>
#include <io.h>
#else
#include <unistd.h>
#endif

static FILE *frame_output = NULL;

/* Same arrangement as worker/ipc/framing.cpp protect_frame_output(). */
int xsd_frame_init(void) {
    int private_fd;
#ifdef _WIN32
    _setmode(_fileno(stdin), _O_BINARY);
    fflush(stdout);
    private_fd = _dup(_fileno(stdout));
    if (private_fd < 0) return -1;
    _setmode(private_fd, _O_BINARY);
    frame_output = _fdopen(private_fd, "wb");
    _dup2(_fileno(stderr), _fileno(stdout));
#else
    fflush(stdout);
    private_fd = dup(fileno(stdout));
    if (private_fd < 0) return -1;
    frame_output = fdopen(private_fd, "wb");
    dup2(fileno(stderr), fileno(stdout));
#endif
    return frame_output ? 0 : -1;
}

static int read_exact(unsigned char *target, size_t size) {
    size_t done = 0;
    while (done < size) {
        size_t got = fread(target + done, 1, size - done, stdin);
        if (got == 0) return done == 0 ? 0 : -1;
        done += got;
    }
    return 1;
}

int xsd_frame_read(char **payload, size_t *size, const char **error) {
    unsigned char header[4];
    uint32_t length;
    char *buffer;
    int status = read_exact(header, 4);
    *payload = NULL;
    *size = 0;
    if (status == 0) return 0;
    if (status < 0) {
        *error = "Truncated frame header";
        return -1;
    }
    length = (uint32_t)header[0] | ((uint32_t)header[1] << 8) |
        ((uint32_t)header[2] << 16) | ((uint32_t)header[3] << 24);
    if (length == 0 || length > XSD_FRAME_LIMIT) {
        *error = "Invalid frame length";
        return -1;
    }
    buffer = (char *)malloc((size_t)length + 1);
    if (!buffer) {
        *error = "Out of memory reading a frame";
        return -1;
    }
    if (read_exact((unsigned char *)buffer, length) != 1) {
        free(buffer);
        *error = "Truncated frame";
        return -1;
    }
    buffer[length] = '\0';
    *payload = buffer;
    *size = length;
    return 1;
}

int xsd_frame_write(const char *payload, size_t size) {
    unsigned char header[4];
    FILE *output = frame_output ? frame_output : stdout;
    if (size == 0 || size > XSD_FRAME_LIMIT) return -1;
    header[0] = (unsigned char)size;
    header[1] = (unsigned char)(size >> 8);
    header[2] = (unsigned char)(size >> 16);
    header[3] = (unsigned char)(size >> 24);
    if (fwrite(header, 1, 4, output) != 4) return -1;
    if (fwrite(payload, 1, size, output) != size) return -1;
    return fflush(output) == 0 ? 0 : -1;
}
