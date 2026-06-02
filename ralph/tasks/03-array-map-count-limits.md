NOT_STARTED


# Audit Array And Map Count Limits

Review `codec_array.go`, `codec_map.go`, and `reader.go`.

Focus:
- `Reader.ReadBlockHeader`
- `arrayDecoder.Decode`
- `mapDecoder.Decode`
- `mapDecoderUnmarshaler.Decode`

Verify:
- Array block counts cannot overflow `int` during `size += int(l)`.
- `Config.MaxSliceAllocSize` is consistently enforced for arrays.
- Maps have an equivalent limit for total decoded entries or allocation growth.
- Negative block counts and block sizes are handled according to the Avro spec.
- Tests cover large arrays, large maps, negative block headers, and repeated blocks.
