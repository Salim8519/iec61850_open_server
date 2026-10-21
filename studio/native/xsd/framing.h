/* Native worker framing shared by every ScadaBench helper: a 4-byte little-endian
 * length followed by that many bytes of UTF-8 JSON, at most 16 MiB per frame. */
#ifndef STUDIO_XSD_FRAMING_H
#define STUDIO_XSD_FRAMING_H

#include <stddef.h>

#define XSD_FRAME_LIMIT (16u * 1024u * 1024u)

/* Puts stdin/stdout in binary mode and keeps a private copy of the stdout pipe for
 * frames; descriptor 1 then points at stderr so stray library output cannot corrupt
 * the protocol. Returns 0 on success. */
int xsd_frame_init(void);

/* Reads one frame into a malloc'd, NUL-terminated buffer. Returns 1 for a frame,
 * 0 for a clean end of input, -1 for a broken frame (message in error). */
int xsd_frame_read(char **payload, size_t *size, const char **error);

/* Writes one frame. Returns 0 on success, -1 when the payload is too large or the
 * pipe is gone. */
int xsd_frame_write(const char *payload, size_t size);

#endif
