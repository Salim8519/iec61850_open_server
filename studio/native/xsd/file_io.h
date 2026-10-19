/* Reads the SCL bytes the engine wrote for one validate request. */
#ifndef STUDIO_XSD_FILE_IO_H
#define STUDIO_XSD_FILE_IO_H

#include <stddef.h>

/* Largest source the helper accepts (the engine itself stops at 20 MB). */
#define XSD_SOURCE_LIMIT (64u * 1024u * 1024u)

/* Reads the whole file at a UTF-8 path (wide-character API on Windows) into a
 * malloc'd buffer. Returns 0, or -1 with a static English message in *error. */
int xsd_read_file(const char *utf8_path, char **data, size_t *size, const char **error);

#endif
