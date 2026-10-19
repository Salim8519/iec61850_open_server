/* Parses XML exactly like lxml 6.0.2's
 *   etree.XMLParser(resolve_entities=False, no_network=True, load_dtd=False,
 *                   huge_tree=False)
 * used through etree.fromstring(bytes) and etree.parse(filename) (parser.pxi). */
#ifndef STUDIO_XSD_DOCUMENT_H
#define STUDIO_XSD_DOCUMENT_H

#include <stddef.h>
#include <libxml/parser.h>

/* _XML_DEFAULT_PARSE_OPTIONS = NOENT|NOCDATA|NONET|COMPACT|BIG_LINES, then
 * resolve_entities=False removes NOENT; load_dtd/huge_tree/recover add nothing.
 * = 0x414800 = 4278272. */
#define XSD_PARSE_OPTIONS (XML_PARSE_NOCDATA | XML_PARSE_NONET | XML_PARSE_COMPACT | XML_PARSE_BIG_LINES)

typedef struct {
    xmlDocPtr doc;        /* the document when lxml would return one (caller frees) */
    char *syntax_message; /* otherwise str(XMLSyntaxError), malloc'd UTF-8 */
    long long syntax_line;/* otherwise XMLSyntaxError.lineno */
    int doctype;          /* doc has a DOCTYPE (lxml docinfo.doctype is not empty) */
} xsd_parse_result;

/* etree.fromstring(bytes, parser). Returns 0, or -1 when out of memory. */
int xsd_parse_memory(const char *bytes, size_t size, xsd_parse_result *result);

/* etree.parse(utf8_path, parser). Returns 0, or -1 when out of memory. */
int xsd_parse_file(const char *utf8_path, xsd_parse_result *result);

void xsd_parse_result_free(xsd_parse_result *result);

#endif
