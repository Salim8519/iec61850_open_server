# studio_xsd: SCL schema validation helper

`studio_xsd.exe` validates SCL documents against the bundled IEC XSDs
(`studio/schemas/SCL2003`, `SCL2007B`, `SCL2007B4`) with **libxml2 2.11.9**, the release
inside lxml 6.0.2. It exists so the Go engine reports exactly the messages and line
numbers that `scl/schema_validation.py` reported, without cgo (ADR-0074, go-engine
architecture section 1). The Go side is `studio/engine/internal/store/xsd`.

The helper only does the libxml2 part. The Go engine still decides:

- which schema applies to the document (version/revision/release mapping),
- the "unsupported" verdict when no schema applies,
- the DOCTYPE `ValueError`, raised before the helper is called.

The helper also reports the DOCTYPE as a second check.

## Pin: libxml2 2.11.9 and why

- `lxml.etree.LIBXML_VERSION == (2, 11, 9)` and `LIBXML_COMPILED_VERSION == (2, 11, 9)` in
  `studio/.venv` (lxml 6.0.2). XSD messages and parser messages change between libxml2
  releases (wording, "Expected is one of" lists, line attribution), so any other release
  would break byte parity with the Python engine.
- The verdict cache key contains `2.11.9`, and the Go client refuses a helper whose hello
  reports another runtime or compiled version.
- **Feature set.** The libxml2 inside the wheel was built with the switches recorded in
  that wheel's `lxml/includes/libxml/xmlversion.h`:

  | Feature | Setting |
  |---|---|
  | THREAD, TREE, OUTPUT, PUSH, READER, PATTERN, WRITER, SAX1, FTP, HTTP, VALID, HTML, LEGACY, C14N, CATALOG, XPATH, XPTR, XINCLUDE, ICONV, DEBUG, UNICODE, REGEXP, AUTOMATA, SCHEMAS, SCHEMATRON, MODULES, ZLIB | on |
  | THREAD_ALLOC, XPTR_LOCS, ICU, ISO8859X, DEBUG_RUNTIME, EXPR, LZMA | off |

  `cmake/lxml_features.cmake` sets the same `LIBXML2_WITH_*` options. After libxml2 is
  configured, `cmake/check_features.cmake` compares the generated `xmlversion.h` with
  that list and stops the configure step on any difference or on a version other than
  2.11.9. This check was tested against the wheel's own header: it passes, and a changed
  zlib switch makes it fail.
- **iconv matters for messages.** Iconv decides which `encoding="..."` declarations
  parse. The wheel's iconv is GNU libiconv:
  - accepted: GEORGIAN-PS, MULELAO-1, JAVA, C99, TCVN, CP1133, KOI8-T, PT154, RK1048,
    windows-1252, Shift_JIS, GB18030, BIG5-HKSCS, EUC-TW, JOHAB, UTF-7
  - refused with "Unsupported encoding": WINDOWS-31J, IBM037, CP65001

  The helper loads GNU libiconv 1.17 as `libiconv-2.dll` (no extra encodings, DOS code pages on as
  `iconv.c` does for MSVC). The wheel's release is not documented; its binary has the DOS
  aliases (CP1125, CP1131) and none of the extra ones (TDS565, ATARIST, EUC-JISX0213), and
  82 declared names plus their converted text give the same verdicts (lane X, speed-x.md).
- **zlib does not change results** (memory parsing never decompresses). It stays on so the
  build is identical; 1.3.1 is the release in the wheel (`lxml/includes/extlibs/zlib.h`).
- Source archives (SHA-256, trees kept unmodified, built out of tree):
  - `libxml2-2.11.9.tar.xz` (download.gnome.org; matches GNOME's `.sha256sum`):
    `780157a1efdb57188ec474dca87acaee67a3a839c2525b2214d318228451809f`
  - `libiconv-1.17.tar.gz` (ftp.gnu.org):
    `8f74213b56238c85a50a5329f77e06198771e70dd9a739779f4c02f65d971313`
  - `zlib-1.3.1.tar.gz` (github.com/madler/zlib release v1.3.1):
    `9a93b2b7dfdac77ceba5a558a580e74667dd6fede4585b91eefb60f03b72df23`

## Behaviour reproduced (lxml 6.0.2 source, `parser.pxi`, `xmlerror.pxi`, `xmlschema.pxi`)

- **Parse options.** The parser is
  `XMLParser(resolve_entities=False, no_network=True, load_dtd=False, huge_tree=False)`.
  lxml's defaults are NOENT|NOCDATA|NONET|COMPACT|BIG_LINES, and
  `resolve_entities=False` removes NOENT. The result is
  `XML_PARSE_NOCDATA | XML_PARSE_NONET | XML_PARSE_COMPACT | XML_PARSE_BIG_LINES`
  = `0x414800` (4278272), reported by hello as `parse_options`.
- **Parsing the document.** `etree.fromstring(bytes)` becomes `xmlCtxtReadMemory` with
  no URL. As in lxml:
  - UTF-32 byte-order marks and detected UCS-4 input select `UTF-32LE`/`UTF-32BE`;
  - CDATA blocks are removed and the start-of-document hook is lxml's (dictionary, ID table);
  - parser errors go through `sax->serror` into an ordered log.
- **Accepted or refused.** The decision copies lxml's `_handleParseResult`: undefined
  entities alone are tolerated, and Latin-1 fallback with invalid characters is refused.
- **Syntax error text.** The message is `str(XMLSyntaxError)`. It is built from the first
  log entry of level ERROR or worse: message, `, line L, column C` (when L, C > 0), then
  Python's `SyntaxError` suffix ` (<string>, line L)`. Example:
  `Premature end of data in tag SCL line 1, line 3, column 1 (<string>, line 3)`.
  The rare fallbacks of `_raiseParseError` are reproduced too:
  - only warnings logged: `Document is not well formed (line 0)`;
  - nothing logged: the bytes repr of `lastError` with ` (line N)`.
- **Message text** (`_LogEntry.message`):
  - "unknown error" for an empty message;
  - otherwise one trailing newline removed and strict UTF-8, or ASCII with `\xNN`
    escapes when the bytes are not UTF-8.
- **Line numbers.** The line is `error->line`; a node's line replaces it only beyond
  INT_MAX.
- **Schemas.** `etree.parse(SCL.xsd)` is followed by `XMLSchema(tree)`: the root is copied
  into a new document, then `xmlSchemaNewDocParserCtxt` and `xmlSchemaParse` run;
  includes resolve relative to the file. Each schema is compiled once per process, up to
  3 (Python's `lru_cache(maxsize=3)`).
- **Validation.** `xmlSchemaValidateDoc` runs with a new context per document. Every
  structured error is kept in order, warnings included; the first 100 are returned plus
  `error_count`.
- **Order.** Python's order is kept: parse, then the DOCTYPE check, then load the schema,
  then validate.

## Protocol (version 1)

Frames use the native worker framing: a 4-byte little-endian length, then UTF-8 JSON,
1 byte to 16 MiB per frame. Every request carries `version: 1` and an `id`, and the reply
repeats that `id`. Requests are handled one at a time, in order. Text a library prints
cannot corrupt the stream, because frames use a private copy of the stdout pipe and
descriptor 1 is pointed at stderr.

| Request | Reply `result` |
|---|---|
| `{"version":1,"id":"1","op":"hello"}` | `{"protocol":1,"capabilities":["xsd_validate_v1"],"libxml2":"2.11.9","libxml2_compiled":"2.11.9","build":"<STUDIO_XSD_BUILD>","parse_options":4278272}` |
| `{"version":1,"id":"2","op":"validate","schema_path":"<abs>\\SCL2007B\\SCL.xsd","source_path":"<abs work file>"}` | see below |
| `{"version":1,"id":"3","op":"stop"}` | no result; the helper exits 0 |

Validate results:

- `{"phase":"parse","doctype":false,"status":"invalid","error_count":1,"errors":[{"line":3,"message":"<str(XMLSyntaxError)>"}]}`
- `{"phase":"parse","doctype":true}`: lxml parsed a DOCTYPE. The engine raises
  `ValueError("DTD declarations are not allowed in SCL")` and does not cache it.
- `{"phase":"schema","doctype":false,"status":"valid"|"invalid","error_count":N,"errors":[...at most 100]}`

Replies:

- Success: `{"id","ok":true,"result":{...}}`.
- Failure: `{"id","ok":false,"error":"<English text>"}`. Causes include an unreadable
  work file, a bundled XSD that cannot be compiled, "Internal error in XML Schema
  validation.", an unknown operation, a wrong protocol version, and a result above
  16 MiB.

A broken frame makes the helper print the reason on stderr and exit 1.

**Why a work file instead of document bytes in the frame.** SCL imports go up to 20 MB.
That does not fit a 16 MiB frame, and base64 would add a third. Chunked frames would
need reassembly state on both sides and a second copy of the document.

The engine already holds the bytes, so it writes them to
`<data>/cache/xsd/work/source-*.scl` and passes the absolute UTF-8 path. The helper opens
it with `_wfopen`, so non-ASCII data folders work, and still parses from memory, exactly
like `fromstring`. The engine deletes the file as soon as the answer arrives; files left
by a crashed engine are removed after one hour.

## Go client (`internal/store/xsd`)

- **Configuration.**
  `xsd.New(xsd.Config{HelperPath, SchemasDir, DataDir, Timeout, ExpectedBuild, MemoryEntries})`
  returns an `scl.SchemaValidator`.
- **Cache key.** The key is
  `sha256(source) NUL schema NUL "2.11.9" NUL digest(schema folder .xsd files)`.
  Verdicts live in a memory LRU and in `<data>/cache/xsd/<sha256(key)>.json`, which holds
  `json.dumps({"status","errors":[{"line","message"}]})` written to a temporary file then
  renamed. A corrupt file is a miss.
- **When the helper runs.** A cache hit never starts the helper. The helper starts on the
  first miss and must pass hello: protocol, capability, libxml2 2.11.9, and the build id
  when `ExpectedBuild` is set.
- **One at a time.** Only one validation runs at a time, matching Python's global lock.
- **Timeout and crashes.** On timeout the helper is killed and restarted by the next
  request. A crash fails the request in progress, and the next request restarts the
  helper.
- **Errors.** Failures are `apperr.RuntimeError` (503) with actionable text, for example
  "SCL schema validation is not available: the studio_xsd helper is missing. Reinstall
  ScadaBench."
- **Seeding.** `xsd.DecodeSeed` and `(*Validator).ImportSeed` load Python-produced
  verdicts, such as `scl/testdata/golden/xsd_seed.json`, into the cache.

## Build on Windows (VS2019 BuildTools, SDK 10.0.19041)

Nothing is downloaded by the build. The three source trees above are unpacked unmodified
in `.local/native/` (`libxml2-2.11.9`, `libiconv-1.17`, `zlib-1.3.1`). The Visual Studio
generator finds the compiler itself; pass the SDK explicitly (quoted in PowerShell): the
10.0.26100 SDK on this machine has no UCRT headers.

1. Static zlib and the GNU libiconv DLL (`cmake/deps`): zlib's own CMakeLists renames `zconf.h`
   inside the source tree and libiconv has no MSVC project, so `cmake/deps/CMakeLists.txt`
   compiles their library sources (zlib's `zlib_SRCS`; libiconv's `lib/iconv.c` and
   `libcharset/lib/localcharset.c`, the objects of `lib/Makefile.in` minus `relocatable.c`,
   only used for installation-prefix relocation) into `libiconv-2.dll` (import library
   `libiconv.lib`). `iconv.h`/`localcharset.h` come from the `*.build.in` templates filled as
   configure does for a Windows DLL; `cmake/deps/libiconv_config.h` replaces configure's
   `config.h` with the macros those two files read.

   ```
   cmake -S studio/native/xsd/cmake/deps -B studio/native/build/xsd-deps -G "Visual Studio 16 2019" -A x64 ^
     "-DCMAKE_SYSTEM_VERSION=10.0.19041.0" -DZLIB_SOURCE=%CD%/.local/native/zlib-1.3.1 ^
     -DLIBICONV_SOURCE=%CD%/.local/native/libiconv-1.17 ^
     -DCMAKE_INSTALL_PREFIX=%CD%/studio/native/build/xsd-deps/install
   cmake --build studio/native/build/xsd-deps --config Release --parallel 4
   cmake --install studio/native/build/xsd-deps --config Release
   ```

   Expected warnings come from the libiconv sources only (C4311 in gperf tables, one
   C4273 for its own `mbrtowc` declaration, one C4090 in `iconvlist`).
2. The helper (`D` = `studio/native/build/xsd-deps/install`):

   ```
   cmake -S studio/native/xsd -B studio/native/build/xsd -G "Visual Studio 16 2019" -A x64 ^
     "-DCMAKE_SYSTEM_VERSION=10.0.19041.0" -DLIBXML2_SOURCE=%CD%/.local/native/libxml2-2.11.9 ^
     -DIconv_INCLUDE_DIR=%D%/include -DIconv_LIBRARY=%D%/lib/libiconv.lib ^
     -DZLIB_INCLUDE_DIR=%D%/include -DZLIB_LIBRARY=%D%/lib/zlib.lib -DSTUDIO_XSD_BUILD=<release id>
   cmake --build studio/native/build/xsd --config Release --parallel 4 --target studio_xsd
   ```

   The configure step prints `studio_xsd: libxml2 2.11.9 with lxml 6.0.2 feature set`
   (`LIBXML_EXPR_ENABLED` is `#if 0` in 2.11.9's own template, as in the wheel). The result
   is `studio/native/build/xsd/Release/studio_xsd.exe` (about 1.7 MB, `/MD`: needs
   `VCRUNTIME140.dll`, the UCRT and `libiconv-2.dll` beside it; no libxml2/zlib DLLs).
   `tools/build_xsd_helper.py` runs steps 1-2, checks hello and copies both files.
3. Smoke test: hello reports `2.11.9` and `parse_options` 4278272; then the parity test.

The Go engine starts `<native folder>/studio_xsd.exe` (`app.Options.XSDHelper`), the folder of
`--native-exe`. Development uses `studio/native/build/Release`, so copy the helper there; the
parity build (`-tags parity`) also reads `STUDIO_XSD_HELPER` because tests pass a scratch
`--native-exe`.

**Licences.** libxml2 (static): MIT (`Copyright` in its tree). zlib (static): zlib licence
(`LICENSE`). GNU libiconv: LGPL 2.1 or later (`COPYING.LIB`; the `iconv` program, not built,
is GPL). It ships as the replaceable `libiconv-2.dll` with its licence, a notice
(`studio/packaging/libiconv-NOTICE.txt`) and its source on download.scadabench.com
(ADR-0083; uploaded by `website/deploy.cmd`).

## Parity test plan

1. **Golden seed (runs now).** `TestGoldenSeedIsServedWithoutHelper` imports all 559
   verdicts from `studio/engine/internal/scl/testdata/golden/xsd_seed.json` (60 invalid),
   which Python produced with lxml 6.0.2. It then reads each one back through a fresh
   validator whose helper path does not exist, so any cache miss would fail.
2. **Helper against Python** (`TestHelperMatchesPython`, opt-in: `STUDIO_XSD_HELPER` =
   the built exe, `STUDIO_XSD_PARITY` = dump folders). `internal/store/xsd/testdata/
   parity_dump.py` (studio/.venv) records Python's verdicts *with the validated bytes*:
   `corpus` (every validate_schema call of golden_gen.py, checked against the committed
   seed), `edge` (items 3-4), `seed` (a recorded seed plus its files) and, as a pytest
   plugin, the whole Python suite. The test sends every document through the Go client to
   the real helper with an empty cache and compares `status` and every `{line, message}`
   byte for byte. Any difference blocks release. Lane X: 907 / 907 identical.
3. **Malformed and edge inputs** (`testdata/parity_edge.py`; Python results recorded with
   the venv's lxml, compared byte for byte):
   - unclosed tags, an empty document and leading junk;
   - an undefined entity and an undefined namespace prefix;
   - Latin-1 bytes without a declaration, and invalid UTF-8 in attribute values;
   - every encoding in the iconv list above;
   - UTF-16 and UTF-32 with and without BOM;
   - CDATA, a DOCTYPE (doctype flag), lines above 65535 (BIG_LINES), and more than 100
     schema errors (`error_count`).
4. **Large documents.** A 20 MB SCL goes through the work-file path and must match
   Python's verdict.
5. **Protocol.** The Go package tests use a fake helper (the test binary re-executed). They
   cover framing, hello refusal (other libxml2 or build), a crash followed by a restart,
   timeout and kill, concurrent callers sharing one helper, DOCTYPE giving `ValueError`
   without caching, and the missing-helper error.
