NOT_STARTED


# Audit OCF Block Allocation From Untrusted Input

Review `ocf/ocf.go` around `Decoder.readBlock`.

Focus:
- `count := d.reader.ReadLong()`
- `size := d.reader.ReadLong()`
- `make([]byte, size)` for both normal and skipped blocks

Verify:
- Negative `size` cannot panic or bypass error handling.
- Excessive `size` cannot cause uncontrolled memory allocation.
- `count > 0` with malformed `size` is rejected cleanly.
- `count == 0` with large `size` cannot be used for memory DoS.
- Tests cover zero, negative, max-int, and very large size values.
