# Native worker and independent client

Build on Windows x64 with CMake and Visual Studio 2019 Build Tools. The pinned
sources belong under `.local/native/` at the repository root:

- libiec61850 v1.6.1, commit `a13961110b8238d2d8ea577c1fb7592ba3017ad8`.
- nlohmann/json v3.12.0, commit `55f93686c01528224f448c19128836e7df245f72`.

From the repository root, after all workers using the normal output have stopped:

```powershell
cmake -S studio/native -B studio/native/build -G "Visual Studio 16 2019" -A x64
cmake --build studio/native/build --config Release --target studio_ied studio_probe --parallel 4
```

Never rebuild/replace a worker executable used by an active run. For isolated
candidate checks, use a distinct build folder (wait for configuration to finish):

```powershell
cmake -S studio/native -B studio/native/build/array-fix -G "Visual Studio 16 2019" -A x64
cmake --build studio/native/build/array-fix --config Release --target studio_ied studio_probe --parallel 4
$env:STUDIO_NATIVE_BIN = (Resolve-Path studio/native/build/array-fix/Release).Path
studio/.venv/Scripts/python.exe -m pytest -c studio/pyproject.toml studio/tests/protocol -q
```

`STUDIO_NATIVE_BIN` selects protocol-test binaries only; it does not change the
application or packaging default paths. Clear that variable for normal-build tests.
Original vendor sources remain unmodified. CMake checks the audited parser hash
and applies the documented correction to a generated copy under the build folder.
See [array-parser decision](../../docs/decisions/ADR-0006-native-array-parser.md).

The probe supports `read`, `control`, `reports` and `browse`. Browsing obtains the
model, type descriptions and dataset membership over TCP/MMS, not from source XML.
