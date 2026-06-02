NOT_STARTED

Title: Schema parsing accepts unbounded nesting and named-type cache growth

Impact: High. If schemas come from a registry, uploaded OCF metadata, tenant configuration, or any other untrusted source, a malicious schema can consume excessive CPU, memory, or stack. This can crash schema-loading services before record data is processed.

Attack path:
- Attacker provides a deeply nested schema, or a schema with a very large number of named records/enums/fixed types and fields.
- `ParseBytesWithCache()` fully unmarshals JSON into `any`, then recursively descends through `parseType`, `parseComplexType`, `parseRecord`, `parseArray`, `parseMap`, and `parseUnion`.
- Parsed named schemas are copied into the caller-provided cache.
- There is no maximum schema byte size, recursion depth, union width, field count, or named schema count.

Evidence:
- `schema_parse.go`, `ParseBytesWithCache`, unmarshals the entire schema and then calls recursive parsing without a depth budget.
- `schema_parse.go`, `parseType` recursively handles maps, arrays, records, and unions.
- `schema.go`, `SchemaCache`, is an unbounded `sync.Map`; successful parses can add arbitrary named schemas.
- `ocf/ocf.go`, `readHeader`, obtains schemas from file metadata, so untrusted OCF files can trigger schema parsing.

Exploit sketch:
- Submit an OCF file whose `avro.schema` metadata is a deeply nested array/map/record schema.
- Alternatively, return a huge schema from a configured schema registry.
- Parsing consumes memory for the JSON tree and schema objects, or overflows the goroutine stack via recursive descent.

Recommended remediation:
- Introduce parser limits for input byte length, recursion depth, field count, union branch count, and named-schema count.
- Expose conservative defaults and allow callers to opt into larger limits.
- Avoid committing parsed schemas into shared caches until the entire parse has passed all limits.
