# 4. Golden SCL corpus

The SCL layer is where a Go rewrite most easily changes behaviour without any test
noticing: it feeds the native workers (`cfg`), the frontend (model JSON), the user's files
(export) and the user's eyes (error text). Every corpus file is run through both engines
and every output is compared.

## Public fixtures that exist today

| File | Content | Used by |
| --- | --- | --- |
| `scd/open_substation.scd` (43 KB) | 4 IEDs + `Substation` S12 "Baden", 2007A | 17 test references (SLD, signal flow, health, test mode, GOOSE) |
| `scd/model_substation.scd` (167 KB) | OpenSCD example, 6 IEDs | 4 references; source of the demo station |
| `scd/protection_relay.scd` (36 KB) | 1 IED, 2007A | not referenced by tests yet - add to corpus |
| `marketing/videos/demo/scadabench_demo.scd` (136 KB) | public demo station, 6 IEDs, LNode links (built by `marketing/videos/demo_station.py`) | videos; **the** golden/UI station |
| `make_template(name)` (code) | built-in sample IED, 2007B4, XCBR/XSWI/GGIO, 2 RCBs | 19 test files |
| `tests/unit/goose_samples.py` | generated GOOSE senders/receivers incl. vendor `ge:Function` attributes | GOOSE unit tests |
| `tests/unit/sld_fixtures.py` | generated `Substation` sections, `esld:x/y` | SLD tests |
| `tests/unit/test_real_world_scl.py::variant` | vendor patterns as text substitutions on the template (PhyComAddr, unnamed RCB) | import |
| `tests/frontend/fixtures/model_substation_diagram.json` | diagram built from model_substation | frontend |
| `tests/fixtures/licensing/go_vector.json` | Go<->Python license vector | licensing (pattern to copy) |
| `schemas/SCL2003`, `SCL2007B`, `SCL2007B4` | IEC XSDs (unmodified) | schema validation |

Private, never in the corpus: `test_cids/` (git-excluded via `.git/info/exclude`, used by
`test_goose_link_map_vendor_files.py` with `skipif`). A local-only command
`parity/corpus.py --private test_cids` may run both engines over them and print **only**
case hashes, IED counts and pass/fail - no names, no outputs written to the repo.

## Corpus generator

`studio/tools/parity/scl_corpus.py` (deterministic, Python) writes
`studio/tests/fixtures/scl_corpus/<case>.scd` plus `manifest.json`
`{case, purpose, base, expect: accept|warn|reject, sha256}`. Most cases are
`variant()`-style substitutions on `make_template` or the public files, so every case is
small, public and reviewable. Files are committed (byte-exact inputs matter: BOMs and
encodings cannot survive a regenerate-on-the-fly through text APIs).

## Synthetic edge cases to add (~120 cases)

**Bytes and encoding**
- UTF-8 BOM; UTF-8 without declaration; UTF-16 LE and BE with BOM (expat accepts, then
  `source.decode("utf-8-sig")` raises - the exact `UnicodeDecodeError` text reaches the
  user as a 400); `encoding="ISO-8859-1"` with `é` in `desc` (parses, then the same decode
  failure); `encoding="windows-1252"` (expat: unknown encoding); declared UTF-8 with an
  invalid byte; `US-ASCII` with a numeric character reference `&#233;`.
- CRLF vs LF vs CR line endings; tabs and newlines inside attribute values (normalised to
  spaces by the parser, re-escaped as `&#9;`/`&#10;` by lxml on export); trailing
  whitespace/garbage after the root; empty file; 0 bytes; an HTML page; a ZIP renamed .scd.
- Size limits: exactly 20 MB (`MAX_XML`), 20 MB + 1 byte, 100 MB upload limit (413 text).

**XML features**
- `<!DOCTYPE SCL>` without subset (allowed); internal subset with an entity
  (defusedxml `EntitiesForbidden(...)` text); external entity `SYSTEM "file:///..."`;
  billion-laughs; parameter entities; XInclude element (kept as data, not processed).
- Comments: before root, inside `IED`, inside `DataTypeTemplates`, between `Val`
  elements, containing `--`-adjacent text, non-ASCII comment text - must survive export.
- Processing instructions; CDATA in `Val`; `xml:space="preserve"`; entity-escaped
  `&lt;&amp;&gt;&quot;&apos;` in `desc`.
- Namespaces: `scl:` prefix on every element; default namespace redeclared on a child;
  2003 vs 2007 namespace use; foreign attributes (`ge:Function`, `esld:x`); `Private`
  elements with foreign children and `type` - byte-preserved on export.

**Editions and schema**
- `version` absent / `2003` / `2007` + `A` / `B` / `B` + `release="4"` / unknown
  (`status: unsupported`); schema-invalid but importable (validation errors listed with
  libxml2 text and line numbers, import still works or fails exactly as Python decides).

**Names and identity** (`identifier()` uses `{value!r}`: Go must reproduce Python `repr`)
- IED name with space, leading digit, non-ASCII (`'Ä1'`), embedded quote (`"it's"` ->
  repr picks double quotes), missing name (`None`), 65+ chars, duplicate IED names,
  `selected_names` empty, duplicated, unknown, non-string.
- LD `inst` edge values; LN `prefix` with digits; `lnClass` unknown; duplicate LN.

**Type resolution**
- Missing `DataTypeTemplates`; unresolved `lnType`/`type` (`Unresolved {tag} {id}`);
  recursive `DAType` cycle; unknown basic type; unknown FC; `count` on DA/BDA (arrays ->
  "Array attribute ... needs array-attribute support"); `DOI`/`SDI`/`DAI` with `ix`;
  arrays of SDO; `sGroup` values and several `Val` (export refuses with the setting-group
  message); `valKind`/`valImport` present.
- Enums: `EnumType` with negative `ord`, gaps, duplicate `ord`, empty literal, non-ASCII
  literal, enum referenced but missing, `Val` text not in the enum, `Val` given as ord.

**Values** (`initial_value()` per basic type)
- FLOAT32/64: `1e3`, `1E-7`, `-0`, `0.1`, ` 2.5 ` (whitespace), `2,5`, `NaN`, `INF`,
  `-INF`, `1e309`, `3.4028236e38` (FLOAT32 overflow), 17-significant-digit values.
- INT8..INT64, INT8U..INT32U: min-1, min, max, max+1, `+5`, `05`, `0x10`, `1.0`.
- BOOLEAN: `true/false/1/0/TRUE/True/yes/empty`.
- Timestamp, Quality, Dbpos/Tcmd, VisString64/255 over length, Unicode255 with emoji
  (astral -> surrogate pair in `ensure_ascii` JSON), Octet64 hex/odd length, PhyComAddr,
  ObjRef, Check, EntryTime.

**Unicode text**
- `desc` with emoji, Arabic RTL, combining marks, zero-width joiner, `U+2028`, a
  4 KB description, control characters as character references (`&#1;` - invalid XML 1.0).

**Communication and control blocks**
- Missing `Communication`; `ConnectedAP` without IP; duplicate IPs; IPv6; `GSE`/`SMV`
  `MAC-Address` upper/lower/colon forms; `APPID` `0x`/hex/decimal; `VLAN-ID` and
  `VLAN-PRIORITY` edges; GSEControl without `datSet`; SampledValueControl variants;
  `ReportControl` unnamed (warning text), `RptEnabled max` 0/absent, `indexed`, buffered
  flags; `ExtRef` with `intAddr` containing `|`, missing `iedName`, pointing outside the file.
- Multiple `AccessPoint`/`Server` (preview `unavailable_reason` text), `ServerAt`.

**Substation section**
- No `Substation`; several substations; `ConnectivityNode` names with spaces; `Terminal`
  `cNodeName="grounded"`; `LNode` to absent IEDs; `esld:x/y` floats and negatives;
  power transformer windings (open_substation style).

## Outputs: byte-for-byte vs JSON-for-JSON

| Output | Route / producer | Compare |
| --- | --- | --- |
| `cfg` per selected IED (libiec61850 model text fed to `studio_ied.exe`) | `import_scl(...)[i]["cfg"]`; black-box via IPC tap or `scl-dump` | **bytes** |
| `source` stored per IED (`decode("utf-8-sig")`, BOM dropped) | `GET /ieds/<id>/source` | **bytes** |
| Exported SCL with overrides (lxml: `<?xml version='1.0' encoding='UTF-8'?>` single quotes, attribute order kept, `<x/>` form, comments/Private kept, `&#13;`/`&#10;` escapes) | `POST /ieds/<id>/export-scl`, `projects.export` | **bytes** |
| Error text: `StudioError` messages, parser messages (expat/defusedxml), `UnicodeDecodeError`, pydantic details | 400/409 bodies `{error, code, details}` | **bytes** of `error`, JSON of `details` |
| Warnings list (`model["warnings"]`) | import response, project | **bytes** per string, order exact |
| Schema validation result `{status, schema, errors[]}` (libxml2 text + line) | `POST /scl/validate` | JSON; `errors[*]` strings **bytes** |
| `project.json` after import | project download | **bytes** after id rule |
| IED preview list | `POST /scl/preview` | JSON |
| Import model (signals, datasets, reports, goose publishes/subscribes, sv, enums, controls) | `GET /project`, `GET /ieds/<id>/signals` | JSON (key order + numtype strict) |
| Substation sources, SLD from-SCL preview (float positions) | `GET /project/substation-sources`, `POST /project/diagram/from-scl/preview` | JSON; floats compared by Python `repr` |
| Signal flow / GOOSE link map / report clients | `GET /signal-flow`, `/signal-flow/sources`, `/goose/links` | JSON |
| CSV derived from SCL (signal-flow edge CSV, gateway `points.csv`) | downloads | **bytes** |

## Implementation consequences to decide in the ADR

- **XML parsing**: Python uses defusedxml over CPython's expat. Go `encoding/xml` differs in
  encodings (needs a charset reader; expat knows only UTF-8/16, ISO-8859-1, US-ASCII),
  entity/DOCTYPE handling, line/column numbering (expat columns are 0-based) and message
  text. Either Go maps its errors to expat's codes and text (corpus proves it) or the
  parse runs through expat in a native helper. Same for **XSD validation** (lxml 6.0.2 /
  libxml2): no pure-Go equivalent produces identical messages; the realistic options are a
  native `studio_scl.exe` helper linking libxml2 (fits the worker model, `CGO_ENABLED=0`
  stays) or cgo. Either way the corpus is the acceptance test.
- **Serialisation**: Go must reproduce lxml output rules exactly, including attribute
  order and namespace prefixes taken from the source document.
- **Python `repr`/`str` emulation**: identifier errors (`{value!r}`), float `str()` in cfg
  and CSV, `str(KeyError)` quoting.

## Commands

```powershell
python studio/tools/parity/scl_corpus.py --write            # regenerate inputs + manifest
python studio/tools/parity/corpus.py --engines python,go     # all outputs, both engines
python studio/tools/parity/corpus.py --engines python,python # self-diff (must be clean)
cd studio/core; go test ./internal/scl/... -run Corpus       # Go reads the same files + Python goldens
```

`corpus.py` writes `<scratch>/corpus/<engine>/<case>/{preview,validate,import,export-*,cfg-*,error}.{json,txt,scd}`
and exits 0 only when every byte-compare file is identical and every JSON file passes the
section 3 rules. Committed Python goldens live in `studio/tests/golden/scl/` so the Go
unit test can compare without Python installed.
