DONE


# OCF block size allows panic and memory exhaustion before validation

## Summary

The OCF decoder trusts the per-block `size` value read from untrusted container data and uses it directly as a Go slice length. A crafted Avro Object Container File can make `Decoder.HasNext()` panic with a negative block size or attempt a very large allocation with a large positive block size.

## Affected code

- `repos/avro/ocf/ocf.go:203` reads attacker-controlled `count`.
- `repos/avro/ocf/ocf.go:204` reads attacker-controlled `size`.
- `repos/avro/ocf/ocf.go:209` calls `make([]byte, size)` when `count > 0`.
- `repos/avro/ocf/ocf.go:221` calls `make([]byte, size)` when `count == 0 && size > 0`.

The normal Avro value reader has explicit byte/string allocation limits in `repos/avro/reader.go:280` through `repos/avro/reader.go:310`, backed by `Config.MaxByteSliceSize`. OCF block buffering does not apply those limits before allocating the compressed block buffer.

## Attack path

An attacker supplies an OCF file with a valid header and a malicious block header:

1. `NewDecoder` accepts the valid OCF header.
2. Application code calls `HasNext()`, which calls `readBlock()`.
3. `readBlock()` reads `count` and `size` from the file.
4. If `count > 0`, any negative `size` reaches `make([]byte, size)` and panics with `runtime error: makeslice: len out of range`.
5. If `count > 0` and `size` is huge, the decoder attempts to allocate that many bytes before verifying that the input actually contains that block.
6. If `count == 0` and `size` is huge, the skip path still allocates `make([]byte, size)`, so a non-data block can be used for memory exhaustion.

This happens before sync-marker validation and before `reader.Error` can be returned through the library error path.

## Impact

This is a denial-of-service vulnerability for services that decode OCF data from untrusted users, queues, object storage, partner feeds, or ingestion pipelines. The attacker does not need to provide a large physical file for the negative-size panic case. For large positive sizes, the attacker can force memory pressure or process termination by advertising a large block size.

## Security expectation failure

The task requirements are not met:

- Negative `size` is not rejected cleanly; it can panic.
- Excessive `size` is not bounded before allocation.
- `count > 0` with malformed `size` is not rejected before allocation.
- `count == 0` with large `size` still allocates the skipped block.
- Existing tests cover invalid blocks and configured value-size limits, but not OCF block sizes of zero, negative, max-int, or very large values.

## Recommended remediation

Validate OCF block headers before allocation:

- Reject `size < 0` with `reader.Error`.
- Reject positive `size` above a configurable OCF block-size limit.
- For skipped blocks, stream-skip with bounded buffering instead of allocating the entire advertised block.
- Add regression tests for negative size, zero count with large size, positive count with negative size, `math.MaxInt64`, and values above the configured limit.

Consider wiring the limit through decoder configuration rather than reusing `MaxByteSliceSize`, because OCF block sizes describe encoded block payloads, not individual Avro byte/string values.
