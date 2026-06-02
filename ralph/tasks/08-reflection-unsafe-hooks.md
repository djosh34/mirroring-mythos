NOT_STARTED


# Audit Reflection Unsafe And Hook Execution Paths

Review reflection and `unsafe` codec paths.

Focus:
- `codec_union.go`
- `codec_marshaler.go`
- `codec_enum.go`
- `codec_map.go`
- type converters
- union converters
- `encoding.TextMarshaler` and `encoding.TextUnmarshaler`

Verify:
- Malformed data cannot trigger panics through type confusion or nil handling.
- User-provided hooks receive only validated data in expected forms.
- Errors from hooks are propagated cleanly.
- Union type resolution cannot instantiate unexpected application types from untrusted schema names.
- Tests cover nil pointers, invalid union indexes, bad enum values, bad text unmarshaling, and converter errors.
