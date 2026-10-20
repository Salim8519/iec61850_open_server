/* config.h for GNU libiconv 1.17 built as a static library with MSVC x64 for studio_xsd.
 *
 * GNU libiconv has no MSVC project; its configure script writes config.h. This file
 * replaces that output for the two library sources the static library needs
 * (lib/iconv.c and libcharset/lib/localcharset.c, the objects of lib/Makefile.in minus
 * relocatable.c, which only matters for a relocatable DLL). Only macros those sources
 * read are set; every other configure result is unused by them.
 */
#ifndef STUDIO_XSD_LIBICONV_CONFIG_H
#define STUDIO_XSD_LIBICONV_CONFIG_H

/* No --enable-extra-encodings: the lxml 6.0.2 wheel's iconv has no extra encodings
   (no TDS565, ATARIST, EUC-JISX0213, BIG5-2003). iconv.c still turns on the DOS code
   pages (CP437, CP850, CP1125, ...) for _MSC_VER, as the wheel's build does. */
#define ENABLE_EXTRA 0

/* x64 Windows. */
#define WORDS_LITTLEENDIAN 1

/* iconv() takes char** (the declaration in the generated iconv.h matches). */
#define ICONV_CONST

/* The MSVC C runtime has these (the WCHAR_T pseudo-encoding uses them). */
#define HAVE_WCHAR_T 1
#define HAVE_MBRTOWC 1
#define HAVE_WCRTOMB 1
#define HAVE_MBSINIT 1
#define HAVE_SETLOCALE 1

/* localcharset.c uses GetACP()/setlocale() on native Windows, never nl_langinfo. */
#undef HAVE_LANGINFO_CODESET

#endif
