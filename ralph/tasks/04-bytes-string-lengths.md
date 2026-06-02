NOT_STARTED


# Audit Bytes And String Length Handling

Review `reader.go` around `Reader.readBytes`, `ReadBytes`, and `ReadString`.

Focus:
- `size := int(r.ReadLong())`
- negative length handling
- `Config.MaxByteSliceSize`
- 32-bit platform behavior

Verify:
- `int64` to `int` conversion cannot wrap on 32-bit builds.
- Negative lengths are rejected before allocation.
- Very large lengths are rejected before allocation.
- `MaxByteSliceSize` cannot be disabled accidentally in production configurations.
- Tests cover boundary values around 0, 1, max configured size, and oversized values.
