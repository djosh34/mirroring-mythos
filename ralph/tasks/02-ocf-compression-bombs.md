NOT_STARTED


# Audit OCF Compression Bomb Handling

Review `ocf/codec.go`.

Focus:
- `DeflateCodec.Decode` using `io.ReadAll`
- `SnappyCodec.Decode` using `snappy.Decode`
- `ZStandardCodec.Decode` using `DecodeAll`

Verify:
- Compressed blocks have an enforced maximum decoded size.
- Small compressed inputs cannot expand into unbounded memory use.
- Existing `Config.MaxByteSliceSize` limits are not bypassed before Avro value decoding.
- Corrupt compressed data returns errors without panics.
- Tests cover deflate, snappy, and zstd decompression bombs.
