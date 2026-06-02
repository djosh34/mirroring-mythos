NOT_STARTED

Title: OCF decoder trusts block size and can be forced into unbounded allocation

Impact: High. A single malicious Avro Object Container File can make a service allocate attacker-controlled memory before any decoded record is returned. In a production ingestion path this can cause process OOM, container eviction, or node-level memory pressure.

Attack path:
- Attacker supplies an OCF file or stream consumed by `ocf.NewDecoder`.
- `ocf.Decoder.HasNext()` calls `readBlock()`.
- `readBlock()` reads `count := d.reader.ReadLong()` and `size := d.reader.ReadLong()`.
- If `count > 0`, it immediately executes `data := make([]byte, size)` and reads that many bytes.

Evidence:
- `ocf/ocf.go`, `Decoder.readBlock`, allocates `make([]byte, size)` without an upper bound.
- `size` comes directly from Avro block metadata in the untrusted file.
- The lower-level reader has a configurable `MaxByteSliceSize` for byte/string values, but that protection is not applied to OCF block buffers.

Exploit sketch:
- Emit a valid OCF header with a trusted-looking schema.
- Emit a block header with `count = 1` and `size = 0x40000000` or larger.
- The decoder attempts to allocate the declared block buffer before it can reject the file based on sync marker or payload validity.

Recommended remediation:
- Add a decoder option for maximum compressed block size and enforce it before allocation.
- Reject negative or implausibly large sizes with a reader error.
- Consider streaming decompression into a bounded reader instead of fully materializing compressed blocks.
