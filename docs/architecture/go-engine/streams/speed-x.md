# Speed lane X: studio_xsd.exe built and proven against lxml

Branch `speed-x` from master 527f0d2. Closes gap 1 of [0132](../../../changes/2026/0132-speed-lane-i-composed-go-server.md)
(`studio_xsd.exe` not built). Owns `studio/native/xsd` (new `cmake/deps`), the XSD parity
test in `internal/store/xsd` and two test seams. The Python backend, the helper's C sources,
packaging and `studio/native/build/Release` are unchanged. No EXE of the app was built.

## Sources (downloaded by the integrator with the owner's permission)

Unpacked unmodified in the main checkout's `.local/native/`; every file was compared with its
archive (`tarfile`, byte for byte) before the build: no file missing or changed (the antivirus
did not remove anything this time).

| Tree | Archive SHA-256 | Files |
|---|---|---|
| libxml2-2.11.9 | `780157a1...451809f` (= GNOME `.sha256sum`) | 3970 |
| libiconv-1.17 | `8f74213b...d971313` | 1036 |
| zlib-1.3.1 | `9a93b2b7...b72df23` | 253 |

## Build (VS2019 BuildTools 14.29, SDK 10.0.19041, `--parallel 4`)

Full commands: [studio/native/xsd/README.md](../../../../studio/native/xsd/README.md), "Build on Windows".

1. `cmake/deps/CMakeLists.txt` (new) builds static `zlib.lib` and `libiconv.lib` out of tree:
   - zlib: zlib's own CMakeLists renames `zconf.h` *inside the source tree* on out-of-tree
     builds, so the 15 `zlib_SRCS` files are compiled directly with the shipped `zconf.h`.
   - GNU libiconv has no MSVC project: `lib/iconv.c` + `libcharset/lib/localcharset.c` (the
     objects of `lib/Makefile.in`; `relocatable.c` is DLL-only), `iconv.h`/`localcharset.h`
     from the `*.build.in` templates with configure's static-build values, and a 33-line
     `libiconv_config.h` instead of configure's `config.h` (ENABLE_EXTRA 0,
     WORDS_LITTLEENDIAN, empty ICONV_CONST, MSVC has mbrtowc/wcrtomb/mbsinit). `iconv.c`
     itself turns on the DOS code pages for `_MSC_VER`. Warnings: libiconv sources only.
2. `studio/native/xsd` configured as documented: FindIconv reports 1.17, FindZLIB 1.3.1,
   **check_features passes** (`studio_xsd: libxml2 2.11.9 with lxml 6.0.2 feature set`).
   `LIBXML2_WITH_EXPR` is forced ON in 2.11.9's CMakeLists, but its `xmlversion.h.in` has
   `#if 0` for EXPR, so the generated header matches the wheel. No warning in the helper's
   own files at /W4.
3. Result: `studio/native/build/xsd/Release/studio_xsd.exe`, 1.75 MB, imports only
   KERNEL32, WS2_32, VCRUNTIME140 and the UCRT (like the other `/MD` workers). Hello:
   `libxml2 2.11.9`, compiled 2.11.9, `parse_options` 4278272, protocol 1.

Which libiconv the wheel uses is still not documented. Evidence for 1.17-equivalent: the
wheel's `etree.pyd` contains the DOS aliases (CP1125, CP1131) and none of the extra
encodings (TDS565, ATARIST, EUC-JISX0213, RISCOS-LATIN1); every probed name and conversion
below gives the same verdict.

## Parity (byte for byte: status, every line, every message)

`internal/store/xsd/parity_test.go` (`TestHelperMatchesPython`, opt-in) sends each recorded
document through the Go client (`xsd.New`, empty cache per folder) to the real helper.
`testdata/parity_dump.py` records Python's verdicts with the bytes (studio/.venv, lxml 6.0.2).

| Set | Documents | Identical | Invalid | Notes |
|---|---|---|---|---|
| corpus (every validate_schema call of golden_gen.py) | 559 | 559 | 60 | re-recorded today = committed `xsd_seed.json` 559/559 |
| Python suite in process (`-p parity_dump`, unit + integration, 446 passed) | 58 | 58 | 6 | = speed-i2's 58 recorded verdicts, 58/58 same |
| edge (`parity_edge.py`) | 250 | 250 | 230 | + 2 DOCTYPE -> ValueError, not cached |
| stress bench seeds `stress2/xsd_seed.json`, `xsd_seed4.json` + their SCDs | 20 + 20 | 40 | 0 | |
| **Total** | **907** | **907** | | 0 differences |

Edge set: empty/blank, junk and blank before the declaration, unclosed, mismatched,
duplicate attribute, content after the root, undefined entities (attribute and text),
undefined prefix, NUL, `&#0;`, Latin-1 without and with a declaration, invalid UTF-8,
CDATA, comment/PI, DOCTYPE (plain and internal subset), wrong/no namespace, enum error,
150 schema errors (100 kept), lines above 65535 (schema and syntax), UTF-16/32 LE/BE with
and without BOM and declaration, UTF-8 BOM, SCL2003/SCL2007B, a 20 MB document (work-file
path), and **82 declared encodings**, each as a minimal document, with Python-encoded sample
text in an attribute the error echoes, and with raw bytes 0xA0-0xFF (iconv conversion
errors such as "Growing input buffer" and "input conversion failed" match too).

Encodings (minimal document), Python = helper for all 82: 11 refused with "Unsupported
encoding" (ATARIST, BIG5-2003, CP65001, EBCDIC-US, EUC-JISX0213, IBM-1047, IBM037,
RISCOS-LATIN1, TDS565, WINDOWS-31J, bogus-enc); 59 parse and reach the schema, including the
README list (GEORGIAN-PS, MULELAO-1, JAVA, C99, TCVN, CP1133, KOI8-T, PT154, RK1048,
windows-1252, Shift_JIS, GB18030, BIG5-HKSCS, EUC-TW, JOHAB, UTF-7), the DOS pages (CP437,
CP850, CP866, CP1125, CP1131) and `char`; 12 fail earlier with the same parser text (names
with `:` are not XML encoding names, an empty name, and UTF-16/32, UCS-2/4, WCHAR_T,
ISO-IR-165 declared on single-byte text).

Control: a copy of the pytest dump with one message, one line and one status changed fails
with exactly those 3 differences.

## Wiring

- Product: unchanged. `app.Options.XSDHelper` defaults to `<NativeDir>/studio_xsd.exe`,
  `NativeDir` = folder of `--native-exe` (else `native` beside the server, else
  `studio/native/build/Release`). So the helper must sit beside `studio_ied.exe`.
- Parity build only (`-tags parity`): `STUDIO_XSD_HELPER` overrides the path, because tests
  pass a scratch `--native-exe`.
- `projectapi` tests: `STUDIO_XSD_HELPER` replaces the recorded verdicts with the real helper.

| Check (real helper, no `STUDIO_XSD_SEED`) | Result |
|---|---|
| `projectapi` Flask parity (68 recorded requests + design routes) | pass, helper started |
| `STUDIO_ENGINE=go` pytest, whole suite | **456 passed, 104 skipped, 0 failed** (with seeds before: 456/104) |
| same, SCL export/SLD import tests with a missing helper (control) | 4 errors (503), so the helper is really used |
| `parity.corpus.run dual --engines python-http,go`, all 182 cases (preview, validate, import, project, signals, source, export-scl, download over HTTP) | **182 / 182 with 0 differences** (1231 s) |
| `go vet` (also `-tags parity`), `check_file_lengths.py` | clean / PASS |
| `verify.py` | PASS except GOOSE engine `internal/uplink`: "file being used by another process" on its temp test exe (Windows lock); that package alone passes on rerun, untouched here |

`internal/scl` corpus tests cannot use the helper (`store/xsd` imports `scl`: import cycle);
they stay on the seed, whose 559 verdicts are proven identical above.

## Packaging (not changed; for the Go engine release)

- `compile_protected.py` / `studio_protected.spec` (and `studio.spec`) must build and list
  `studio_xsd.exe` under `native/` beside the workers: deps + helper CMake steps above, with
  `-DSTUDIO_XSD_BUILD=<release id>`, and `required` must include it.
- Pass the same id as `xsd.Config.ExpectedBuild` (`wire_project.go` passes none today), so a
  stale helper is refused at hello.
- The helper has no licence-enforcement gate like the other workers (owner's call; it only
  answers validation for the engine).
- Ship licences: `.local/native/libxml2-2.11.9/Copyright` (MIT) -> `licenses/libxml2`,
  `.local/native/zlib-1.3.1/LICENSE` -> `licenses/zlib`,
  `.local/native/libiconv-1.17/COPYING.LIB` (LGPL 2.1) -> `licenses/libiconv`.
- **LGPL and static linking (owner decides).** LGPL 2.1 section 6 lets a program link the
  library statically only if users can relink it with a modified libiconv: ship the
  helper's object files (or its source, `studio/native/xsd` is already plain C) plus these
  build steps, keep the licence text, and offer the libiconv source. Alternatives: link
  libiconv as a DLL (`libiconv.dll` beside the helper, users may replace it; changes only
  the deps CMake), or publish `studio/native/xsd` itself. Either keeps the verdicts.
- Development: copy `studio/native/build/xsd/Release/studio_xsd.exe` into
  `studio/native/build/Release` (not done here: that folder belongs to the owner's app).

## Open items

1. Owner: LGPL route for libiconv (above).
2. Integrator: put the helper in `studio/native/build/Release` (or teach the native build
   to produce it there), then drop `STUDIO_XSD_SEED` from the go-mode runs.
3. Packaging steps above when the Go engine ships; set `ExpectedBuild`.
4. The wheel's libiconv release stays inferred (1.17 behaviour proven on 82 names).
