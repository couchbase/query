//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of the
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package external

import (
	"time"

	"github.com/apache/arrow-go/v18/arrow"
)

// This file renders Iceberg TIMESTAMP/TIMESTAMPTZ/DATE/TIME column values as
// ISO-8601-style strings instead of raw epoch integers, and provides the
// exact inverse so the row-level filter (iceberg_row_filter.go) can compare
// an already-formatted row value against a bound iceberg.Literal without
// losing precision.
//
// The formats are deliberately NOT arbitrary: they are exactly what
// iceberg-go's own literal-from-string binding (literals.go,
// StringLiteral.To) parses back, at full source precision, with no
// millisecond-style truncation anywhere in the round trip. That parity is
// the whole point — see MB-73900 for what happens when a query engine's
// display/comparison precision silently diverges from the precision Iceberg
// itself uses to prune data files and Parquet row groups: a stricter pushed
// predicate than the one actually evaluated silently drops matching rows.
//
//	iceberg.TimestampType   (no zone):  "2006-01-02T15:04:05.999999"       — StringLiteral.To parses via time.Parse("2006-01-02T15:04:05", s)
//	iceberg.TimestampTzType (has zone): "2006-01-02T15:04:05.999999Z07:00" — parses via time.Parse(time.RFC3339, s)
//	iceberg.DateType:                   "2006-01-02"                      — parses via time.Parse("2006-01-02", s)
//	iceberg.TimeType:                   "15:04:05.999999"                 — parses via arrow.Time64FromString(s, arrow.Microsecond)
//
// Go's time.Parse accepts a fractional-second suffix even when the layout
// doesn't declare one, so a full 6-digit (or 9-digit, for the _ns variants)
// fraction round-trips exactly regardless of which of the layouts above
// iceberg-go uses to parse a value back.
//
// timestamp_ns/timestamptz_ns (Iceberg v3) values are formatted the same way
// at nanosecond precision, for display and for our own row-filter
// comparison. iceberg-go's StringLiteral.To has no case at all for the _ns
// types, so a literal typed against one of these columns cannot itself be
// pushed down as a string today — a separate, pre-existing iceberg-go
// limitation this file does not attempt to work around.

// icebergUTCAliases mirrors iceberg-go's table/arrow_utils.go utcAliases: the
// set of Arrow TimeZone strings iceberg-go treats as UTC when deciding
// whether an Arrow *TimestampType round-trips to a zoned (TimestampTzType) or
// unzoned (TimestampType) Iceberg logical type.
var icebergUTCAliases = map[string]bool{"UTC": true, "+00:00": true, "Etc/UTC": true, "Z": true}

func isIcebergUTCAlias(tz string) bool {
	return icebergUTCAliases[tz]
}

// scaleToMicros converts an integer value in the given Arrow time unit to microseconds.
func scaleToMicros(v int64, unit arrow.TimeUnit) int64 {
	switch unit {
	case arrow.Second:
		return v * 1_000_000
	case arrow.Millisecond:
		return v * 1_000
	case arrow.Nanosecond:
		return v / 1_000
	default: // arrow.Microsecond
		return v
	}
}

// formatIcebergTimestamp renders a microsecond epoch value as the canonical
// TIMESTAMP/TIMESTAMPTZ string described above.
func formatIcebergTimestamp(micros int64, hasTZ bool) string {
	t := time.UnixMicro(micros).UTC()
	if hasTZ {
		return t.Format("2006-01-02T15:04:05.999999Z07:00")
	}
	return t.Format("2006-01-02T15:04:05.999999")
}

// formatIcebergTimestampNano is the nanosecond (timestamp_ns/timestamptz_ns) twin.
func formatIcebergTimestampNano(nanos int64, hasTZ bool) string {
	t := time.Unix(0, nanos).UTC()
	if hasTZ {
		return t.Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	return t.Format("2006-01-02T15:04:05.999999999")
}

// parseIcebergTimestampTime reverses formatIcebergTimestamp/formatIcebergTimestampNano.
// Used by the row-level filter to compare an already-formatted row value
// against a bound iceberg.Timestamp/TimestampNano literal without
// re-truncating precision. nano selects the returned unit, independent of
// how many fractional digits the string actually carries.
func parseIcebergTimestampTime(s string, nano bool) (int64, bool) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999999"} {
		if t, err := time.Parse(layout, s); err == nil {
			if nano {
				return t.UTC().UnixNano(), true
			}
			return t.UTC().UnixMicro(), true
		}
	}
	return 0, false
}

// formatIcebergDate renders an Iceberg DATE value (epoch-day count) as "YYYY-MM-DD".
func formatIcebergDate(days int32) string {
	return time.Unix(int64(days)*86400, 0).UTC().Format("2006-01-02")
}

// parseIcebergDate reverses formatIcebergDate into an epoch-day count.
func parseIcebergDate(s string) (int32, bool) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return 0, false
	}
	return int32(t.UTC().Unix() / 86400), true
}

// formatIcebergTime renders an Iceberg TIME-of-day value (microseconds since
// midnight) as "HH:MM:SS.ffffff".
func formatIcebergTime(micros int64) string {
	t := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(micros) * time.Microsecond)
	return t.Format("15:04:05.999999")
}

// parseIcebergTime reverses formatIcebergTime into microseconds since midnight.
func parseIcebergTime(s string) (int64, bool) {
	for _, layout := range []string{"15:04:05.999999999", "15:04"} {
		if t, err := time.Parse(layout, s); err == nil {
			// Read the clock components directly rather than subtracting a
			// fixed midnight reference: time.Parse defaults the date portion
			// of a time-only layout to year 0, not the Unix epoch, so a
			// Sub() against a 1970-01-01 reference silently overflows.
			h, m, sec := t.Clock()
			micros := int64(h)*3_600_000_000 + int64(m)*60_000_000 + int64(sec)*1_000_000 + int64(t.Nanosecond())/1_000
			return micros, true
		}
	}
	return 0, false
}
