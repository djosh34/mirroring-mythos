NOT_STARTED


# Audit Schema Parser Resource Exhaustion

Review `schema_parse.go`, `schema.go`, `schema_walk.go`, and schema cache behavior.

Focus:
- `ParseBytesWithCache`
- recursive parsing of records, arrays, maps, unions, and references
- `DefaultSchemaCache`
- default values and custom props

Verify:
- Deeply nested schemas cannot cause stack exhaustion.
- Very large schemas cannot cause uncontrolled CPU or memory use.
- Circular or repeated references terminate safely.
- Shared/default schema cache behavior is safe across tenants or request boundaries.
- Tests cover deep nesting, many fields, many union branches, large defaults, and circular references.
