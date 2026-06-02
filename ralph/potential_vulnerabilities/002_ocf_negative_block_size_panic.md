NOT_STARTED

Title: OCF decoder can panic on negative block size

Impact: High. Malicious input can crash a service if it decodes OCF data without an outer panic recovery boundary. This is a denial-of-service risk for ingestion APIs, file processors, and message consumers.

Attack path:
- Attacker supplies an OCF file with a positive block count and a negative block size.
- `ocf.Decoder.readBlock()` reads both fields from the untrusted stream.
- With `count > 0`, it calls `make([]byte, size)` using the negative `int64` size.
- Go panics for a negative slice length.

Evidence:
- `ocf/ocf.go`, `Decoder.readBlock`, has no validation that `size >= 0` before allocation.
- The method returns a count and records reader errors for some invalid conditions, but the panic occurs before that error path.

Exploit sketch:
- Construct an OCF file with a normal header and sync marker.
- Encode block count as `1`.
- Encode block size as `-1`.
- Call `HasNext()` on the decoder; allocation panics.

Recommended remediation:
- Validate `size < 0` immediately after reading the block header and set `d.reader.Error`.
- Add regression tests for negative block size with both `count > 0` and `count == 0`.
