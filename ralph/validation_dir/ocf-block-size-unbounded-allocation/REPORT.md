CHECKED

# OCF block size allows panic before validation

## Result

The vulnerability is real against the current checkout in `repos/avro`.

Calling the normal public OCF decoding API on a crafted container file can panic inside `Decoder.HasNext()` before the library reports an error. The reproduced panic is:

```text
runtime error: makeslice: len out of range
```

The stack reaches:

```text
github.com/hamba/avro/v2/ocf.(*Decoder).readBlock
	.../repos/avro/ocf/ocf.go:209
github.com/hamba/avro/v2/ocf.(*Decoder).HasNext
	.../repos/avro/ocf/ocf.go:157
```

`dec.Error()` was still `<nil>` after the panic was recovered, so this is not converted into the library's normal error path.

## Reproduction

The reproduction in this directory is a standalone Go program using the library as an application would:

```text
go run .
```

The module uses:

```text
replace github.com/hamba/avro/v2 => ../../../repos/avro
```

so it exercises the local repository under review.

The program performs these steps:

1. Creates a valid Avro Object Container File through `ocf.NewEncoder`.
2. Uses a fixed sync marker so the generated block can be located deterministically.
3. Encodes one normal `int32` record.
4. Mutates only the OCF block-size field from Avro long `1` (`0x02`) to Avro long `-1` (`0x01`).
5. Creates an OCF decoder through `ocf.NewDecoder`.
6. Calls `dec.HasNext()`.
7. Recovers and prints the panic.

Observed command output:

```text
decoder error after HasNext: <nil>
observed panic: runtime error: makeslice: len out of range
```

The stack trace printed by the program confirms the panic occurs at `repos/avro/ocf/ocf.go:209`, where `readBlock` does:

```go
data := make([]byte, size)
```

with attacker-controlled `size == -1`.

## Prerequisites for exploitability

The consuming codebase must decode Avro Object Container Files from data that is not fully trusted. Examples include uploaded files, partner feeds, queues, object storage, log ingestion, or any network path that can supply OCF bytes.

The application must call the normal OCF read flow:

```go
dec, err := ocf.NewDecoder(reader)
if err != nil {
	return err
}
for dec.HasNext() {
	// decode values
}
```

The attacker only needs to provide a syntactically valid OCF header and a malicious block header where `count > 0` and `size < 0`. The file does not need to contain a large payload for the negative-size panic case.

This reproduction verifies the negative block-size panic. The same code path also accepts large positive `size` values before allocation, so the memory-exhaustion concern is credible, but I did not force a huge allocation in this validation to avoid intentionally destabilizing the process.

## Security impact

This is a denial-of-service issue. A crafted OCF file can terminate a process if the application does not recover from panics around decoding. Even with panic recovery, the library violates the expected contract by panicking instead of returning a decoder error.

## Recommended fix

Validate OCF block headers before allocating:

- Reject `size < 0`.
- Reject unreasonable positive block sizes using an OCF block-size limit.
- Avoid allocating skipped blocks whole when `count == 0`.
- Add regression tests for negative sizes and excessive sizes in OCF block headers.
