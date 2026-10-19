#include "document.h"

#include "error_log.h"
#include "syntax_error.h"

#include <limits.h>
#include <stdlib.h>
#include <string.h>
#include <libxml/SAX2.h>
#include <libxml/dict.h>
#include <libxml/encoding.h>
#include <libxml/hash.h>
#include <libxml/parserInternals.h>

/* lxml _initSaxDocument: shared dictionary for names and an XML ID table per document
 * (collect_ids=True). */
static void start_document(void *data) {
    xmlParserCtxtPtr context = (xmlParserCtxtPtr)data;
    xmlDocPtr doc;
    xmlSAX2StartDocument(data);
    doc = context->myDoc;
    if (doc && context->dict && !doc->dict) {
        context->dictNames = 1;
        doc->dict = context->dict;
        xmlDictReference(context->dict);
    }
    if (doc && !doc->ids) {
        xmlDictPtr ids_dict = xmlDictCreate();
        if (ids_dict) {
            doc->ids = xmlHashCreateDict(0, ids_dict);
            xmlDictFree(ids_dict);
        } else {
            doc->ids = xmlHashCreate(0);
        }
    }
}

/* lxml _newParserCtxt + _configureSaxContext + _ParserContext.prepare. */
static xmlParserCtxtPtr new_context(xsd_log *log) {
    xmlParserCtxtPtr context = xmlNewParserCtxt();
    if (!context) return NULL;
    context->sax->startDocument = start_document;
    context->sax->cdataBlock = NULL; /* strip_cdata=True */
    context->sax->serror = xsd_log_receive_parser;
    context->_private = log;
    return context;
}

static int log_has_code(const xsd_log *log, int code) {
    size_t index;
    for (index = 0; index < log->count; index++)
        if (log->items[index].code == code) return 1;
    return 0;
}

/* lxml _handleParseResult with recover=False: decides whether the document lxml got
 * from libxml2 is returned to the caller or raised as XMLSyntaxError. */
static xmlDocPtr accept_result(xmlParserCtxtPtr context, xmlDocPtr doc, const xsd_log *log) {
    int well_formed;
    size_t index;
    if (context->myDoc && context->myDoc != doc) xmlFreeDoc(context->myDoc);
    context->myDoc = NULL;
    if (!doc) return NULL;
    if (!context->wellFormed && !context->html && context->charset == XML_CHAR_ENCODING_8859_1 &&
        log_has_code(log, XML_ERR_INVALID_CHAR)) {
        well_formed = 0;
    } else if (context->wellFormed && context->lastError.level < XML_ERR_ERROR) {
        well_formed = 1;
    } else if (!context->replaceEntities && !context->validate) {
        /* undefined entities alone do not make the document unusable */
        well_formed = 1;
        for (index = 0; index < log->count; index++) {
            const xsd_entry *entry = &log->items[index];
            if (entry->level >= XML_ERR_ERROR && entry->code != XML_WAR_UNDECLARED_ENTITY &&
                entry->code != XML_ERR_UNDECLARED_ENTITY) {
                well_formed = 0;
                break;
            }
        }
    } else {
        well_formed = 0;
    }
    if (!well_formed) {
        xmlFreeDoc(doc);
        return NULL;
    }
    return doc;
}

/* lxml DocInfo.doctype is non-empty for an internal subset or a public/system id. */
static int has_doctype(xmlDocPtr doc) {
    xmlDtdPtr external = doc->extSubset;
    if (doc->intSubset) return 1;
    return external && ((external->ExternalID && external->ExternalID[0]) ||
                        (external->SystemID && external->SystemID[0]));
}

static int finish(xmlParserCtxtPtr context, xmlDocPtr doc, xsd_log *log, xsd_parse_result *result) {
    result->doc = accept_result(context, doc, log);
    if (result->doc) {
        result->doctype = has_doctype(result->doc);
        if (!result->doc->encoding) result->doc->encoding = xmlStrdup((const xmlChar *)"UTF-8");
    } else {
        result->syntax_message = xsd_syntax_error(log, context, &result->syntax_line);
    }
    context->sax->serror = NULL;
    context->_private = NULL;
    xmlFreeParserCtxt(context);
    xsd_log_free(log);
    return result->doc || result->syntax_message ? 0 : -1;
}

int xsd_parse_memory(const char *bytes, size_t size, xsd_parse_result *result) {
    const unsigned char *text = (const unsigned char *)bytes;
    const char *encoding = NULL;
    xsd_log log;
    xmlParserCtxtPtr context;
    xmlDocPtr doc;
    memset(result, 0, sizeof *result);
    if (size > INT_MAX) return -1;
    /* lxml _BaseParser._parseDoc: libxml2 does not recognise UTF-32 BOMs */
    if (size >= 4 && text[0] == 0xFF && text[1] == 0xFE && text[2] == 0 && text[3] == 0) {
        encoding = "UTF-32LE";
        text += 4;
        size -= 4;
    } else if (size >= 4 && text[0] == 0 && text[1] == 0 && text[2] == 0xFE && text[3] == 0xFF) {
        encoding = "UTF-32BE";
        text += 4;
        size -= 4;
    } else {
        xmlCharEncoding detected = xmlDetectCharEncoding(text, (int)size);
        if (detected == XML_CHAR_ENCODING_UCS4LE) encoding = "UTF-32LE";
        else if (detected == XML_CHAR_ENCODING_UCS4BE) encoding = "UTF-32BE";
    }
    xsd_log_init(&log);
    context = new_context(&log);
    if (!context) return -1;
    doc = xmlCtxtReadMemory(context, (const char *)text, (int)size, NULL, encoding, XSD_PARSE_OPTIONS);
    return finish(context, doc, &log, result);
}

int xsd_parse_file(const char *utf8_path, xsd_parse_result *result) {
    xsd_log log;
    xmlParserCtxtPtr context;
    xmlDocPtr doc;
    memset(result, 0, sizeof *result);
    xsd_log_init(&log);
    context = new_context(&log);
    if (!context) return -1;
    doc = xmlCtxtReadFile(context, utf8_path, NULL, XSD_PARSE_OPTIONS);
    return finish(context, doc, &log, result);
}

void xsd_parse_result_free(xsd_parse_result *result) {
    if (result->doc) xmlFreeDoc(result->doc);
    free(result->syntax_message);
    memset(result, 0, sizeof *result);
}
