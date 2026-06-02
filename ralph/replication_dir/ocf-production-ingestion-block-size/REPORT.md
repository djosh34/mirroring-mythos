CHECKED_AND_WORKING

# OCF block size panic survives production-style ingestion

## Summary

I re-tested the reported Avro Object Container File block-size issue against `repos/avro` with a larger example that uses the library for its intended purpose: a partner order-event ingestion path that reads `.avro` Object Container Files with `ocf.NewDecoder`, checks source-file properties, validates OCF metadata, decodes typed records, and checks decoded business data.

The vulnerability is real. A crafted OCF file can pass normal header and metadata validation, then panic inside the library when application code calls the normal public `dec.HasNext()` loop. The application does not intentionally create a vulnerable decoder path; the panic is caused by `repos/avro/ocf/ocf.go` allocating a slice from an attacker-controlled block `size` before rejecting negative or unreasonable values.

## What I Built

Replication directory:

```text
ralph/replication_dir/ocf-production-ingestion-block-size
```

The program creates a realistic partner ingestion scenario:

1. Writes a legitimate OCF file using `ocf.NewEncoder` with a record schema named `OrderEvent`.
2. Adds normal metadata identifying the expected producer.
3. Decodes the legitimate file using `ocf.NewDecoder` and the standard `for dec.HasNext() { dec.Decode(...) }` API.
4. Applies defensive application checks:
   - `.avro` extension required.
   - File must be a regular file.
   - File size must be non-zero and at most 1 MiB.
   - Decoder header must parse successfully.
   - Metadata key `producer` must equal `partner-orders-v1`.
   - OCF codec must be `null`.
   - Parsed schema must be the expected `OrderEvent` schema.
   - Decoded records must contain expected non-empty IDs, positive amounts, and `EUR` currency.
   - The opened file is wrapped in an `io.LimitReader`.
5. Creates a crafted variant by changing only the first data block's encoded block-size long to Avro's zig-zag encoding for `-1`.
6. Runs the same ingestion path against the crafted file.

This is not an intentionally vulnerable app-level parser. The app uses the public OCF API as documented by the package examples and performs reasonable source and metadata validation before decoding records.

## Result

Command:

```text
go run .
```

Observed output:

```text
valid file decoded records: 3
crafted file decoded records before panic: 0
decoder error after crafted HasNext: <nil>
observed panic: runtime error: makeslice: len out of range
```

The program also demonstrates the same attack as a received in-memory event payload rather than as a file-only scenario:

```text
received event bytes demonstration:
valid OCF event batch length: 452 bytes
malicious field offset: 334
bytes around block header before: 06 c8 01 10 6f 72 64 2d
bytes around block header after:  06 01 01 10 6f 72 64 2d
meaning: record count stays 0x06 (3 records), block size becomes 0x01 (Avro long -1)
received crafted bytes decoded records before panic: 0
received crafted bytes decoder error after HasNext: <nil>
received crafted bytes panic: runtime error: makeslice: len out of range
```

The stack trace reaches:

```text
github.com/hamba/avro/v2/ocf.(*Decoder).readBlock
	.../repos/avro/ocf/ocf.go:209
github.com/hamba/avro/v2/ocf.(*Decoder).HasNext
	.../repos/avro/ocf/ocf.go:157
```

`dec.Error()` is still `<nil>` after the recovered panic, so the issue bypasses the library's normal error contract. A production process that does not recover panics around decoding can be terminated by the crafted input.

## Why The Vulnerability Is Reachable In Real Code

The relevant library flow is:

```go
dec, err := ocf.NewDecoder(reader)
if err != nil {
	return err
}
for dec.HasNext() {
	var event OrderEvent
	if err := dec.Decode(&event); err != nil {
		return err
	}
}
if err := dec.Error(); err != nil {
	return err
}
```

That is the intended OCF API pattern shown by the library examples. `ocf.NewDecoder` validates and parses the OCF header, schema, metadata, and codec. The malicious value is not in the header; it is in the following data block header. The block header is consumed later by `HasNext()`.

In `readBlock`, the library does:

```go
count := d.reader.ReadLong()
size := d.reader.ReadLong()
...
data := make([]byte, size)
```

No check rejects `size < 0` before `make`, so a block size of `-1` causes `runtime error: makeslice: len out of range`.

## Prerequisites

For this to matter in a consuming codebase, all of the following must be true:

1. The application accepts Avro Object Container File bytes from a source that is not fully trusted. Examples include partner feeds, uploads, object storage drops, queues, log ingestion, batch import directories, or network file transfer.
2. The application uses `github.com/hamba/avro/v2/ocf.NewDecoder` and then calls `HasNext()` to iterate records.
3. The attacker can provide a syntactically valid OCF header and a data block header with `count > 0` and `size < 0`.
4. The application does not isolate decoding in a subprocess or recover from panics at the trust boundary.

The attacker does not need to provide a huge file. In this replication the crafted file remains small and passes a 1 MiB application file-size limit because the panic is caused by the encoded negative size, not by actual payload length.

In an event-ingestion context, the payload that matters is a complete Avro OCF byte sequence, such as a batch/message body containing the OCF magic bytes, metadata, schema, sync marker, and then a crafted data block header. The schema and metadata can remain expected. The malicious change is in the block header immediately after the sync marker: keep the block count positive and encode the block size as Avro long `-1` (`0x01`). In the reproduction's generated payload, that changes the block header window from:

```text
06 c8 01 10 6f 72 64 2d
```

to:

```text
06 01 01 10 6f 72 64 2d
```

The first byte, `0x06`, is Avro zig-zag encoding for count `3`. The next byte, changed to `0x01`, is Avro zig-zag encoding for block size `-1`. That is sufficient to make `HasNext()` reach the unchecked `make([]byte, size)` allocation.

## Practical Impact

This is a denial-of-service vulnerability. A crafted OCF file can crash a service or batch worker that decodes untrusted OCF data. Panic recovery can prevent process death, but even with recovery the library still violates the expected decoder behavior because malformed input should become an error returned by `dec.Error()` or `Decode`, not a runtime panic.

The same unchecked allocation site also makes excessive positive block sizes concerning, because `make([]byte, size)` occurs before a policy limit is applied. I did not force a huge positive allocation in this replication because the negative-size case already proves the issue without intentionally destabilizing the host.

## Recommended Fix

Validate OCF block sizes before allocation in `ocf.Decoder.readBlock`:

1. Reject `size < 0`.
2. Add a configurable maximum block-size limit and reject values above it.
3. Return malformed block headers through the decoder error path instead of panicking.
4. Add regression tests for negative block sizes and excessive positive block sizes.
