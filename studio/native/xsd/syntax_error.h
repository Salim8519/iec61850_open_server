/* The text of str(lxml.etree.XMLSyntaxError) and its lineno for a document lxml
 * refused to parse (parser.pxi _raiseParseError + CPython SyntaxError.__str__). */
#ifndef STUDIO_XSD_SYNTAX_ERROR_H
#define STUDIO_XSD_SYNTAX_ERROR_H

#include "error_log.h"
#include <libxml/parser.h>

/* Returns a malloc'd UTF-8 message (NULL when out of memory) and stores the
 * exception's lineno in *line. */
char *xsd_syntax_error(const xsd_log *log, const xmlParserCtxt *context, long long *line);

#endif
