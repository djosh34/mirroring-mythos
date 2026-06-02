package main

import (
	"bytes"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/hamba/avro/v2/ocf"
)

func main() {
	data, sync, err := validOCF()
	if err != nil {
		fmt.Printf("setup failed: %v\n", err)
		os.Exit(2)
	}

	mutated, err := withNegativeBlockSize(data, sync)
	if err != nil {
		fmt.Printf("mutation failed: %v\n", err)
		os.Exit(2)
	}

	dec, err := ocf.NewDecoder(bytes.NewReader(mutated))
	if err != nil {
		fmt.Printf("NewDecoder unexpectedly rejected header: %v\n", err)
		os.Exit(1)
	}

	panicked := false
	var panicValue any
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				panicValue = r
				debug.PrintStack()
			}
		}()

		_ = dec.HasNext()
	}()

	fmt.Printf("decoder error after HasNext: %v\n", dec.Error())
	if !panicked {
		fmt.Println("expected HasNext to panic for negative OCF block size, but it returned normally")
		os.Exit(1)
	}

	fmt.Printf("observed panic: %v\n", panicValue)
}

func validOCF() ([]byte, [16]byte, error) {
	var sync [16]byte
	copy(sync[:], []byte("0123456789abcdef"))

	var buf bytes.Buffer
	enc, err := ocf.NewEncoder(`"int"`, &buf, ocf.WithSyncBlock(sync))
	if err != nil {
		return nil, sync, err
	}
	if err := enc.Encode(int32(7)); err != nil {
		return nil, sync, err
	}
	if err := enc.Close(); err != nil {
		return nil, sync, err
	}

	return buf.Bytes(), sync, nil
}

func withNegativeBlockSize(data []byte, sync [16]byte) ([]byte, error) {
	syncAt := bytes.Index(data, sync[:])
	if syncAt == -1 {
		return nil, fmt.Errorf("sync marker not found")
	}

	// A one-record int OCF block is:
	// count=1 (0x02), size=1 (0x02), payload int32(7) (0x0e), sync marker.
	block := data[syncAt+len(sync):]
	pattern := append([]byte{0x02, 0x02, 0x0e}, sync[:]...)
	blockAt := bytes.Index(block, pattern)
	if blockAt == -1 {
		return nil, fmt.Errorf("expected generated block pattern not found")
	}

	out := append([]byte(nil), data...)
	sizeOffset := syncAt + len(sync) + blockAt + 1
	out[sizeOffset] = 0x01 // Avro zig-zag long encoding for -1.
	return out, nil
}
