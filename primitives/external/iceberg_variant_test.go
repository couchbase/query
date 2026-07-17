//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of the
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package external

import (
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/decimal"
	"github.com/apache/arrow-go/v18/parquet/variant"
	"github.com/google/uuid"
)

// buildTestVariant constructs a variant value equivalent to:
//
//	{
//	  "name": "widget",
//	  "qty": 42,
//	  "price": 12.50,     // decimal128, scale 2
//	  "active": true,
//	  "note": null,
//	  "id": <uuid>,
//	  "raw": []byte{0xDE, 0xAD, 0xBE, 0xEF},
//	  "tags": ["a", "b"]
//	}
func buildTestVariant(t *testing.T, testUUID uuid.UUID) variant.Value {
	t.Helper()

	var b variant.Builder
	start := b.Offset()
	fields := make([]variant.FieldEntry, 0)

	fields = append(fields, b.NextField(start, "name"))
	if err := b.AppendString("widget"); err != nil {
		t.Fatalf("AppendString: %v", err)
	}

	fields = append(fields, b.NextField(start, "qty"))
	if err := b.AppendInt(42); err != nil {
		t.Fatalf("AppendInt: %v", err)
	}

	priceDec, err := decimal.Decimal128FromFloat(12.50, 10, 2)
	if err != nil {
		t.Fatalf("Decimal128FromFloat: %v", err)
	}
	fields = append(fields, b.NextField(start, "price"))
	if err := b.AppendDecimal16(2, priceDec); err != nil {
		t.Fatalf("AppendDecimal16: %v", err)
	}

	fields = append(fields, b.NextField(start, "active"))
	if err := b.AppendBool(true); err != nil {
		t.Fatalf("AppendBool: %v", err)
	}

	fields = append(fields, b.NextField(start, "note"))
	if err := b.AppendNull(); err != nil {
		t.Fatalf("AppendNull: %v", err)
	}

	fields = append(fields, b.NextField(start, "id"))
	if err := b.AppendUUID(testUUID); err != nil {
		t.Fatalf("AppendUUID: %v", err)
	}

	fields = append(fields, b.NextField(start, "raw"))
	if err := b.AppendBinary([]byte{0xDE, 0xAD, 0xBE, 0xEF}); err != nil {
		t.Fatalf("AppendBinary: %v", err)
	}

	fields = append(fields, b.NextField(start, "tags"))
	arrStart, offsets := b.Offset(), make([]int, 0)
	offsets = append(offsets, b.NextElement(arrStart))
	if err := b.AppendString("a"); err != nil {
		t.Fatalf("AppendString: %v", err)
	}
	offsets = append(offsets, b.NextElement(arrStart))
	if err := b.AppendString("b"); err != nil {
		t.Fatalf("AppendString: %v", err)
	}
	if err := b.FinishArray(arrStart, offsets); err != nil {
		t.Fatalf("FinishArray: %v", err)
	}

	if err := b.FinishObject(start, fields); err != nil {
		t.Fatalf("FinishObject: %v", err)
	}

	vv, err := b.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return vv
}

func TestDecodeVariantScalar(t *testing.T) {
	testUUID := uuid.MustParse("12345678-1234-5678-1234-567812345678")
	vv := buildTestVariant(t, testUUID)

	t.Run("decimalToDouble=false", func(t *testing.T) {
		decoded := decodeVariantScalar(vv, false)
		obj, ok := decoded.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map[string]interface{}, got %T", decoded)
		}

		if obj["name"] != "widget" {
			t.Errorf("name: got %v (%T), want %q", obj["name"], obj["name"], "widget")
		}
		if obj["qty"] != int8(42) {
			t.Errorf("qty: got %v (%T), want int8(42)", obj["qty"], obj["qty"])
		}
		if obj["price"] != "12.50" {
			t.Errorf("price: got %v (%T), want %q", obj["price"], obj["price"], "12.50")
		}
		if obj["active"] != true {
			t.Errorf("active: got %v (%T), want true", obj["active"], obj["active"])
		}
		if obj["note"] != nil {
			t.Errorf("note: got %v, want nil", obj["note"])
		}
		if obj["id"] != testUUID.String() {
			t.Errorf("id: got %v, want %q", obj["id"], testUUID.String())
		}
		rawBytes, ok := obj["raw"].([]byte)
		if !ok || string(rawBytes) != string([]byte{0xDE, 0xAD, 0xBE, 0xEF}) {
			t.Errorf("raw: got %v (%T), want []byte{0xDE, 0xAD, 0xBE, 0xEF}", obj["raw"], obj["raw"])
		}
		tags, ok := obj["tags"].([]interface{})
		if !ok || len(tags) != 2 || tags[0] != "a" || tags[1] != "b" {
			t.Errorf("tags: got %v (%T), want [a b]", obj["tags"], obj["tags"])
		}
	})

	t.Run("decimalToDouble=true", func(t *testing.T) {
		decoded := decodeVariantScalar(vv, true)
		obj := decoded.(map[string]interface{})
		price, ok := obj["price"].(float64)
		if !ok || price != 12.50 {
			t.Errorf("price: got %v (%T), want float64(12.50)", obj["price"], obj["price"])
		}
	})
}

func TestDecodeVariantScalarTemporalTypes(t *testing.T) {
	var b variant.Builder
	start := b.Offset()
	fields := make([]variant.FieldEntry, 0)

	fields = append(fields, b.NextField(start, "day"))
	if err := b.AppendDate(arrow.Date32(19723)); err != nil { // 2023-12-01
		t.Fatalf("AppendDate: %v", err)
	}

	fields = append(fields, b.NextField(start, "created"))
	if err := b.AppendTimestamp(arrow.Timestamp(1701388800000000), true, true); err != nil {
		t.Fatalf("AppendTimestamp: %v", err)
	}

	if err := b.FinishObject(start, fields); err != nil {
		t.Fatalf("FinishObject: %v", err)
	}

	vv, err := b.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	decoded := decodeVariantScalar(vv, false)
	obj, ok := decoded.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", decoded)
	}

	if obj["day"] != int32(19723) {
		t.Errorf("day: got %v (%T), want int32(19723)", obj["day"], obj["day"])
	}
	if obj["created"] != int64(1701388800000000) {
		t.Errorf("created: got %v (%T), want int64(1701388800000000)", obj["created"], obj["created"])
	}
}
