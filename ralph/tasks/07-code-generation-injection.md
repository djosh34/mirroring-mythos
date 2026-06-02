NOT_STARTED


# Audit Code Generation Injection

Review `gen/gen.go`, `gen/output_template.tmpl`, and command code under `cmd/avrogen`.

Focus:
- schema names
- field names
- enum symbols
- docs
- tags
- logical type imports
- custom templates
- metadata

Verify:
- Untrusted schemas cannot inject arbitrary Go code into generated output.
- Generated identifiers are sanitized and collision-safe.
- Generated struct tags cannot be escaped into arbitrary tags or code.
- Custom template usage is treated as trusted-only or documented as unsafe with untrusted input.
- Tests cover malicious schema names, field names, enum symbols, docs, tags, and imports.
