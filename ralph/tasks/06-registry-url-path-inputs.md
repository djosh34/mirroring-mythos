NOT_STARTED


# Audit Schema Registry URL And Path Inputs

Review `registry/client.go`.

Focus:
- `NewClient(baseURL string)`
- methods accepting `subject`
- `path.Join`
- `c.base.Parse(path)`
- `Client.request`

Verify:
- User-controlled `baseURL` cannot be used for SSRF in consuming applications without explicit trust controls.
- Subject names containing `../`, leading slashes, encoded slashes, query strings, or fragments do not alter the intended endpoint.
- Registry response bodies have reasonable read/decode limits.
- Basic auth credentials are not sent to unintended hosts.
- Tests cover path normalization and malicious subject names.
