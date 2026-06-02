NOT_STARTED

Title: Map decoder has no maximum entry count or allocation budget

Impact: High. An attacker-controlled Avro map can force unbounded key/value allocations and CPU work. This affects ordinary map fields and OCF header metadata because OCF headers decode `meta` as `map[string][]byte`.

Attack path:
- Attacker sends Avro data for a schema containing a map, or an OCF file with a metadata map.
- `mapDecoder.Decode()` reads each block header count from the stream.
- For each entry it allocates a key string, allocates a new element, decodes the value, and inserts into the Go map.
- There is no equivalent to `Config.MaxSliceAllocSize` for maps and no total entry cap.

Evidence:
- `codec_map.go`, `mapDecoder.Decode`, loops `for range l` for every block count.
- `codec_map.go`, `mapDecoderUnmarshaler.Decode`, has the same unbounded entry loop.
- `config.go` exposes `MaxSliceAllocSize`, but that is only enforced in `codec_array.go`.
- OCF `Header.Meta` is `map[string][]byte`, so this can be reached before the user has decoded any records.

Exploit sketch:
- Provide a map block header with a very large positive count.
- Encode many short keys and tiny values to keep the payload compact relative to map overhead.
- The decoder repeatedly allocates Go map entries until memory or CPU is exhausted.

Recommended remediation:
- Add a `MaxMapAllocSize` or shared collection allocation budget.
- Track cumulative map entries across blocks and reject excessive values.
- Apply the same bound to OCF header metadata parsing.
