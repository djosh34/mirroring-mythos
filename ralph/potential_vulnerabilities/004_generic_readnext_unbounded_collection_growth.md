NOT_STARTED

Title: Generic ReadNext bypasses collection size limits

Impact: High. Services using generic decoding can be forced to allocate attacker-controlled arrays, maps, and nested object graphs without the configured typed-decoder slice limit.

Attack path:
- Application calls `Reader.ReadNext(schema)` or uses a decode path that resolves into generic `any` values.
- For arrays, `ReadNext` initializes `arr := []any{}` and appends every attacker-declared element.
- For maps, it initializes `obj := map[string]any{}` and inserts every attacker-declared key.
- No cumulative item limit or allocation budget is checked.

Evidence:
- `reader_generic.go`, array case, appends in `ReadArrayCB`.
- `reader_generic.go`, map case, inserts in `ReadMapCB`.
- `reader_generic.go`, `ReadArrayCB` and `ReadMapCB` loop directly over block counts from `ReadBlockHeader()`.
- `codec_array.go` enforces `Config.MaxSliceAllocSize` for typed slice decoding, showing the intended protection exists but is not applied here.

Exploit sketch:
- Use a schema such as `{"type":"array","items":"null"}` or `{"type":"map","values":"null"}`.
- Supply block headers with very large counts and compact null values.
- The generic decoder grows `[]any` or `map[string]any` until the process exhausts memory.

Recommended remediation:
- Enforce `MaxSliceAllocSize` or a new generic collection limit in `ReadArrayCB`, `ReadMapCB`, and `ReadNext`.
- Stop decoding immediately when cumulative counts exceed the configured budget.
- Add tests proving generic decoding honors the same resource limits as typed decoding.
