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
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/iceberg-go"
)

// TestFormatIcebergTimestampMatchesIcebergGoParsing verifies that
// formatIcebergTimestamp/formatIcebergTimestampNano produce strings that
// iceberg-go's own literal binding can parse back to the exact same
// microsecond/nanosecond value -- the whole point being to never repeat
// MB-73900 (a query engine silently comparing at coarser precision than
// Iceberg uses to prune files).
func TestFormatIcebergTimestampMatchesIcebergGoParsing(t *testing.T) {
	cases := []struct {
		name   string
		micros int64
		hasTZ  bool
		typ    iceberg.Type
	}{
		{"no-zone whole second", 1767268800000000, false, iceberg.PrimitiveTypes.Timestamp},
		{"no-zone sub-millisecond", 1767268800123456, false, iceberg.PrimitiveTypes.Timestamp},
		{"zoned sub-millisecond", 1767268800123456, true, iceberg.PrimitiveTypes.TimestampTz},
		{"zoned whole second", 1767268800000000, true, iceberg.PrimitiveTypes.TimestampTz},
		{"pre-1970", -1000, true, iceberg.PrimitiveTypes.TimestampTz},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := formatIcebergTimestamp(c.micros, c.hasTZ)
			lit, err := iceberg.StringLiteral(s).To(c.typ)
			if err != nil {
				t.Fatalf("iceberg-go rejected formatted timestamp %q: %v", s, err)
			}
			got := int64(lit.(iceberg.TimestampLiteral).Value())
			if got != c.micros {
				t.Errorf("round trip mismatch: formatted %q, iceberg-go parsed back %d, want %d", s, got, c.micros)
			}
		})
	}
}

func TestFormatIcebergDateMatchesIcebergGoParsing(t *testing.T) {
	cases := []int32{0, 1, 20454, -1}
	for _, days := range cases {
		s := formatIcebergDate(days)
		lit, err := iceberg.StringLiteral(s).To(iceberg.PrimitiveTypes.Date)
		if err != nil {
			t.Fatalf("iceberg-go rejected formatted date %q: %v", s, err)
		}
		got := int32(lit.(iceberg.DateLiteral).Value())
		if got != days {
			t.Errorf("round trip mismatch: formatted %q, iceberg-go parsed back %d, want %d", s, got, days)
		}
	}
}

func TestFormatIcebergTimeMatchesIcebergGoParsing(t *testing.T) {
	cases := []int64{0, 1, 43200000000, 86399999999}
	for _, micros := range cases {
		s := formatIcebergTime(micros)
		lit, err := iceberg.StringLiteral(s).To(iceberg.PrimitiveTypes.Time)
		if err != nil {
			t.Fatalf("iceberg-go rejected formatted time %q: %v", s, err)
		}
		got := int64(lit.(iceberg.TimeLiteral).Value())
		if got != micros {
			t.Errorf("round trip mismatch: formatted %q, iceberg-go parsed back %d, want %d", s, got, micros)
		}
	}
}

// TestParseIcebergTemporalRoundTrip checks our own reverse parsers (used by
// compareValues, not iceberg-go) invert the format functions exactly.
func TestParseIcebergTemporalRoundTrip(t *testing.T) {
	t.Run("timestamp micros no zone", func(t *testing.T) {
		want := int64(1767268800123456)
		s := formatIcebergTimestamp(want, false)
		got, ok := parseIcebergTimestampTime(s, false)
		if !ok || got != want {
			t.Errorf("parseIcebergTimestampTime(%q) = %d, %v; want %d, true", s, got, ok, want)
		}
	})
	t.Run("timestamp micros zoned", func(t *testing.T) {
		want := int64(1767268800123456)
		s := formatIcebergTimestamp(want, true)
		got, ok := parseIcebergTimestampTime(s, false)
		if !ok || got != want {
			t.Errorf("parseIcebergTimestampTime(%q) = %d, %v; want %d, true", s, got, ok, want)
		}
	})
	t.Run("timestamp nanos", func(t *testing.T) {
		want := int64(1767268800123456789)
		s := formatIcebergTimestampNano(want, true)
		got, ok := parseIcebergTimestampTime(s, true)
		if !ok || got != want {
			t.Errorf("parseIcebergTimestampTime(%q, nano) = %d, %v; want %d, true", s, got, ok, want)
		}
	})
	t.Run("date", func(t *testing.T) {
		want := int32(20454)
		s := formatIcebergDate(want)
		got, ok := parseIcebergDate(s)
		if !ok || got != want {
			t.Errorf("parseIcebergDate(%q) = %d, %v; want %d, true", s, got, ok, want)
		}
	})
	t.Run("time", func(t *testing.T) {
		want := int64(43200123456)
		s := formatIcebergTime(want)
		got, ok := parseIcebergTime(s)
		if !ok || got != want {
			t.Errorf("parseIcebergTime(%q) = %d, %v; want %d, true", s, got, ok, want)
		}
	})
	t.Run("bad input", func(t *testing.T) {
		if _, ok := parseIcebergTimestampTime("not a timestamp", false); ok {
			t.Error("expected ok=false for garbage input")
		}
		if _, ok := parseIcebergDate("not a date"); ok {
			t.Error("expected ok=false for garbage input")
		}
		if _, ok := parseIcebergTime("not a time"); ok {
			t.Error("expected ok=false for garbage input")
		}
	})
}

// TestScaleToMicros checks the unit-conversion helper across every Arrow time unit.
func TestScaleToMicros(t *testing.T) {
	cases := []struct {
		v    int64
		unit arrow.TimeUnit
		want int64
	}{
		{1, arrow.Second, 1_000_000},
		{1, arrow.Millisecond, 1_000},
		{1, arrow.Microsecond, 1},
		{1000, arrow.Nanosecond, 1},
	}
	for _, c := range cases {
		if got := scaleToMicros(c.v, c.unit); got != c.want {
			t.Errorf("scaleToMicros(%d, %v) = %d, want %d", c.v, c.unit, got, c.want)
		}
	}
}

// TestCompareValuesTemporalStringReversal is the regression test for the bug
// this file exists to avoid: once getColumnValue/convertArrowValue render
// TIMESTAMP/DATE/TIME row values as strings, compareValues (the row-level,
// post-scan filter used by buildRowMatcher) must still compare them against
// a bound iceberg.Literal at full precision -- not silently fail to match, and
// not truncate to a coarser unit than the file/row-group pruning already used.
func TestCompareValuesTemporalStringReversal(t *testing.T) {
	t.Run("timestamp equal, sub-millisecond, row is minimum of its group", func(t *testing.T) {
		// The exact MB-73900 shape: the stored value is the row-group/file
		// minimum, carrying sub-millisecond microseconds. If this engine ever
		// went through a lossy millisecond-only representation, the row would
		// become unreachable by equality exactly as described in that ticket.
		micros := int64(1767268800123456)
		rowVal := formatIcebergTimestamp(micros, true)
		lit := iceberg.TimestampLiteral(iceberg.Timestamp(micros))
		c, ok := compareToLiteral(rowVal, lit)
		if !ok || c != 0 {
			t.Fatalf("compareToLiteral(%q, %d) = %d, %v; want 0, true", rowVal, micros, c, ok)
		}
	})
	t.Run("timestamp less-than at microsecond boundary", func(t *testing.T) {
		rowMicros := int64(1767268800123456)
		litMicros := int64(1767268800123457) // 1us later
		rowVal := formatIcebergTimestamp(rowMicros, true)
		lit := iceberg.TimestampLiteral(iceberg.Timestamp(litMicros))
		c, ok := compareToLiteral(rowVal, lit)
		if !ok || c != -1 {
			t.Fatalf("compareToLiteral(%q, %d) = %d, %v; want -1, true", rowVal, litMicros, c, ok)
		}
	})
	t.Run("timestamp_ns literal, previously unhandled", func(t *testing.T) {
		nanos := int64(1767268800123456789)
		rowVal := formatIcebergTimestampNano(nanos, true)
		lit := iceberg.TimestampNsLiteral(iceberg.TimestampNano(nanos))
		c, ok := compareToLiteral(rowVal, lit)
		if !ok || c != 0 {
			t.Fatalf("compareToLiteral(%q, %d) = %d, %v; want 0, true (iceberg.TimestampNano case)", rowVal, nanos, c, ok)
		}
	})
	t.Run("date equal", func(t *testing.T) {
		days := int32(20454)
		rowVal := formatIcebergDate(days)
		lit := iceberg.DateLiteral(iceberg.Date(days))
		c, ok := compareToLiteral(rowVal, lit)
		if !ok || c != 0 {
			t.Fatalf("compareToLiteral(%q, %d) = %d, %v; want 0, true", rowVal, days, c, ok)
		}
	})
	t.Run("time equal", func(t *testing.T) {
		micros := int64(43200123456)
		rowVal := formatIcebergTime(micros)
		lit := iceberg.TimeLiteral(iceberg.Time(micros))
		c, ok := compareToLiteral(rowVal, lit)
		if !ok || c != 0 {
			t.Fatalf("compareToLiteral(%q, %d) = %d, %v; want 0, true", rowVal, micros, c, ok)
		}
	})
	t.Run("garbage row value does not panic, reports not comparable", func(t *testing.T) {
		lit := iceberg.TimestampLiteral(iceberg.Timestamp(0))
		if _, ok := compareToLiteral("definitely not a timestamp", lit); ok {
			t.Error("expected ok=false for an unparseable row value")
		}
	})
}

// TestReaderTemporalToStringToggle exercises Reader.getColumnValue directly
// against real Arrow arrays (not just the standalone format functions above)
// to confirm the temporalToString flag actually gates the behavior:
// default false preserves the pre-existing raw-int output exactly, and true
// switches to the ISO-8601 string.
func TestReaderTemporalToStringToggle(t *testing.T) {
	mem := memory.NewGoAllocator()

	t.Run("Timestamp (zoned, microsecond)", func(t *testing.T) {
		dt := &arrow.TimestampType{Unit: arrow.Microsecond, TimeZone: "UTC"}
		b := array.NewTimestampBuilder(mem, dt)
		defer b.Release()
		const micros = int64(1767268800123456)
		b.Append(arrow.Timestamp(micros))
		arr := b.NewArray()
		defer arr.Release()

		off := &Reader{}
		if got := off.getColumnValue(arr, 0); got != micros {
			t.Errorf("default (off): got %v (%T), want raw int64 %d", got, got, micros)
		}

		on := &Reader{temporalToString: true}
		want := formatIcebergTimestamp(micros, true)
		if got := on.getColumnValue(arr, 0); got != want {
			t.Errorf("on: got %v, want %q", got, want)
		}
	})

	t.Run("Date32", func(t *testing.T) {
		b := array.NewDate32Builder(mem)
		defer b.Release()
		const days = int32(20454)
		b.Append(arrow.Date32(days))
		arr := b.NewArray()
		defer arr.Release()

		off := &Reader{}
		if got := off.getColumnValue(arr, 0); got != days {
			t.Errorf("default (off): got %v (%T), want raw int32 %d", got, got, days)
		}

		on := &Reader{temporalToString: true}
		want := formatIcebergDate(days)
		if got := on.getColumnValue(arr, 0); got != want {
			t.Errorf("on: got %v, want %q", got, want)
		}
	})

	t.Run("Time64 (microsecond)", func(t *testing.T) {
		dt := &arrow.Time64Type{Unit: arrow.Microsecond}
		b := array.NewTime64Builder(mem, dt)
		defer b.Release()
		const micros = int64(43200123456)
		b.Append(arrow.Time64(micros))
		arr := b.NewArray()
		defer arr.Release()

		off := &Reader{}
		if got := off.getColumnValue(arr, 0); got != micros {
			t.Errorf("default (off): got %v (%T), want raw int64 %d", got, got, micros)
		}

		on := &Reader{temporalToString: true}
		want := formatIcebergTime(micros)
		if got := on.getColumnValue(arr, 0); got != want {
			t.Errorf("on: got %v, want %q", got, want)
		}
	})
}
